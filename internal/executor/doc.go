// Package executor provides the core execution engine for AI agent operations,
// implementing a robust system for running commands with support for streaming,
// tool calls, and asynchronous operations through a Future/Promise pattern.
//
// Design decisions:
//   - Command pattern: Encapsulates execution parameters in RunCommand struct
//   - Future/Promise: Async operations with type-safe result handling
//   - Structured output: JSON Schema validation for responses
//   - Context awareness: All operations respect context cancellation
//   - Thread safety: Concurrent execution support with proper synchronization
//   - Flexible unmarshaling: Support for different response types (JSON, string, gjson)
//
// Key components:
//
//   - Executor: Interface defining the core execution contract
//     ├── Run: Executes agent commands with streaming support
//     └── handleToolCalls: Manages tool invocations during execution
//
//   - RunCommand: Configuration for execution
//     ├── Agent: The AI agent to execute
//     ├── Thread: Memory aggregator for context
//     ├── Stream: Enable/disable streaming mode
//     └── Hook: Event handler for execution lifecycle
//
//   - Future/Promise pattern:
//     ├── CompletableFuture: Combined interface for async operations
//     ├── Promise: Write interface for results
//     └── Future: Read interface for retrieving results
//
// Example usage:
//
//	// Create and configure a run command
//	cmd, err := NewRunCommand(agent, thread, hook)
//	if err != nil {
//	    return err
//	}
//	cmd = cmd.WithStream(true).
//	    WithMaxTurns(5).
//	    WithContextVariables(vars)
//
//	// Create a future for the result
//	future := NewFuture(DefaultUnmarshal[MyResponse]())
//
//	// Execute the command
//	if err := executor.Run(ctx, cmd, future); err != nil {
//	    return err
//	}
//
//	// Get the result (blocks until complete)
//	result, err := future.Get()
//
// The package is designed to be internal, providing the execution engine while
// keeping implementation details private. It handles:
//   - Command execution lifecycle
//   - Asynchronous operation management
//   - Tool call coordination
//   - Response validation and unmarshaling
//   - Event distribution through hooks
//   - Context and cancellation management
package executor
