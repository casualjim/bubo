package executor

import (
	"context"

	"github.com/casualjim/bubo/executor/pubsub"
	"github.com/casualjim/bubo/pkg/messages"
)

type mockHook struct {
	pubsub.Hook
}

func (h *mockHook) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {}

func (h *mockHook) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
}

func (h *mockHook) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
}

func (h *mockHook) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
}

func (h *mockHook) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
}

func (h *mockHook) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
}

func (h *mockHook) OnError(ctx context.Context, err error) {}
