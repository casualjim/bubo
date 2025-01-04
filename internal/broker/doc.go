// Package broker implements a pub/sub message broker for distributing events
// between AI agents, tools, and other system components. It provides a clean,
// minimal interface for topic-based event distribution with context awareness.
//
// Design decisions:
//   - Context-first: All operations accept context.Context for cancellation/timeout
//   - Topic-based: Events are distributed through named topics for logical separation
//   - Clean interfaces: Minimal, focused interfaces for each responsibility
//   - Hook integration: Direct support for events.Hook for event handling
//   - Subscription management: Explicit subscription lifecycle with cleanup
//   - Thread safety: Safe for concurrent publishing and subscribing
//
// Interface hierarchy:
//   - Broker: Top-level interface for accessing topics
//     └── Topic: Interface for publishing/subscribing to events
//     └── Subscription: Interface for managing subscriptions
//
// Key concepts:
//   - Topics provide isolated channels for specific event streams
//   - Subscriptions are managed explicitly with unique IDs
//   - Hooks define how events are processed by subscribers
//   - Context support enables proper cleanup and cancellation
//
// Example usage:
//
//	// Create a broker and get a topic
//	broker := NewLocalBroker()
//	topic := broker.Topic(ctx, "agent-events")
//
//	// Create a subscription with a hook
//	hook := &MyEventHandler{}
//	sub, err := topic.Subscribe(ctx, hook)
//	if err != nil {
//	    return err
//	}
//	defer sub.Unsubscribe() // Ensure cleanup
//
//	// Publish events to the topic
//	event := events.Request[messages.UserMessage]{
//	    RunID:  uuid.New(),
//	    TurnID: uuid.New(),
//	    Message: messages.UserMessage{...},
//	}
//	if err := topic.Publish(ctx, event); err != nil {
//	    return err
//	}
//
// The broker package is designed to be internal to avoid exposing implementation
// details while providing a robust foundation for event distribution throughout
// the system. It works closely with the events package to ensure type-safe
// event handling and proper context management.
package broker
