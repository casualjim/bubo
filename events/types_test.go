package events

import (
	"errors"
	"testing"
	"time"

	"github.com/casualjim/bubo/messages"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDelimJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	delim := Delim{
		RunID:  runID,
		TurnID: turnID,
		Delim:  "test",
	}

	t.Run("marshal", func(t *testing.T) {
		data, err := delim.MarshalJSON()
		require.NoError(t, err)

		result := gjson.ParseBytes(data)
		assert.Equal(t, "delim", result.Get("type").String())
		assert.Equal(t, runID.String(), result.Get("run_id").String())
		assert.Equal(t, turnID.String(), result.Get("turn_id").String())
		assert.Equal(t, "test", result.Get("delim").String())
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := []byte(`{
			"type": "delim",
			"run_id": "` + runID.String() + `",
			"turn_id": "` + turnID.String() + `",
			"delim": "test"
		}`)

		var d Delim
		err := d.UnmarshalJSON(input)
		require.NoError(t, err)
		assert.Equal(t, delim, d)
	})

	t.Run("unmarshal errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "invalid json",
				input: "invalid",
			},
			{
				name:  "missing type",
				input: `{"run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "wrong type",
				input: `{"type": "wrong", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "missing run_id",
				input: `{"type": "delim"}`,
			},
			{
				name:  "invalid run_id",
				input: `{"type": "delim", "run_id": "invalid"}`,
			},
			{
				name:  "missing turn_id",
				input: `{"type": "delim", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "invalid turn_id",
				input: `{"type": "delim", "run_id": "` + runID.String() + `", "turn_id": "invalid"}`,
			},
			{
				name:  "missing delim",
				input: `{"type": "delim", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `"}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var d Delim
				err := d.UnmarshalJSON([]byte(tt.input))
				assert.Error(t, err)
			})
		}
	})
}

func TestChunkJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))
	meta := gjson.Parse(`{"key":"value"}`)

	msg := messages.New().AssistantMessage("test")
	chunk := Chunk[messages.AssistantMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Chunk:     msg.Payload,
		Sender:    "test",
		Timestamp: timestamp,
		Meta:      meta,
	}

	t.Run("marshal", func(t *testing.T) {
		data, err := chunk.MarshalJSON()
		require.NoError(t, err)

		result := gjson.ParseBytes(data)
		assert.Equal(t, "chunk", result.Get("type").String())
		assert.Equal(t, runID.String(), result.Get("run_id").String())
		assert.Equal(t, turnID.String(), result.Get("turn_id").String())
		assert.True(t, result.Get("chunk").Exists())
		assert.Equal(t, "test", result.Get("sender").String())
		assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
		assert.Equal(t, "value", result.Get("meta.key").String())
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := []byte(`{
			"type": "chunk",
			"run_id": "` + runID.String() + `",
			"turn_id": "` + turnID.String() + `",
			"chunk": {"type": "assistant", "content": "test"},
			"sender": "test",
			"timestamp": "` + timestamp.String() + `",
			"meta": {"key":"value"}
		}`)

		var c Chunk[messages.AssistantMessage]
		err := c.UnmarshalJSON(input)
		require.NoError(t, err)
		assert.Equal(t, chunk.RunID, c.RunID)
		assert.Equal(t, chunk.TurnID, c.TurnID)
		assert.Equal(t, chunk.Sender, c.Sender)
		assert.Equal(t, chunk.Timestamp, c.Timestamp)
		assert.Equal(t, chunk.Meta.Raw, c.Meta.Raw)
	})

	t.Run("unmarshal errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "invalid json",
				input: "invalid",
			},
			{
				name:  "missing type",
				input: `{"run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "wrong type",
				input: `{"type": "wrong", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "missing run_id",
				input: `{"type": "chunk"}`,
			},
			{
				name:  "invalid run_id",
				input: `{"type": "chunk", "run_id": "invalid"}`,
			},
			{
				name:  "missing turn_id",
				input: `{"type": "chunk", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "invalid turn_id",
				input: `{"type": "chunk", "run_id": "` + runID.String() + `", "turn_id": "invalid"}`,
			},
			{
				name:  "missing chunk",
				input: `{"type": "chunk", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `"}`,
			},
			{
				name:  "invalid chunk",
				input: `{"type": "chunk", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "chunk": "invalid"}`,
			},
			{
				name:  "invalid timestamp",
				input: `{"type": "chunk", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "chunk": {}, "timestamp": "invalid"}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var c Chunk[messages.AssistantMessage]
				err := c.UnmarshalJSON([]byte(tt.input))
				assert.Error(t, err)
			})
		}
	})
}

func TestRequestJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))
	meta := gjson.Parse(`{"key":"value"}`)

	msg := messages.New().UserPrompt("test")
	request := Request[messages.UserMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Message:   msg.Payload,
		Sender:    "test",
		Timestamp: timestamp,
		Meta:      meta,
	}

	t.Run("marshal", func(t *testing.T) {
		data, err := request.MarshalJSON()
		require.NoError(t, err)

		result := gjson.ParseBytes(data)
		assert.Equal(t, "request", result.Get("type").String())
		assert.Equal(t, runID.String(), result.Get("run_id").String())
		assert.Equal(t, turnID.String(), result.Get("turn_id").String())
		assert.True(t, result.Get("message").Exists())
		assert.Equal(t, "test", result.Get("sender").String())
		assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
		assert.Equal(t, "value", result.Get("meta.key").String())
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := []byte(`{
			"type": "request",
			"run_id": "` + runID.String() + `",
			"turn_id": "` + turnID.String() + `",
			"message": {"type": "user", "content": "test"},
			"sender": "test",
			"timestamp": "` + timestamp.String() + `",
			"meta": {"key":"value"}
		}`)

		var r Request[messages.UserMessage]
		err := r.UnmarshalJSON(input)
		require.NoError(t, err)
		assert.Equal(t, request.RunID, r.RunID)
		assert.Equal(t, request.TurnID, r.TurnID)
		assert.Equal(t, request.Sender, r.Sender)
		assert.Equal(t, request.Timestamp, r.Timestamp)
		assert.Equal(t, request.Meta.Raw, r.Meta.Raw)
	})

	t.Run("unmarshal errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "invalid json",
				input: "invalid",
			},
			{
				name:  "missing type",
				input: `{"run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "wrong type",
				input: `{"type": "wrong", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "missing run_id",
				input: `{"type": "request"}`,
			},
			{
				name:  "invalid run_id",
				input: `{"type": "request", "run_id": "invalid"}`,
			},
			{
				name:  "missing turn_id",
				input: `{"type": "request", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "invalid turn_id",
				input: `{"type": "request", "run_id": "` + runID.String() + `", "turn_id": "invalid"}`,
			},
			{
				name:  "missing message",
				input: `{"type": "request", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `"}`,
			},
			{
				name:  "invalid message",
				input: `{"type": "request", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "message": "invalid"}`,
			},
			{
				name:  "invalid timestamp",
				input: `{"type": "request", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "message": {}, "timestamp": "invalid"}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var r Request[messages.UserMessage]
				err := r.UnmarshalJSON([]byte(tt.input))
				assert.Error(t, err)
			})
		}
	})
}

func TestResponseJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))
	meta := gjson.Parse(`{"key":"value"}`)

	msg := messages.New().AssistantMessage("test")
	response := Response[messages.AssistantMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Response:  msg.Payload,
		Sender:    "test",
		Timestamp: timestamp,
		Meta:      meta,
	}

	t.Run("marshal", func(t *testing.T) {
		data, err := response.MarshalJSON()
		require.NoError(t, err)

		result := gjson.ParseBytes(data)
		assert.Equal(t, "response", result.Get("type").String())
		assert.Equal(t, runID.String(), result.Get("run_id").String())
		assert.Equal(t, turnID.String(), result.Get("turn_id").String())
		assert.True(t, result.Get("response").Exists())
		assert.Equal(t, "test", result.Get("sender").String())
		assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
		assert.Equal(t, "value", result.Get("meta.key").String())
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := []byte(`{
			"type": "response",
			"run_id": "` + runID.String() + `",
			"turn_id": "` + turnID.String() + `",
			"response": {"type": "assistant", "content": "test"},
			"sender": "test",
			"timestamp": "` + timestamp.String() + `",
			"meta": {"key":"value"}
		}`)

		var r Response[messages.AssistantMessage]
		err := r.UnmarshalJSON(input)
		require.NoError(t, err)
		assert.Equal(t, response.RunID, r.RunID)
		assert.Equal(t, response.TurnID, r.TurnID)
		assert.Equal(t, response.Sender, r.Sender)
		assert.Equal(t, response.Timestamp, r.Timestamp)
		assert.Equal(t, response.Meta.Raw, r.Meta.Raw)
	})

	t.Run("unmarshal errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "invalid json",
				input: "invalid",
			},
			{
				name:  "missing type",
				input: `{"run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "wrong type",
				input: `{"type": "wrong", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "missing run_id",
				input: `{"type": "response"}`,
			},
			{
				name:  "invalid run_id",
				input: `{"type": "response", "run_id": "invalid"}`,
			},
			{
				name:  "missing turn_id",
				input: `{"type": "response", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "invalid turn_id",
				input: `{"type": "response", "run_id": "` + runID.String() + `", "turn_id": "invalid"}`,
			},
			{
				name:  "missing response",
				input: `{"type": "response", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `"}`,
			},
			{
				name:  "invalid response",
				input: `{"type": "response", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "response": "invalid"}`,
			},
			{
				name:  "invalid timestamp",
				input: `{"type": "response", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "response": {}, "timestamp": "invalid"}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var r Response[messages.AssistantMessage]
				err := r.UnmarshalJSON([]byte(tt.input))
				assert.Error(t, err)
			})
		}
	})
}

func TestEventSerialization(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))
	meta := gjson.Parse(`{"key":"value"}`)

	t.Run("ToJSON", func(t *testing.T) {
		tests := []struct {
			name    string
			event   Event
			wantErr bool
		}{
			{
				name: "Delim",
				event: Delim{
					RunID:  runID,
					TurnID: turnID,
					Delim:  "test",
				},
			},
			{
				name: "Chunk AssistantMessage",
				event: Chunk[messages.AssistantMessage]{
					RunID:     runID,
					TurnID:    turnID,
					Chunk:     messages.New().AssistantMessage("test").Payload,
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
			{
				name: "Chunk ToolCallMessage",
				event: Chunk[messages.ToolCallMessage]{
					RunID:     runID,
					TurnID:    turnID,
					Chunk:     messages.New().ToolCall([]messages.ToolCallData{{Name: "test", Arguments: "{}"}}).Payload,
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
			{
				name: "Request UserMessage",
				event: Request[messages.UserMessage]{
					RunID:     runID,
					TurnID:    turnID,
					Message:   messages.New().UserPrompt("test").Payload,
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
			{
				name: "Request ToolResponse",
				event: Request[messages.ToolResponse]{
					RunID:     runID,
					TurnID:    turnID,
					Message:   messages.New().ToolResponse("test12", "test", "{}").Payload,
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
			{
				name: "Response AssistantMessage",
				event: Response[messages.AssistantMessage]{
					RunID:     runID,
					TurnID:    turnID,
					Response:  messages.New().AssistantMessage("test").Payload,
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
			{
				name: "Response ToolCallMessage",
				event: Response[messages.ToolCallMessage]{
					RunID:     runID,
					TurnID:    turnID,
					Response:  messages.New().ToolCall([]messages.ToolCallData{{Name: "test", Arguments: "{}"}}).Payload,
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
			{
				name: "Error",
				event: Error{
					RunID:     runID,
					TurnID:    turnID,
					Err:       errors.New("test error"),
					Sender:    "test",
					Timestamp: timestamp,
					Meta:      meta,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := ToJSON(tt.event)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.NotNil(t, data)

				// Verify we can unmarshal it back
				event, err := FromJSON(data)
				require.NoError(t, err)
				assert.IsType(t, tt.event, event)
			})
		}
	})

	t.Run("FromJSON errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "invalid json",
				input: "invalid",
			},
			{
				name:  "missing type",
				input: `{"run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "unknown type",
				input: `{"type": "unknown"}`,
			},
			{
				name:  "invalid chunk type",
				input: `{"type": "chunk", "chunk": {"type": "unknown"}}`,
			},
			{
				name:  "invalid request type",
				input: `{"type": "request", "message": {"type": "unknown"}}`,
			},
			{
				name:  "invalid response type",
				input: `{"type": "response", "message": {"type": "unknown"}}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := FromJSON([]byte(tt.input))
				assert.Error(t, err)
			})
		}
	})
}

func TestErrorJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))
	meta := gjson.Parse(`{"key":"value"}`)
	testErr := errors.New("test error")

	errEvent := Error{
		RunID:     runID,
		TurnID:    turnID,
		Err:       testErr,
		Sender:    "test",
		Timestamp: timestamp,
		Meta:      meta,
	}

	t.Run("marshal", func(t *testing.T) {
		data, err := errEvent.MarshalJSON()
		require.NoError(t, err)

		result := gjson.ParseBytes(data)
		assert.Equal(t, "error", result.Get("type").String())
		assert.Equal(t, runID.String(), result.Get("run_id").String())
		assert.Equal(t, turnID.String(), result.Get("turn_id").String())
		assert.Equal(t, testErr.Error(), result.Get("error").String())
		assert.Equal(t, "test", result.Get("sender").String())
		assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
		assert.Equal(t, "value", result.Get("meta.key").String())
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := []byte(`{
			"type": "error",
			"run_id": "` + runID.String() + `",
			"turn_id": "` + turnID.String() + `",
			"error": "test error",
			"sender": "test",
			"timestamp": "` + timestamp.String() + `",
			"meta": {"key": "value"}
		}`)

		var e Error
		err := e.UnmarshalJSON(input)
		require.NoError(t, err)
		assert.Equal(t, errEvent.RunID, e.RunID)
		assert.Equal(t, errEvent.TurnID, e.TurnID)
		assert.Equal(t, errEvent.Err.Error(), e.Err.Error())
		assert.Equal(t, errEvent.Sender, e.Sender)
		assert.Equal(t, errEvent.Timestamp, e.Timestamp)
		assert.Equal(t, `{"key": "value"}`, e.Meta.Raw)
	})

	t.Run("Error() method", func(t *testing.T) {
		errStr := errEvent.Error()
		assert.Contains(t, errStr, testErr.Error())
		assert.Contains(t, errStr, runID.String())
		assert.Contains(t, errStr, turnID.String())

		// Test with nil error
		errEvent.Err = nil
		errStr = errEvent.Error()
		assert.Contains(t, errStr, "<nil>")
	})

	t.Run("unmarshal errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "invalid json",
				input: "invalid",
			},
			{
				name:  "missing type",
				input: `{"run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "wrong type",
				input: `{"type": "wrong", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "missing run_id",
				input: `{"type": "error"}`,
			},
			{
				name:  "invalid run_id",
				input: `{"type": "error", "run_id": "invalid"}`,
			},
			{
				name:  "missing turn_id",
				input: `{"type": "error", "run_id": "` + runID.String() + `"}`,
			},
			{
				name:  "invalid turn_id",
				input: `{"type": "error", "run_id": "` + runID.String() + `", "turn_id": "invalid"}`,
			},
			{
				name:  "missing error",
				input: `{"type": "error", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `"}`,
			},
			{
				name:  "invalid timestamp",
				input: `{"type": "error", "run_id": "` + runID.String() + `", "turn_id": "` + turnID.String() + `", "error": "test", "timestamp": "invalid"}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var e Error
				err := e.UnmarshalJSON([]byte(tt.input))
				assert.Error(t, err)
			})
		}
	})
}
