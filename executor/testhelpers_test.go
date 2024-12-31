package executor

import (
	"context"

	"github.com/casualjim/bubo/executor/pubsub"
	"github.com/casualjim/bubo/pkg/messages"
)

type mockHook[T any] struct {
	pubsub.Hook[T]
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
