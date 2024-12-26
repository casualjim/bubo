package messages

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestInstructions_message(t *testing.T) {
	i := Instructions{}
	i.message()
}

func TestInstructions(t *testing.T) {
	i := Instructions{
		Content: "test content",
		Sender:  "test sender",
	}
	assert.Equal(t, "test content", i.Content)
	assert.Equal(t, "test sender", i.Sender)
}

func TestUserPrompt_message(t *testing.T) {
	u := UserPrompt{}
	u.message()
}

func TestUserPrompt_request(t *testing.T) {
	u := UserPrompt{}
	u.request()
}

func TestUserPrompt(t *testing.T) {
	now := strfmt.DateTime(time.Now())
	content := ContentOrParts{Content: "test content"}
	u := UserPrompt{
		Content:   content,
		Sender:    "test sender",
		Timestamp: now,
	}
	assert.Equal(t, content, u.Content)
	assert.Equal(t, "test sender", u.Sender)
	assert.Equal(t, now, u.Timestamp)
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
	now := strfmt.DateTime(time.Now())
	a := AssistantMessage{
		Content:   "test content",
		Refusal:   "test refusal",
		Sender:    "test sender",
		Timestamp: now,
	}
	assert.Equal(t, "test content", a.Content)
	assert.Equal(t, "test refusal", a.Refusal)
	assert.Equal(t, "test sender", a.Sender)
	assert.Equal(t, now, a.Timestamp)
}

func TestToolCall_message(t *testing.T) {
	tc := ToolCall{}
	tc.message()
}

func TestToolCall_response(t *testing.T) {
	tc := ToolCall{}
	tc.response()
}

func TestToolCall(t *testing.T) {
	now := strfmt.DateTime(time.Now())
	tc := ToolCall{
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
	now := strfmt.DateTime(time.Now())
	tr := ToolResponse{
		ToolName:   "test tool",
		ToolCallID: "test-call-id",
		Content:    "test content",
		Timestamp:  now,
	}
	assert.Equal(t, "test tool", tr.ToolName)
	assert.Equal(t, "test-call-id", tr.ToolCallID)
	assert.Equal(t, "test content", tr.Content)
	assert.Equal(t, now, tr.Timestamp)
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
	now := strfmt.DateTime(time.Now())
	err := assert.AnError
	r := Retry{
		Error:      err,
		ToolName:   "test tool",
		ToolCallID: "test-call-id",
		Timestamp:  now,
	}
	assert.Equal(t, err, r.Error)
	assert.Equal(t, "test tool", r.ToolName)
	assert.Equal(t, "test-call-id", r.ToolCallID)
	assert.Equal(t, now, r.Timestamp)
}
