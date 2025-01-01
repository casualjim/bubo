package api

import (
	"github.com/casualjim/bubo/tool"
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

	// Tools returns the agent's function definitions
	Tools() []tool.Definition

	// ParallelToolCalls returns whether the agent supports parallel tool calls
	ParallelToolCalls() bool

	// RenderInstructions renders the agent's instructions with the provided context variables.
	RenderInstructions(types.ContextVars) (string, error)
}
