package bubo

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
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
	responseSchema *jsonschema.Schema
}

func (e *ExecutionContext) createCommand(owl api.Owl, mem *shorttermmemory.Aggregator) (executor.RunCommand, error) {
	return executor.NewRunCommand(owl, mem, e.hook)
}

type Future[T any] interface {
	// can't type alias this (yet) because of the type parameter

	Get() (T, error)
}

func Local[T any](hook Hook[T]) ExecutionContext {
	fut := executor.NewFuture(executor.DefaultUnmarshal[T]())
	go func() {
		val, err := fut.Get()
		if err != nil {
			hook.OnError(context.Background(), err)
			return
		}
		hook.OnResult(context.Background(), val)
	}()
	return ExecutionContext{
		executor:       executor.NewLocal(),
		responseSchema: jsonSchema[T](),
		hook:           hook,
		promise:        fut,
	}
}

func (p *Parliament) Run(ctx context.Context, rc ExecutionContext) error {
	for _, step := range p.steps {
		if err := p.runStep(ctx, step.owlName, step.task, rc); err != nil {
			return err
		}
	}
	return nil
}

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
	OnResult(ctx context.Context, result T)
}

func HookFuture[T any](hook Hook[T]) (Hook[T], Future[T]) {
	fhook := &hookFuture[T]{
		inner: hook,
	}

	fut := executor.NewFuture(executor.DefaultUnmarshal[T]())
	go func() {
		result, err := fut.Get()
		if err != nil {
			hook.OnError(context.Background(), err)
			return
		}
		hook.OnResult(context.Background(), result)
	}()
	fhook.fut = fut
	return fhook, fhook.fut
}

type hookFuture[T any] struct {
	inner Hook[T]
	fut   executor.CompletableFuture[T]
}

func (f *hookFuture[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	f.inner.OnUserPrompt(ctx, msg)
}

func (f *hookFuture[T]) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	f.inner.OnAssistantChunk(ctx, msg)
}

func (f *hookFuture[T]) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	f.inner.OnToolCallChunk(ctx, msg)
}

func (f *hookFuture[T]) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	f.inner.OnAssistantMessage(ctx, msg)
}

func (f *hookFuture[T]) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	f.inner.OnToolCallMessage(ctx, msg)
}

func (f *hookFuture[T]) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	f.inner.OnToolCallResponse(ctx, msg)
}

func (f *hookFuture[T]) OnError(ctx context.Context, err error) {
	f.inner.OnError(ctx, err)
}

func (f *hookFuture[T]) OnResult(ctx context.Context, result T) {
	f.inner.OnResult(ctx, result)
}

func (f hookFuture[T]) AsFuture() Future[T] {
	return f.fut
}

type AsFuture[T any] interface {
	AsFuture() Future[T]
}
