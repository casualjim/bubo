package executor

import (
	"context"

	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/messages"
)

type mockHook[T any] struct {
	events.Hook[T]
}

func (h *mockHook[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {}

func (h *mockHook[T]) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
}

func (h *mockHook[T]) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
}

func (h *mockHook[T]) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
}

func (h *mockHook[T]) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
}

func (h *mockHook[T]) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
}

func (h *mockHook[T]) OnError(ctx context.Context, err error) {}
