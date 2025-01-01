package bubo

import (
	"strings"
	"text/template"

	"github.com/casualjim/bubo/types"
)

// Owl represents an interface for an agent with various capabilities.
// It provides methods to retrieve the agent's name, model, instructions,
// function definitions, tool choice, and whether it supports parallel tool calls.
// available functions, tool choice, and whether parallel tool calls are supported.
type Owl interface {
	// Name returns the agent's name
	Name() string

	// Model returns the agent's model
	Model() Model

	// Instructions returns the agent's instructions
	Instructions() string

	// Tools returns the agent's function definitions
	Tools() []AgentToolDefinition

	// ToolChoice returns the agent's tool choice
	ToolChoice() string

	// ParallelToolCalls returns whether the agent supports parallel tool calls
	ParallelToolCalls() bool

	// RenderInstructions renders the agent's instructions with the provided context variables.
	RenderInstructions(types.ContextVars) (string, error)
}

var _ Owl = (*DefaultOwl)(nil)

// DefaultOwl represents an agent with specific attributes and capabilities.
// It includes the agent's name, model, instructions, function definitions, tool choice,
// and whether it supports parallel tool calls.
type DefaultOwl struct {
	name              string
	model             Model
	instructions      string
	tools             []AgentToolDefinition
	toolChoice        string
	parallelToolCalls bool
}

// Name returns the agent's name.
func (a *DefaultOwl) Name() string {
	return a.name
}

// Model returns the agent's model.
func (a *DefaultOwl) Model() Model {
	return a.model
}

// Instructions returns the agent's instructions.
func (a *DefaultOwl) Instructions() string {
	return a.instructions
}

// Tools returns the agent's function definitions.
func (a *DefaultOwl) Tools() []AgentToolDefinition {
	return a.tools
}

// ToolChoice returns the agent's tool choice.
func (a *DefaultOwl) ToolChoice() string {
	return a.toolChoice
}

// ParallelToolCalls returns whether the agent supports parallel tool calls.
func (a *DefaultOwl) ParallelToolCalls() bool {
	return a.parallelToolCalls
}

// RenderInstructions renders the agent's instructions with the provided context variables.
func (a *DefaultOwl) RenderInstructions(cv types.ContextVars) (string, error) {
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

// AddTool adds a function definition to the agent.
func (a *DefaultOwl) AddTool(f1 AgentToolDefinition, frest ...AgentToolDefinition) {
	a.tools = append(a.tools, append([]AgentToolDefinition{f1}, frest...)...)
}

// WithTool adds a function definition to the agent.
func (a *DefaultOwl) WithTool(f1 AgentToolDefinition, frest ...AgentToolDefinition) *DefaultOwl {
	a.tools = append(a.tools, append([]AgentToolDefinition{f1}, frest...)...)
	return a
}

// SetToolChoice sets the agent's tool choice.
func (a *DefaultOwl) SetToolChoice(toolChoice string) {
	a.toolChoice = toolChoice
}

// WithToolChoice sets the agent's tool choice.
func (a *DefaultOwl) WithToolChoice(toolChoice string) *DefaultOwl {
	a.toolChoice = toolChoice
	return a
}

// EnableParallelToolCalls enables parallel tool calls for the agent.
func (a *DefaultOwl) EnableParallelToolCalls() {
	a.parallelToolCalls = true
}

// DisableParallelToolCalls disables parallel tool calls for the agent.
func (a *DefaultOwl) DisableParallelToolCalls() {
	a.parallelToolCalls = false
}

// WithParallelToolCalls enables parallel tool calls for the agent.
func (a *DefaultOwl) WithParallelToolCalls() *DefaultOwl {
	a.parallelToolCalls = true
	return a
}

// WithoutParallelToolCalls disables parallel tool calls for the agent.
func (a *DefaultOwl) WithoutParallelToolCalls() *DefaultOwl {
	a.parallelToolCalls = false
	return a
}

// NewAgent creates a new DefaultAgent with the provided parameters.
func NewAgent(name string, model Model, instructions string) *DefaultOwl {
	if model == nil {
		panic("model is required")
	}
	return &DefaultOwl{
		name:              name,
		model:             model,
		instructions:      instructions,
		parallelToolCalls: true,
	}
}
