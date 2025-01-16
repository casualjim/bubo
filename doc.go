/*
Package bubo provides a framework for building and orchestrating AI agents with a focus on
reliability, extensibility, and maintainable workflows.

The package implements a robust foundation for creating AI-powered applications through several
key abstractions:

  - Agents: Autonomous entities that can process tasks and make decisions
  - Workflows: Sequences of steps that coordinate agent interactions
  - Tools: Extensible capabilities that agents can use
  - Events: Communication system between components
  - Memory: Context retention across interactions

# Basic Usage

A typical workflow involves creating agents, defining their capabilities, and orchestrating
their interactions:

	englishAgent := agent.New(
		agent.Name("English Agent"),
		agent.Model(openai.GPT4oMini()),
		agent.Instructions("You only speak English"),
		agent.Tools(transferTool),
	)

	p := bubo.New(
		bubo.Agents(englishAgent),
		bubo.Steps(
			bubo.Step(englishAgent.Name(), "Process this text"),
		),
	)

	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
		// Handle error
	}

# Architecture

The package is built around several core concepts:

1. Execution Engine (execution.go)
  - Manages the lifecycle of workflows
  - Handles state transitions and error recovery
  - Coordinates between agents and tools

2. Hooks (hook.go)
  - Provides extension points for workflow customization
  - Enables monitoring and logging
  - Allows integration with external systems

3. Promises (promise.go)
  - Implements asynchronous operation handling
  - Manages concurrent agent tasks
  - Provides error propagation

4. Tasks (task.go)
  - Represents units of work
  - Encapsulates agent instructions
  - Manages execution context

5. Knots (knot.go)
  - Implements workflow synchronization points
  - Manages agent handoffs
  - Coordinates multi-agent interactions

# Integration

Bubo integrates with several backend systems:

  - NATS for message brokering
  - Temporal for workflow orchestration
  - Various AI providers (e.g., OpenAI)

# Examples

The examples directory contains various usage patterns:

  - Basic agent interactions
  - Context variable handling
  - Function calling
  - Agent handoff scenarios
  - Temporal workflow integration
  - Triage system implementation

See the examples directory for complete implementations.

# Best Practices

1. Agent Design
  - Keep agent instructions focused and specific
  - Use tools for extending agent capabilities
  - Implement proper error handling

2. Workflow Management
  - Break complex workflows into manageable steps
  - Use hooks for monitoring and debugging
  - Implement proper context cancellation

3. Tool Implementation
  - Keep tools simple and focused
  - Document tool capabilities clearly
  - Use the bubo-tool-gen for consistent generation

4. Error Handling
  - Always check for errors from Run()
  - Implement proper cleanup in hooks
  - Use context for cancellation

# Thread Safety

The package is designed to be thread-safe when used correctly:
  - Agents can be shared across goroutines
  - Hooks should be implemented in a thread-safe manner
  - Context is used for cancellation and deadlines

For more information about specific components, see their respective documentation:
  - agent.Agent for agent implementation details
  - provider.Provider for AI provider integration
  - tool.Tool for implementing custom capabilities
*/
package bubo
