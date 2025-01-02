package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/tool"
	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	p := New()
	assert.NotNil(t, p)
	assert.NotNil(t, p.client)
}

func TestProvider_buildRequest_Error(t *testing.T) {
	p := New()
	ctx := context.Background()
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	// Create a tool with an invalid function that will fail JSON conversion
	invalidTool := tool.Definition{
		Name:        "invalid_tool",
		Description: "A test tool",
		Parameters:  map[string]string{"param1": "value1"},
		Function:    nil,
	}

	params := &provider.CompletionParams{
		RunID:        runID,
		Instructions: "Test instructions",
		Thread:       aggregator,
		Tools:        []tool.Definition{invalidTool},
	}

	_, err := p.buildRequest(ctx, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool invalid_tool has nil function")
}

func TestProvider_buildRequest(t *testing.T) {
	p := New()
	ctx := context.Background()
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	// Add user message to the aggregator
	userMsg := messages.Message[messages.UserMessage]{
		RunID:  runID,
		TurnID: aggregator.ID(),
		Sender: "testUser",
		Payload: messages.UserMessage{
			Content: messages.ContentOrParts{
				Content: "Hello",
			},
		},
	}
	aggregator.AddUserPrompt(userMsg)

	// Create a test tool
	toolDef := tool.Definition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]string{
			"param1": "value1",
		},
		Function: func(s string) string { return s },
	}

	params := &provider.CompletionParams{
		RunID:        runID,
		Instructions: "Test instructions",
		Thread:       aggregator,
		Stream:       false,
		Model:        GPT4oMini(),
		Tools:        []tool.Definition{toolDef},
	}

	chatParams, err := p.buildRequest(ctx, params)
	require.NoError(t, err)

	// Verify the built request
	assert.Equal(t, GPT4oMini().Name(), string(chatParams.Model.Value))
	assert.Equal(t, int64(1), chatParams.N.Value)
	assert.True(t, chatParams.ParallelToolCalls.Value)
	assert.Equal(t, 0.1, chatParams.Temperature.Value)
	assert.Equal(t, "testUser", chatParams.User.Value)

	// Verify messages
	messages := chatParams.Messages.Value
	require.Len(t, messages, 2) // System message + user message

	// Verify system message
	systemMsg := messages[0].(openai.ChatCompletionSystemMessageParam)
	assert.Equal(t, "Test instructions", systemMsg.Content.Value[0].Text.Value)

	// Verify user message
	userMsg2 := messages[1].(openai.ChatCompletionUserMessageParam)
	assert.Equal(t, "Hello", userMsg2.Content.Value[0].(openai.ChatCompletionContentPartTextParam).Text.Value)

	// Verify tools
	tools := chatParams.Tools.Value
	assert.Len(t, tools, 1)
	assert.Equal(t, openai.ChatCompletionToolTypeFunction, tools[0].Type.Value)
	assert.Equal(t, "test_tool", tools[0].Function.Value.Name.Value)
	assert.Equal(t, "A test tool", tools[0].Function.Value.Description.Value)
}

func TestProvider_ChatCompletion_ContextCancellation(t *testing.T) {
	serverDone := make(chan struct{})
	p := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)

		// Set up SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		// Write initial event
		event := openai.ChatCompletionChunk{
			ID: "test-id",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						Content: "Hello",
					},
				},
			},
		}
		data, err := json.Marshal(event)
		require.NoError(t, err)
		_, err = fmt.Fprintf(w, "data: %s\n\n", data)
		require.NoError(t, err)
		flusher.Flush()

		// Wait for context cancellation
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	params := provider.CompletionParams{
		RunID:        runID,
		Instructions: "Test instructions",
		Thread:       aggregator,
		Stream:       true,
		Model:        GPT4oMini(),
	}

	events, err := p.ChatCompletion(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, events)

	// Read the start delimiter and first chunk
	event := <-events
	assert.IsType(t, provider.Delim{}, event)
	assert.Equal(t, "start", event.(provider.Delim).Delim)

	event = <-events
	chunk, ok := event.(provider.Chunk[messages.AssistantMessage])
	assert.True(t, ok)
	assert.Equal(t, "Hello", chunk.Chunk.Content.Content)

	// Cancel the context
	cancel()

	// Wait for server to finish
	<-serverDone

	// Should receive error and channel close
	event = <-events
	errEvent, ok := event.(provider.Error)
	assert.True(t, ok)
	assert.Equal(t, context.Canceled, errEvent.Err)

	// Channel should be closed
	_, ok = <-events
	assert.False(t, ok, "Channel should be closed after context cancellation")
}

