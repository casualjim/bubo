package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/casualjim/bubo/pkg/messages"
	"github.com/stretchr/testify/assert"
)

func TestMustJSON(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		expected, _ := json.Marshal(data)
		assert.Equal(t, string(expected), mustJSON(data))
	})

	t.Run("panic on invalid json", func(t *testing.T) {
		assert.Panics(t, func() {
			mustJSON(make(chan int))
		})
	})
}

type recordingHook struct {
	userPrompts       []messages.Message[messages.UserMessage]
	assistantChunks   []messages.Message[messages.AssistantMessage]
	toolCallChunks    []messages.Message[messages.ToolCallMessage]
	assistantMessages []messages.Message[messages.AssistantMessage]
	toolCallMessages  []messages.Message[messages.ToolCallMessage]
	toolCallResponses []messages.Message[messages.ToolResponse]
	responses         []any
	errors            []error
	wg                *sync.WaitGroup // Optional WaitGroup for synchronization
	mu                sync.Mutex      // Protect concurrent access to slices
	delay             time.Duration   // Optional delay for testing overflow scenarios
}

func (h *recordingHook) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	h.mu.Lock()
	h.userPrompts = append(h.userPrompts, msg)
	h.mu.Unlock()
}

func (h *recordingHook) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	h.mu.Lock()
	h.assistantChunks = append(h.assistantChunks, msg)
	h.mu.Unlock()
}

func (h *recordingHook) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	h.mu.Lock()
	h.toolCallChunks = append(h.toolCallChunks, msg)
	h.mu.Unlock()
}

func (h *recordingHook) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.assistantMessages = append(h.assistantMessages, msg)
	if h.wg != nil {
		h.wg.Done()
	}
}

func (h *recordingHook) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toolCallMessages = append(h.toolCallMessages, msg)
	if h.wg != nil {
		h.wg.Done()
	}
}

func (h *recordingHook) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	h.mu.Lock()
	h.toolCallResponses = append(h.toolCallResponses, msg)
	h.mu.Unlock()
	if h.wg != nil {
		h.wg.Done()
	}
}

func (h *recordingHook) OnResponse(ctx context.Context, response any) {
	h.mu.Lock()
	h.responses = append(h.responses, response)
	h.mu.Unlock()
}

func (h *recordingHook) OnError(ctx context.Context, err error) {
	h.mu.Lock()
	h.errors = append(h.errors, err)
	h.mu.Unlock()
}

func TestCompositeHook(t *testing.T) {
	ctx := context.Background()
	recorder1 := &recordingHook{}
	recorder2 := &recordingHook{}
	composite := NewCompositeHook(recorder1, recorder2)

	t.Run("forwards user prompt", func(t *testing.T) {
		msg := messages.New().UserPrompt("test prompt")
		composite.OnUserPrompt(ctx, msg)
		assert.Equal(t, []messages.Message[messages.UserMessage]{msg}, recorder1.userPrompts)
		assert.Equal(t, []messages.Message[messages.UserMessage]{msg}, recorder2.userPrompts)
	})

	t.Run("forwards assistant chunk", func(t *testing.T) {
		msg := messages.New().AssistantMessage("test chunk")
		composite.OnAssistantChunk(ctx, msg)
		assert.Equal(t, []messages.Message[messages.AssistantMessage]{msg}, recorder1.assistantChunks)
		assert.Equal(t, []messages.Message[messages.AssistantMessage]{msg}, recorder2.assistantChunks)
	})

	t.Run("forwards tool call chunk", func(t *testing.T) {
		msg := messages.New().ToolCall([]messages.ToolCallData{{
			ID:        "test-id",
			Name:      "test-tool",
			Arguments: `{"arg":"value"}`,
		}})
		composite.OnToolCallChunk(ctx, msg)
		assert.Equal(t, []messages.Message[messages.ToolCallMessage]{msg}, recorder1.toolCallChunks)
		assert.Equal(t, []messages.Message[messages.ToolCallMessage]{msg}, recorder2.toolCallChunks)
	})

	t.Run("forwards assistant message", func(t *testing.T) {
		msg := messages.New().AssistantMessage("test message")
		composite.OnAssistantMessage(ctx, msg)
		assert.Equal(t, []messages.Message[messages.AssistantMessage]{msg}, recorder1.assistantMessages)
		assert.Equal(t, []messages.Message[messages.AssistantMessage]{msg}, recorder2.assistantMessages)
	})

	t.Run("forwards tool call message", func(t *testing.T) {
		msg := messages.New().ToolCall([]messages.ToolCallData{{
			ID:        "test-id",
			Name:      "test-tool",
			Arguments: `{"arg":"value"}`,
		}})
		composite.OnToolCallMessage(ctx, msg)
		assert.Equal(t, []messages.Message[messages.ToolCallMessage]{msg}, recorder1.toolCallMessages)
		assert.Equal(t, []messages.Message[messages.ToolCallMessage]{msg}, recorder2.toolCallMessages)
	})

	t.Run("forwards tool call response", func(t *testing.T) {
		msg := messages.New().ToolResponse("test-id", "test-tool", "test response")
		composite.OnToolCallResponse(ctx, msg)
		assert.Equal(t, []messages.Message[messages.ToolResponse]{msg}, recorder1.toolCallResponses)
		assert.Equal(t, []messages.Message[messages.ToolResponse]{msg}, recorder2.toolCallResponses)
	})

	t.Run("forwards errors", func(t *testing.T) {
		err := errors.New("test error")
		composite.OnError(ctx, err)
		assert.Equal(t, []error{err}, recorder1.errors)
		assert.Equal(t, []error{err}, recorder2.errors)
	})
}
