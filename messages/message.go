// Package messages provides a messaging system for handling various types of communication
// between different parts of the application. It includes support for user messages,
// assistant messages, tool calls, and responses.
package messages

import (
	"errors"
	"fmt"
	"time"

	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/go-openapi/strfmt"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	instructionsJSON = []byte(`{"type":"instructions"}`)
	userMessageJSON  = []byte(`{"type":"user"}`)
	assistantJSON    = []byte(`{"type":"assistant"}`)
	toolCallJSON     = []byte(`{"type":"tool_call"}`)
	toolResponseJSON = []byte(`{"type":"tool_response"}`)
	retryJSON        = []byte(`{"type":"retry"}`)
)

// ModelMessage is a marker interface that all message types must implement.
// It ensures type safety for message handling throughout the application.
type ModelMessage interface {
	message()
}

// Request is a marker interface that identifies messages that can be used
// as requests in the system. This includes user messages and tool responses.
type Request interface {
	request()
}

// Response is a marker interface that identifies messages that can be used
// as responses in the system. This includes assistant messages and tool calls.
type Response interface {
	response()
}

// messageBuilder is a helper struct that implements the builder pattern for creating
// different types of messages. It maintains common fields like sender and timestamp
// that are shared across all message types.
type messageBuilder struct {
	runID     uuid.UUID
	turnID    uuid.UUID
	sender    string          // The entity that sent the message
	timestamp strfmt.DateTime // When the message was created
	metadata  gjson.Result    // Additional metadata to be included with the message
}

func wrap[T ModelMessage](bldr *messageBuilder, msg T) Message[T] {
	return Message[T]{
		RunID:     bldr.runID,
		TurnID:    bldr.turnID,
		Sender:    bldr.sender,
		Timestamp: bldr.timestamp,
		Meta:      bldr.metadata,
		Payload:   msg,
	}
}

// New creates a new messageBuilder instance with the current timestamp.
// It initializes a builder that can be used to create various types of messages.
func New() messageBuilder {
	return messageBuilder{
		runID:     uuidx.New(),
		turnID:    uuidx.New(),
		timestamp: strfmt.DateTime(time.Now()),
	}
}

// WithSender sets the sender field in the messageBuilder and returns a new builder instance.
// This method is part of the builder pattern and allows for method chaining.
func (b messageBuilder) WithSender(sender string) messageBuilder {
	b.sender = sender
	return b
}

// WithTimestamp sets a custom timestamp in the messageBuilder and returns a new builder instance.
// This method is part of the builder pattern and allows for method chaining.
func (b messageBuilder) WithTimestamp(timestamp strfmt.DateTime) messageBuilder {
	b.timestamp = timestamp
	return b
}

// WithMetadata sets custom metadata in the messageBuilder and returns a new builder instance.
// This method is part of the builder pattern and allows for method chaining.
func (b messageBuilder) WithMetadata(metadata gjson.Result) messageBuilder {
	b.metadata = metadata
	return b
}

// Instructions creates a new instruction message with the given content.
// This type of message is typically used to provide system-level instructions.
func (b messageBuilder) Instructions(content string) Message[InstructionsMessage] {
	return wrap(&b, InstructionsMessage{Content: content})
}

// UserPrompt creates a new user message with single-part content.
// This is used when the user sends a simple text message.
func (b messageBuilder) UserPrompt(content string) Message[UserMessage] {
	return wrap(&b, UserMessage{Content: ContentOrParts{Content: content}})
}

// UserPromptMultipart creates a new user message with multiple content parts.
// This is used when the user message contains multiple components (e.g., text and images).
func (b messageBuilder) UserPromptMultipart(parts ...ContentPart) Message[UserMessage] {
	return wrap(&b, UserMessage{Content: ContentOrParts{Parts: parts}})
}

// AssistantMessage creates a new assistant message with single-part content.
// This is used when the assistant responds with a simple text message.
func (b messageBuilder) AssistantMessage(content string) Message[AssistantMessage] {
	return wrap(&b, AssistantMessage{
		Content: AssistantContentOrParts{Content: content},
	})
}

// AssistantRefusal creates a new assistant message indicating a refusal to perform an action.
// This is used when the assistant needs to decline a request with an explanation.
func (b messageBuilder) AssistantRefusal(refusal string) Message[AssistantMessage] {
	return wrap(&b, AssistantMessage{
		Refusal: refusal,
	})
}

// AssistantMessageMultipart creates a new assistant message with multiple content parts.
// This is used when the assistant needs to respond with structured or multi-component content.
func (b messageBuilder) AssistantMessageMultipart(parts ...AssistantContentPart) Message[AssistantMessage] {
	return wrap(&b, AssistantMessage{
		Content: AssistantContentOrParts{Parts: parts},
	})
}

