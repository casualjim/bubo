package messages

import (
	"errors"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

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
	now := strfmt.DateTime(time.Now())
	tc := ToolCallMessage{
		ID: "test-id",
		Function: []ToolCallData{
			{
				Name:      "test name",
				Arguments: "test args",
			},
		},
		Sender:    "test sender",
		Timestamp: now,
	}
	assert.Equal(t, "test-id", tc.ID)
	assert.Len(t, tc.Function, 1)
	assert.Equal(t, "test name", tc.Function[0].Name)
	assert.Equal(t, "test args", tc.Function[0].Arguments)
	assert.Equal(t, "test sender", tc.Sender)
	assert.Equal(t, now, tc.Timestamp)
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

	t.Run("WithSender", func(t *testing.T) {
		result := builder.WithSender("test-sender")
		assert.Equal(t, "test-sender", result.sender)
	})

	t.Run("WithTimestamp", func(t *testing.T) {
		result := builder.WithTimestamp(now)
		assert.Equal(t, now, result.timestamp)
	})

	t.Run("Instructions", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).Instructions("test content")
		assert.Equal(t, "test content", msg.Payload.Content)
		assert.Equal(t, "test", msg.Sender)
		assert.Equal(t, now, msg.Timestamp)
	})

	t.Run("UserPrompt", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).UserPrompt("test content")
		assert.Equal(t, "test content", msg.Payload.Content.Content)
		assert.Equal(t, "test", msg.Sender)
		assert.Equal(t, now, msg.Timestamp)
	})

	t.Run("UserPromptMultipart", func(t *testing.T) {
		parts := []ContentPart{
			TextContentPart{Text: "part1"},
			ImageContentPart{URL: "image.jpg"},
		}
		msg := builder.WithSender("test").WithTimestamp(now).UserPromptMultipart(parts...)
		assert.Equal(t, parts, msg.Payload.Content.Parts)
		assert.Equal(t, "test", msg.Sender)
	})

	t.Run("AssistantMessage", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).AssistantMessage("test content")
		assert.Equal(t, "test content", msg.Payload.Content.Content)
		assert.Empty(t, msg.Payload.Refusal)
	})

	t.Run("AssistantRefusal", func(t *testing.T) {
		msg := builder.WithSender("test").WithTimestamp(now).AssistantRefusal("not allowed")
		assert.Equal(t, "not allowed", msg.Payload.Refusal)
		assert.Empty(t, msg.Payload.Content.Content)
	})

	t.Run("AssistantMessageMultipart", func(t *testing.T) {
		parts := []AssistantContentPart{
			TextContentPart{Text: "part1"},
			RefusalContentPart{Refusal: "not allowed"},
		}
		msg := builder.WithSender("test").WithTimestamp(now).AssistantMessageMultipart(parts...)
		assert.Equal(t, parts, msg.Payload.Content.Parts)
	})
}

func TestToolOperations(t *testing.T) {
	t.Run("CallTool", func(t *testing.T) {
		args := gjson.Parse(`{"key": "value"}`)
		toolCall := CallTool("test-tool", args)
		assert.Equal(t, "test-tool", toolCall.Name)
		assert.Equal(t, args.Raw, toolCall.Arguments)
	})

	t.Run("ToolCall", func(t *testing.T) {
		now := strfmt.DateTime(time.Now())
		builder := messageBuilder{}
		toolData := CallTool("test-tool", gjson.Parse(`{"key": "value"}`))

		msg := builder.WithSender("test").WithTimestamp(now).ToolCall("call-id", toolData)
		assert.Equal(t, "call-id", msg.Payload.ID)
		assert.Equal(t, toolData, msg.Payload.Function[0])
	})

	t.Run("ToolResponse", func(t *testing.T) {
		now := strfmt.DateTime(time.Now())
		builder := messageBuilder{}

		msg := builder.WithSender("test").WithTimestamp(now).ToolResponse("call-id", "test-tool", "result")
		assert.Equal(t, "call-id", msg.Payload.ToolCallID)
		assert.Equal(t, "test-tool", msg.Payload.ToolName)
		assert.Equal(t, "result", msg.Payload.Content)
	})

	t.Run("ToolError", func(t *testing.T) {
		now := strfmt.DateTime(time.Now())
		builder := messageBuilder{}
		testErr := errors.New("test error")

		msg := builder.WithSender("test").WithTimestamp(now).ToolError("call-id", "test-tool", testErr)
		assert.Equal(t, "call-id", msg.Payload.ToolCallID)
		assert.Equal(t, "test-tool", msg.Payload.ToolName)
		assert.Equal(t, testErr, msg.Payload.Error)
	})
}
