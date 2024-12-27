// Package messages provides a messaging system for handling various types of communication
// between different parts of the application. It includes support for user messages,
// assistant messages, tool calls, and responses.
package messages

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/tidwall/gjson"
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
	sender    string          // The entity that sent the message
	timestamp strfmt.DateTime // When the message was created
}

// New creates a new messageBuilder instance with the current timestamp.
// It initializes a builder that can be used to create various types of messages.
func New() messageBuilder {
	return messageBuilder{
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

// Instructions creates a new instruction message with the given content.
// This type of message is typically used to provide system-level instructions.
func (b messageBuilder) Instructions(content string) Message[InstructionsMessage] {
	return Message[InstructionsMessage]{
		Payload:   InstructionsMessage{Content: content},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// UserPrompt creates a new user message with single-part content.
// This is used when the user sends a simple text message.
func (b messageBuilder) UserPrompt(content string) Message[UserMessage] {
	return Message[UserMessage]{
		Payload:   UserMessage{Content: ContentOrParts{Content: content}},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// UserPromptMultipart creates a new user message with multiple content parts.
// This is used when the user message contains multiple components (e.g., text and images).
func (b messageBuilder) UserPromptMultipart(parts ...ContentPart) Message[UserMessage] {
	return Message[UserMessage]{
		Payload:   UserMessage{Content: ContentOrParts{Parts: parts}},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// AssistantMessage creates a new assistant message with single-part content.
// This is used when the assistant responds with a simple text message.
func (b messageBuilder) AssistantMessage(content string) Message[AssistantMessage] {
	return Message[AssistantMessage]{
		Payload: AssistantMessage{
			Content: AssistantContentOrParts{Content: content},
		},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// AssistantRefusal creates a new assistant message indicating a refusal to perform an action.
// This is used when the assistant needs to decline a request with an explanation.
func (b messageBuilder) AssistantRefusal(refusal string) Message[AssistantMessage] {
	return Message[AssistantMessage]{
		Payload: AssistantMessage{
			Refusal: refusal,
		},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// AssistantMessageMultipart creates a new assistant message with multiple content parts.
// This is used when the assistant needs to respond with structured or multi-component content.
func (b messageBuilder) AssistantMessageMultipart(parts ...AssistantContentPart) Message[AssistantMessage] {
	return Message[AssistantMessage]{
		Payload: AssistantMessage{
			Content: AssistantContentOrParts{Parts: parts},
		},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// CallTool creates a new ToolCallData instance with the given name and arguments.
// This is a helper function for creating tool call requests with JSON arguments.
func CallTool(name string, args gjson.Result) ToolCallData {
	return ToolCallData{
		Name:      name,
		Arguments: args.Raw,
	}
}

// ToolCall creates a new tool call message with the specified ID and tool call data.
// This is used when the assistant needs to invoke one or more external tools.
func (b messageBuilder) ToolCall(id string, invokeTool ...ToolCallData) Message[ToolCallMessage] {
	return Message[ToolCallMessage]{
		Payload: ToolCallMessage{
			ID:       id,
			Function: invokeTool,
		},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// ToolResponse creates a new tool response message with the result of a tool execution.
// This is used to return the results of a successful tool call.
func (b messageBuilder) ToolResponse(id, name, result string) Message[ToolResponse] {
	return Message[ToolResponse]{
		Payload: ToolResponse{
			ToolCallID: id,
			ToolName:   name,
			Content:    result,
		},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// ToolError creates a new retry message when a tool execution fails.
// This is used to handle and communicate tool execution errors.
func (b messageBuilder) ToolError(id, name string, error error) Message[Retry] {
	return Message[Retry]{
		Payload: Retry{
			ToolCallID: id,
			ToolName:   name,
			Error:      error,
		},
		Sender:    b.sender,
		Timestamp: b.timestamp,
	}
}

// Message is a generic container for all message types in the system.
// It includes common metadata like sender and timestamp alongside the specific message payload.
type Message[T ModelMessage] struct {
	Payload   T               `json:",inline"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp,omitempty"`
}

// InstructionsMessage represents system-level instructions.
// It contains content that provides guidance or directives to the system.
type InstructionsMessage struct {
	Content string   `json:"content"`
	_       struct{} // require keyed usage
}

func (InstructionsMessage) message() {}

// UserMessage represents a message from a user.
// It can contain either simple text content or multiple content parts.
type UserMessage struct {
	Content ContentOrParts `json:"content"`
	_       struct{}       // require keyed usage
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

func (AssistantMessage) message()  {}
func (AssistantMessage) response() {}

// ToolCallData contains the information needed to execute a tool.
// It includes the tool name and its arguments as a JSON string.
type ToolCallData struct {
	Name      string   `json:"name"`
	Arguments string   `json:"arguments"`
	_         struct{} // require keyed usage
}

// ToolCallMessage represents a request to execute one or more tools.
// It includes an ID for tracking and the function calls to be made.
type ToolCallMessage struct {
	ID        string          `json:"id"`
	Function  []ToolCallData  `json:"function"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp"`
	_         struct{}        // require keyed usage
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

func (Retry) message() {}
func (Retry) request() {}
