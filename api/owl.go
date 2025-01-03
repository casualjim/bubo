package api

import (
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
)

// Owl represents the core interface for AI agents in the system. It defines the essential
// capabilities and configuration options that every agent must implement.
//
// Design decisions:
//   - Minimal interface: Only includes essential methods needed for agent operation
//   - Immutable configuration: Methods return values rather than allowing runtime changes
//   - Flexible instruction rendering: Supports dynamic instructions based on context
//   - Optional parallel execution: Agents can opt-in to parallel tool execution
//
// Example usage:
//
//	agent := NewAgent(config)
//	if agent.ParallelToolCalls() {
//	    // Handle parallel tool execution
//	}
//	instructions, err := agent.RenderInstructions(contextVars)
//	if err != nil {
//	    // Handle error
//	}
//
// The interface is designed to be implementation-agnostic, allowing different types
// of agents (e.g., local models, API-based services) to provide these capabilities
// while maintaining a consistent integration surface.
type Owl interface {
	// Name returns the agent's unique identifier.
	// This name should be consistent across sessions and is used for logging,
	// debugging, and distinguishing between multiple agents in the system.
	Name() string

	// Model returns the configuration and capabilities of the underlying AI model.
	// This includes model-specific settings, limitations, and supported features
	// that affect how the agent operates.
	Model() Model

	// Tools returns the set of functions this agent can execute.
	// These definitions describe the available tools, their parameters,
	// and how they should be invoked. The agent uses these definitions
	// to understand its capabilities and make informed decisions.
	Tools() []tool.Definition

	// ParallelToolCalls indicates if this agent can execute multiple tools concurrently.
	// When true, the agent may batch compatible tool calls together for improved performance.
	// This is particularly useful for I/O-bound operations or independent tasks.
	ParallelToolCalls() bool

	// RenderInstructions generates the agent's operational instructions using the provided context.
	// The context variables allow for dynamic customization of the agent's behavior
	// based on runtime conditions, user preferences, or system state.
	//
	// Returns an error if the instructions cannot be rendered with the given context,
	// for example if required variables are missing or template syntax is invalid.
	RenderInstructions(types.ContextVars) (string, error)
}
