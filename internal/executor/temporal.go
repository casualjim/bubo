package executor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/casualjim/bubo/agent"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/broker"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/provider/models"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type TemporalProxy struct {
	client client.Client
	broker broker.Broker
}

func (t *TemporalProxy) Run(ctx context.Context, cmd RunCommand, promise Promise) error {
	if err := cmd.Validate(); err != nil {
		promise.Error(err)
		return err
	}

	params := runParams{
		runID:  cmd.ID(),
		turnID: cmd.Thread.ID(),
		agent:  cmd.Agent,
		hook:   cmd.Hook,
	}
	if err := t.validateAgentAndProvider(ctx, &params); err != nil {
		promise.Error(err)
		return err
	}

	topic := t.broker.Topic(ctx, cmd.id.String())
	sub, err := topic.Subscribe(ctx, cmd.Hook)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	fut, err := t.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s-%s", cmd.Agent.Name(), cmd.id),
		TaskQueue: "agent-" + nameAsID(cmd.Agent.Name()),
	}, RemoteRunCommandFromRunCommand(cmd))
	if err != nil {
		promise.Error(err)
		return err
	}

	var result RemoteRunResult
	err = fut.Get(ctx, &result)
	if err != nil {
		promise.Error(err)
		return err
	}

	result.Checkpoint.MergeInto(cmd.Thread)
	promise.Complete(result.Result)
	return nil
}

func (t *TemporalProxy) validateAgentAndProvider(ctx context.Context, params *runParams) error {
	model := params.agent.Model()
	if model == nil {
		err := fmt.Errorf("agent model cannot be nil")
		t.publishError(ctx, params, err)
		return err
	}

	prov := model.Provider()
	if prov == nil {
		err := fmt.Errorf("model provider cannot be nil")
		t.publishError(ctx, params, err)
		return err
	}

	return nil
}

type runParams struct {
	runID  uuid.UUID
	turnID uuid.UUID
	agent  api.Agent
	hook   events.Hook
}

func (t *TemporalProxy) publishError(ctx context.Context, params *runParams, err error) {
	if ee, hasErr := wrapErr(params.runID, params.turnID, params.agent.Name(), err); hasErr {
		params.hook.OnError(ctx, ee)
	}
}

type Temporal struct {
	broker broker.Broker
}

type RemoteRunCommand struct {
	ID               uuid.UUID                  `json:"id"`
	Agent            RemoteAgent                `json:"agent"`
	StructuredOutput *provider.StructuredOutput `json:"structured_output,omitempty"`
	Stream           bool                       `json:"stream"`
	MaxTurns         int                        `json:"max_turns"`
	ContextVariables types.ContextVars          `json:"context_variables,omitempty"`
	Checkpoint       shorttermmemory.Checkpoint `json:"checkpoint"`
}

type RemoteAgent struct {
	Name              string `json:"name"`
	Model             string `json:"model"`
	Instructions      string `json:"instructions"`
	ParallelToolCalls bool   `json:"parallelToolCalls"`
}

// RenderInstructions renders the agent's instructions with the provided context variables.
func (a *RemoteAgent) RenderInstructions(cv types.ContextVars) (string, error) {
	if !strings.Contains(a.Instructions, "{{") {
		return a.Instructions, nil
	}
	return renderTemplate("instructions", a.Instructions, cv)
}

func renderTemplate(name, templateStr string, cv types.ContextVars) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, cv); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type RemoteRunResultType uint8

const (
	RemoteRunResultTypeIncomplete RemoteRunResultType = iota
	RemoteRunResultTypeCompletion
	RemoteRunResultTypeToolCall
)

type RemoteRunResult struct {
	ID               uuid.UUID                  `json:"id"`
	Checkpoint       shorttermmemory.Checkpoint `json:"checkpoint"`
	Result           string                     `json:"result"`
	Type             RemoteRunResultType        `json:"type"`
	ToolCalls        *messages.ToolCallMessage  `json:"tool_calls,omitempty"`
	ContextVariables types.ContextVars          `json:"context_variables,omitempty"`
}

func RemoteRunCommandFromRunCommand(cmd RunCommand) RemoteRunCommand {
	return RemoteRunCommand{
		ID: cmd.id,
		Agent: RemoteAgent{
			Name:              cmd.Agent.Name(),
			Model:             cmd.Agent.Model().Name(),
			Instructions:      cmd.Agent.Instructions(),
			ParallelToolCalls: cmd.Agent.ParallelToolCalls(),
		},
		StructuredOutput: cmd.StructuredOutput,
		Stream:           cmd.Stream,
		MaxTurns:         cmd.MaxTurns,
		ContextVariables: cmd.ContextVariables,
	}
}

