package messages

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
	Name    string `json:"name,omitempty"`
}

func (Instructions) message() {}

type UserPrompt struct {
	Content ContentOrParts `json:"content"`
	Name    string         `json:"name,omitempty"`
}

func (UserPrompt) message() {}
func (UserPrompt) request() {}

type AssistantMessage struct {
	Content      string `json:"content"`
	Refusal      string `json:"refusal,omitempty"`
	Name         string `json:"name,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
}

func (AssistantMessage) message()  {}
func (AssistantMessage) response() {}

type ToolCallData struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string         `json:"id"`
	Function []ToolCallData `json:"function"`
}

func (ToolCall) message()  {}
func (ToolCall) response() {}

type ToolResponse struct{}

func (ToolResponse) message() {}
func (ToolResponse) request() {}

type Retry struct {
	Error error
	Count int
}

func (Retry) message() {}
func (Retry) request() {}
