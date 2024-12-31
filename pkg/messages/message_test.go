package messages

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Keep all existing test functions unchanged...
func TestInstructions_message(t *testing.T) {
	i := InstructionsMessage{}
	i.message()
}

func TestInstructions(t *testing.T) {
	i := InstructionsMessage{
		Content: "test content",
	}
	assert.Equal(t, "test content", i.Content)
}

func TestUserPrompt_message(t *testing.T) {
	u := UserMessage{}
	u.message()
}

func TestUserPrompt_request(t *testing.T) {
	u := UserMessage{}
	u.request()
}

func TestUserPrompt(t *testing.T) {
	content := ContentOrParts{Content: "test content"}
	u := UserMessage{
		Content: content,
	}
	assert.Equal(t, content, u.Content)
}

func TestAssistantMessage_message(t *testing.T) {
	a := AssistantMessage{}
	a.message()
}

func TestAssistantMessage_response(t *testing.T) {
	a := AssistantMessage{}
	a.response()
}

func TestAssistantMessage(t *testing.T) {
	content := AssistantContentOrParts{Content: "test content"}
	a := AssistantMessage{
		Content: content,
		Refusal: "test refusal",
	}
	assert.Equal(t, content, a.Content)
	assert.Equal(t, "test refusal", a.Refusal)
}

func TestToolCall_message(t *testing.T) {
	tc := ToolCallMessage{}
	tc.message()
}

func TestToolCall_response(t *testing.T) {
	tc := ToolCallMessage{}
	tc.response()
}

func TestToolCall(t *testing.T) {
	tc := ToolCallMessage{
		ToolCalls: []ToolCallData{
			{
				ID:        "test-id",
				Name:      "test name",
				Arguments: "test args",
			},
		},
	}
	assert.Len(t, tc.ToolCalls, 1)
	assert.Equal(t, "test-id", tc.ToolCalls[0].ID)
	assert.Equal(t, "test name", tc.ToolCalls[0].Name)
	assert.Equal(t, "test args", tc.ToolCalls[0].Arguments)
}

func TestToolResponse_message(t *testing.T) {
	tr := ToolResponse{}
	tr.message()
}

func TestToolResponse_request(t *testing.T) {
	tr := ToolResponse{}
	tr.request()
}

func TestToolResponse(t *testing.T) {
	tr := ToolResponse{
		ToolName:   "test tool",
		ToolCallID: "test-call-id",
		Content:    "test content",
	}
	assert.Equal(t, "test tool", tr.ToolName)
	assert.Equal(t, "test-call-id", tr.ToolCallID)
	assert.Equal(t, "test content", tr.Content)
}

func TestRetry_message(t *testing.T) {
	r := Retry{}
	r.message()
}

func TestRetry_request(t *testing.T) {
	r := Retry{}
	r.request()
}

func TestRetry(t *testing.T) {
	err := assert.AnError
	r := Retry{
		Error:      err,
		ToolName:   "test tool",
		ToolCallID: "test-call-id",
	}
	assert.Equal(t, err, r.Error)
	assert.Equal(t, "test tool", r.ToolName)
	assert.Equal(t, "test-call-id", r.ToolCallID)
}

func TestNew(t *testing.T) {
	builder := New()
	assert.NotZero(t, builder.timestamp)
}