func (t *Temporal) RunChildWorkflow(ctx workflow.Context, cmd RemoteRunCommand) (string, error) {
	return t.Run(ctx, cmd)
}

func (t *Temporal) Run(ctx workflow.Context, cmd RemoteRunCommand) (string, error) {
	mem := shorttermmemory.New()
	cmd.Checkpoint.MergeInto(mem)

	ctxVars := maps.Clone(cmd.ContextVariables)
	if ctxVars == nil {
		ctxVars = make(types.ContextVars)
	}

	activeAgent := cmd.Agent

	remainingTurns := cmd.MaxTurns - mem.TurnLen()
	for remainingTurns > 0 {
		remainingTurns--
		res, err := t.runCompletionActivity(ctx, completionParams{
			RunID:            cmd.ID,
			Agent:            activeAgent,
			Checkpoint:       mem.Checkpoint(),
			ContextVariables: ctxVars,
			StructuredOutput: cmd.StructuredOutput,
			Stream:           cmd.Stream,
		})
		if err != nil {
			var continueErr *continueError
			if errors.As(err, &continueErr) {
				continue // Agent transfer occurred
			}
			return "", err
		}

		switch res.Type {
		case RemoteRunResultTypeCompletion:
			res.Checkpoint.MergeInto(mem)
			return res.Result, nil
		case RemoteRunResultTypeToolCall:
			if res.ToolCalls == nil {
				continue
			}
			// Handle each tool call as a separate activity
			for _, call := range res.ToolCalls.ToolCalls {
				toolResult, err := t.runToolCallActivity(ctx, remoteToolCallParams{
					RunID:    cmd.ID,
					TurnID:   mem.ID(),
					Agent:    activeAgent,
					ToolCall: call,
					CtxVars:  ctxVars,
				})

				// Update context variables from tool result
				if toolResult.CtxVars != nil {
					maps.Copy(ctxVars, toolResult.CtxVars)
				}
				if err != nil {
					return "", err
				}

				// Handle potential agent transfer
				if toolResult.Agent != nil {
					// Start a child workflow for the transferred agent
					cwo := workflow.ChildWorkflowOptions{
						WorkflowID: fmt.Sprintf("%s-child-%s", cmd.ID, toolResult.Agent.Name),
						TaskQueue:  "agent-" + nameAsID(toolResult.Agent.Name),
					}
					ctx = workflow.WithChildOptions(ctx, cwo)

					var childResult string
					childFuture := workflow.ExecuteChildWorkflow(ctx, t.RunChildWorkflow, RemoteRunCommand{
						ID:               cmd.ID,
						Agent:            *toolResult.Agent,
						StructuredOutput: cmd.StructuredOutput,
						Stream:           cmd.Stream,
						MaxTurns:         remainingTurns,
						ContextVariables: ctxVars,
						Checkpoint:       mem.Checkpoint(),
					})

					if err := childFuture.Get(ctx, &childResult); err != nil {
						return "", fmt.Errorf("child workflow failed: %w", err)
					}
					continue
				}

				// Add tool response to memory
				if toolResult.Message != nil {
					mem.AddToolResponse(*toolResult.Message)
				}
			}
		}
	}

	// Publish max turns error through activity
	cctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    10 * time.Second, // Error publishing should be quick
		ScheduleToStartTimeout: 5 * time.Second,  // Short scheduling timeout
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    100 * time.Millisecond,
			MaximumInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5, // More retries for error handling
		},
	})
	if err := workflow.ExecuteActivity(cctx, t.PublishError, completionParams{
		RunID:      cmd.ID,
		Agent:      activeAgent,
		Checkpoint: mem.Checkpoint(),
	}, "max turns reached").Get(ctx, nil); err != nil {
		return "", fmt.Errorf("failed to publish max turns error: %w", err)
	}
	return "", errors.New("max turns reached")
}

func (t *Temporal) runCompletionActivity(ctx workflow.Context, cmd completionParams) (RemoteRunResult, error) {
	log := workflow.GetLogger(ctx)
	log.Info("running completion", "agent", cmd.Agent.Name)
	cctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 30 * time.Second, // Allow time for worker pickup
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			MaximumInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	})

	var result RemoteRunResult
	if err := workflow.ExecuteActivity(cctx, t.RunCompletion, cmd).Get(ctx, &result); err != nil {
		return RemoteRunResult{}, err
	}
	return result, nil
}

type remoteToolCallParams struct {
	RunID    uuid.UUID
	TurnID   uuid.UUID
	Agent    RemoteAgent
	ToolCall messages.ToolCallData
	CtxVars  types.ContextVars
}

type remoteToolCallResult struct {
	Message *messages.Message[messages.ToolResponse] `json:"message,omitempty"`
	Agent   *RemoteAgent                             `json:"agent,omitempty"`
	CtxVars types.ContextVars                        `json:"context_variables,omitempty"`
}

