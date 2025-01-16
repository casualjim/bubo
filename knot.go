package bubo

import (
	"context"
	"fmt"
	"slices"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/fogfish/opts"
)

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
