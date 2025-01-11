package bubo

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sync"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/fogfish/opts"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"
)

type task interface {
	task()
}

type stringTask string

func (s stringTask) task() {}

type messageTask messages.Message[messages.UserMessage]

func (m messageTask) task() {}

type ConversationStep struct {
	agentName string
	task      task
}

func Step[T Task](agentName string, tsk T) ConversationStep {
	var t task
	switch xt := any(tsk).(type) {
	case string:
		t = stringTask(xt)
	case messages.Message[messages.UserMessage]:
		t = messageTask(xt)
	default:
		panic(fmt.Sprintf("invalid task type: %T", xt))
	}
	return ConversationStep{
		agentName: agentName,
		task:      t,
	}
}

type Task interface {
	~string | messages.Message[messages.UserMessage]
}

type Knot struct {
	name   string
	agents *haxmap.Map[string, api.Agent]
	steps  []ConversationStep
}

func Agents(agent api.Agent, extraAgents ...api.Agent) opts.Option[Knot] {
	return opts.Type[Knot](func(o *Knot) error {
		o.agents.Set(agent.Name(), agent)
		for elem := range slices.Values(extraAgents) {
			o.agents.Set(elem.Name(), elem)
		}
		return nil
	})
}

func Steps(step ConversationStep, extraSteps ...ConversationStep) opts.Option[Knot] {
	return opts.Type[Knot](func(o *Knot) error {
		o.steps = append(o.steps, step)
		o.steps = append(o.steps, extraSteps...)
		return nil
	})
}

var Name = opts.ForName[Knot, string]("name")

func New(options ...opts.Option[Knot]) *Knot {
	p := &Knot{
		name:   "User",
		agents: haxmap.New[string, api.Agent](),
	}
	if err := opts.Apply(p, options); err != nil {
		panic(err)
	}
	return p
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

type Future[T any] interface {
	// can't type alias this (yet) because of the type parameter

	Get() (T, error)
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

type deferredPromise[T any] struct {
	promise executor.CompletableFuture[T]
	hook    Hook[T]
	mu      sync.Mutex
	value   string
	err     error
	once    sync.Once
}

func (d *deferredPromise[T]) Forward(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		d.promise.Error(d.err)
		return
	}

	d.promise.Complete(d.value)
	res, err := d.promise.Get()
	if err != nil {
		d.hook.OnError(ctx, err)
		return
	}
	d.hook.OnResult(ctx, res)
}

func (d *deferredPromise[T]) Complete(result string) {
	d.once.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.value = result
	})
}

func (d *deferredPromise[T]) Error(err error) {
	d.once.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.err = err
	})
}

func Local[T any](hook Hook[T], options ...opts.Option[ExecutionContext]) ExecutionContext {
	fut := executor.NewFuture(executor.DefaultUnmarshal[T]())
	dp := &deferredPromise[T]{
		promise: fut,
		hook:    hook,
	}

	execCtx := ExecutionContext{
		executor: executor.NewLocal(),
		hook:     hook,
		promise:  dp,
		onClose: func(ctx context.Context) {
			dp.Forward(ctx)
			hook.OnClose(ctx)
		},
	}

	if err := opts.Apply(&execCtx, options); err != nil {
		panic(err)
	}

	return execCtx
}

func (p *Knot) Run(ctx context.Context, rc ExecutionContext) error {
	defer rc.onClose(ctx)

	maxItems := len(p.steps) - 1

	for i, step := range p.steps {
		var promise executor.Promise
		var schema *provider.StructuredOutput
		if i < maxItems {
			promise = noopPromise{}
		} else {
			promise = rc.promise
			schema = rc.responseSchema
		}

		if err := p.runStep(ctx, step.agentName, step.task, ExecutionContext{
			executor:       rc.executor,
			hook:           rc.hook,
			promise:        promise,
			contextVars:    rc.contextVars,
			onClose:        rc.onClose,
			responseSchema: schema,
		}); err != nil {
			return err
		}
	}

	return nil
}

type noopPromise struct{}

func (noopPromise) Complete(string) {}
func (noopPromise) Error(error)     {}

func (p *Knot) runStep(ctx context.Context, agentName string, prompt task, rc ExecutionContext) error {
	agent, found := p.agents.Get(agentName)
	if !found {
		return fmt.Errorf("agent %s not found", agentName)
	}

	state := shorttermmemory.New()

	var message messages.Message[messages.UserMessage]
	switch tsk := prompt.(type) {
	case stringTask:
		message = messages.New().WithSender(p.name).UserPrompt(string(tsk))
	case messageTask:
		message = messages.Message[messages.UserMessage](tsk)
	default:
		return fmt.Errorf("unknown task type %T", tsk)
	}
	state.AddUserPrompt(message)
	rc.hook.OnUserPrompt(ctx, message)

	cmd, err := rc.createCommand(agent, state)
	if err != nil {
		return err
	}

	if err := rc.executor.Run(ctx, cmd, rc.promise); err != nil {
		return err
	}
	return nil
}

type Hook[T any] interface {
	events.Hook
	OnResult(context.Context, T)
	OnClose(context.Context)
}
