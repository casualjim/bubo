package provider

import (
	"errors"
	"fmt"

	"github.com/casualjim/bubo/pkg/messages"
	"github.com/casualjim/bubo/pkg/runstate"
	"github.com/go-openapi/strfmt"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	delimJSON    = []byte(`{"type":"delim"}`)
	chunkJSON    = []byte(`{"type":"chunk"}`)
	responseJSON = []byte(`{"type":"response"}`)
	errorJSON    = []byte(`{"type":"error"}`)
)

type StreamEvent interface {
	streamEvent()
}

type Delim struct {
	RunID  uuid.UUID `json:"run_id"`
	TurnID uuid.UUID `json:"turn_id"`
	Delim  string    `json:"delim"`
}

func (Delim) streamEvent() {}

type Chunk[T messages.Response] struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Chunk     T               `json:"chunk"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Chunk[T]) streamEvent() {}

func ChunkToMessage[T messages.Response, M messages.ModelMessage](dst *messages.Message[M], src Chunk[T]) {
	dst.Meta = src.Meta
	dst.RunID = src.RunID
	dst.Timestamp = src.Timestamp
	dst.TurnID = src.TurnID
	dst.Meta = src.Meta
	if payload, ok := any(src.Chunk).(M); ok {
		dst.Payload = payload
	} else {
		// This should never occur, if it does definitely raise an issue.
		panic(fmt.Sprintf("invalid chunk type: %T", src.Chunk))
	}
}

type Response[T messages.Response] struct {
	RunID      uuid.UUID           `json:"run_id"`
	TurnID     uuid.UUID           `json:"turn_id"`
	Checkpoint runstate.Checkpoint `json:"checkpoint"`
	Response   T                   `json:"response"`
	Timestamp  strfmt.DateTime     `json:"timestamp,omitempty"`
	Meta       gjson.Result        `json:"meta,omitempty"`
}

func (Response[T]) streamEvent() {}

func ResponseToMessage[T messages.Response, M messages.ModelMessage](dst *messages.Message[M], src Response[T]) {
	dst.Meta = src.Meta
	dst.RunID = src.RunID
	dst.Timestamp = src.Timestamp
	dst.TurnID = src.TurnID
	dst.Meta = src.Meta
	if payload, ok := any(src.Response).(M); ok {
		dst.Payload = payload
	} else {
		// This should never occur, if it does definitely raise an issue.
		panic(fmt.Sprintf("invalid response type: %T", src.Response))
	}
}

type Error struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Err       error           `json:"error"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Error) streamEvent() {}

func (e Error) Error() string {
	return fmt.Sprintf("run_id: %s, turn_id: %s, timestamp: %s, error: %v", e.RunID, e.TurnID, e.Timestamp, e.Err)
}

// MarshalJSON implements custom JSON marshaling for Delim
func (d Delim) MarshalJSON() ([]byte, error) {
	result := delimJSON

	var err error
	result, err = sjson.SetBytes(result, "run_id", d.RunID.String())
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "turn_id", d.TurnID.String())
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "delim", d.Delim)
	return result, err
}

// UnmarshalJSON implements custom JSON unmarshaling for Delim
func (d *Delim) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "delim" {
		return fmt.Errorf("missing or invalid type, expected 'delim'")
	}

	runID := gjson.GetBytes(data, "run_id")
	if !runID.Exists() {
		return fmt.Errorf("missing required field 'run_id'")
	}
	if err := d.RunID.UnmarshalText([]byte(runID.String())); err != nil {
		return fmt.Errorf("invalid run_id: %w", err)
	}

	turnID := gjson.GetBytes(data, "turn_id")
	if !turnID.Exists() {
		return fmt.Errorf("missing required field 'turn_id'")
	}
	if err := d.TurnID.UnmarshalText([]byte(turnID.String())); err != nil {
		return fmt.Errorf("invalid turn_id: %w", err)
	}

	delim := gjson.GetBytes(data, "delim")
	if !delim.Exists() {
		return fmt.Errorf("missing required field 'delim'")
	}
	d.Delim = delim.String()

	return nil
}

