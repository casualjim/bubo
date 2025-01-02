package provider

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestDelim_MarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	delim := Delim{
		RunID:  runID,
		TurnID: turnID,
		Delim:  "test-delim",
	}

	data, err := json.Marshal(delim)
	assert.NoError(t, err)

	// Verify JSON structure
	assert.True(t, gjson.ValidBytes(data))
	result := gjson.ParseBytes(data)
	assert.Equal(t, "delim", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, "test-delim", result.Get("delim").String())
}

func TestDelim_UnmarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	jsonData := []byte(`{
    "type": "delim",
    "run_id": "` + runID.String() + `",
    "turn_id": "` + turnID.String() + `",
    "delim": "test-delim"
  }`)

	var delim Delim
	err := json.Unmarshal(jsonData, &delim)
	assert.NoError(t, err)
	assert.Equal(t, runID, delim.RunID)
	assert.Equal(t, turnID, delim.TurnID)
	assert.Equal(t, "test-delim", delim.Delim)
}

func TestChunk_MarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	chunk := Chunk[messages.AssistantMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Timestamp: timestamp,
		Chunk: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: "test content",
			},
		},
		Meta: gjson.Parse(`{"key": "value"}`),
	}

	data, err := json.Marshal(chunk)
	assert.NoError(t, err)

	// Verify JSON structure
	assert.True(t, gjson.ValidBytes(data))
	result := gjson.ParseBytes(data)
	assert.Equal(t, "chunk", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "value", result.Get("meta.key").String())
}

func TestChunk_UnmarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	jsonData := []byte(`{
    "type": "chunk",
    "run_id": "` + runID.String() + `",
    "turn_id": "` + turnID.String() + `",
    "timestamp": "` + timestamp.String() + `",
    "chunk": {
      "type": "assistant",
      "content": "test content"
    },
    "meta": {
      "key": "value"
    }
  }`)

	var chunk Chunk[messages.AssistantMessage]
	err := json.Unmarshal(jsonData, &chunk)
	assert.NoError(t, err)
	assert.Equal(t, runID, chunk.RunID)
	assert.Equal(t, turnID, chunk.TurnID)
	assert.Equal(t, timestamp, chunk.Timestamp)
	assert.Equal(t, "test content", chunk.Chunk.Content.Content)
	assert.Equal(t, "value", chunk.Meta.Get("key").String())
}

func TestResponse_MarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	aggregator := shorttermmemory.New()
	response := Response[messages.AssistantMessage]{
		RunID:      runID,
		TurnID:     turnID,
		Timestamp:  timestamp,
		Checkpoint: aggregator.Checkpoint(),
		Response: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: "test response",
			},
		},
		Meta: gjson.Parse(`{"key": "value"}`),
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	// Verify JSON structure
	assert.True(t, gjson.ValidBytes(data))
	result := gjson.ParseBytes(data)
	assert.Equal(t, "response", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "value", result.Get("meta.key").String())
}

func TestResponse_UnmarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	jsonData := []byte(`{
    "type": "response",
    "run_id": "` + runID.String() + `",
    "turn_id": "` + turnID.String() + `",
    "timestamp": "` + timestamp.String() + `",
    "checkpoint": {
      "messages": [],
      "id": "` + uuid.New().String() + `"
    },
    "response": {
      "type": "assistant",
      "content": "test response"
    },
    "meta": {
      "key": "value"
    }
  }`)

	var response Response[messages.AssistantMessage]
	err := json.Unmarshal(jsonData, &response)
	assert.NoError(t, err)
	assert.Equal(t, runID, response.RunID)
	assert.Equal(t, turnID, response.TurnID)
	assert.Equal(t, timestamp, response.Timestamp)
	assert.Equal(t, "test response", response.Response.Content.Content)
	assert.Equal(t, "value", response.Meta.Get("key").String())
}

func TestError_MarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	errEvent := Error{
		RunID:     runID,
		TurnID:    turnID,
		Timestamp: timestamp,
		Err:       errors.New("test error"),
		Meta:      gjson.Parse(`{"key": "value"}`),
	}

	data, err := json.Marshal(errEvent)
	assert.NoError(t, err)

	// Verify JSON structure
	assert.True(t, gjson.ValidBytes(data))
	result := gjson.ParseBytes(data)
	assert.Equal(t, "error", result.Get("type").String())
	assert.Equal(t, runID.String(), result.Get("run_id").String())
	assert.Equal(t, turnID.String(), result.Get("turn_id").String())
	assert.Equal(t, timestamp.String(), result.Get("timestamp").String())
	assert.Equal(t, "test error", result.Get("error").String())
	assert.Equal(t, "value", result.Get("meta.key").String())
}

func TestError_UnmarshalJSON(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	jsonData := []byte(`{
    "type": "error",
    "run_id": "` + runID.String() + `",
    "turn_id": "` + turnID.String() + `",
    "timestamp": "` + timestamp.String() + `",
    "error": "test error",
    "meta": {
      "key": "value"
    }
  }`)

	var errEvent Error
	err := json.Unmarshal(jsonData, &errEvent)
	assert.NoError(t, err)
	assert.Equal(t, runID, errEvent.RunID)
	assert.Equal(t, turnID, errEvent.TurnID)
	assert.Equal(t, timestamp, errEvent.Timestamp)
	assert.Equal(t, "test error", errEvent.Err.Error())
	assert.Equal(t, "value", errEvent.Meta.Get("key").String())
}

func TestChunkToMessage(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	chunk := Chunk[messages.AssistantMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Timestamp: timestamp,
		Chunk: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: "test content",
			},
		},
		Meta: gjson.Parse(`{"key": "value"}`),
	}

	var msg messages.Message[messages.AssistantMessage]
	ChunkToMessage[messages.AssistantMessage, messages.AssistantMessage](&msg, chunk)

	assert.Equal(t, runID, msg.RunID)
	assert.Equal(t, turnID, msg.TurnID)
	assert.Equal(t, timestamp, msg.Timestamp)
	assert.Equal(t, "test content", msg.Payload.Content.Content)
	assert.Equal(t, "value", msg.Meta.Get("key").String())
}

func TestResponseToMessage(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now().UTC().Truncate(time.Millisecond))

	response := Response[messages.AssistantMessage]{
		RunID:     runID,
		TurnID:    turnID,
		Timestamp: timestamp,
		Response: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: "test response",
			},
		},
		Meta: gjson.Parse(`{"key": "value"}`),
	}

	var msg messages.Message[messages.AssistantMessage]
	ResponseToMessage[messages.AssistantMessage, messages.AssistantMessage](&msg, response)

	assert.Equal(t, runID, msg.RunID)
	assert.Equal(t, turnID, msg.TurnID)
	assert.Equal(t, timestamp, msg.Timestamp)
	assert.Equal(t, "test response", msg.Payload.Content.Content)
	assert.Equal(t, "value", msg.Meta.Get("key").String())
}
