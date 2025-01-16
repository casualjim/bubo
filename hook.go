// Package bubo provides a framework for building conversational AI agents that can interact
// in a structured manner. It supports multi-agent conversations, structured output,
// and flexible execution contexts.
package bubo

import (
	"context"

	"github.com/casualjim/bubo/events"
)

// Hook extends the events.Hook interface to provide type-safe result handling
// and cleanup functionality. It's parameterized by type T which represents
// the expected type of conversation results.
//
// Hooks serve several important purposes in the system:
//   - Monitoring conversation progress
//   - Handling intermediate and final results
//   - Managing resources and cleanup
//   - Implementing custom logging or metrics
//   - Integrating with external systems
//
// Example implementation:
//
//	type LoggingHook[T any] struct {
//	    events.BaseHook // Provides default implementations
//	    logger *log.Logger
//	}
//
//	func (h *LoggingHook[T]) OnResult(ctx context.Context, result T) {
//	    h.logger.Printf("Received result: %v", result)
//	}
//
//	func (h *LoggingHook[T]) OnClose(ctx context.Context) {
//	    h.logger.Print("Conversation completed")
//	}
//
// Usage:
//
//	hook := &LoggingHook[string]{
//	    logger: log.New(os.Stdout, "", log.LstdFlags),
//	}
//
//	p := bubo.New(
//	    bubo.Agents(agent),
//	    bubo.Steps(step),
//	)
//
//	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
//	    // Handle error
//	}
type Hook[T any] interface {
	// events.Hook provides the base event handling interface:
	//   - OnStart: Called when a conversation starts
	//   - OnError: Called when an error occurs
	//   - OnMessage: Called for each message in the conversation
	//   - OnToolCall: Called when a tool is invoked
	//   - OnToolResponse: Called when a tool returns a result
	events.Hook // Embeds the base Hook interface for common event handling

	// OnResult is called when a conversation step produces a result of type T.
	// This method allows for type-safe handling of conversation outputs.
	//
	// The context.Context parameter can be used for cancellation and timeouts.
	// The type parameter T represents the expected result type, which must match
	// the type parameter used when implementing the Hook interface.
	//
	// This method should:
	//   - Process results quickly to avoid blocking the conversation
	//   - Handle results in a thread-safe manner if shared across goroutines
	//   - Respect context cancellation
	//   - Not modify the result parameter as it may be shared
	OnResult(context.Context, T)

	// OnClose is called when a conversation workflow completes.
	// It provides an opportunity for cleanup and resource release.
	//
	// This method is called exactly once per conversation, regardless of whether
	// the conversation completed successfully or encountered an error. It should:
	//   - Release any resources acquired during the conversation
	//   - Close any open connections or files
	//   - Flush any buffered data
	//   - Complete any final logging or metrics
	//
	// The context.Context parameter may be used for timeout management during cleanup,
	// but implementations should attempt to clean up resources even if the context
	// is cancelled.
	OnClose(context.Context)
}
