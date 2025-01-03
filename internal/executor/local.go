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

type Temporal struct{}

type Local struct {
	// broker broker.Broker
}

func NewLocal() *Local {
	// if broker == nil {
	// 	panic("broker cannot be nil")
	// }
	return &Local{
		// broker: broker,
	}
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

	// topic, subscription, err := l.setupTopicAndSubscription(ctx, command)
	// if err != nil {
	// 	return err
	// }
	// defer subscription.Unsubscribe()

	contextVars := command.initializeContextVars()
	thread := command.Thread.Fork()
	activeAgent := command.Agent

	return l.runReactorLoop(ctx, reactorParams{
		command:     command,
		thread:      thread,
		activeAgent: activeAgent,
		contextVars: contextVars,
		// topic:       topic,
		promise: promise,
	})
}

// func (l *Local) setupTopicAndSubscription(ctx context.Context, command RunCommand) (broker.Topic, broker.Subscription, error) {
// 	topic := l.Topic(ctx, command.ID().String())
// 	subscription, err := topic.Subscribe(ctx, command.Hook)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to subscribe to topic: %w", err)
// 	}
// 	if subscription == nil {
// 		return nil, nil, fmt.Errorf("received nil subscription from topic")
// 	}
// 	return topic, subscription, nil
// }

type reactorParams struct {
	command     RunCommand
	thread      *shorttermmemory.Aggregator
	activeAgent api.Owl
	contextVars types.ContextVars
	// topic       broker.Topic
	promise Promise
}

