// Package bubo provides a framework for building conversational AI agents that can interact
// in a structured manner. It supports multi-agent conversations, structured output,
// and flexible execution contexts.
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

// Knot represents a conversational workflow that coordinates multiple AI agents
// through a series of predefined steps. It manages agent registration and execution
// of conversation steps in sequence.
type Knot struct {
	name   string                         // The name of the conversation initiator
	agents *haxmap.Map[string, api.Agent] // Registry of available agents
	steps  []ConversationStep             // Ordered sequence of conversation steps
}

// Agents creates an option to register one or more agents with the Knot.
// It requires at least one agent and can accept additional agents as variadic arguments.
func Agents(agent api.Agent, extraAgents ...api.Agent) opts.Option[Knot] {
	return opts.Type[Knot](func(o *Knot) error {
		o.agents.Set(agent.Name(), agent)
		for elem := range slices.Values(extraAgents) {
			o.agents.Set(elem.Name(), elem)
		}
		return nil
	})
}

// Steps creates an option to add one or more conversation steps to the Knot.
// Each step represents a single interaction in the conversation flow.
func Steps(step ConversationStep, extraSteps ...ConversationStep) opts.Option[Knot] {
	return opts.Type[Knot](func(o *Knot) error {
		o.steps = append(o.steps, step)
		o.steps = append(o.steps, extraSteps...)
		return nil
	})
}

// Name is an option to set the name of the conversation initiator.
var Name = opts.ForName[Knot, string]("name")

// New creates a new Knot instance with the provided options.
// It initializes a default name of "User" and an empty agent registry.
// Options can be used to customize the name, add agents, and define conversation steps.
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

// Run executes the conversation workflow defined by the Knot's steps.
// It processes each step sequentially using the provided execution context.
// The last step's output can be structured according to the response schema if specified.
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
