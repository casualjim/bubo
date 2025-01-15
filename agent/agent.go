package agent

import (
	"strings"
	"text/template"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/provider/openai"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
	"github.com/fogfish/opts"
)

var _ api.Agent = (*defaultAgent)(nil)

// defaultAgent represents an agent with specific attributes and capabilities.
// It includes the agent's name, model, instructions, function definitions, tool choice,
// and whether it supports parallel tool calls.
type defaultAgent struct {
	name              string
	model             api.Model
	instructions      string
	tools             []tool.Definition
	parallelToolCalls bool
}

// Name returns the agent's name.
func (a *defaultAgent) Name() string {
	return a.name
}

// Model returns the agent's model.
func (a *defaultAgent) Model() api.Model {
	return a.model
}

// Tools returns the agent's function definitions.
func (a *defaultAgent) Tools() []tool.Definition {
	return a.tools
}

func (a *defaultAgent) Instructions() string {
	return a.instructions
}

// ParallelToolCalls returns whether the agent supports parallel tool calls.
func (a *defaultAgent) ParallelToolCalls() bool {
	return a.parallelToolCalls
}

// RenderInstructions renders the agent's instructions with the provided context variables.
func (a *defaultAgent) RenderInstructions(cv types.ContextVars) (string, error) {
	if !strings.Contains(a.instructions, "{{") {
		return a.instructions, nil
	}
	return renderTemplate("instructions", a.instructions, cv)
}

func renderTemplate(name, templateStr string, cv types.ContextVars) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, cv); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var (
	Name              = opts.ForName[defaultAgent, string]("name")
	Model             = opts.ForName[defaultAgent, api.Model]("model")
	Instructions      = opts.ForName[defaultAgent, string]("instructions")
	ParallelToolCalls = opts.ForName[defaultAgent, bool]("parallelToolCalls")
)

func Tools(tool tool.Definition, extraTools ...tool.Definition) opts.Option[defaultAgent] {
	return opts.Type[defaultAgent](func(o *defaultAgent) error {
		o.tools = append(o.tools, tool)
		o.tools = append(o.tools, extraTools...)
		return nil
	})
}

// New creates a new DefaultAgent with the provided parameters.
func New(options ...opts.Option[defaultAgent]) api.Agent {
	agent := &defaultAgent{
		model:             openai.GPT4oMini(),
		parallelToolCalls: true,
	}
	if err := opts.Apply(agent, options); err != nil {
		panic(err)
	}
	return agent
}