func TestProvider_buildRequest_ComplexTools(t *testing.T) {
	p := New()
	ctx := context.Background()
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	// Create tools with string parameters
	toolDefs := []tool.Definition{
		{
			Name:        "complex_tool",
			Description: "A tool with multiple parameters",
			Parameters: map[string]string{
				"param0": "s",
				"param1": "i",
			},
			Function: func(s string, i int) string { return fmt.Sprintf("%s %d", s, i) },
		},
		{
			Name: "simple_tool",
			Parameters: map[string]string{
				"param0": "s",
			},
			Function: func(s string) string { return s },
		},
	}

	params := &provider.CompletionParams{
		RunID:        runID,
		Instructions: "Test instructions",
		Thread:       aggregator,
		Tools:        toolDefs,
		Model:        GPT4oMini(),
	}

	chatParams, err := p.buildRequest(ctx, params)
	require.NoError(t, err)

	// Verify tools were properly converted
	tools := chatParams.Tools.Value
	assert.Len(t, tools, 2)

	// Verify complex tool
	assert.Equal(t, "complex_tool", tools[0].Function.Value.Name.Value)
	assert.NotNil(t, tools[0].Function.Value.Parameters.Value)

	// Verify simple tool
	assert.Equal(t, "simple_tool", tools[1].Function.Value.Name.Value)
	assert.NotNil(t, tools[1].Function.Value.Parameters.Value)
}

func setupTestServer(t *testing.T, handler http.HandlerFunc) *Provider {
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})

	p := New(option.WithBaseURL(server.URL + "/v1"))
	return p
}

func TestProvider_ChatCompletion(t *testing.T) {
	mockResp := openai.ChatCompletion{
		ID: "test-id",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "Test response",
				},
			},
		},
	}

	p := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	})

	ctx := context.Background()
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	params := provider.CompletionParams{
		RunID:        runID,
		Instructions: "Test instructions",
		Thread:       aggregator,
		Stream:       false,
		Model:        GPT4oMini(),
	}

	events, err := p.ChatCompletion(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, events)

	// Read the response from the channel
	event := <-events
	resp, ok := event.(provider.Response[messages.AssistantMessage])
	require.True(t, ok)
	assert.Equal(t, "Test response", resp.Response.Content.Content)

	// Channel should be closed after the response
	_, ok = <-events
	assert.False(t, ok)
}

func TestMessagesToOpenAI_EmptyMessages(t *testing.T) {
	result, user := messagesToOpenAI("Test instructions", slices.Values([]messages.Message[messages.ModelMessage]{}))

	assert.Len(t, result, 1) // Only system message
	systemMsg := result[0].(openai.ChatCompletionSystemMessageParam)
	assert.Equal(t, "Test instructions", systemMsg.Content.Value[0].Text.Value)
	assert.Empty(t, user)
}

