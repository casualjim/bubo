package executor

import (
	"context"
	"sync"
	"time"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/broker"
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
}

func (h *mockHook) OnError(ctx context.Context, err error) {}

// Mock Subscription

type mockSubscription struct {
	broker.Subscription
}

func (m *mockSubscription) Unsubscribe() {}

// Mock Topic

type mockTopic struct {
	broker.Topic
	mu         sync.RWMutex
	published  []events.Event
	hook       events.Hook
	eventsChan chan events.Event
	subscribe  func(ctx context.Context, hook events.Hook) (broker.Subscription, error)
}

func (m *mockTopic) Publish(ctx context.Context, event events.Event) error {
	m.mu.Lock()
	m.published = append(m.published, event)
	m.mu.Unlock()

	// Send to channel if it exists - no lock needed as channels are thread-safe
	if m.eventsChan != nil {
		m.eventsChan <- event
	}
	return nil
}

func (m *mockTopic) Subscribe(ctx context.Context, hook events.Hook) (broker.Subscription, error) {
	if m.subscribe != nil {
		return m.subscribe(ctx, hook)
	}
	m.mu.Lock()
	m.hook = hook
	m.mu.Unlock()
	return &mockSubscription{}, nil
}

// waitForEvent waits for an event that matches the given predicate
func (m *mockTopic) waitForEvent(timeout time.Duration, predicate func(events.Event) bool) (events.Event, error) {
	// Initialize channel if needed
	m.mu.Lock()
	if m.eventsChan == nil {
		m.eventsChan = make(chan events.Event, 100)
	}
	m.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// First check already published events under read lock
	m.mu.RLock()
	for _, event := range m.published {
		if predicate(event) {
			m.mu.RUnlock()
			return event, nil
		}
	}
	m.mu.RUnlock()

	// Then wait for new events
	for {
		select {
		case event := <-m.eventsChan:
			if predicate(event) {
				return event, nil
			}
		case <-timer.C:
			return nil, nil
		}
	}
}

// Mock Broker

type mockBroker struct {
	broker.Broker
	mu     sync.RWMutex
	topics map[string]*mockTopic
}

func newMockBroker() *mockBroker {
	return &mockBroker{
		topics: make(map[string]*mockTopic),
	}
}

func (m *mockBroker) Topic(_ context.Context, id string) broker.Topic {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.topics[id]; ok {
		return t
	}

	t := &mockTopic{
		eventsChan: make(chan events.Event, 100),
	}
	m.topics[id] = t
	return t
}

// Helper function to wait for a specific event type and optionally validate its content
func (m *mockBroker) waitForEvent(id string, timeout time.Duration, predicate func(events.Event) bool) (events.Event, error) {
	m.mu.RLock()
	topic, ok := m.topics[id]
	m.mu.RUnlock()

	if !ok {
		// If topic doesn't exist yet, wait for it to be created
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.mu.RLock()
				topic, ok = m.topics[id]
				m.mu.RUnlock()
				if ok {
					return topic.waitForEvent(timeout, predicate)
				}
			case <-timer.C:
				return nil, nil
			}
		}
	}

	return topic.waitForEvent(timeout, predicate)
}

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