func (l *Local) runReactorLoop(ctx context.Context, params reactorParams) error {
	for params.thread.TurnLen() < params.command.MaxTurns {
		if err := l.validateAgentAndProvider(ctx, &params); err != nil {
			return err
		}

		stream, err := l.initiateChatCompletion(ctx, &params)
		if err != nil {
			return err
		}

		if err := l.handleStreamEvents(ctx, stream, &params); err != nil {
			var breakErr *breakError
			if errors.As(err, &breakErr) {
				return nil // Normal completion, exit without error
			}
			return err
		}
	}
	return nil
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
	if len(params.thread.Messages()) > 0 {
		lastMsg := params.thread.Messages()[len(params.thread.Messages())-1]
		if assistantMsg, ok := lastMsg.Payload.(messages.AssistantMessage); ok {
			params.promise.Complete(assistantMsg.Content.Content)
		}
	}
	return nil
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
		// return l.publishEvent(ctx, *params, event)
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
		// Signal that we should break after the stream is closed
		// params.command.MaxTurns = params.thread.TurnLen()
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

func (l *Local) handleToolCallResponse(ctx context.Context, event provider.Response[messages.ToolCallMessage], params *reactorParams) error {
	event.Checkpoint.MergeInto(params.thread)
	params.command.Hook.OnToolCallMessage(ctx, messages.Message[messages.ToolCallMessage]{
		RunID:     event.RunID,
		TurnID:    event.TurnID,
		Payload:   event.Response,
		Sender:    params.activeAgent.Name(),
		Timestamp: event.Timestamp,
		Meta:      event.Meta,
	})

	agent, err := l.handleToolCalls(ctx, toolCallParams{
		mem:         params.thread.Fork(),
		agent:       params.activeAgent,
		runID:       params.command.ID(),
		hook:        params.command.Hook,
		toolCalls:   event.Response,
		contextVars: params.contextVars,
	})
	if err != nil {
		l.publishError(ctx, params, err)
		return err
	}
	if agent != nil {
		params.activeAgent = agent
	}
	return nil
}

func (l *Local) handleAssistantResponse(ctx context.Context, event provider.Response[messages.AssistantMessage], params *reactorParams) error {
	event.Checkpoint.MergeInto(params.thread)

	msg := messages.Message[messages.AssistantMessage]{
		RunID:     params.command.ID(),
		TurnID:    params.thread.ID(),
		Payload:   event.Response,
		Sender:    params.activeAgent.Name(),
		Timestamp: strfmt.DateTime(time.Now()),
	}
	params.thread.AddAssistantMessage(msg)
	params.command.Hook.OnAssistantMessage(ctx, msg)

	// Signal that we should break after the stream is closed
	params.command.MaxTurns = params.thread.TurnLen()
	return nil
}

func (l *Local) handleToolCalls(ctx context.Context, params toolCallParams) (api.Owl, error) {
	agentTools := make(map[string]tool.Definition, len(params.agent.Tools()))
	for tool := range slices.Values(params.agent.Tools()) {
		agentTools[tool.Name] = tool
	}

	// Initialize context variables
	var contextVars types.ContextVars
	if params.contextVars != nil {
		contextVars = maps.Clone(params.contextVars)
	}

	// Partition tool calls into agent transfers and regular tools while preserving order
	var agentCalls, regularCalls []struct {
		index int
		call  messages.ToolCallData
	}
	for i, call := range params.toolCalls.ToolCalls {
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

		// Check if tool returns an Agent by examining its return type
		if reflect.TypeOf(tool.Function).Out(0) == reflect.TypeOf((*api.Owl)(nil)).Elem() {
			agentCalls = append(agentCalls, struct {
				index int
				call  messages.ToolCallData
			}{i, call})
		} else {
			regularCalls = append(regularCalls, struct {
				index int
				call  messages.ToolCallData
			}{i, call})
		}
	}

	// Sort by original index to maintain received order within each partition
	slices.SortFunc(agentCalls, func(a, b struct {
		index int
		call  messages.ToolCallData
	},
	) int {
		return a.index - b.index
	})
	slices.SortFunc(regularCalls, func(a, b struct {
		index int
		call  messages.ToolCallData
	},
	) int {
		return a.index - b.index
	})

	// Handle agent transfers first - return on first successful transfer
	for _, item := range agentCalls {
		tool := agentTools[item.call.Name]
		args := buildArgList(item.call.Arguments, tool.Parameters)
		result, err := callFunction(tool.Function, args, contextVars)
		if err != nil {
			return nil, events.Error{
				RunID:     params.runID,
				TurnID:    params.mem.ID(),
				Sender:    params.agent.Name(),
				Err:       err,
				Timestamp: strfmt.DateTime(time.Now()),
			}
		}

		// Update memory and context variables
		params.mem.AddToolResponse(messages.Message[messages.ToolResponse]{
			RunID:  params.runID,
			TurnID: params.mem.ID(),
			Payload: messages.ToolResponse{
				ToolName:   tool.Name,
				ToolCallID: item.call.ID,
				Content:    result.Value,
			},
			Sender:    params.agent.Name(),
			Timestamp: strfmt.DateTime(time.Now()),
			Meta:      gjson.Result{},
		})

		if result.ContextVariables != nil {
			if contextVars == nil {
				contextVars = make(types.ContextVars, len(result.ContextVariables))
			}
			maps.Copy(contextVars, result.ContextVariables)
			// Update parent context variables
			if params.contextVars == nil {
				params.contextVars = make(types.ContextVars, len(result.ContextVariables))
			}
			maps.Copy(params.contextVars, result.ContextVariables)
		}

		// Publish tool response
		params.hook.OnToolCallResponse(ctx, messages.Message[messages.ToolResponse]{
			RunID:  params.runID,
			TurnID: params.mem.ID(),
			Payload: messages.ToolResponse{
				ToolName:   tool.Name,
				ToolCallID: item.call.ID,
				Content:    fmt.Sprintf("transfer to agent %s", result.Agent.Name()),
			},
			Sender:    params.agent.Name(),
			Timestamp: strfmt.DateTime(time.Now()),
		})

		if result.Agent != nil {
			return result.Agent, nil // Return first successful agent transfer
		}
	}

	// Handle regular tool calls
	for _, item := range regularCalls {
		tool := agentTools[item.call.Name]
		args := buildArgList(item.call.Arguments, tool.Parameters)
		result, err := callFunction(tool.Function, args, contextVars)
		if err != nil {
			return nil, events.Error{
				RunID:     params.runID,
				TurnID:    params.mem.ID(),
				Sender:    params.agent.Name(),
				Err:       err,
				Timestamp: strfmt.DateTime(time.Now()),
			}
		}

		// Update memory and context variables
		params.mem.AddToolResponse(messages.Message[messages.ToolResponse]{
			RunID:  params.runID,
			TurnID: params.mem.ID(),
			Payload: messages.ToolResponse{
				ToolName:   tool.Name,
				ToolCallID: item.call.ID,
				Content:    result.Value,
			},
			Sender:    params.agent.Name(),
			Timestamp: strfmt.DateTime(time.Now()),
			Meta:      gjson.Result{},
		})

		if result.ContextVariables != nil {
			if contextVars == nil {
				contextVars = make(types.ContextVars, len(result.ContextVariables))
			}
			maps.Copy(contextVars, result.ContextVariables)
			// Update parent context variables
			if params.contextVars == nil {
				params.contextVars = make(types.ContextVars, len(result.ContextVariables))
			}
			maps.Copy(params.contextVars, result.ContextVariables)
		}

		// Publish tool response
		params.hook.OnToolCallResponse(ctx, messages.Message[messages.ToolResponse]{
			RunID:  params.runID,
			TurnID: params.mem.ID(),
			Payload: messages.ToolResponse{
				ToolName:   tool.Name,
				ToolCallID: item.call.ID,
				Content:    result.Value,
			},
			Sender:    params.agent.Name(),
			Timestamp: strfmt.DateTime(time.Now()),
		})
	}

	return nil, nil
}

func buildArgList(arguments string, parameters map[string]string) []reflect.Value {
	args := gjson.Parse(arguments)
	// build an ordered list of arguments
	targs := make([]string, len(parameters))
	for k, v := range parameters {
		ns := strings.TrimPrefix(k, "param")
		i, _ := strconv.Atoi(ns)
		if i < 0 || i >= len(targs) {
			continue
		}
		targs[i] = v
	}

	toolArgs := make([]reflect.Value, 0) //nolint: prealloc
	for _, arg := range targs {
		if arg == "" {
			continue
		}

		val := args.Get(arg)
		if !val.Exists() {
			continue
		}

		// TODO: this needs to support a runtime context argument
		// that is optionally passed to the function
		// TODO: this needs verification of complex types
		toolArgs = append(toolArgs, reflect.ValueOf(val.Value()))
	}
	return toolArgs
}

type result struct {
	Value            string
	Agent            api.Owl
	ContextVariables types.ContextVars
}

func callFunction(fn any, args []reflect.Value, contextVars types.ContextVars) (result, error) {
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
		return result{}, nil
	}

	res := results[0]
	if !res.IsValid() {
		return result{}, nil
	}

	switch vtpe := res.Interface().(type) {
	case api.Owl:
		return result{Value: fmt.Sprintf(`{"assistant":%q}`, vtpe.Name()), Agent: vtpe}, nil
	case error:
		return result{}, vtpe
	case types.ContextVars:
		return result{Value: "", ContextVariables: vtpe}, nil
	case string:
		return result{Value: vtpe}, nil
	case time.Time:
		return result{Value: vtpe.Format(time.RFC3339)}, nil
	case int, int8, int16, int32, int64:
		val := reflect.ValueOf(vtpe)
		return result{Value: strconv.FormatInt(val.Int(), 10)}, nil
	case uint, uint8, uint16, uint32, uint64:
		val := reflect.ValueOf(vtpe)
		return result{Value: strconv.FormatUint(val.Uint(), 10)}, nil
	case float32, float64:
		return result{Value: strconv.FormatFloat(vtpe.(float64), 'f', -1, 64)}, nil
	case encoding.TextMarshaler:
		b, err := vtpe.MarshalText()
		if err != nil {
			slog.Error("Error marshalling function return", slogx.Error(err))
			return result{}, err
		}
		return result{Value: string(b)}, nil
	case fmt.Stringer:
		return result{Value: vtpe.String()}, nil
	default:
		b, err := json.Marshal(vtpe)
		if err != nil {
			slog.Error("Error marshalling function return", slogx.Error(err))
			return result{}, err
		}
		return result{Value: string(b)}, nil
	}
}
