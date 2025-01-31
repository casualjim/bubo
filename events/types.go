package events

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	delimJSON    = []byte(`{"type":"delim"}`)
	chunkJSON    = []byte(`{"type":"chunk"}`)
	requestJSON  = []byte(`{"type":"request"}`)
	responseJSON = []byte(`{"type":"response"}`)
	errorJSON    = []byte(`{"type":"error"}`)
)

type Event interface {
	pubsubEvent()
}

func FromStreamEvent(e provider.StreamEvent, sender string) Event {
	switch event := e.(type) {
	case provider.Delim:
		return Delim(event)
	case provider.Chunk[messages.ToolCallMessage]:
		return Chunk[messages.ToolCallMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Chunk:     event.Chunk,
			Sender:    sender,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		}
	case provider.Chunk[messages.AssistantMessage]:
		return Chunk[messages.AssistantMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Chunk:     event.Chunk,
			Sender:    sender,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		}
	case provider.Response[messages.ToolCallMessage]:
		return Response[messages.ToolCallMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Response:  event.Response,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
			Sender:    sender,
		}
	case provider.Response[messages.AssistantMessage]:
		return Response[messages.AssistantMessage]{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Response:  event.Response,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
			Sender:    sender,
		}
	case provider.Error:
		return Error{
			RunID:     event.RunID,
			TurnID:    event.TurnID,
			Err:       event.Err,
			Sender:    sender,
			Timestamp: event.Timestamp,
			Meta:      event.Meta,
		}
	default:
		panic(fmt.Sprintf("unknown event type: %T", event))
	}
}

func ToJSON(event Event) ([]byte, error) {
	switch e := event.(type) {
	case Delim:
		return json.Marshal(e)
	case Chunk[messages.AssistantMessage]:
		return json.Marshal(e)
	case Chunk[messages.ToolCallMessage]:
		return json.Marshal(e)
	case Request[messages.UserMessage]:
		return json.Marshal(e)
	case Request[messages.ToolResponse]:
		return json.Marshal(e)
	case Response[messages.AssistantMessage]:
		return json.Marshal(e)
	case Response[messages.ToolCallMessage]:
		return json.Marshal(e)
	case Error:
		return json.Marshal(e)
	default:
		return nil, fmt.Errorf("unknown event type: %T", event)
	}
}

func FromJSON(jsonData []byte) (Event, error) {
	if !gjson.ValidBytes(jsonData) {
		return nil, fmt.Errorf("invalid JSON: %s", jsonData)
	}
	jv := gjson.ParseBytes(jsonData)
	et := jv.Get("type").String()
	switch et {
	case "delim":
		var d Delim
		if err := json.Unmarshal(jsonData, &d); err != nil {
			return nil, err
		}
		return d, nil
	case "chunk":
		ct := jv.Get("chunk.type").String()
		switch ct {
		case "assistant":
			var d Chunk[messages.AssistantMessage]
			if err := json.Unmarshal(jsonData, &d); err != nil {
				return nil, err
			}
			return d, nil
		case "tool_call":
			var d Chunk[messages.ToolCallMessage]
			if err := json.Unmarshal(jsonData, &d); err != nil {
				return nil, err
			}
			return d, nil
		default:
			return nil, fmt.Errorf("failed to parse event chunk type: %s", ct)
		}
	case "request":
		ct := jv.Get("message.type").String()
		switch ct {
		case "user":
			var d Request[messages.UserMessage]
			if err := json.Unmarshal(jsonData, &d); err != nil {
				return nil, err
			}
			return d, nil
		case "tool_response":
			var d Request[messages.ToolResponse]
			if err := json.Unmarshal(jsonData, &d); err != nil {
				return nil, err
			}
			return d, nil
		default:
			return nil, fmt.Errorf("failed to parse event request type: %s", ct)
		}
	case "response":
		ct := jv.Get("response.type").String()
		switch ct {
		case "assistant":
			var d Response[messages.AssistantMessage]
			if err := json.Unmarshal(jsonData, &d); err != nil {
				return nil, err
			}
			return d, nil
		case "tool_call":
			var d Response[messages.ToolCallMessage]
			if err := json.Unmarshal(jsonData, &d); err != nil {
				return nil, err
			}
			return d, nil
		default:
			return nil, fmt.Errorf("failed to parse event response type: %s", ct)
		}
	case "error":
		var e Error
		if err := json.Unmarshal(jsonData, &e); err != nil {
			return nil, err
		}
		return e, nil
	default:
		return nil, fmt.Errorf("failed to parse event type: %s", et)
	}
}

