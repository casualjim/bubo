package executor

import (
	"context"
	"errors"
	"math"
	"reflect"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/casualjim/bubo/types"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"
)

// Structured Outputs uses a subset of JSON schema
// These flags are necessary to comply with the subset
var reflector = jsonschema.Reflector{
	AllowAdditionalProperties: false,
	DoNotReference:            true,
}

func toJSONSchema[T any]() *jsonschema.Schema {
	var v T
	schema := reflector.Reflect(v)
	return schema
}

func NewRunCommand[T any](agent bubo.Owl, thread *shorttermmemory.Aggregator, hook events.Hook[T]) (RunCommand[T], error) {
	var err error
	if agent == nil {
		err = errors.Join(err, errors.New("agent is required"))
	}
	if thread == nil {
		err = errors.Join(err, errors.New("thread is required"))
	}
	if hook == nil {
		err = errors.Join(err, errors.New("hook is required"))
	}

	if err != nil {
		return RunCommand[T]{}, err
	}

	var responseUnmarshaler func([]byte) (T, error)
	var schema *jsonschema.Schema

	// Check if T is gjson.Result
	var isGjsonResult bool
	var t T
	_, isGjsonResult = any(t).(gjson.Result)
	isString := reflect.TypeFor[T]().Kind() == reflect.String

	if isGjsonResult {
		// For gjson.Result, we parse differently
		responseUnmarshaler = func(data []byte) (T, error) {
			result := gjson.ParseBytes(data)
			return any(result).(T), nil
		}
	} else if isString {
		// For string, we just return the string
		responseUnmarshaler = func(data []byte) (T, error) {
			return any(string(data)).(T), nil
		}
	} else {
		// For all other types, use standard JSON unmarshaling
		responseUnmarshaler = func(data []byte) (T, error) {
			var v T
			if err := json.Unmarshal(data, &v); err != nil {
				return v, err
			}
			return v, nil
		}
		schema = toJSONSchema[T]()
	}

	return RunCommand[T]{
		ID:                uuidx.New(),
		Agent:             agent,
		Thread:            thread,
		ResponseSchema:    schema,
		UnmarshalResponse: responseUnmarshaler,
		Hook:              hook,
		MaxTurns:          math.MaxInt,
	}, nil
}

type RunCommand[T any] struct {
	ID                uuid.UUID
	Agent             bubo.Owl
	Thread            *shorttermmemory.Aggregator
	ResponseSchema    *jsonschema.Schema
	UnmarshalResponse func([]byte) (T, error)
	Stream            bool
	MaxTurns          int
	ContextVariables  types.ContextVars
	Hook              events.Hook[T]
}

func (r RunCommand[T]) WithStream(stream bool) RunCommand[T] {
	r.Stream = stream
	return r
}

func (r RunCommand[T]) WithMaxTurns(maxTurns int) RunCommand[T] {
	r.MaxTurns = maxTurns
	return r
}

func (r RunCommand[T]) WithContextVariables(contextVariables types.ContextVars) RunCommand[T] {
	r.ContextVariables = contextVariables
	return r
}

type Executor[T any] interface {
	Run(context.Context, RunCommand[T]) error
}
