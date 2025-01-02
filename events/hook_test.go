package events

import (
	"context"
	"fmt"
	"testing"

	"github.com/casualjim/bubo/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHook struct {
	userPromptCalled     bool
	assistantChunkCalled bool
	toolCallChunkCalled  bool
	assistantMsgCalled   bool
	toolCallMsgCalled    bool
	toolCallRespCalled   bool
	errorCalled          bool
	lastUserPrompt       messages.Message[messages.UserMessage]
	lastAssistantChunk   messages.Message[messages.AssistantMessage]
	lastToolCallChunk    messages.Message[messages.ToolCallMessage]
	lastAssistantMsg     messages.Message[messages.AssistantMessage]
	lastToolCallMsg      messages.Message[messages.ToolCallMessage]
	lastToolCallResp     messages.Message[messages.ToolResponse]
	lastError            error
}

func (m *mockHook) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	m.userPromptCalled = true
	m.lastUserPrompt = msg
}

func (m *mockHook) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	m.assistantChunkCalled = true
	m.lastAssistantChunk = msg
}

func (m *mockHook) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	m.toolCallChunkCalled = true
	m.lastToolCallChunk = msg
}

func (m *mockHook) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	m.assistantMsgCalled = true
	m.lastAssistantMsg = msg
}

func (m *mockHook) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	m.toolCallMsgCalled = true
	m.lastToolCallMsg = msg
}

func (m *mockHook) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	m.toolCallRespCalled = true
	m.lastToolCallResp = msg
}

func (m *mockHook) OnResult(ctx context.Context, result interface{}) {
}

func (m *mockHook) OnError(ctx context.Context, err error) {
	m.errorCalled = true
	m.lastError = err
}

// func TestLoggingHook(t *testing.T) {
// 	hook := LoggingHook[any]()
// 	ctx := context.Background()

// 	t.Run("OnUserPrompt", func(t *testing.T) {
// 		msg := messages.Message[messages.UserMessage]{
// 			Payload: messages.UserMessage{
// 				Content: messages.ContentOrParts{Content: "test prompt"},
// 			},
// 		}
// 		require.NotPanics(t, func() {
// 			hook.OnUserPrompt(ctx, msg)
// 		})
// 	})

// 	t.Run("OnAssistantChunk", func(t *testing.T) {
// 		msg := messages.Message[messages.AssistantMessage]{
// 			Payload: messages.AssistantMessage{
// 				Content: messages.AssistantContentOrParts{Content: "test chunk"},
// 			},
// 		}
// 		require.NotPanics(t, func() {
// 			hook.OnAssistantChunk(ctx, msg)
// 		})
// 	})

// 	t.Run("OnToolCallChunk", func(t *testing.T) {
// 		msg := messages.Message[messages.ToolCallMessage]{
// 			Payload: messages.ToolCallMessage{
// 				ToolCalls: []messages.ToolCallData{{Name: "test", Arguments: "{}"}},
// 			},
// 		}
// 		require.NotPanics(t, func() {
// 			hook.OnToolCallChunk(ctx, msg)
// 		})
// 	})

// 	t.Run("OnAssistantMessage", func(t *testing.T) {
// 		msg := messages.Message[messages.AssistantMessage]{
// 			Payload: messages.AssistantMessage{
// 				Content: messages.AssistantContentOrParts{Content: "test message"},
// 			},
// 		}
// 		require.NotPanics(t, func() {
// 			hook.OnAssistantMessage(ctx, msg)
// 		})
// 	})

// 	t.Run("OnToolCallMessage", func(t *testing.T) {
// 		msg := messages.Message[messages.ToolCallMessage]{
// 			Payload: messages.ToolCallMessage{
// 				ToolCalls: []messages.ToolCallData{{Name: "test", Arguments: "{}"}},
// 			},
// 		}
// 		require.NotPanics(t, func() {
// 			hook.OnToolCallMessage(ctx, msg)
// 		})
// 	})

// 	t.Run("OnToolCallResponse", func(t *testing.T) {
// 		msg := messages.Message[messages.ToolResponse]{
// 			Payload: messages.ToolResponse{
// 				ToolName: "test",
// 				Content:  "result",
// 			},
// 		}
// 		require.NotPanics(t, func() {
// 			hook.OnToolCallResponse(ctx, msg)
// 		})
// 	})