func TestMessagesToOpenAI_ContentParts(t *testing.T) {
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	// Create a message with different content parts
	userMsg := messages.Message[messages.UserMessage]{
		RunID:  runID,
		TurnID: aggregator.ID(),
		Sender: "user1",
		Payload: messages.UserMessage{
			Content: messages.ContentOrParts{
				Parts: []messages.ContentPart{
					messages.TextContentPart{Text: "Hello"},
					messages.ImageContentPart{
						URL:    "http://example.com/image.jpg",
						Detail: "high",
					},
					&messages.AudioContentPart{
						InputAudio: messages.InputAudio{
							Data:   []byte("audio data"),
							Format: "mp3",
						},
					},
				},
			},
		},
	}
	aggregator.AddUserPrompt(userMsg)

	result, user := messagesToOpenAI("Test instructions", aggregator.MessagesIter())

	// Verify the conversion
	assert.Equal(t, "user1", user)
	assert.Len(t, result, 2) // System message + user message with parts

	// Verify user message parts
	userMsgResult := result[1].(openai.ChatCompletionUserMessageParam)
	parts := userMsgResult.Content.Value
	assert.Len(t, parts, 3)

	// Verify text part
	textPart := parts[0].(openai.ChatCompletionContentPartTextParam)
	assert.Equal(t, "Hello", textPart.Text.Value)

	// Verify image part
	imagePart := parts[1].(openai.ChatCompletionContentPartImageParam)
	assert.Equal(t, "http://example.com/image.jpg", imagePart.ImageURL.Value.URL.Value)
	assert.Equal(t, openai.ChatCompletionContentPartImageImageURLDetailHigh, imagePart.ImageURL.Value.Detail.Value)

	// Verify audio part
	audioPart := parts[2].(openai.ChatCompletionContentPartInputAudioParam)
	assert.Equal(t, "mp3", string(audioPart.InputAudio.Value.Format.Value))
	decodedAudio, _ := base64.StdEncoding.DecodeString(audioPart.InputAudio.Value.Data.Value)
	assert.Equal(t, []byte("audio data"), decodedAudio)
}

func TestMessagesToOpenAI(t *testing.T) {
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	// Add different types of messages
	userMsg := messages.Message[messages.UserMessage]{
		RunID:  runID,
		TurnID: aggregator.ID(),
		Sender: "user1",
		Payload: messages.UserMessage{
			Content: messages.ContentOrParts{
				Content: "Hello",
			},
		},
	}
	aggregator.AddUserPrompt(userMsg)

	assistantMsg := messages.Message[messages.AssistantMessage]{
		RunID:  runID,
		TurnID: aggregator.ID(),
		Payload: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: "Hi there",
			},
		},
	}
	aggregator.AddAssistantMessage(assistantMsg)

	toolCallMsg := messages.Message[messages.ToolCallMessage]{
		RunID:  runID,
		TurnID: aggregator.ID(),
		Payload: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					ID:        "tool1",
					Name:      "test_tool",
					Arguments: `{"param": "value"}`,
				},
			},
		},
	}
	aggregator.AddToolCall(toolCallMsg)

	toolResponseMsg := messages.Message[messages.ToolResponse]{
		RunID:  runID,
		TurnID: aggregator.ID(),
		Payload: messages.ToolResponse{
			ToolCallID: "tool1",
			Content:    "Tool response",
		},
	}
	aggregator.AddToolResponse(toolResponseMsg)

	result, user := messagesToOpenAI("Test instructions", aggregator.MessagesIter())

	// Verify the conversion
	assert.Equal(t, "user1", user)
	assert.Len(t, result, 5) // System message + 4 messages
	firstMsg := result[0].(openai.ChatCompletionSystemMessageParam)
	assert.Equal(t, "Test instructions", firstMsg.Content.Value[0].Text.Value)
}