func TestMessageBuilder(t *testing.T) {
	now := strfmt.DateTime(time.Now())
	builder := messageBuilder{}
	metadata := gjson.Parse(`{"key": "value"}`)

	t.Run("WithSender", func(t *testing.T) {
		result := builder.WithSender("test-sender")
		assert.Equal(t, "test-sender", result.sender)
	})

	t.Run("WithTimestamp", func(t *testing.T) {
		result := builder.WithTimestamp(now)
		assert.Equal(t, now, result.timestamp)
	})

	t.Run("WithMetadata", func(t *testing.T) {
		result := builder.WithMetadata(metadata)
		assert.Equal(t, metadata.Raw, result.metadata.Raw)
	})

	t.Run("Instructions", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).Instructions("test content")
		assert.Equal(t, "test content", msg.Payload.Content)
		assert.Equal(t, "test", msg.Sender)
		assert.Equal(t, now, msg.Timestamp)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("UserPrompt", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).UserPrompt("test content")
		assert.Equal(t, "test content", msg.Payload.Content.Content)
		assert.Equal(t, "test", msg.Sender)
		assert.Equal(t, now, msg.Timestamp)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("UserPromptMultipart", func(t *testing.T) {
		parts := []ContentPart{
			TextContentPart{Text: "part1"},
			ImageContentPart{URL: "image.jpg"},
		}
		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).UserPromptMultipart(parts...)
		assert.Equal(t, parts, msg.Payload.Content.Parts)
		assert.Equal(t, "test", msg.Sender)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("AssistantMessage", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).AssistantMessage("test content")
		assert.Equal(t, "test content", msg.Payload.Content.Content)
		assert.Empty(t, msg.Payload.Refusal)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("AssistantRefusal", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).AssistantRefusal("not allowed")
		assert.Equal(t, "not allowed", msg.Payload.Refusal)
		assert.Empty(t, msg.Payload.Content.Content)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("AssistantMessageMultipart", func(t *testing.T) {
		parts := []AssistantContentPart{
			TextContentPart{Text: "part1"},
			RefusalContentPart{Refusal: "not allowed"},
		}
		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).AssistantMessageMultipart(parts...)
		assert.Equal(t, parts, msg.Payload.Content.Parts)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})
}