// 	t.Run("OnError", func(t *testing.T) {
// 		err := fmt.Errorf("test error")
// 		require.NotPanics(t, func() {
// 			hook.OnError(ctx, err)
// 		})
// 	})
// }

func TestCompositeHook(t *testing.T) {
	mock1 := &mockHook{}
	mock2 := &mockHook{}
	composite := NewCompositeHook(mock1, mock2)
	ctx := context.Background()

	t.Run("OnUserPrompt", func(t *testing.T) {
		msg := messages.Message[messages.UserMessage]{
			Payload: messages.UserMessage{
				Content: messages.ContentOrParts{Content: "test prompt"},
			},
		}
		composite.OnUserPrompt(ctx, msg)
		assert.True(t, mock1.userPromptCalled)
		assert.True(t, mock2.userPromptCalled)
		assert.Equal(t, msg, mock1.lastUserPrompt)
		assert.Equal(t, msg, mock2.lastUserPrompt)
	})

	t.Run("OnAssistantChunk", func(t *testing.T) {
		msg := messages.Message[messages.AssistantMessage]{
			Payload: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{Content: "test chunk"},
			},
		}
		composite.OnAssistantChunk(ctx, msg)
		assert.True(t, mock1.assistantChunkCalled)
		assert.True(t, mock2.assistantChunkCalled)
		assert.Equal(t, msg, mock1.lastAssistantChunk)
		assert.Equal(t, msg, mock2.lastAssistantChunk)
	})

	t.Run("OnToolCallChunk", func(t *testing.T) {
		msg := messages.Message[messages.ToolCallMessage]{
			Payload: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{{Name: "test", Arguments: "{}"}},
			},
		}
		composite.OnToolCallChunk(ctx, msg)
		assert.True(t, mock1.toolCallChunkCalled)
		assert.True(t, mock2.toolCallChunkCalled)
		assert.Equal(t, msg, mock1.lastToolCallChunk)
		assert.Equal(t, msg, mock2.lastToolCallChunk)
	})

	t.Run("OnAssistantMessage", func(t *testing.T) {
		msg := messages.Message[messages.AssistantMessage]{
			Payload: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{Content: "test message"},
			},
		}
		composite.OnAssistantMessage(ctx, msg)
		assert.True(t, mock1.assistantMsgCalled)
		assert.True(t, mock2.assistantMsgCalled)
		assert.Equal(t, msg, mock1.lastAssistantMsg)
		assert.Equal(t, msg, mock2.lastAssistantMsg)
	})

	t.Run("OnToolCallMessage", func(t *testing.T) {
		msg := messages.Message[messages.ToolCallMessage]{
			Payload: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{{Name: "test", Arguments: "{}"}},
			},
		}
		composite.OnToolCallMessage(ctx, msg)
		assert.True(t, mock1.toolCallMsgCalled)
		assert.True(t, mock2.toolCallMsgCalled)
		assert.Equal(t, msg, mock1.lastToolCallMsg)
		assert.Equal(t, msg, mock2.lastToolCallMsg)
	})

	t.Run("OnToolCallResponse", func(t *testing.T) {
		msg := messages.Message[messages.ToolResponse]{
			Payload: messages.ToolResponse{
				ToolName: "test",
				Content:  "result",
			},
		}
		composite.OnToolCallResponse(ctx, msg)
		assert.True(t, mock1.toolCallRespCalled)
		assert.True(t, mock2.toolCallRespCalled)
		assert.Equal(t, msg, mock1.lastToolCallResp)
		assert.Equal(t, msg, mock2.lastToolCallResp)
	})

	t.Run("OnError", func(t *testing.T) {
		err := fmt.Errorf("test error")
		composite.OnError(ctx, err)
		assert.True(t, mock1.errorCalled)
		assert.True(t, mock2.errorCalled)
		assert.Equal(t, err, mock1.lastError)
		assert.Equal(t, err, mock2.lastError)
	})
}

func TestMustJSON(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		require.NotPanics(t, func() {
			result := mustJSON(data)
			assert.Equal(t, `{"key":"value"}`, result)
		})
	})

	t.Run("invalid json", func(t *testing.T) {
		// Create a circular reference that can't be marshaled to JSON
		type circular struct {
			Self *circular
		}
		data := &circular{}
		data.Self = data

		require.Panics(t, func() {
			_ = mustJSON(data)
		})
	})
}
