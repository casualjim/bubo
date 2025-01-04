package bubo

import (
	"context"
	"fmt"
	"log/slog"
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

type ConversationStep struct {
	owlName string
	task    string
}

func Step(owlName string, task string) ConversationStep {
	return ConversationStep{
		owlName: owlName,
		task:    task,
	}
}

type Parliament struct {
	owls  *haxmap.Map[string, api.Owl]
	steps []ConversationStep
}

func Owls(owl api.Owl, extraOwls ...api.Owl) opts.Option[Parliament] {
	return opts.Type[Parliament](func(o *Parliament) error {
		o.owls.Set(owl.Name(), owl)
		for elem := range slices.Values(extraOwls) {
			o.owls.Set(elem.Name(), elem)
		}
		return nil
	})
}

func Steps(step ConversationStep, extraSteps ...ConversationStep) opts.Option[Parliament] {
	return opts.Type[Parliament](func(o *Parliament) error {
		o.steps = append(o.steps, step)
		o.steps = append(o.steps, extraSteps...)
		return nil
	})
}

func New(options ...opts.Option[Parliament]) *Parliament {
	p := &Parliament{
		owls: haxmap.New[string, api.Owl](),
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

func (e *ExecutionContext) createCommand(owl api.Owl, mem *shorttermmemory.Aggregator) (executor.RunCommand, error) {
	cmd, err := executor.NewRunCommand(owl, mem, e.hook)
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

func (p *Parliament) Run(ctx context.Context, rc ExecutionContext) error {
	defer rc.onClose(ctx)

	maxItems := len(p.steps) - 1

	for i, step := range p.steps {
		var promise executor.Promise
		var schema *provider.StructuredOutput
		if i < maxItems {
			slog.Debug("using noop promise, not the last step")
			promise = noopPromise{}
		} else {
			slog.Debug("using input promise, last step")
			promise = rc.promise
			schema = rc.responseSchema
		}

		if err := p.runStep(ctx, step.owlName, step.task, ExecutionContext{
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

func (p *Parliament) runStep(ctx context.Context, owlName, prompt string, rc ExecutionContext) error {
	owl, found := p.owls.Get(owlName)
	if !found {
		return fmt.Errorf("owl %s not found", owlName)
	}

	state := shorttermmemory.New()

	state.AddUserPrompt(messages.New().UserPrompt(prompt))
	cmd, err := rc.createCommand(owl, state)
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