func TestCompletionChunkToStreamEvent(t *testing.T) {
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	tests := []struct {
		name     string
		chunk    *openai.ChatCompletionChunk
		command  *provider.CompletionParams
		validate func(t *testing.T, event provider.StreamEvent)
	}{
		{
			name: "assistant message chunk",
			chunk: &openai.ChatCompletionChunk{
				Choices: []openai.ChatCompletionChunkChoice{
					{
						Delta: openai.ChatCompletionChunkChoicesDelta{
							Content: "Test chunk",
						},
					},
				},
			},
			command: &provider.CompletionParams{
				RunID:  runID,
				Thread: aggregator,
			},
			validate: func(t *testing.T, event provider.StreamEvent) {
				chunk, ok := event.(provider.Chunk[messages.AssistantMessage])
				assert.True(t, ok)
				assert.Equal(t, "Test chunk", chunk.Chunk.Content.Content)
			},
		},
		{
			name: "tool call chunk",
			chunk: &openai.ChatCompletionChunk{
				Choices: []openai.ChatCompletionChunkChoice{
					{
						Delta: openai.ChatCompletionChunkChoicesDelta{
							ToolCalls: []openai.ChatCompletionChunkChoicesDeltaToolCall{
								{
									ID: "tool1",
									Function: openai.ChatCompletionChunkChoicesDeltaToolCallsFunction{
										Name:      "test_tool",
										Arguments: `{"param": "value"}`,
									},
								},
							},
						},
					},
				},
			},
			command: &provider.CompletionParams{
				RunID:  runID,
				Thread: aggregator,
			},
			validate: func(t *testing.T, event provider.StreamEvent) {
				chunk, ok := event.(provider.Chunk[messages.ToolCallMessage])
				assert.True(t, ok)
				assert.Len(t, chunk.Chunk.ToolCalls, 1)
				assert.Equal(t, "tool1", chunk.Chunk.ToolCalls[0].ID)
				assert.Equal(t, "test_tool", chunk.Chunk.ToolCalls[0].Name)
				assert.Equal(t, `{"param": "value"}`, chunk.Chunk.ToolCalls[0].Arguments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := completionChunkToStreamEvent(tt.chunk, tt.command)
			tt.validate(t, event)
		})
	}
}

func TestProvider_ChatCompletion_Stream(t *testing.T) {
	mockEvents := []openai.ChatCompletionChunk{
		{
			ID: "test-id",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						Content: "Hello",
					},
				},
			},
		},
		{
			ID: "test-id",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						ToolCalls: []openai.ChatCompletionChunkChoicesDeltaToolCall{
							{
								ID: "tool1",
								Function: openai.ChatCompletionChunkChoicesDeltaToolCallsFunction{
									Name:      "test_tool",
									Arguments: `{"param": "value"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	p := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		// Write each event with proper SSE format
		for _, event := range mockEvents {
			data, err := json.Marshal(event)
			require.NoError(t, err)
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
			require.NoError(t, err)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond) // Small delay to ensure proper event handling
		}

		// Write completion
		_, err := fmt.Fprintf(w, "data: [DONE]\n\n")
		require.NoError(t, err)
		flusher.Flush()
	})

	ctx := context.Background()
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	params := provider.CompletionParams{
		RunID:        runID,
		Instructions: "Test instructions",
		Thread:       aggregator,
		Stream:       true,
		Model:        GPT4oMini(),
	}

	events, err := p.ChatCompletion(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, events)

	var responses []provider.StreamEvent //nolint:prealloc
	for event := range events {
		responses = append(responses, event)
	}

	// Verify we got start, chunks, end, and final response
	assert.Len(t, responses, 5)

	// Verify start delimiter
	assert.IsType(t, provider.Delim{}, responses[0])
	assert.Equal(t, "start", responses[0].(provider.Delim).Delim)

	// Verify first chunk (text)
	chunk1, ok := responses[1].(provider.Chunk[messages.AssistantMessage])
	assert.True(t, ok)
	assert.Equal(t, "Hello", chunk1.Chunk.Content.Content)

	// Verify second chunk (tool call)
	chunk2, ok := responses[2].(provider.Chunk[messages.ToolCallMessage])
	assert.True(t, ok)
	assert.Len(t, chunk2.Chunk.ToolCalls, 1)
	assert.Equal(t, "tool1", chunk2.Chunk.ToolCalls[0].ID)
	assert.Equal(t, "test_tool", chunk2.Chunk.ToolCalls[0].Name)

	// Verify end delimiter
	assert.IsType(t, provider.Delim{}, responses[3])
	assert.Equal(t, "end", responses[3].(provider.Delim).Delim)
}

func TestCompletionToStreamEvent_MultipleToolCalls(t *testing.T) {
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	chat := &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					ToolCalls: []openai.ChatCompletionMessageToolCall{
						{
							ID: "tool1",
							Function: openai.ChatCompletionMessageToolCallFunction{
								Name:      "test_tool1",
								Arguments: `{"param": "value1"}`,
							},
						},
						{
							ID: "tool2",
							Function: openai.ChatCompletionMessageToolCallFunction{
								Name:      "test_tool2",
								Arguments: `{"param": "value2"}`,
							},
						},
					},
				},
			},
		},
	}

	command := &provider.CompletionParams{
		RunID:  runID,
		Thread: aggregator,
	}

	event := completionToStreamEvent(chat, command)
	resp, ok := event.(provider.Response[messages.ToolCallMessage])
	require.True(t, ok)
	assert.Len(t, resp.Response.ToolCalls, 2)

	// Verify first tool call
	assert.Equal(t, "tool1", resp.Response.ToolCalls[0].ID)
	assert.Equal(t, "test_tool1", resp.Response.ToolCalls[0].Name)
	assert.Equal(t, `{"param": "value1"}`, resp.Response.ToolCalls[0].Arguments)

	// Verify second tool call
	assert.Equal(t, "tool2", resp.Response.ToolCalls[1].ID)
	assert.Equal(t, "test_tool2", resp.Response.ToolCalls[1].Name)
	assert.Equal(t, `{"param": "value2"}`, resp.Response.ToolCalls[1].Arguments)
}

func TestCompletionToStreamEvent(t *testing.T) {
	runID := uuid.New()
	aggregator := shorttermmemory.New()

	tests := []struct {
		name     string
		chat     *openai.ChatCompletion
		command  *provider.CompletionParams
		validate func(t *testing.T, event provider.StreamEvent)
	}{
		{
			name: "empty choices",
			chat: &openai.ChatCompletion{
				Choices: []openai.ChatCompletionChoice{},
			},
			command: &provider.CompletionParams{
				RunID:  runID,
				Thread: aggregator,
			},
			validate: func(t *testing.T, event provider.StreamEvent) {
				_, ok := event.(provider.Delim)
				assert.True(t, ok)
			},
		},
		{
			name: "assistant message",
			chat: &openai.ChatCompletion{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Test response",
						},
					},
				},
			},
			command: &provider.CompletionParams{
				RunID:  runID,
				Thread: aggregator,
			},
			validate: func(t *testing.T, event provider.StreamEvent) {
				resp, ok := event.(provider.Response[messages.AssistantMessage])
				assert.True(t, ok)
				assert.Equal(t, "Test response", resp.Response.Content.Content)
			},
		},
		{
			name: "tool calls",
			chat: &openai.ChatCompletion{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							ToolCalls: []openai.ChatCompletionMessageToolCall{
								{
									ID: "tool1",
									Function: openai.ChatCompletionMessageToolCallFunction{
										Name:      "test_tool",
										Arguments: `{"param": "value"}`,
									},
								},
							},
						},
					},
				},
			},
			command: &provider.CompletionParams{
				RunID:  runID,
				Thread: aggregator,
			},
			validate: func(t *testing.T, event provider.StreamEvent) {
				resp, ok := event.(provider.Response[messages.ToolCallMessage])
				assert.True(t, ok)
				assert.Len(t, resp.Response.ToolCalls, 1)
				assert.Equal(t, "tool1", resp.Response.ToolCalls[0].ID)
				assert.Equal(t, "test_tool", resp.Response.ToolCalls[0].Name)
				assert.Equal(t, `{"param": "value"}`, resp.Response.ToolCalls[0].Arguments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := completionToStreamEvent(tt.chat, tt.command)
			tt.validate(t, event)
		})
	}
}
