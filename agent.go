package bubo

import (
	"os"
	"strings"
	"text/template"

	"github.com/goccy/go-json"
	"github.com/openai/openai-go"
)

type ContextVars map[string]any

func (cv ContextVars) String() string {
	jsonData, err := json.Marshal(cv)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// Agent represents an interface for an agent with various capabilities.
// It provides methods to retrieve the agent's name, model, instructions,
// function definitions, tool choice, and whether it supports parallel tool calls.
// available functions, tool choice, and whether parallel tool calls are supported.
type Agent interface {
	// Name returns the agent's name
	Name() string

	// Model returns the agent's model
	Model() string

	// Instructions returns the agent's instructions
	Instructions() string

	// Functions returns the agent's function definitions
	Functions() []AgentFunctionDefinition

	// ToolChoice returns the agent's tool choice
	ToolChoice() string

	// ParallelToolCalls returns whether the agent supports parallel tool calls
	ParallelToolCalls() bool

	RenderInstructions(ContextVars) (string, error)
}

// DefaultAgent represents an agent with specific attributes and capabilities.
// It includes the agent's name, model, instructions, function definitions, tool choice,
// and whether it supports parallel tool calls.
type DefaultAgent struct {
	name              string
	model             string
	instructions      string
	functions         []AgentFunctionDefinition
	toolChoice        string
	parallelToolCalls bool
}

// Name returns the agent's name.
func (a *DefaultAgent) Name() string {
	return a.name
}

// Model returns the agent's model.
func (a *DefaultAgent) Model() string {
	return a.model
}

// Instructions returns the agent's instructions.
func (a *DefaultAgent) Instructions() string {
	return a.instructions
}

// Functions returns the agent's function definitions.
func (a *DefaultAgent) Functions() []AgentFunctionDefinition {
	return a.functions
}

// ToolChoice returns the agent's tool choice.
func (a *DefaultAgent) ToolChoice() string {
	return a.toolChoice
}

// ParallelToolCalls returns whether the agent supports parallel tool calls.
func (a *DefaultAgent) ParallelToolCalls() bool {
	return a.parallelToolCalls
}

func (a *DefaultAgent) RenderInstructions(cv ContextVars) (string, error) {
	if !strings.Contains(a.instructions, "{{") {
		return a.instructions, nil
	}
	return renderTemplate("instructions", a.instructions, cv)
}

func renderTemplate(name, templateStr string, cv ContextVars) (string, error) {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, cv); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// AddFunction adds a function definition to the agent.
func (a *DefaultAgent) AddFunction(f1 AgentFunctionDefinition, frest ...AgentFunctionDefinition) {
	a.functions = append(a.functions, append([]AgentFunctionDefinition{f1}, frest...)...)
}

// WithFunction adds a function definition to the agent.
func (a *DefaultAgent) WithFunction(f1 AgentFunctionDefinition, frest ...AgentFunctionDefinition) *DefaultAgent {
	a.functions = append(a.functions, append([]AgentFunctionDefinition{f1}, frest...)...)
	return a
}

// SetToolChoice sets the agent's tool choice.
func (a *DefaultAgent) SetToolChoice(toolChoice string) {
	a.toolChoice = toolChoice
}

// WithToolChoice sets the agent's tool choice.
func (a *DefaultAgent) WithToolChoice(toolChoice string) *DefaultAgent {
	a.toolChoice = toolChoice
	return a
}

// EnableParallelToolCalls enables parallel tool calls for the agent.
func (a *DefaultAgent) EnableParallelToolCalls() {
	a.parallelToolCalls = true
}

// DisableParallelToolCalls disables parallel tool calls for the agent.
func (a *DefaultAgent) DisableParallelToolCalls() {
	a.parallelToolCalls = true
}

// WithParallelToolCalls enables parallel tool calls for the agent.
func (a *DefaultAgent) WithParallelToolCalls() *DefaultAgent {
	a.parallelToolCalls = true
	return a
}

// WithoutParallelToolCalls disables parallel tool calls for the agent.
func (a *DefaultAgent) WithoutParallelToolCalls() *DefaultAgent {
	a.parallelToolCalls = false
	return a
}

// NewAgent creates a new DefaultAgent with the provided parameters.
func NewAgent(name, model, instructions string) *DefaultAgent {
	if model == "" {
		model = os.Getenv("OPENAI_DEFAULT_MODEL")
		if model == "" {
			model = openai.ChatModelGPT4oMini
		}
	}
	return &DefaultAgent{
		name:              name,
		model:             model,
		instructions:      instructions,
		parallelToolCalls: true,
	}
}
