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

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/executor/pubsub"
	"github.com/casualjim/bubo/pkg/messages"
	"github.com/casualjim/bubo/pkg/reflectx"
	"github.com/casualjim/bubo/pkg/runstate"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/go-openapi/strfmt"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

var _ Executor[any] = &Local[any]{}

type Temporal struct{}

type Local[T any] struct {
	broker pubsub.Broker
}

func (l *Local[T]) Run(ctx context.Context, command RunCommand[T]) error {
	topic := l.broker.Topic(ctx, command.ID.String())
	events, err := topic.Subscribe(context.Background(), command.Hook)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}
	go func() {
		defer events.Unsubscribe()

		contextVars := maps.Clone(command.ContextVariables)
		thread := command.Thread.Fork()
		activeAgent := command.Agent

		// REACTOR:
		for thread.TurnLen() < command.MaxTurns {
			prov := activeAgent.Model().Provider()
			instructions, err := activeAgent.RenderInstructions(contextVars)
			if err != nil {
				ee, _ := wrapErr(command.ID, thread.ID(), activeAgent.Name(), err)
				_ = topic.Publish(ctx, ee)
				return
			}

			stream, err := prov.ChatCompletion(ctx, provider.CompletionParams{
				RunID:        command.ID,
				Instructions: instructions,
				Thread:       thread,
			})
			if err != nil {
				ee, _ := wrapErr(command.ID, thread.ID(), activeAgent.Name(), err)
				_ = topic.Publish(ctx, ee)
				return
			}

			for {
				select {
				case event, hasMore := <-stream:
					if !hasMore {
						return
					}

					switch event := event.(type) {
					case provider.Delim:
					case provider.Error:
						if perr := topic.Publish(ctx, pubsub.FromStreamEvent(event, activeAgent.Name())); perr != nil {
							slog.ErrorContext(ctx, "failed to publish error event", slogx.Error(perr))
						}
						return
					case provider.Chunk[messages.AssistantMessage]:
						if perr := topic.Publish(ctx, pubsub.FromStreamEvent(event, activeAgent.Name())); perr != nil {
							slog.ErrorContext(ctx, "failed to publish chunk event", slogx.Error(perr))
						}

					case provider.Chunk[messages.ToolCallMessage]:
						if perr := topic.Publish(ctx, pubsub.FromStreamEvent(event, activeAgent.Name())); perr != nil {
							slog.ErrorContext(ctx, "failed to publish chunk event", slogx.Error(perr))
						}

					case provider.Response[messages.ToolCallMessage]:
						event.Checkpoint.MergeInto(thread)
						if perr := topic.Publish(ctx, pubsub.FromStreamEvent(event, activeAgent.Name())); perr != nil {
							slog.ErrorContext(ctx, "failed to publish response event", slogx.Error(perr))
						}

						if agent, err := l.handleToolCalls(ctx, toolCallParams{
							mem:       thread.Fork(),
							agent:     activeAgent,
							runID:     command.ID,
							toolCalls: event.Response,
							topic:     topic,
						}); err != nil {
							evt, _ := wrapErr(command.ID, thread.ID(), activeAgent.Name(), err)
							if perr := topic.Publish(ctx, evt); perr != nil {
								slog.ErrorContext(ctx, "failed to publish response event", slogx.Error(perr))
							}
							return
						} else if agent != nil {
							activeAgent = agent
						}

					case provider.Response[messages.AssistantMessage]:
						event.Checkpoint.MergeInto(thread)
						if perr := topic.Publish(ctx, pubsub.FromStreamEvent(event, activeAgent.Name())); perr != nil {
							slog.ErrorContext(ctx, "failed to publish response event", slogx.Error(perr))
						}

						value, err := command.UnmarshalResponse([]byte(event.Response.Content.Content))
						if err != nil {
							evt, _ := wrapErr(command.ID, thread.ID(), activeAgent.Name(), err)
							if perr := topic.Publish(ctx, evt); perr != nil {
								slog.ErrorContext(ctx, "failed to unmarshal response value", slogx.Error(perr))
							}
						}
						if perr := topic.Publish(ctx, pubsub.Response[T]{
							RunID:     command.ID,
							TurnID:    thread.ID(),
							Response:  value,
							Sender:    activeAgent.Name(),
							Timestamp: strfmt.DateTime(time.Now()),
						}); perr != nil {
							evt, _ := wrapErr(command.ID, thread.ID(), activeAgent.Name(), err)
							if perr := topic.Publish(ctx, evt); perr != nil {
								slog.ErrorContext(ctx, "failed to publish response value", slogx.Error(perr))
							}
						}

						return
					default:
						panic(fmt.Sprintf("unknown event type %T", event))
					}
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return nil
}

func wrapErr(runID, turnID uuid.UUID, sender string, err error) (pubsub.Error, bool) {
	if err == nil {
		return pubsub.Error{}, false
	}
	if errors.Is(err, pubsub.Error{}) {
		return err.(pubsub.Error), true //nolint:errorlint
	}
	return pubsub.Error{
		RunID:     runID,
		TurnID:    turnID,
		Sender:    sender,
		Err:       err,
		Timestamp: strfmt.DateTime(time.Now()),
	}, true
}

type toolCallParams struct {
	runID       uuid.UUID
	agent       bubo.Agent
	contextVars types.ContextVars
	mem         *runstate.Aggregator
	toolCalls   messages.ToolCallMessage
	topic       pubsub.Topic
}

func (l *Local[T]) handleToolCalls(_ context.Context, params toolCallParams) (bubo.Agent, error) {
	agentTools := make(map[string]bubo.AgentToolDefinition, len(params.agent.Tools()))
	for tool := range slices.Values(params.agent.Tools()) {
		agentTools[tool.Name] = tool
	}

	contextVars := maps.Clone(params.contextVars)
	var nextAgent bubo.Agent

	// TODO: partition the tool calls into calls that will return an agent and those that will not.
	// When we have a tool that returns an agent we need to executed those first and return the first successful one.
	for toolCall := range slices.Values(params.toolCalls.ToolCalls) {
		tool, foundTool := agentTools[toolCall.Name]
		if !foundTool {
			return nil, pubsub.Error{
				RunID:     params.runID,
				TurnID:    params.mem.ID(),
				Sender:    params.agent.Name(),
				Err:       fmt.Errorf("unknown tool %s", toolCall.Name),
				Timestamp: strfmt.DateTime(time.Now()),
			}
		}

		args := buildArgList(toolCall.Arguments, tool.Parameters)

		result, err := callFunction(tool.Function, args, contextVars)
		if err != nil {
			return nil, pubsub.Error{
				RunID:     params.runID,
				TurnID:    params.mem.ID(),
				Sender:    params.agent.Name(),
				Err:       err,
				Timestamp: strfmt.DateTime(time.Now()),
			}
		}

		params.mem.AddToolResponse(messages.Message[messages.ToolResponse]{
			RunID:  params.runID,
			TurnID: params.mem.ID(),
			Payload: messages.ToolResponse{
				ToolName:   tool.Name,
				ToolCallID: toolCall.ID,
				Content:    result.Value,
			},
			Sender:    params.agent.Name(),
			Timestamp: strfmt.DateTime(time.Now()),
			Meta:      gjson.Result{},
		})

		if result.ContextVariables != nil {
			if params.contextVars == nil {
				params.contextVars = make(types.ContextVars, len(result.ContextVariables))
			}
			maps.Copy(params.contextVars, result.ContextVariables)
		}
		if result.Agent != nil {
			nextAgent = result.Agent
		}
	}

	return nextAgent, nil
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

	var toolArgs []reflect.Value //nolint: prealloc
	for arg := range slices.Values(targs) {
		if arg == "" {
			continue
		}

		// TODO: this needs to support a runtime context argument
		// that is optionally passed to the function
		// TODO: this needs verification of complex types
		toolArgs = append(toolArgs, reflect.ValueOf(args.Get(arg).Value()))
	}
	return toolArgs
}

type result struct {
	Value            string
	Agent            bubo.Agent
	ContextVariables types.ContextVars
}

func callFunction(fn any, args []reflect.Value, contextVars types.ContextVars) (result, error) {
	val := reflect.ValueOf(fn)
	vtpe := val.Type()

	for fi := 0; fi < vtpe.NumIn(); fi++ {
		if reflectx.IsRefinedType[types.ContextVars](vtpe.In(fi)) {
			args[fi] = reflect.ValueOf(contextVars)
		} else {
			vv := args[fi]
			if vv.Type().ConvertibleTo(vtpe.In(fi)) {
				args[fi] = vv.Convert(vtpe.In(fi))
			}
		}
	}

	results := val.Call(args)

	switch vtpe := results[0].Interface().(type) {
	case bubo.Agent:
		return result{Value: fmt.Sprintf(`{"assistant":%q}`, vtpe.Name()), Agent: vtpe}, nil
	case error:
		return result{}, vtpe
	case string:
		return result{Value: vtpe}, nil
	case time.Time:
		return result{Value: vtpe.Format(time.RFC3339)}, nil
	case int, int8, int16, int32, int64:
		return result{Value: strconv.FormatInt(vtpe.(int64), 10)}, nil
	case uint, uint8, uint16, uint32, uint64:
		return result{Value: strconv.FormatUint(vtpe.(uint64), 10)}, nil
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