// MarshalJSON implements custom JSON marshaling for Chunk[T]
func (c Chunk[T]) MarshalJSON() ([]byte, error) {
	result := chunkJSON

	var err error
	result, err = sjson.SetBytes(result, "run_id", c.RunID.String())
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "turn_id", c.TurnID.String())
	if err != nil {
		return nil, err
	}

	chunkBytes, err := json.Marshal(c.Chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk: %w", err)
	}
	result, err = sjson.SetRawBytes(result, "chunk", chunkBytes)
	if err != nil {
		return nil, err
	}

	if !c.Timestamp.IsZero() {
		result, err = sjson.SetBytes(result, "timestamp", c.Timestamp.String())
		if err != nil {
			return nil, err
		}
	}

	if c.Meta.Exists() {
		result, err = sjson.SetRawBytes(result, "meta", []byte(c.Meta.Raw))
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Chunk[T]
func (c *Chunk[T]) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "chunk" {
		return fmt.Errorf("missing or invalid type, expected 'chunk'")
	}

	runID := gjson.GetBytes(data, "run_id")
	if !runID.Exists() {
		return fmt.Errorf("missing required field 'run_id'")
	}
	if err := c.RunID.UnmarshalText([]byte(runID.String())); err != nil {
		return fmt.Errorf("invalid run_id: %w", err)
	}

	turnID := gjson.GetBytes(data, "turn_id")
	if !turnID.Exists() {
		return fmt.Errorf("missing required field 'turn_id'")
	}
	if err := c.TurnID.UnmarshalText([]byte(turnID.String())); err != nil {
		return fmt.Errorf("invalid turn_id: %w", err)
	}

	chunk := gjson.GetBytes(data, "chunk")
	if !chunk.Exists() {
		return fmt.Errorf("missing required field 'chunk'")
	}
	if err := json.Unmarshal([]byte(chunk.Raw), &c.Chunk); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	if timestamp := gjson.GetBytes(data, "timestamp"); timestamp.Exists() {
		if err := c.Timestamp.UnmarshalText([]byte(timestamp.String())); err != nil {
			return fmt.Errorf("invalid timestamp: %w", err)
		}
	}

	if meta := gjson.GetBytes(data, "meta"); meta.Exists() {
		c.Meta = meta
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Response[T]
func (r Response[T]) MarshalJSON() ([]byte, error) {
	result := responseJSON

	var err error
	result, err = sjson.SetBytes(result, "run_id", r.RunID.String())
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "turn_id", r.TurnID.String())
	if err != nil {
		return nil, err
	}

	cpj, err := json.Marshal(r.Checkpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	result, err = sjson.SetRawBytes(result, "checkpoint", cpj)
	if err != nil {
		return nil, err
	}

	responseBytes, err := json.Marshal(r.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	result, err = sjson.SetRawBytes(result, "response", responseBytes)
	if err != nil {
		return nil, err
	}

	if !r.Timestamp.IsZero() {
		result, err = sjson.SetBytes(result, "timestamp", r.Timestamp.String())
		if err != nil {
			return nil, err
		}
	}

	if r.Meta.Exists() {
		result, err = sjson.SetRawBytes(result, "meta", []byte(r.Meta.Raw))
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Response[T]
func (r *Response[T]) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "response" {
		return fmt.Errorf("missing or invalid type, expected 'response'")
	}

	runID := gjson.GetBytes(data, "run_id")
	if !runID.Exists() {
		return fmt.Errorf("missing required field 'run_id'")
	}
	if err := r.RunID.UnmarshalText([]byte(runID.String())); err != nil {
		return fmt.Errorf("invalid run_id: %w", err)
	}

	turnID := gjson.GetBytes(data, "turn_id")
	if !turnID.Exists() {
		return fmt.Errorf("missing required field 'turn_id'")
	}
	if err := r.TurnID.UnmarshalText([]byte(turnID.String())); err != nil {
		return fmt.Errorf("invalid turn_id: %w", err)
	}

	checkpoint := gjson.GetBytes(data, "checkpoint")
	if !checkpoint.Exists() {
		return fmt.Errorf("missing required field 'checkpoint'")
	}

	if err := json.Unmarshal([]byte(checkpoint.Raw), &r.Checkpoint); err != nil {
		return fmt.Errorf("invalid checkpoint: %w", err)
	}

	response := gjson.GetBytes(data, "response")
	if !response.Exists() {
		return fmt.Errorf("missing required field 'response'")
	}
	if err := json.Unmarshal([]byte(response.Raw), &r.Response); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	if timestamp := gjson.GetBytes(data, "timestamp"); timestamp.Exists() {
		if err := r.Timestamp.UnmarshalText([]byte(timestamp.String())); err != nil {
			return fmt.Errorf("invalid timestamp: %w", err)
		}
	}

	if meta := gjson.GetBytes(data, "meta"); meta.Exists() {
		r.Meta = meta
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Error
func (e Error) MarshalJSON() ([]byte, error) {
	result := errorJSON

	var err error
	result, err = sjson.SetBytes(result, "run_id", e.RunID.String())
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "turn_id", e.TurnID.String())
	if err != nil {
		return nil, err
	}

	if e.Err != nil {
		result, err = sjson.SetBytes(result, "error", e.Err.Error())
		if err != nil {
			return nil, err
		}
	}

	if !e.Timestamp.IsZero() {
		result, err = sjson.SetBytes(result, "timestamp", e.Timestamp.String())
		if err != nil {
			return nil, err
		}
	}

	if e.Meta.Exists() {
		result, err = sjson.SetRawBytes(result, "meta", []byte(e.Meta.Raw))
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Error
func (e *Error) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "error" {
		return fmt.Errorf("missing or invalid type, expected 'error'")
	}

	runID := gjson.GetBytes(data, "run_id")
	if !runID.Exists() {
		return fmt.Errorf("missing required field 'run_id'")
	}
	if err := e.RunID.UnmarshalText([]byte(runID.String())); err != nil {
		return fmt.Errorf("invalid run_id: %w", err)
	}

	turnID := gjson.GetBytes(data, "turn_id")
	if !turnID.Exists() {
		return fmt.Errorf("missing required field 'turn_id'")
	}
	if err := e.TurnID.UnmarshalText([]byte(turnID.String())); err != nil {
		return fmt.Errorf("invalid turn_id: %w", err)
	}

	errMsg := gjson.GetBytes(data, "error")
	if !errMsg.Exists() {
		return errors.New("missing required field 'error'")
	}
	e.Err = errors.New(errMsg.String())

	if timestamp := gjson.GetBytes(data, "timestamp"); timestamp.Exists() {
		if err := e.Timestamp.UnmarshalText([]byte(timestamp.String())); err != nil {
			return fmt.Errorf("invalid timestamp: %w", err)
		}
	}

	if meta := gjson.GetBytes(data, "meta"); meta.Exists() {
		e.Meta = meta
	}

	return nil
}
