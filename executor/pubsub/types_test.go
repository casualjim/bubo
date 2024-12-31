package pubsub

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/casualjim/bubo/pkg/messages"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDelimSerialization(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	delim := Delim{
		RunID:  runID,
		TurnID: turnID,
		Delim:  "test-delim",
	}

	// Test marshaling
	data, err := json.Marshal(delim)
	require.NoError(t, err)

	// Verify JSON structure
	result := gjson.ParseBytes(data)
	assert.Equal(t, "delim", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, "test-delim", result.Get("delim").String())

	// Test unmarshaling
	var unmarshaled Delim
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, delim, unmarshaled)

	// Test error cases
	testCases := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "missing type",
			json:    `{"run_id":"` + runID.String() + `","turn_id":"` + turnID.String() + `","delim":"test"}`,
			wantErr: "missing or invalid type, expected 'delim'",
		},
		{
			name:    "wrong type",
			json:    `{"type":"wrong","run_id":"` + runID.String() + `","turn_id":"` + turnID.String() + `","delim":"test"}`,
			wantErr: "missing or invalid type, expected 'delim'",
		},
		{
			name:    "missing run_id",
			json:    `{"type":"delim","turn_id":"` + turnID.String() + `","delim":"test"}`,
			wantErr: "missing required field 'run_id'",
		},
		{
			name:    "missing turn_id",
			json:    `{"type":"delim","run_id":"` + runID.String() + `","delim":"test"}`,
			wantErr: "missing required field 'turn_id'",
		},
		{
			name:    "missing delim",
			json:    `{"type":"delim","run_id":"` + runID.String() + `","turn_id":"` + turnID.String() + `"}`,
			wantErr: "missing required field 'delim'",
		},
		{
			name:    "invalid run_id",
			json:    `{"type":"delim","run_id":"invalid","turn_id":"` + turnID.String() + `","delim":"test"}`,
			wantErr: "invalid run_id",
		},
		{
			name:    "invalid turn_id",
			json:    `{"type":"delim","run_id":"` + runID.String() + `","turn_id":"invalid","delim":"test"}`,
			wantErr: "invalid turn_id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var d Delim
			err := json.Unmarshal([]byte(tc.json), &d)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestChunkSerialization(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
	toolCall := messages.ToolCallMessage{
		ToolCalls: []messages.ToolCallData{
			{
				ID:        "test-id",
				Name:      "test-tool",
				Arguments: `{"arg":"value"}`,
			},
		},
	}
	chunk := Chunk[messages.ToolCallMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Chunk:     toolCall,
		Sender:    "test-sender",
		Timestamp: timestamp,
		Meta:      gjson.Parse(`{"key":"value"}`),
	}

	// Test marshaling
	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	// Verify JSON structure
	result := gjson.ParseBytes(data)
	assert.Equal(t, "chunk", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, "test-id", result.Get("chunk.tool_calls.0.id").String())
	assert.Equal(t, "test-tool", result.Get("chunk.tool_calls.0.name").String())
	assert.Equal(t, `{"arg":"value"}`, result.Get("chunk.tool_calls.0.arguments").String())
	assert.Equal(t, "test-sender", result.Get("sender").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "value", result.Get("meta.key").String())

	// Test unmarshaling
	var unmarshaled Chunk[messages.ToolCallMessage]
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, chunk.RunID, unmarshaled.RunID)
	assert.Equal(t, chunk.TurnID, unmarshaled.TurnID)
	assert.Equal(t, chunk.Chunk, unmarshaled.Chunk)
	assert.Equal(t, chunk.Sender, unmarshaled.Sender)
	assert.Equal(t, chunk.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, chunk.Meta.Raw, unmarshaled.Meta.Raw)
}

