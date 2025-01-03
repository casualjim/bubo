// Package events provides a pub/sub event system for AI agent interactions,
// supporting type-safe event handling with rich metadata and serialization.
// It builds on top of the provider package's streaming events, adding sender
// tracking and pub/sub capabilities.
//
// Design decisions:
//   - Type safety: Generic event types ensure compile-time correctness
//   - Rich metadata: Every event includes run/turn IDs, timestamps, and optional metadata
//   - Sender tracking: Events maintain origin information through the system
//   - Efficient JSON: Custom marshaling with pre-allocated type markers
//   - Error context: Errors preserve full execution context for debugging
//   - Provider integration: Seamless conversion from provider.StreamEvent
//
// Event hierarchy:
//   - Event: Base interface for all pub/sub events
//     ├── Delim: Stream boundary markers
//     ├── Chunk[T]: Incremental response fragments
//     ├── Request[T]: Incoming requests (user prompts, tool calls)
//     ├── Response[T]: Complete responses with context
//     ├── Result[T]: Final computation results
//     └── Error: Error events with preserved context
//
// Each event type includes:
//   - RunID: Unique identifier for the execution run
//   - TurnID: Identifier for the specific interaction turn
//   - Timestamp: When the event occurred
//   - Sender: Origin of the event (agent, tool, etc.)
//   - Meta: Optional structured metadata
//
// Example usage:
//
//	// Convert provider events to pub/sub events
//	providerEvent := provider.Chunk[messages.AssistantMessage]{...}
//	pubsubEvent := events.FromStreamEvent(providerEvent, "openai")
//
//	// Create and handle events
//	event := events.Request[messages.UserMessage]{
//	    RunID:  uuid.New(),
//	    TurnID: uuid.New(),
//	    Message: messages.UserMessage{
//	        Content: messages.ContentOrParts{Content: "Hello"},
//	    },
//	    Sender: "user",
//	}
//
//	// Type-safe event handling
//	switch e := event.(type) {
//	case events.Request[messages.UserMessage]:
//	    // Handle user request
//	case events.Response[messages.AssistantMessage]:
//	    // Handle assistant response
//	case events.Error:
//	    // Handle error with context
//	}
//
// The package is designed to work seamlessly with the provider and messages
// packages, providing a complete system for handling AI agent interactions
// with proper typing, context preservation, and error handling.
package events
