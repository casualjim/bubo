package bubo

import (
	"context"
	"reflect"
	"slices"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/broker"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/fogfish/opts"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"
)

type Step[T any] struct {
	owlName  string            //nolint:unused // Reserved for future use
	task     string            //nolint:unused // Reserved for future use
	executor executor.Executor //nolint:unused // Reserved for future use
}

func (s *Step[T]) step() {}

type step interface { //nolint:unused // Reserved for future use
	step()
}

type Parliament struct {
	owls  []api.Owl
	steps []step //nolint:unused // Reserved for future use
}

func WithOwls(owl api.Owl, extraOwls ...api.Owl) opts.Option[Parliament] {
	return opts.Type[Parliament](func(o *Parliament) error {
		o.owls = append(o.owls, owl)
		o.owls = append(o.owls, extraOwls...)
		return nil
	})
}

func WithStep(owlName string, task string) opts.Option[Parliament] {
	return opts.Type[Parliament](func(o *Parliament) error {
		return nil
	})
}

func New(options ...opts.Option[Parliament]) *Parliament {
	p := &Parliament{}
	if err := opts.Apply(p, options); err != nil {
		panic(err)
	}
	return p
}

func WithStructuredOutput[T any](output T) opts.Option[executor.RunCommand] {
	return opts.Type[executor.RunCommand](func(o *executor.RunCommand) error {
		o.ResponseSchema = jsonSchema[T]()
		return nil
	})
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

type RunConfig struct {
	executor       executor.Executor
	createCommand  func(api.Owl, *shorttermmemory.Aggregator) (executor.RunCommand, error)
	promise        executor.Promise
	responseSchema *jsonschema.Schema
}

// can't type alias this (yet) because of the type parameter

type Future[T any] interface {
	Get() (T, error)
}

func Local[T any](hook Hook[T]) (Future[T], RunConfig) {
	fut := executor.NewFuture(executor.DefaultUnmarshal[T]())

	return fut, RunConfig{
		executor:       executor.NewLocal(broker.Local()),
		promise:        fut,
		responseSchema: jsonSchema[T](),
		createCommand: func(o api.Owl, a *shorttermmemory.Aggregator) (executor.RunCommand, error) {
			return executor.NewRunCommand(o, a, hook)
		},
	}
}

func (p *Parliament) Run(ctx context.Context, prompt string, rc RunConfig) error {
	state := shorttermmemory.New()
	for owl := range slices.Values(p.owls) {
		state.AddUserPrompt(messages.New().UserPrompt(prompt))
		cmd, err := rc.createCommand(owl, state)
		if err != nil {
			return err
		}

		if err := rc.executor.Run(ctx, cmd, rc.promise); err != nil {
			return err
		}
	}
	return nil
}

type Hook[T any] interface {
	events.Hook
	OnResult(ctx context.Context, result T)
}
