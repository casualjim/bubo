package messages

import "github.com/go-openapi/strfmt"

type Message interface {
	message()
}

type Request interface {
	request()
}

type Response interface {
	response()
}

type Instructions struct {
	Content string `json:"content"`
	Sender  string `json:"sender,omitempty"`
}

func (Instructions) message() {}

type UserPrompt struct {
	Content   ContentOrParts  `json:"content"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp"`
}

func (UserPrompt) message() {}
func (UserPrompt) request() {}

type AssistantMessage struct {
	Content   string          `json:"content"`
	Refusal   string          `json:"refusal,omitempty"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp"`
}

func (AssistantMessage) message()  {}
func (AssistantMessage) response() {}

type ToolCallData struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Function  []ToolCallData  `json:"function"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp strfmt.DateTime `json:"timestamp"`
}

func (ToolCall) message()  {}
func (ToolCall) response() {}

type ToolResponse struct {
	ToolName   string          `json:"tool_name"`
	ToolCallID string          `json:"tool_call_id"`
	Content    string          `json:"content"`
	Timestamp  strfmt.DateTime `json:"timestamp"`
}

func (ToolResponse) message() {}
func (ToolResponse) request() {}

type Retry struct {
	Error      error           `json:"error"`
	ToolName   string          `json:"tool_name,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Timestamp  strfmt.DateTime `json:"timestamp"`
}

func (Retry) message() {}
func (Retry) request() {}
