package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"

	"github.com/casualjim/bubo/pkg/messages"
	"github.com/casualjim/bubo/pkg/slogx"
)

// Hook defines the interface for handling all possible event types in the execution flow.
// This interface is deliberately designed without a base "no-op" implementation to ensure
// consumers make explicit decisions about handling each event type.
//
// Design decisions:
//  1. All methods must be implemented: This is a conscious choice to ensure compile-time
//     safety. When new event types are added, all implementations will need to be updated.
//  2. No provided no-op implementation: While it might be convenient to provide a NoOpHook,
//     doing so would undermine the interface's primary benefit of forcing conscious
//     decisions about event handling.
//  3. Complete coverage: The interface covers all possible event types to ensure no events
//     can be accidentally missed in implementations.
//
// Implementation guidelines:
//   - Implement all methods explicitly, even if some events don't require handling
//   - Consider logging or monitoring for events that aren't actively handled
//   - Be prepared for new methods to be added as the system evolves
//
// Example implementation:
//
//	type MyHandler struct{}
//
//	func (h *MyHandler) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
//	    // Explicit handling of user prompts
//	}
//	func (h *MyHandler) OnAssistantChunk(...) {
//	    // Explicit decision to not handle chunks
//	    log.Debug("ignoring assistant chunk")
//	}
//	// ... must implement all other methods
type Hook[T any] interface {
	OnUserPrompt(context.Context, messages.Message[messages.UserMessage])

	OnAssistantChunk(context.Context, messages.Message[messages.AssistantMessage])

	OnToolCallChunk(context.Context, messages.Message[messages.ToolCallMessage])

	OnAssistantMessage(context.Context, messages.Message[messages.AssistantMessage])

	OnToolCallMessage(context.Context, messages.Message[messages.ToolCallMessage])

	OnToolCallResponse(context.Context, messages.Message[messages.ToolResponse])

	OnResult(context.Context, T)

	OnError(context.Context, error)
}

func LoggingHook[T any]() Hook[T] {
	return &loggingHook[T]{}
}

type loggingHook[T any] struct{}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (loggingHook[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	slog.InfoContext(ctx, "User prompt", "message", mustJSON(msg))
}

func (loggingHook[T]) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	slog.InfoContext(ctx, "Assistant chunk", "message", mustJSON(msg))
}

func (loggingHook[T]) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	slog.InfoContext(ctx, "Tool call chunk", "message", mustJSON(msg))
}

func (loggingHook[T]) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	slog.InfoContext(ctx, "Assistant message", "message", mustJSON(msg))
}

func (loggingHook[T]) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	slog.InfoContext(ctx, "Tool call", "message", mustJSON(msg))
}

func (loggingHook[T]) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	slog.InfoContext(ctx, "Tool call response", "message", mustJSON(msg))
}

func (loggingHook[T]) OnResult(ctx context.Context, result T) {
	slog.InfoContext(ctx, "completion result", "result", mustJSON(result))
}

func (loggingHook[T]) OnError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "completion error", slogx.Error(err))
}

func NewCompositeHook[T any](hooks ...Hook[T]) Hook[T] {
	return CompositeHook[T](hooks)
}

// CompositeHook allows combining multiple hooks into a single hook implementation.
// Note: This is provided as a utility for combining hooks, not as a way to avoid
// implementing the full interface.
type CompositeHook[T any] []Hook[T]

func (c CompositeHook[T]) OnUserPrompt(ctx context.Context, up messages.Message[messages.UserMessage]) {
	for h := range slices.Values(c) {
		h.OnUserPrompt(ctx, up)
	}
}

func (c CompositeHook[T]) OnAssistantChunk(ctx context.Context, ac messages.Message[messages.AssistantMessage]) {
	for h := range slices.Values(c) {
		h.OnAssistantChunk(ctx, ac)
	}
}

func (c CompositeHook[T]) OnToolCallChunk(ctx context.Context, tc messages.Message[messages.ToolCallMessage]) {
	for h := range slices.Values(c) {
		h.OnToolCallChunk(ctx, tc)
	}
}

func (c CompositeHook[T]) OnAssistantMessage(ctx context.Context, am messages.Message[messages.AssistantMessage]) {
	for h := range slices.Values(c) {
		h.OnAssistantMessage(ctx, am)
	}
}

func (c CompositeHook[T]) OnToolCallMessage(ctx context.Context, tm messages.Message[messages.ToolCallMessage]) {
	for h := range slices.Values(c) {
		h.OnToolCallMessage(ctx, tm)
	}
}

func (c CompositeHook[T]) OnToolCallResponse(ctx context.Context, tr messages.Message[messages.ToolResponse]) {
	for h := range slices.Values(c) {
		h.OnToolCallResponse(ctx, tr)
	}
}

func (c CompositeHook[T]) OnResult(ctx context.Context, result T) {
	for h := range slices.Values(c) {
		h.OnResult(ctx, result)
	}
}

func (c CompositeHook[T]) OnError(ctx context.Context, err error) {
	for h := range slices.Values(c) {
		h.OnError(ctx, err)
	}
}