// CallTool creates a new ToolCallData instance with the given name and arguments.
// This is a helper function for creating tool call requests with JSON arguments.
func CallTool(id, name string, args gjson.Result) ToolCallData {
	return ToolCallData{
		ID:        id,
		Name:      name,
		Arguments: args.Raw,
	}
}

// ToolCall creates a new tool call message with the specified ID and tool call data.
// This is used when the assistant needs to invoke one or more external tools.
func (b messageBuilder) ToolCall(toolCalls []ToolCallData) Message[ToolCallMessage] {
	return wrap(&b, ToolCallMessage{
		ToolCalls: toolCalls,
	})
}

// ToolResponse creates a new tool response message with the result of a tool execution.
// This is used to return the results of a successful tool call.
func (b messageBuilder) ToolResponse(id, name, result string) Message[ToolResponse] {
	return wrap(&b, ToolResponse{
		ToolCallID: id,
		ToolName:   name,
		Content:    result,
	})
}

// ToolError creates a new retry message when a tool execution fails.
// This is used to handle and communicate tool execution errors.
func (b messageBuilder) ToolError(id, name string, error error) Message[Retry] {
	return wrap(&b, Retry{
		ToolCallID: id,
		ToolName:   name,
		Error:      error,
	})
}

// Message is a generic container for all message types in the system.
// It includes common metadata like sender and timestamp alongside the specific message payload.
type Message[T ModelMessage] struct {
	RunID     uuid.UUID       `json:"run_id,omitempty"`  // ID of the run this message belongs to
	TurnID    uuid.UUID       `json:"turn_id,omitempty"` // ID of the turn this message belongs to
	Payload   T               `json:",inline"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
	Meta      gjson.Result    `json:"meta,omitempty"` // Additional metadata that can be attached to any message type
}

// MarshalJSON implements custom JSON marshaling for Message[T]
func (m Message[T]) MarshalJSON() ([]byte, error) {
	// Marshal the payload first
	payloadBytes, err := json.Marshal(m.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Start with the payload as base
	result := payloadBytes

	// Add other fields if they're present
	if m.RunID != uuid.Nil {
		if result, err = sjson.SetBytes(result, "run_id", m.RunID.String()); err != nil {
			return nil, err
		}
	}
	if m.TurnID != uuid.Nil {
		if result, err = sjson.SetBytes(result, "turn_id", m.TurnID.String()); err != nil {
			return nil, err
		}
	}
	if m.Sender != "" {
		if result, err = sjson.SetBytes(result, "sender", m.Sender); err != nil {
			return nil, err
		}
	}
	if !m.Timestamp.IsZero() {
		if result, err = sjson.SetBytes(result, "timestamp", m.Timestamp.String()); err != nil {
			return nil, err
		}
	}
	if m.Meta.Exists() {
		if result, err = sjson.SetRawBytes(result, "meta", []byte(m.Meta.Raw)); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Message[T]
func (m *Message[T]) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	// Parse the JSON data
	parsed := gjson.ParseBytes(data)
	if !parsed.IsObject() {
		return fmt.Errorf("expected JSON object, got: %s", data)
	}

	// Check message type first
	msgType := parsed.Get("type")
	if !msgType.Exists() {
		return fmt.Errorf("missing required field 'type'")
	}

	// Create a new instance of the appropriate type based on the type field
	var payload T
	switch msgType.String() {
	case "instructions":
		var msg InstructionsMessage
		if err := msg.UnmarshalJSON(data); err != nil {
			return err
		}
		var ok bool
		payload, ok = any(msg).(T)
		if !ok {
			return fmt.Errorf("type mismatch: expected InstructionsMessage")
		}
	case "user":
		var msg UserMessage
		if err := msg.UnmarshalJSON(data); err != nil {
			return err
		}
		var ok bool
		payload, ok = any(msg).(T)
		if !ok {
			return fmt.Errorf("type mismatch: expected UserMessage")
		}
	case "assistant":
		var msg AssistantMessage
		if err := msg.UnmarshalJSON(data); err != nil {
			return err
		}
		var ok bool
		payload, ok = any(msg).(T)
		if !ok {
			return fmt.Errorf("type mismatch: expected AssistantMessage")
		}
	case "tool_call":
		var msg ToolCallMessage
		if err := msg.UnmarshalJSON(data); err != nil {
			return err
		}
		var ok bool
		payload, ok = any(msg).(T)
		if !ok {
			return fmt.Errorf("type mismatch: expected ToolCallMessage")
		}
	case "tool_response":
		var msg ToolResponse
		if err := msg.UnmarshalJSON(data); err != nil {
			return err
		}
		var ok bool
		payload, ok = any(msg).(T)
		if !ok {
			return fmt.Errorf("type mismatch: expected ToolResponse")
		}
	case "retry":
		var msg Retry
		if err := msg.UnmarshalJSON(data); err != nil {
			return err
		}
		var ok bool
		payload, ok = any(msg).(T)
		if !ok {
			return fmt.Errorf("type mismatch: expected Retry")
		}
	default:
		return fmt.Errorf("unknown message type: %s", msgType.String())
	}

	// Extract the basic fields
	if runID := parsed.Get("run_id"); runID.Exists() {
		if err := m.RunID.UnmarshalText([]byte(runID.String())); err != nil {
			return fmt.Errorf("invalid run_id: %w", err)
		}
	}

	if turnID := parsed.Get("turn_id"); turnID.Exists() {
		if err := m.TurnID.UnmarshalText([]byte(turnID.String())); err != nil {
			return fmt.Errorf("invalid turn_id: %w", err)
		}
	}

	if sender := parsed.Get("sender"); sender.Exists() {
		m.Sender = sender.String()
	}

	if timestamp := parsed.Get("timestamp"); timestamp.Exists() {
		if err := m.Timestamp.UnmarshalText([]byte(timestamp.String())); err != nil {
			return fmt.Errorf("invalid timestamp: %w", err)
		}
	}

	if meta := parsed.Get("meta"); meta.Exists() {
		m.Meta = meta
	}

	m.Payload = payload
	return nil
}

// InstructionsMessage represents system-level instructions.
// It contains content that provides guidance or directives to the system.
type InstructionsMessage struct {
	Content string   `json:"content"`
	_       struct{} // require keyed usage
}

// MarshalJSON implements custom JSON marshaling for InstructionsMessage
func (i InstructionsMessage) MarshalJSON() ([]byte, error) {
	return sjson.SetBytes(instructionsJSON, "content", i.Content)
}

// UnmarshalJSON implements custom JSON unmarshaling for InstructionsMessage
func (i *InstructionsMessage) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "instructions" {
		return fmt.Errorf("missing or invalid message type, expected 'instructions'")
	}

	content := gjson.GetBytes(data, "content")
	if !content.Exists() {
		return fmt.Errorf("missing required field 'content'")
	}

	i.Content = content.String()
	return nil
}

func (InstructionsMessage) message() {}

// UserMessage represents a message from a user.
// It can contain either simple text content or multiple content parts.
type UserMessage struct {
	Content ContentOrParts `json:"content"`
	_       struct{}       // require keyed usage
}

// MarshalJSON implements custom JSON marshaling for UserMessage
func (u UserMessage) MarshalJSON() ([]byte, error) {
	contentBytes, err := json.Marshal(u.Content)
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(userMessageJSON, "content", contentBytes)
}

// UnmarshalJSON implements custom JSON unmarshaling for UserMessage
func (u *UserMessage) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "user" {
		return fmt.Errorf("missing or invalid message type, expected 'user'")
	}

	content := gjson.GetBytes(data, "content")
	if !content.Exists() {
		return fmt.Errorf("missing required field 'content'")
	}

	return json.Unmarshal([]byte(content.Raw), &u.Content)
}

func (UserMessage) message() {}
func (UserMessage) request() {}

// AssistantMessage represents a response from the assistant.
// It can contain content, multiple content parts, or a refusal message.
type AssistantMessage struct {
	Content AssistantContentOrParts `json:"content,omitempty"`
	Refusal string                  `json:"refusal,omitempty"`
	_       struct{}                // require keyed usage
}

// MarshalJSON implements custom JSON marshaling for AssistantMessage
func (a AssistantMessage) MarshalJSON() ([]byte, error) {
	if a.Content.Content != "" && a.Refusal != "" {
		return nil, fmt.Errorf("both Content and Refusal cannot be set")
	}

	result := assistantJSON

	var err error
	if a.Content.Content != "" || len(a.Content.Parts) > 0 {
		var contentBytes []byte
		contentBytes, err = json.Marshal(a.Content)
		if err != nil {
			return nil, err
		}
		result, err = sjson.SetRawBytes(result, "content", contentBytes)
	} else if a.Refusal != "" {
		result, err = sjson.SetBytes(result, "refusal", a.Refusal)
	}

	return result, err
}

// UnmarshalJSON implements custom JSON unmarshaling for AssistantMessage
func (a *AssistantMessage) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "assistant" {
		return fmt.Errorf("missing or invalid message type, expected 'assistant'")
	}

	content := gjson.GetBytes(data, "content")
	refusal := gjson.GetBytes(data, "refusal")

	if content.Exists() && refusal.Exists() {
		return fmt.Errorf("both 'content' and 'refusal' cannot be present")
	}

	if content.Exists() {
		if err := json.Unmarshal([]byte(content.Raw), &a.Content); err != nil {
			return fmt.Errorf("invalid content: %w", err)
		}
	}

	if refusal.Exists() {
		a.Refusal = refusal.String()
	}

	return nil
}

func (AssistantMessage) message()  {}
func (AssistantMessage) response() {}

// ToolCallData contains the information needed to execute a tool.
// It includes the tool name and its arguments as a JSON string.
type ToolCallData struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Arguments string   `json:"arguments"`
	_         struct{} // require keyed usage
}

// ToolCallMessage represents a request to execute one or more tools.
type ToolCallMessage struct {
	ToolCalls []ToolCallData `json:"tool_calls"`
	_         struct{}       // require keyed usage
}

func (t ToolCallMessage) MarshalJSON() ([]byte, error) {
	result := toolCallJSON

	toolCalls, err := json.Marshal(t.ToolCalls)
	if err != nil {
		return nil, err
	}
	result, err = sjson.SetRawBytes(result, "tool_calls", toolCalls)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UnmarshalJSON implements custom JSON unmarshaling for ToolCallMessage
func (t *ToolCallMessage) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "tool_call" {
		return fmt.Errorf("missing or invalid message type, expected 'tool_call'")
	}

	toolCalls := gjson.GetBytes(data, "tool_calls")
	if !toolCalls.Exists() {
		return fmt.Errorf("missing required field 'tool_calls'")
	}

	if !toolCalls.IsArray() {
		return fmt.Errorf("'tool_calls' must be an array")
	}

	return json.Unmarshal([]byte(toolCalls.Raw), &t.ToolCalls)
}

func (ToolCallMessage) message()  {}
func (ToolCallMessage) response() {}

// ToolResponse represents the successful result of a tool execution.
// It includes the tool name, call ID, and the execution result.
type ToolResponse struct {
	ToolName   string   `json:"tool_name"`
	ToolCallID string   `json:"tool_call_id"`
	Content    string   `json:"content"`
	_          struct{} // require keyed usage
}

// MarshalJSON implements custom JSON marshaling for ToolResponse
func (t ToolResponse) MarshalJSON() ([]byte, error) {
	result := toolResponseJSON

	var err error
	result, err = sjson.SetBytes(result, "tool_name", t.ToolName)
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "tool_call_id", t.ToolCallID)
	if err != nil {
		return nil, err
	}

	result, err = sjson.SetBytes(result, "content", t.Content)
	return result, err
}

// UnmarshalJSON implements custom JSON unmarshaling for ToolResponse
func (t *ToolResponse) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "tool_response" {
		return fmt.Errorf("missing or invalid message type, expected 'tool_response'")
	}

	toolName := gjson.GetBytes(data, "tool_name")
	if !toolName.Exists() {
		return fmt.Errorf("missing required field 'tool_name'")
	}

	toolCallID := gjson.GetBytes(data, "tool_call_id")
	if !toolCallID.Exists() {
		return fmt.Errorf("missing required field 'tool_call_id'")
	}

	content := gjson.GetBytes(data, "content")
	if !content.Exists() {
		return fmt.Errorf("missing required field 'content'")
	}

	t.ToolName = toolName.String()
	t.ToolCallID = toolCallID.String()
	t.Content = content.String()
	return nil
}

func (ToolResponse) message() {}
func (ToolResponse) request() {}

// Retry represents a failed tool execution that may need to be retried.
// It includes error information and details about the failed tool call.
type Retry struct {
	Error      error    `json:"error"`
	ToolName   string   `json:"tool_name,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
	_          struct{} // require keyed usage
}

