package owl

import (
	"strings"
	"text/template"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/provider/openai"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
	"github.com/fogfish/opts"
)

var _ api.Owl = (*defaultOwl)(nil)

// defaultOwl represents an agent with specific attributes and capabilities.
// It includes the agent's name, model, instructions, function definitions, tool choice,
// and whether it supports parallel tool calls.
type defaultOwl struct {
	name              string
	model             api.Model
	instructions      string
	tools             []tool.Definition
	parallelToolCalls bool
}

// Name returns the agent's name.
func (a *defaultOwl) Name() string {
	return a.name
}

// Model returns the agent's model.
func (a *defaultOwl) Model() api.Model {
	return a.model
}

// Tools returns the agent's function definitions.
func (a *defaultOwl) Tools() []tool.Definition {
	return a.tools
}

// ParallelToolCalls returns whether the agent supports parallel tool calls.
func (a *defaultOwl) ParallelToolCalls() bool {
	return a.parallelToolCalls
}

// RenderInstructions renders the agent's instructions with the provided context variables.
func (a *defaultOwl) RenderInstructions(cv types.ContextVars) (string, error) {
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
	Name              = opts.ForName[defaultOwl, string]("name")
	Model             = opts.ForName[defaultOwl, api.Model]("model")
	Instructions      = opts.ForName[defaultOwl, string]("instructions")
	ParallelToolCalls = opts.ForName[defaultOwl, bool]("parallelToolCalls")
)

func Tools(tool tool.Definition, extraTools ...tool.Definition) opts.Option[defaultOwl] {
	return opts.Type[defaultOwl](func(o *defaultOwl) error {
		o.tools = append(o.tools, tool)
		o.tools = append(o.tools, extraTools...)
		return nil
	})
}

// New creates a new DefaultOwl with the provided parameters.
func New(options ...opts.Option[defaultOwl]) api.Owl {
	owl := &defaultOwl{
		model:             openai.GPT4oMini(),
		parallelToolCalls: true,
	}
	if err := opts.Apply(owl, options); err != nil {
		panic(err)
	}
	return owl
}
