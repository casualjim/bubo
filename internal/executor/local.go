package executor

import (
	"context"
	"encoding"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/reflectx"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
	"github.com/go-openapi/strfmt"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

var _ Executor = &Local{}

type breakError struct{}

func (e *breakError) Error() string {
	return "break"
}

type continueError struct{}

func (e *continueError) Error() string {
	return "continue"
}

type Temporal struct{}

type Local struct{}

func NewLocal() *Local {
	return &Local{}
}

func wrapErr(runID, turnID uuid.UUID, sender string, err error) (events.Error, bool) {
	if err == nil {
		return events.Error{}, false
	}
	if pErr, ok := err.(events.Error); ok { //nolint: errorlint
		return pErr, true
	}
	return events.Error{
		RunID:     runID,
		TurnID:    turnID,
		Sender:    sender,
		Err:       err,
		Timestamp: strfmt.DateTime(time.Now()),
	}, true
}

type toolCallParams struct {
	runID       uuid.UUID
	agent       api.Owl
	contextVars types.ContextVars
	mem         *shorttermmemory.Aggregator
	hook        events.Hook
	toolCalls   messages.ToolCallMessage
}

func (l *Local) Run(ctx context.Context, command RunCommand, promise Promise) error {
	if err := command.Validate(); err != nil {
		return err
	}

	contextVars := command.initializeContextVars()
	thread := command.Thread.Fork()
	activeAgent := command.Agent

	err := l.runReactorLoop(ctx, reactorParams{
		command:     command,
		thread:      thread,
		activeAgent: activeAgent,
		contextVars: contextVars,
		promise:     promise,
	})
	if err != nil {
		var breakErr *breakError
		if errors.As(err, &breakErr) {
			// Break error means successful completion
			command.Thread.Join(thread)
			return nil
		}

		return err
	}

	// Always join the thread back to the command's thread
	command.Thread.Join(thread)
	return nil
}

type reactorParams struct {
	command     RunCommand
	thread      *shorttermmemory.Aggregator
	activeAgent api.Owl
	contextVars types.ContextVars
	promise     Promise
}

func (l *Local) runReactorLoop(ctx context.Context, params reactorParams) error {
	for params.thread.TurnLen() < params.command.MaxTurns {
		// Validate current agent and provider
		if err := l.validateAgentAndProvider(ctx, &params); err != nil {
			return err
		}

		// Get chat completion stream
		stream, err := l.initiateChatCompletion(ctx, &params)
		if err != nil {
			return err
		}

		// Process stream events
		if err := l.handleStreamEvents(ctx, stream, &params); err != nil {
			var continueErr *continueError
			if errors.As(err, &continueErr) {
				continue // Agent transfer occurred, retry with new agent
			}
			return err
		}

		// Handle completion
		return l.handleStreamCompletion(&params)
	}
	return errors.New("max turns exceeded")
}

func (l *Local) validateAgentAndProvider(ctx context.Context, params *reactorParams) error {
	model := params.activeAgent.Model()
	if model == nil {
		err := fmt.Errorf("agent model cannot be nil")
		l.publishError(ctx, params, err)
		return err
	}

	prov := model.Provider()
	if prov == nil {
		err := fmt.Errorf("model provider cannot be nil")
		l.publishError(ctx, params, err)
		return err
	}

	return nil
}

func (l *Local) initiateChatCompletion(ctx context.Context, params *reactorParams) (<-chan provider.StreamEvent, error) {
	instructions, err := params.activeAgent.RenderInstructions(params.contextVars)
	if err != nil {
		l.publishError(ctx, params, fmt.Errorf("failed to render instructions: %w", err))
		return nil, fmt.Errorf("failed to render instructions: %w", err)
	}

	stream, err := params.activeAgent.Model().Provider().ChatCompletion(ctx, provider.CompletionParams{
		RunID:          params.command.ID(),
		Instructions:   instructions,
		Thread:         params.thread,
		Stream:         params.command.Stream,
		Model:          params.activeAgent.Model(),
		ResponseSchema: params.command.ResponseSchema,
		Tools:          params.activeAgent.Tools(),
	})
	if err != nil {
		l.publishError(ctx, params, fmt.Errorf("failed to get chat completion: %w", err))
		return nil, fmt.Errorf("failed to get chat completion: %w", err)
	}

	return stream, nil
}

func (l *Local) handleStreamEvents(ctx context.Context, stream <-chan provider.StreamEvent, params *reactorParams) error {
	for {
		select {
		case event, hasMore := <-stream:
			if !hasMore {
				return l.handleStreamCompletion(params)
			}

			if err := l.processStreamEvent(ctx, event, params); err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *Local) handleStreamCompletion(params *reactorParams) error {
	msgs := params.thread.Messages()
	if len(msgs) == 0 {
		return fmt.Errorf("no messages in thread")
	}

	// The last message must be from the current agent
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Sender != params.activeAgent.Name() {
		return fmt.Errorf("last message is not from current agent %s", params.activeAgent.Name())
	}

	// If it's a tool response, continue to allow agent transfer
	if _, ok := lastMsg.Payload.(messages.ToolResponse); ok {
		return &continueError{}
	}

	// If it's an assistant message, complete the promise
	// We know it's safe because handleToolCallResponse would have returned continueError
	// if there was an agent transfer
	if assistantMsg, ok := lastMsg.Payload.(messages.AssistantMessage); ok {
		params.promise.Complete(assistantMsg.Content.Content)
		return &breakError{}
	}

	// Last message was neither assistant message nor tool response
	return fmt.Errorf("last message from agent %s was neither assistant message nor tool response", params.activeAgent.Name())
}

func (l *Local) processStreamEvent(ctx context.Context, event provider.StreamEvent, params *reactorParams) error {
	switch event := event.(type) {
	case provider.Delim:
		return nil
	case provider.Error:
		l.publishError(ctx, params, event)
		params.promise.Error(event.Err)
		return event.Err
	case provider.Chunk[messages.AssistantMessage]:
		params.command.Hook.OnAssistantChunk(ctx, messages.Message[messages.AssistantMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Payload:   event.Chunk,
			Sender:    params.activeAgent.Name(),
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		})
		return nil
	case provider.Chunk[messages.ToolCallMessage]:
		params.command.Hook.OnToolCallChunk(ctx, messages.Message[messages.ToolCallMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Payload:   event.Chunk,
			Sender:    params.activeAgent.Name(),
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		})
		return nil
	case provider.Response[messages.ToolCallMessage]:
		return l.handleToolCallResponse(ctx, event, params)
	case provider.Response[messages.AssistantMessage]:
		if err := l.handleAssistantResponse(ctx, event, params); err != nil {
			return err
		}
		return nil
	default:
		panic(fmt.Sprintf("unknown event type %T", event))
	}
}

func (l *Local) publishError(ctx context.Context, params *reactorParams, err error) {
	if ee, hasErr := wrapErr(params.command.ID(), params.thread.ID(), params.activeAgent.Name(), err); hasErr {
		params.command.Hook.OnError(ctx, ee)
	}
}

func (l *Local) handleAssistantResponse(ctx context.Context, event provider.Response[messages.AssistantMessage], params *reactorParams) error {
	// The provider guarantees that tool calls and responses are processed before
	// any assistant messages. If we get here, it means all tool calls have been
	// handled and there were no agent transfers.
	event.Checkpoint.MergeInto(params.thread)

	msg := messages.Message[messages.AssistantMessage]{
		RunID:     event.RunID,
		TurnID:    event.TurnID,
		Payload:   event.Response,
		Sender:    params.activeAgent.Name(),
		Timestamp: event.Timestamp,
		Meta:      event.Meta,
	}
	params.thread.AddAssistantMessage(msg)
	params.command.Hook.OnAssistantMessage(ctx, msg)
	return nil
}

func (l *Local) handleToolCallResponse(ctx context.Context, event provider.Response[messages.ToolCallMessage], params *reactorParams) error {
	forked := params.thread.Fork()
	event.Checkpoint.MergeInto(forked)

	toolCallMsg := messages.Message[messages.ToolCallMessage]{
		RunID:     event.RunID,
		TurnID:    event.TurnID,
		Payload:   event.Response,
		Sender:    params.activeAgent.Name(),
		Timestamp: event.Timestamp,
		Meta:      event.Meta,
	}
	forked.AddToolCall(toolCallMsg)
	params.command.Hook.OnToolCallMessage(ctx, toolCallMsg)

	toolParams := toolCallParams{
		mem:         forked,
		agent:       params.activeAgent,
		runID:       event.RunID,
		hook:        params.command.Hook,
		toolCalls:   event.Response,
		contextVars: make(types.ContextVars),
	}
	if params.contextVars != nil {
		maps.Copy(toolParams.contextVars, params.contextVars)
	}

	nextAgent, err := l.handleToolCalls(ctx, toolParams)
	if err != nil {
		l.publishError(ctx, params, err)
		return err
	}

	// Handle agent transfer after joining threads
	if nextAgent != nil {
		params.activeAgent = nextAgent
		return &continueError{}
	}

	return nil
}

func (l *Local) handleToolCalls(ctx context.Context, params toolCallParams) (api.Owl, error) {
	agentTools := make(map[string]tool.Definition, len(params.agent.Tools()))
	for tool := range slices.Values(params.agent.Tools()) {
		agentTools[tool.Name] = tool
	}

	var agentTransfers []messages.ToolCallData
	var otherTools []messages.ToolCallData

	for _, call := range params.toolCalls.ToolCalls {
		tool, exists := agentTools[call.Name]
		if !exists {
			return nil, events.Error{
				RunID:     params.runID,
				TurnID:    params.mem.ID(),
				Sender:    params.agent.Name(),
				Err:       fmt.Errorf("unknown tool %s", call.Name),
				Timestamp: strfmt.DateTime(time.Now()),
			}
		}

		if reflectx.ResultImplements[api.Owl](tool.Function) {
			agentTransfers = append(agentTransfers, call)
		} else {
			otherTools = append(otherTools, call)
		}
	}

	for _, call := range append(agentTransfers, otherTools...) {
		tool := agentTools[call.Name]
		args := buildArgList(call.Arguments, tool.Parameters)
		result, err := callFunction(tool.Function, args, params.contextVars)
		if err != nil {
			return nil, err
		}

		// Check for agent transfer before adding response
		if result.Agent != nil {
			return result.Agent, nil
		}

		msg := messages.New().ToolResponse(call.ID, call.Name, fmt.Sprintf("%v", result.Value))
		msg.RunID = params.runID
		msg.TurnID = params.mem.ID()
		msg.Sender = params.agent.Name()
		params.mem.AddToolResponse(msg)
		params.hook.OnToolCallResponse(ctx, msg)

		if result.ContextVariables != nil {
			if params.contextVars == nil {
				params.contextVars = make(types.ContextVars)
			}
			maps.Copy(params.contextVars, result.ContextVariables)
		}
	}

	return nil, nil
}

func buildArgList(arguments string, parameters map[string]string) []reflect.Value {
	args := gjson.Parse(arguments)
	targs := make([]string, len(parameters))
	for k, v := range parameters {
		ns := strings.TrimPrefix(k, "param")
		i, _ := strconv.Atoi(ns)
		if i < 0 || i >= len(targs) {
			continue
		}
		targs[i] = v
	}

	toolArgs := make([]reflect.Value, 0)
	for _, arg := range targs {
		if arg == "" {
			continue
		}

		val := args.Get(arg)
		if !val.Exists() {
			continue
		}

		toolArgs = append(toolArgs, reflect.ValueOf(val.Value()))
	}
	return toolArgs
}

type toolResult struct {
	Value            string
	Agent            api.Owl
	ContextVariables types.ContextVars
}

func callFunction(fn any, args []reflect.Value, contextVars types.ContextVars) (toolResult, error) {
	val := reflect.ValueOf(fn)
	vtpe := val.Type()

	numIn := vtpe.NumIn()
	callArgs := make([]reflect.Value, numIn)

	for fi := 0; fi < numIn; fi++ {
		paramType := vtpe.In(fi)
		if reflectx.IsRefinedType[types.ContextVars](paramType) {
			callArgs[fi] = reflect.ValueOf(contextVars)
		} else if fi < len(args) {
			vv := args[fi]
			if vv.Type().ConvertibleTo(paramType) {
				callArgs[fi] = vv.Convert(paramType)
			}
		}
	}

	results := val.Call(callArgs)
	if len(results) == 0 {
		return toolResult{}, nil
	}

	res := results[0]
	if !res.IsValid() {
		return toolResult{}, nil
	}

	switch vtpe := res.Interface().(type) {
	case api.Owl:
		return toolResult{Value: fmt.Sprintf(`{"assistant":%q}`, vtpe.Name()), Agent: vtpe}, nil
	case error:
		return toolResult{}, vtpe
	case types.ContextVars:
		return toolResult{Value: "", ContextVariables: vtpe}, nil
	case string:
		return toolResult{Value: vtpe}, nil
	case time.Time:
		return toolResult{Value: vtpe.Format(time.RFC3339)}, nil
	case int, int8, int16, int32, int64:
		val := reflect.ValueOf(vtpe)
		return toolResult{Value: strconv.FormatInt(val.Int(), 10)}, nil
	case uint, uint8, uint16, uint32, uint64:
		val := reflect.ValueOf(vtpe)
		return toolResult{Value: strconv.FormatUint(val.Uint(), 10)}, nil
	case float32, float64:
		return toolResult{Value: strconv.FormatFloat(vtpe.(float64), 'f', -1, 64)}, nil
	case encoding.TextMarshaler:
		b, err := vtpe.MarshalText()
		if err != nil {
			slog.Error("Error marshalling function return", slogx.Error(err))
			return toolResult{}, err
		}
		return toolResult{Value: string(b)}, nil
	case fmt.Stringer:
		return toolResult{Value: vtpe.String()}, nil
	default:
		b, err := json.Marshal(vtpe)
		if err != nil {
			slog.Error("Error marshalling function return", slogx.Error(err))
			return toolResult{}, err
		}
		return toolResult{Value: string(b)}, nil
	}
}
