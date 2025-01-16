package bubo

import (
	"context"
	"reflect"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/fogfish/opts"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"
)

type ExecutionContext struct {
	executor       executor.Executor
	hook           events.Hook
	promise        executor.Promise
	responseSchema *provider.StructuredOutput
	contextVars    types.ContextVars
	onClose        func(context.Context)
	stream         bool
	maxTurns       int
}

func (e *ExecutionContext) createCommand(agent api.Agent, mem *shorttermmemory.Aggregator) (executor.RunCommand, error) {
	cmd, err := executor.NewRunCommand(agent, mem, e.hook)
	if err != nil {
		return executor.RunCommand{}, err
	}
	if len(e.contextVars) > 0 {
		cmd = cmd.WithContextVariables(e.contextVars)
	}
	if e.responseSchema != nil {
		cmd = cmd.WithStructuredOutput(e.responseSchema)
	}
	if e.stream {
		cmd = cmd.WithStream(e.stream)
	}
	if e.maxTurns > 0 {
		cmd = cmd.WithMaxTurns(e.maxTurns)
	}
	return cmd, nil
}

func jsonSchema[T any]() *jsonschema.Schema {
	var schema *jsonschema.Schema
	var isGjsonResult bool
	var t T
	_, isGjsonResult = any(t).(gjson.Result)
	isString := reflect.TypeFor[T]().Kind() == reflect.String

	if !isGjsonResult && !isString {
		schema = executor.ToJSONSchema[T]()
	}

	return schema
}

var (
	WithContextVars = opts.ForName[ExecutionContext, types.ContextVars]("contextVars")
	Streaming       = opts.ForName[ExecutionContext, bool]("stream")
	WithMaxTurns    = opts.ForName[ExecutionContext, int]("maxTurns")
)

func StructuredOutput[T any](name, description string) opts.Option[ExecutionContext] {
	return opts.Type[ExecutionContext](func(s *ExecutionContext) error {
		schema := jsonSchema[T]()
		if schema != nil {
			s.responseSchema = &provider.StructuredOutput{
				Name:        name,
				Description: description,
				Schema:      schema,
			}
		}
		return nil
	})
}
