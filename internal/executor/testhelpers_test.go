package executor

import (
	"context"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
)

// Mock Provider

type mockProvider struct {
	provider.Provider
	responses          []provider.StreamEvent
	err                error
	lastParams         provider.CompletionParams // Track the last params received
	streamCh           chan provider.StreamEvent // Channel for controlling response timing in tests
	chatCompletionHook func()                    // Hook to signal when ChatCompletion is called
}

func (m *mockProvider) ChatCompletion(ctx context.Context, params provider.CompletionParams) (<-chan provider.StreamEvent, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.lastParams = params // Store the params for verification

	if m.chatCompletionHook != nil {
		m.chatCompletionHook()
	}

	if m.streamCh != nil {
		return m.streamCh, nil
	}

	// Default behavior when streamCh is not set
	ch := make(chan provider.StreamEvent, len(m.responses))
	for _, resp := range m.responses {
		ch <- resp
	}
	close(ch)
	return ch, nil
}

// Mock Agent

type mockAgent struct {
	api.Owl
	testName  string
	testModel testModel
	testTools []tool.Definition
}

func (m *mockAgent) Name() string {
	if m.testName == "" {
		return "mock_agent"
	}
	return m.testName
}

func (m *mockAgent) Model() api.Model {
	return m.testModel
}

func (m *mockAgent) Instructions() string {
	return "mock instructions"
}

func (m *mockAgent) Tools() []tool.Definition {
	return m.testTools
}

func (m *mockAgent) ToolChoice() string {
	return ""
}

func (m *mockAgent) ParallelToolCalls() bool {
	return false
}

func (m *mockAgent) RenderInstructions(cv types.ContextVars) (string, error) {
	return m.Instructions(), nil
}

// Mock Hook

type mockHook struct {
	events.Hook
	onAssistantMessage func(ctx context.Context, msg messages.Message[messages.AssistantMessage])
	onToolCallResponse func(ctx context.Context, msg messages.Message[messages.ToolResponse])
}

func (h *mockHook) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {}

func (h *mockHook) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
}

func (h *mockHook) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
}

func (h *mockHook) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	if h.onAssistantMessage != nil {
		h.onAssistantMessage(ctx, msg)
	}
}

func (h *mockHook) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
}

func (h *mockHook) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	if h.onToolCallResponse != nil {
		h.onToolCallResponse(ctx, msg)
	}
}

func (h *mockHook) OnError(ctx context.Context, err error) {}

// Test Model

type testModel struct {
	provider provider.Provider
}

func (m testModel) Provider() provider.Provider { return m.provider }
func (m testModel) String() string              { return "test_model" }
func (m testModel) Name() string                { return "test_model" }

// Helper Functions

func newTestAgent() *mockAgent {
	responseCh := make(chan provider.StreamEvent, 1)
	prov := &mockProvider{
		streamCh: responseCh,
		chatCompletionHook: func() {
			// Default hook does nothing
		},
		responses: []provider.StreamEvent{
			provider.Response[messages.AssistantMessage]{
				Response: messages.AssistantMessage{
					Content: messages.AssistantContentOrParts{
						Content: "test result",
					},
				},
			},
		},
	}
	return &mockAgent{
		testName:  "test_agent",
		testModel: testModel{provider: prov},
		testTools: []tool.Definition{
			{
				Name:     "test_tool",
				Function: func() string { return "result" },
			},
		},
	}
}