type Delim struct {
	RunID  uuid.UUID `json:"run_id"`
	TurnID uuid.UUID `json:"turn_id"`
	Delim  string    `json:"delim"`
}

func (Delim) pubsubEvent() {}

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

type Chunk[T messages.Response] struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Chunk     T               `json:"chunk"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Chunk[T]) pubsubEvent() {}

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

	if c.Sender != "" {
		result, err = sjson.SetBytes(result, "sender", c.Sender)
		if err != nil {
			return nil, err
		}
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

	if sender := gjson.GetBytes(data, "sender"); sender.Exists() {
		c.Sender = sender.String()
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

type Request[T messages.Request] struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Message   T               `json:"message"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Request[T]) pubsubEvent() {}

// MarshalJSON implements custom JSON marshaling for Request[T]
func (r Request[T]) MarshalJSON() ([]byte, error) {
	result := requestJSON

	var err error
	result, err = sjson.SetBytes(result, "run_id", r.RunID.String())
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "turn_id", r.TurnID.String())
	if err != nil {
		return nil, err
	}

	messageBytes, err := json.Marshal(r.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	result, err = sjson.SetRawBytes(result, "message", messageBytes)
	if err != nil {
		return nil, err
	}

	if r.Sender != "" {
		result, err = sjson.SetBytes(result, "sender", r.Sender)
		if err != nil {
			return nil, err
		}
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

// UnmarshalJSON implements custom JSON unmarshaling for Request[T]
func (r *Request[T]) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "request" {
		return fmt.Errorf("missing or invalid type, expected 'request'")
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

	message := gjson.GetBytes(data, "message")
	if !message.Exists() {
		return fmt.Errorf("missing required field 'message'")
	}
	if err := json.Unmarshal([]byte(message.Raw), &r.Message); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	if sender := gjson.GetBytes(data, "sender"); sender.Exists() {
		r.Sender = sender.String()
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

type Response[T messages.Response] struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Response  T               `json:"response"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Response[T]) pubsubEvent() {}

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

	responseBytes, err := json.Marshal(r.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	result, err = sjson.SetRawBytes(result, "response", responseBytes)
	if err != nil {
		return nil, err
	}

	if r.Sender != "" {
		result, err = sjson.SetBytes(result, "sender", r.Sender)
		if err != nil {
			return nil, err
		}
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

	response := gjson.GetBytes(data, "response")
	if !response.Exists() {
		return fmt.Errorf("missing required field 'response'")
	}
	if err := json.Unmarshal([]byte(response.Raw), &r.Response); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	if sender := gjson.GetBytes(data, "sender"); sender.Exists() {
		r.Sender = sender.String()
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

type Result[T any] struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Result    T               `json:"result"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Result[T]) pubsubEvent() {}

// MarshalJSON implements custom JSON marshaling for Result[T]
func (r Result[T]) MarshalJSON() ([]byte, error) {
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

	responseBytes, err := json.Marshal(r.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Result: %w", err)
	}
	result, err = sjson.SetRawBytes(result, "result", responseBytes)
	if err != nil {
		return nil, err
	}

	if r.Sender != "" {
		result, err = sjson.SetBytes(result, "sender", r.Sender)
		if err != nil {
			return nil, err
		}
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

// UnmarshalJSON implements custom JSON unmarshaling for Result[T]
func (r *Result[T]) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "result" {
		return fmt.Errorf("missing or invalid type, expected 'result'")
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

	response := gjson.GetBytes(data, "result")
	if !response.Exists() {
		return fmt.Errorf("missing required field 'result'")
	}
	if err := json.Unmarshal([]byte(response.Raw), &r.Result); err != nil {
		return fmt.Errorf("invalid result: %w", err)
	}

	if sender := gjson.GetBytes(data, "sender"); sender.Exists() {
		r.Sender = sender.String()
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

type Error struct {
	RunID     uuid.UUID       `json:"run_id"`
	TurnID    uuid.UUID       `json:"turn_id"`
	Err       error           `json:"error"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"`
}

func (Error) pubsubEvent() {}

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

	if e.Sender != "" {
		result, err = sjson.SetBytes(result, "sender", e.Sender)
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

	if sender := gjson.GetBytes(data, "sender"); sender.Exists() {
		e.Sender = sender.String()
	}

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

func (e Error) Error() string {
	errStr := "<nil>"
	if e.Err != nil {
		errStr = e.Err.Error()
	}
	return fmt.Sprintf("%s run_id=%s turn_id=%s", errStr, e.RunID, e.TurnID)
}
