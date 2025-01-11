package executor

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/pkg/stdx"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/casualjim/bubo/provider"
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

func ToJSONSchema[T any]() *jsonschema.Schema {
	var v T
	schema := reflector.Reflect(v)
	return schema
}

func NewRunCommand(agent api.Agent, thread *shorttermmemory.Aggregator, hook events.Hook) (RunCommand, error) {
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
		return RunCommand{}, err
	}

	return RunCommand{
		id:       uuidx.New(),
		Agent:    agent,
		Thread:   thread,
		Hook:     hook,
		MaxTurns: math.MaxInt,
	}, nil
}

type RunCommand struct {
	id               uuid.UUID
	Agent            api.Agent
	Thread           *shorttermmemory.Aggregator
	StructuredOutput *provider.StructuredOutput
	Stream           bool
	MaxTurns         int
	ContextVariables types.ContextVars
	Hook             events.Hook
}

func (r *RunCommand) Validate() error {
	if r.Agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}
	if r.Thread == nil {
		return fmt.Errorf("thread cannot be nil")
	}
	if r.Hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}
	return nil
}

func (r *RunCommand) initializeContextVars() types.ContextVars {
	if r.ContextVariables != nil {
		return maps.Clone(r.ContextVariables)
	}
	return nil
}

func (r *RunCommand) ID() uuid.UUID {
	return r.id
}

func (r RunCommand) WithStream(stream bool) RunCommand {
	r.Stream = stream
	return r
}

func (r RunCommand) WithMaxTurns(maxTurns int) RunCommand {
	r.MaxTurns = maxTurns
	return r
}

func (r RunCommand) WithContextVariables(contextVariables types.ContextVars) RunCommand {
	r.ContextVariables = contextVariables
	return r
}

func (r RunCommand) WithStructuredOutput(output *provider.StructuredOutput) RunCommand {
	r.StructuredOutput = output
	return r
}

func DefaultUnmarshal[T any]() func([]byte) (T, error) {
	var responseUnmarshaler func([]byte) (T, error)

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
	}
	return responseUnmarshaler
}

type CompletableFuture[T any] interface {
	Future[T]
	Promise
}

type Promise interface {
	Complete(string)
	Error(error)
}

type Future[T any] interface {
	Get() (T, error)
}

type futState struct {
	value string
	err   error
}

type futResult[T any] struct {
	result T
	err    error
	done   bool
}

type future[T any] struct {
	unmarshal func([]byte) (T, error)
	ch        chan futState
	result    atomic.Value // holds *futResult[T]
	once      sync.Once
	mu        sync.Mutex
}

func NewFuture[T any](unmarshal func([]byte) (T, error)) CompletableFuture[T] {
	f := &future[T]{
		unmarshal: unmarshal,
		ch:        make(chan futState, 1),
	}
	f.result.Store(&futResult[T]{})
	return f
}

func (f *future[T]) Get() (T, error) {
	res := f.result.Load().(*futResult[T])
	if res.done {
		return res.result, res.err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring lock
	res = f.result.Load().(*futResult[T])
	if res.done {
		return res.result, res.err
	}

	r := <-f.ch
	var newResult futResult[T]
	if r.err != nil {
		newResult = futResult[T]{
			result: stdx.Zero[T](),
			err:    r.err,
			done:   true,
		}
	} else {
		result, err := f.unmarshal([]byte(r.value))
		newResult = futResult[T]{
			result: result,
			err:    err,
			done:   true,
		}
	}
	f.result.Store(&newResult)
	return newResult.result, newResult.err
}

func (f *future[T]) Complete(data string) {
	f.once.Do(func() {
		f.ch <- futState{value: data}
	})
}

func (f *future[T]) Error(err error) {
	f.once.Do(func() {
		f.ch <- futState{err: err}
	})
}

type Executor interface {
	Run(context.Context, RunCommand, Promise) error
	// Topic(context.Context, string) broker.Topic
	handleToolCalls(ctx context.Context, params toolCallParams) (api.Agent, error)
}