func (t *Temporal) runToolCallActivity(ctx workflow.Context, toolCall remoteToolCallParams) (remoteToolCallResult, error) {
	cctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    1 * time.Minute,  // Most tool calls should complete faster
		ScheduleToStartTimeout: 10 * time.Second, // Allow reasonable time for worker pickup
		HeartbeatTimeout:       10 * time.Second, // For long-running tools
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			MaximumInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	})

	var result remoteToolCallResult
	if err := workflow.ExecuteActivity(cctx, t.CallTool, toolCall).Get(ctx, &result); err != nil {
		return remoteToolCallResult{}, err
	}
	return result, nil
}

type completionParams struct {
	RunID            uuid.UUID                  `json:"run_id"`
	Agent            RemoteAgent                `json:"agent"`
	Checkpoint       shorttermmemory.Checkpoint `json:"checkpoint"`
	ContextVariables types.ContextVars          `json:"context_variables,omitempty"`
	StructuredOutput *provider.StructuredOutput `json:"strutured_output,omitempty"`
	Stream           bool                       `json:"stream,omitempty"`
}

func (t *Temporal) RunCompletion(ctx context.Context, cmd completionParams) (RemoteRunResult, error) {
	log := activity.GetLogger(ctx)
	log.Info("running completion activity", "agent", cmd.Agent.Name)

	ctxVars := maps.Clone(cmd.ContextVariables)
	if ctxVars == nil {
		ctxVars = make(types.ContextVars)
	}

	instructions, err := cmd.Agent.RenderInstructions(ctxVars)
	if err != nil {
		return RemoteRunResult{}, fmt.Errorf("failed to render instructions: %w", err)
	}

	model, exist := models.Get(cmd.Agent.Model)
	if !exist {
		return RemoteRunResult{}, fmt.Errorf("model %s not found", cmd.Agent.Model)
	}

	agg := shorttermmemory.New()
	cmd.Checkpoint.MergeInto(agg)

	stream, err := model.Provider().ChatCompletion(ctx, provider.CompletionParams{
		RunID:          cmd.RunID,
		Instructions:   instructions,
		Thread:         agg,
		Stream:         cmd.Stream,
		ResponseSchema: cmd.StructuredOutput,
		Model:          model,
	})
	if err != nil {
		return RemoteRunResult{}, fmt.Errorf("failed to get chat completion: %w", err)
	}

	for {
		select {
		case event, hasMore := <-stream:
			if !hasMore {
				// Return last message from thread
				msgs := agg.Messages()
				if len(msgs) == 0 {
					return RemoteRunResult{}, fmt.Errorf("no messages in thread")
				}
				lastMsg := msgs[len(msgs)-1]

				if assistantMsg, ok := lastMsg.Payload.(messages.AssistantMessage); ok {
					return RemoteRunResult{
						ID:         cmd.RunID,
						Result:     assistantMsg.Content.Content,
						Checkpoint: agg.Checkpoint(),
						Type:       RemoteRunResultTypeCompletion,
					}, nil
				}

				if toolCallMsg, ok := lastMsg.Payload.(messages.ToolCallMessage); ok {
					return RemoteRunResult{
						ID:         cmd.RunID,
						Checkpoint: agg.Checkpoint(),
						Type:       RemoteRunResultTypeToolCall,
						ToolCalls:  &toolCallMsg,
					}, nil
				}

				return RemoteRunResult{}, fmt.Errorf("unexpected last message type")
			}

			if err := t.processStreamEvent(ctx, event, &cmd, agg); err != nil {
				return RemoteRunResult{}, err
			}
		case <-ctx.Done():
			return RemoteRunResult{}, ctx.Err()
		}
	}
}

func (t *Temporal) processStreamEvent(ctx context.Context, event provider.StreamEvent, params *completionParams, agg *shorttermmemory.Aggregator) error {
	switch event := event.(type) {
	case provider.Delim:
		return nil
	case provider.Error:
		if err := t.PublishError(ctx, *params, event.Err.Error()); err != nil {
			return err
		}
		return event.Err
	case provider.Chunk[messages.AssistantMessage]:
		return publishEvent[messages.AssistantMessage](ctx, t.broker, event.RunID.String(), params.Agent.Name, event)
	case provider.Chunk[messages.ToolCallMessage]:
		return publishEvent[messages.ToolCallMessage](ctx, t.broker, event.RunID.String(), params.Agent.Name, event)
	case provider.Response[messages.ToolCallMessage]:
		event.Checkpoint.MergeInto(agg)
		msg := messages.Message[messages.ToolCallMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Payload:   event.Response,
			Sender:    params.Agent.Name,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		}
		agg.AddToolCall(msg)
		return publishEvent[messages.ToolCallMessage](ctx, t.broker, event.RunID.String(), params.Agent.Name, event)
	case provider.Response[messages.AssistantMessage]:
		event.Checkpoint.MergeInto(agg)
		msg := messages.Message[messages.AssistantMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Payload:   event.Response,
			Sender:    params.Agent.Name,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		}
		agg.AddAssistantMessage(msg)
		return publishEvent[messages.AssistantMessage](ctx, t.broker, event.RunID.String(), params.Agent.Name, event)
	default:
		panic(fmt.Sprintf("unknown event type %T", event))
	}
}