func TestRequestSerialization(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
	request := Request[messages.ToolResponse]{
		RunID:  runID,
		TurnID: turnID,
		Message: messages.ToolResponse{
			ToolName:   "test-tool",
			ToolCallID: "test-id",
			Content:    "test-response",
		},
		Sender:    "test-sender",
		Timestamp: timestamp,
		Meta:      gjson.Parse(`{"key":"value"}`),
	}

	// Test marshaling
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Verify JSON structure
	result := gjson.ParseBytes(data)
	assert.Equal(t, "request", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, "test-tool", result.Get("message.tool_name").String())
	assert.Equal(t, "test-id", result.Get("message.tool_call_id").String())
	assert.Equal(t, "test-response", result.Get("message.content").String())
	assert.Equal(t, "test-sender", result.Get("sender").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "value", result.Get("meta.key").String())

	// Test unmarshaling
	var unmarshaled Request[messages.ToolResponse]
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, request.RunID, unmarshaled.RunID)
	assert.Equal(t, request.TurnID, unmarshaled.TurnID)
	assert.Equal(t, request.Message, unmarshaled.Message)
	assert.Equal(t, request.Sender, unmarshaled.Sender)
	assert.Equal(t, request.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, request.Meta.Raw, unmarshaled.Meta.Raw)
}

func TestResponseSerialization(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
	toolCall := messages.ToolCallMessage{
		ToolCalls: []messages.ToolCallData{
			{
				ID:        "test-id",
				Name:      "test-tool",
				Arguments: `{"arg":"value"}`,
			},
		},
	}
	response := Response[messages.ToolCallMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Response:  toolCall,
		Sender:    "test-sender",
		Timestamp: timestamp,
		Meta:      gjson.Parse(`{"key":"value"}`),
	}

	// Test marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Verify JSON structure
	result := gjson.ParseBytes(data)
	assert.Equal(t, "response", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, "test-id", result.Get("response.tool_calls.0.id").String())
	assert.Equal(t, "test-tool", result.Get("response.tool_calls.0.name").String())
	assert.Equal(t, `{"arg":"value"}`, result.Get("response.tool_calls.0.arguments").String())
	assert.Equal(t, "test-sender", result.Get("sender").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "value", result.Get("meta.key").String())

	// Test unmarshaling
	var unmarshaled Response[messages.ToolCallMessage]
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.RunID, unmarshaled.RunID)
	assert.Equal(t, response.TurnID, unmarshaled.TurnID)
	assert.Equal(t, response.Response, unmarshaled.Response)
	assert.Equal(t, response.Sender, unmarshaled.Sender)
	assert.Equal(t, response.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, response.Meta.Raw, unmarshaled.Meta.Raw)
}

func TestErrorSerialization(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
	errEvent := Error{
		RunID:     runID,
		TurnID:    turnID,
		Err:       errors.New("test-error"),
		Sender:    "test-sender",
		Timestamp: timestamp,
		Meta:      gjson.Parse(`{"key":"value"}`),
	}

	// Test marshaling
	data, err := json.Marshal(errEvent)
	require.NoError(t, err)

	// Verify JSON structure
	result := gjson.ParseBytes(data)
	assert.Equal(t, "error", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, "test-error", result.Get("error").String())
	assert.Equal(t, "test-sender", result.Get("sender").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "value", result.Get("meta.key").String())

	// Test unmarshaling
	var unmarshaled Error
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, errEvent.RunID, unmarshaled.RunID)
	assert.Equal(t, errEvent.TurnID, unmarshaled.TurnID)
	assert.Equal(t, errEvent.Err.Error(), unmarshaled.Err.Error())
	assert.Equal(t, errEvent.Sender, unmarshaled.Sender)
	assert.Equal(t, errEvent.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, errEvent.Meta.Raw, unmarshaled.Meta.Raw)

	// Test Error() string method
	assert.Contains(t, errEvent.Error(), "test-error")
	assert.Contains(t, errEvent.Error(), runID.String())
	assert.Contains(t, errEvent.Error(), turnID.String())
}