// MarshalJSON implements custom JSON marshaling for Retry
func (r Retry) MarshalJSON() ([]byte, error) {
	result := retryJSON

	var err error
	if r.Error != nil {
		result, err = sjson.SetBytes(result, "error", r.Error.Error())
		if err != nil {
			return nil, err
		}
	}

	if r.ToolName != "" {
		result, err = sjson.SetBytes(result, "tool_name", r.ToolName)
		if err != nil {
			return nil, err
		}
	}

	if r.ToolCallID != "" {
		result, err = sjson.SetBytes(result, "tool_call_id", r.ToolCallID)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Retry
func (r *Retry) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid json: %s", data)
	}

	msgType := gjson.GetBytes(data, "type")
	if !msgType.Exists() || msgType.String() != "retry" {
		return fmt.Errorf("missing or invalid message type, expected 'retry'")
	}

	errMsg := gjson.GetBytes(data, "error")
	if !errMsg.Exists() {
		return fmt.Errorf("missing required field 'error'")
	}

	r.Error = errors.New(errMsg.String())

	if toolName := gjson.GetBytes(data, "tool_name"); toolName.Exists() {
		r.ToolName = toolName.String()
	}

	if toolCallID := gjson.GetBytes(data, "tool_call_id"); toolCallID.Exists() {
		r.ToolCallID = toolCallID.String()
	}

	return nil
}

func (Retry) message() {}
func (Retry) request() {}