func publishEvent[T messages.ModelMessage](ctx context.Context, broker broker.Broker, topic, sender string, event provider.StreamEvent) error {
	log := activity.GetLogger(ctx)
	if err := broker.Topic(ctx, topic).Publish(ctx, events.FromStreamEvent(event, sender)); err != nil {
		log.Error("failed to publish event", "error", err)
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}

// PublishError is an activity that publishes error events
func (t *Temporal) PublishError(ctx context.Context, params completionParams, errMsg string) error {
	log := activity.GetLogger(ctx)
	err := errors.New(errMsg)
	if ee, hasErr := wrapErr(params.RunID, params.Checkpoint.ID(), params.Agent.Name, err); hasErr {
		if perr := t.broker.Topic(ctx, params.RunID.String()).Publish(ctx, ee); perr != nil {
			log.Error("failed to publish error", "error", perr)
			return fmt.Errorf("failed to publish error: %w", perr)
		}
	}
	return nil
}

func (t *Temporal) CallTool(ctx context.Context, tc remoteToolCallParams) (remoteToolCallResult, error) {
	log := activity.GetLogger(ctx)
	log.Info("calling tool", "name", tc.ToolCall.Name, "args", tc.ToolCall.Arguments)

	agent, hasAgent := agent.Get(tc.Agent.Name)
	if !hasAgent {
		return remoteToolCallResult{}, fmt.Errorf("agent not found: %s", tc.Agent.Name)
	}

	var agentTool *tool.Definition
	for tool := range slices.Values(agent.Tools()) {
		if tool.Name == tc.ToolCall.Name {
			agentTool = &tool
			break
		}
	}

	if agentTool == nil {
		return remoteToolCallResult{}, events.Error{
			RunID:     tc.RunID,
			TurnID:    tc.TurnID,
			Sender:    agent.Name(),
			Err:       fmt.Errorf("unknown tool %s", tc.ToolCall.Name),
			Timestamp: strfmt.DateTime(time.Now()),
		}
	}

	args := buildArgList(tc.ToolCall.Arguments, agentTool.Parameters)
	// Create a copy of context variables to avoid modifying the original
	ctxVars := maps.Clone(tc.CtxVars)
	if ctxVars == nil {
		ctxVars = make(types.ContextVars)
	}

	result, err := callFunction(agentTool.Function, args, ctxVars)
	if err != nil {
		return remoteToolCallResult{}, err
	}

	// Update original context variables with any new values
	if result.ContextVariables != nil {
		maps.Copy(ctxVars, result.ContextVariables)
	}

	if result.Agent != nil {
		return remoteToolCallResult{
			Agent: &RemoteAgent{
				Name:              result.Agent.Name(),
				Model:             result.Agent.Model().Name(),
				Instructions:      result.Agent.Instructions(),
				ParallelToolCalls: result.Agent.ParallelToolCalls(),
			},
			CtxVars: ctxVars,
		}, nil
	}

	msg := messages.Message[messages.ToolResponse]{
		RunID:  tc.RunID,
		TurnID: tc.TurnID,
		Payload: messages.ToolResponse{
			ToolName:   tc.ToolCall.Name,
			ToolCallID: tc.ToolCall.ID,
			Content:    result.Value,
		},
		Sender:    agentTool.Name,
		Timestamp: strfmt.DateTime(time.Now()),
	}

	// Publish tool response event
	if err := t.broker.Topic(ctx, tc.RunID.String()).Publish(ctx, events.Request[messages.ToolResponse]{
		Message: msg.Payload,
		RunID:   tc.RunID,
		TurnID:  tc.TurnID,
		Sender:  agentTool.Name,
	}); err != nil {
		log.Error("failed to publish tool response", "error", err)
		return remoteToolCallResult{}, fmt.Errorf("failed to publish tool response: %w", err)
	}

	return remoteToolCallResult{
		Message: &msg,
		CtxVars: ctxVars,
	}, nil
}

func nameAsID(name string) string {
	hashVal := sha256.Sum256([]byte(name))
	return hex.EncodeToString(hashVal[:])
}