func TestMessageJSONMarshaling(t *testing.T) {
	now := strfmt.DateTime(time.Now().UTC().Truncate(time.Second))
	runID := uuid.New()
	turnID := uuid.New()

	testCases := []struct {
		name     string
		message  interface{}
		expected string
	}{
		{
			name: "instructions message",
			message: Message[InstructionsMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "system",
				Timestamp: now,
				Meta:      gjson.Parse(`{"key":"value"}`),
				Payload:   InstructionsMessage{Content: "test instructions"},
			},
			expected: fmt.Sprintf(`{
				"type": "instructions",
				"content": "test instructions",
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "system",
				"timestamp": "%s",
				"meta": {"key":"value"}
			}`, runID, turnID, now),
		},
		{
			name: "user message with text",
			message: Message[UserMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "user",
				Timestamp: now,
				Payload: UserMessage{
					Content: ContentOrParts{Content: "hello"},
				},
			},
			expected: fmt.Sprintf(`{
				"type": "user",
				"content": "hello",
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "user",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "user message with parts",
			message: Message[UserMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "user",
				Timestamp: now,
				Payload: UserMessage{
					Content: ContentOrParts{
						Parts: []ContentPart{
							Text("hello"),
							Image("http://example.com/image.jpg"),
						},
					},
				},
			},
			expected: fmt.Sprintf(`{
				"type": "user",
				"content": [
					{"type":"text","text":"hello"},
					{"type":"image","image_url":"http://example.com/image.jpg"}
				],
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "user",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "assistant message with text",
			message: Message[AssistantMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "assistant",
				Timestamp: now,
				Payload: AssistantMessage{
					Content: AssistantContentOrParts{Content: "hello"},
				},
			},
			expected: fmt.Sprintf(`{
				"type": "assistant",
				"content": "hello",
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "assistant",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "assistant message with parts",
			message: Message[AssistantMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "assistant",
				Timestamp: now,
				Payload: AssistantMessage{
					Content: AssistantContentOrParts{
						Parts: []AssistantContentPart{
							Text("hello"),
							Refusal("cannot do that"),
						},
					},
				},
			},
			expected: fmt.Sprintf(`{
				"type": "assistant",
				"content": [
					{"type":"text","text":"hello"},
					{"type":"refusal","refusal":"cannot do that"}
				],
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "assistant",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "assistant refusal message",
			message: Message[AssistantMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "assistant",
				Timestamp: now,
				Payload: AssistantMessage{
					Refusal: "cannot do that",
				},
			},
			expected: fmt.Sprintf(`{
				"type": "assistant",
				"refusal": "cannot do that",
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "assistant",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "tool call message",
			message: Message[ToolCallMessage]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "assistant",
				Timestamp: now,
				Payload: ToolCallMessage{
					ToolCalls: []ToolCallData{
						{
							ID:        "123",
							Name:      "test_tool",
							Arguments: `{"arg":"value"}`,
						},
					},
				},
			},
			expected: fmt.Sprintf(`{
				"type": "tool_call",
				"tool_calls": [
					{"id": "123","name":"test_tool","arguments":"{\"arg\":\"value\"}"}
				],
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "assistant",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "tool response message",
			message: Message[ToolResponse]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "tool",
				Timestamp: now,
				Payload: ToolResponse{
					ToolName:   "test_tool",
					ToolCallID: "123",
					Content:    "tool result",
				},
			},
			expected: fmt.Sprintf(`{
				"type": "tool_response",
				"tool_name": "test_tool",
				"tool_call_id": "123",
				"content": "tool result",
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "tool",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
		{
			name: "retry message",
			message: Message[Retry]{
				RunID:     runID,
				TurnID:    turnID,
				Sender:    "tool",
				Timestamp: now,
				Payload: Retry{
					Error:      fmt.Errorf("test error"),
					ToolName:   "test_tool",
					ToolCallID: "123",
				},
			},
			expected: fmt.Sprintf(`{
				"type": "retry",
				"error": "test error",
				"tool_name": "test_tool",
				"tool_call_id": "123",
				"run_id": "%s",
				"turn_id": "%s",
				"sender": "tool",
				"timestamp": "%s"
			}`, runID, turnID, now),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tc.message)
			require.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(data))

			// Test unmarshaling
			switch msg := tc.message.(type) {
			case Message[InstructionsMessage]:
				var decoded Message[InstructionsMessage]
				require.NoError(t, json.Unmarshal(data, &decoded))
				assert.Equal(t, msg.RunID, decoded.RunID)
				assert.Equal(t, msg.TurnID, decoded.TurnID)
				assert.Equal(t, msg.Sender, decoded.Sender)
				assert.Equal(t, msg.Timestamp, decoded.Timestamp)
				assert.Equal(t, msg.Meta.Raw, decoded.Meta.Raw)
				assert.Equal(t, msg.Payload, decoded.Payload)
			case Message[UserMessage]:
				var decoded Message[UserMessage]
				require.NoError(t, json.Unmarshal(data, &decoded))
				assert.Equal(t, msg.RunID, decoded.RunID)
				assert.Equal(t, msg.TurnID, decoded.TurnID)
				assert.Equal(t, msg.Sender, decoded.Sender)
				assert.Equal(t, msg.Timestamp, decoded.Timestamp)
				assert.Equal(t, msg.Meta.Raw, decoded.Meta.Raw)
				assert.Equal(t, msg.Payload, decoded.Payload)
			case Message[AssistantMessage]:
				var decoded Message[AssistantMessage]
				require.NoError(t, json.Unmarshal(data, &decoded))
				assert.Equal(t, msg.RunID, decoded.RunID)
				assert.Equal(t, msg.TurnID, decoded.TurnID)
				assert.Equal(t, msg.Sender, decoded.Sender)
				assert.Equal(t, msg.Timestamp, decoded.Timestamp)
				assert.Equal(t, msg.Meta.Raw, decoded.Meta.Raw)
				assert.Equal(t, msg.Payload, decoded.Payload)
			case Message[ToolCallMessage]:
				var decoded Message[ToolCallMessage]
				require.NoError(t, json.Unmarshal(data, &decoded))
				assert.Equal(t, msg.RunID, decoded.RunID)
				assert.Equal(t, msg.TurnID, decoded.TurnID)
				assert.Equal(t, msg.Sender, decoded.Sender)
				assert.Equal(t, msg.Timestamp, decoded.Timestamp)
				assert.Equal(t, msg.Meta.Raw, decoded.Meta.Raw)
				assert.Equal(t, msg.Payload, decoded.Payload)
			case Message[ToolResponse]:
				var decoded Message[ToolResponse]
				require.NoError(t, json.Unmarshal(data, &decoded))
				assert.Equal(t, msg.RunID, decoded.RunID)
				assert.Equal(t, msg.TurnID, decoded.TurnID)
				assert.Equal(t, msg.Sender, decoded.Sender)
				assert.Equal(t, msg.Timestamp, decoded.Timestamp)
				assert.Equal(t, msg.Meta.Raw, decoded.Meta.Raw)
				assert.Equal(t, msg.Payload, decoded.Payload)
			case Message[Retry]:
				var decoded Message[Retry]
				require.NoError(t, json.Unmarshal(data, &decoded))
				assert.Equal(t, msg.RunID, decoded.RunID)
				assert.Equal(t, msg.TurnID, decoded.TurnID)
				assert.Equal(t, msg.Sender, decoded.Sender)
				assert.Equal(t, msg.Timestamp, decoded.Timestamp)
				assert.Equal(t, msg.Meta.Raw, decoded.Meta.Raw)
				assert.Equal(t, msg.Payload.Error.Error(), decoded.Payload.Error.Error())
				assert.Equal(t, msg.Payload.ToolName, decoded.Payload.ToolName)
				assert.Equal(t, msg.Payload.ToolCallID, decoded.Payload.ToolCallID)
			}
		})
	}
}

func TestMessageJSONUnmarshalingErrors(t *testing.T) {
	testCases := []struct {
		name          string
		json          string
		expectedError string
	}{
		{
			name:          "invalid json",
			json:          `{invalid`,
			expectedError: "invalid character",
		},
		{
			name:          "missing type field",
			json:          `{"content":"test"}`,
			expectedError: "missing required field 'type'",
		},
		{
			name:          "invalid type field",
			json:          `{"type":"unknown","content":"test"}`,
			expectedError: "unknown message type: unknown",
		},
		{
			name:          "missing required content field for instructions",
			json:          `{"type":"instructions"}`,
			expectedError: "missing required field 'content'",
		},
		{
			name:          "missing required content field for user message",
			json:          `{"type":"user"}`,
			expectedError: "missing required field 'content'",
		},
		{
			name:          "both content and refusal in assistant message",
			json:          `{"type":"assistant","content":"hello","refusal":"cannot"}`,
			expectedError: "both 'content' and 'refusal' cannot be present",
		},
		{
			name:          "missing tool_calls in tool call",
			json:          `{"type":"tool_call"}`,
			expectedError: "missing required field 'tool_calls'",
		},
		{
			name:          "invalid tool_calls type in tool call",
			json:          `{"type":"tool_call","tool_calls":"not_array"}`,
			expectedError: "'tool_calls' must be an array",
		},
		{
			name:          "missing tool_name in tool response",
			json:          `{"type":"tool_response","tool_call_id":"123","content":"result"}`,
			expectedError: "missing required field 'tool_name'",
		},
		{
			name:          "missing tool_call_id in tool response",
			json:          `{"type":"tool_response","tool_name":"test","content":"result"}`,
			expectedError: "missing required field 'tool_call_id'",
		},
		{
			name:          "missing content in tool response",
			json:          `{"type":"tool_response","tool_name":"test","tool_call_id":"123"}`,
			expectedError: "missing required field 'content'",
		},
		{
			name:          "missing error in retry",
			json:          `{"type":"retry","tool_name":"test","tool_call_id":"123"}`,
			expectedError: "missing required field 'error'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var msg Message[ModelMessage]
			err := json.Unmarshal([]byte(tc.json), &msg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestToolOperations(t *testing.T) {
	now := strfmt.DateTime(time.Now())
	metadata := gjson.Parse(`{"key": "value"}`)

	t.Run("CallTool", func(t *testing.T) {
		args := gjson.Parse(`{"key": "value"}`)
		toolCall := CallTool("", "test-tool", args)
		assert.Equal(t, "test-tool", toolCall.Name)
		assert.Equal(t, args.Raw, toolCall.Arguments)
	})

	t.Run("ToolCall", func(t *testing.T) {
		builder := messageBuilder{}
		toolData := CallTool("call-id", "test-tool", gjson.Parse(`{"key": "value"}`))

		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).ToolCall([]ToolCallData{toolData})
		assert.Equal(t, "call-id", msg.Payload.ToolCalls[0].ID)
		assert.Equal(t, toolData, msg.Payload.ToolCalls[0])
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("ToolResponse", func(t *testing.T) {
		builder := messageBuilder{}

		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).ToolResponse("call-id", "test-tool", "result")
		assert.Equal(t, "call-id", msg.Payload.ToolCallID)
		assert.Equal(t, "test-tool", msg.Payload.ToolName)
		assert.Equal(t, "result", msg.Payload.Content)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})

	t.Run("ToolError", func(t *testing.T) {
		builder := messageBuilder{}
		testErr := errors.New("test error")

		msg := builder.WithSender("test").WithTimestamp(now).WithMetadata(metadata).ToolError("call-id", "test-tool", testErr)
		assert.Equal(t, "call-id", msg.Payload.ToolCallID)
		assert.Equal(t, "test-tool", msg.Payload.ToolName)
		assert.Equal(t, testErr, msg.Payload.Error)
		assert.Equal(t, metadata.Raw, msg.Meta.Raw)
	})
}
