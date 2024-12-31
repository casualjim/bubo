package executor

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/executor/pubsub"
	"github.com/casualjim/bubo/pkg/messages"
	"github.com/casualjim/bubo/pkg/runstate"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildArgList(t *testing.T) {
	tests := []struct {
		name       string
		arguments  string
		parameters map[string]string
		want       []string
	}{
		{
			name:      "empty arguments",
			arguments: "{}",
			parameters: map[string]string{
				"param0": "arg1",
			},
			want: []string{},
		},
		{
			name:      "single argument",
			arguments: `{"arg1": "value1"}`,
			parameters: map[string]string{
				"param0": "arg1",
			},
			want: []string{"value1"},
		},
		{
			name:      "multiple arguments",
			arguments: `{"arg1": "value1", "arg2": "value2"}`,
			parameters: map[string]string{
				"param0": "arg1",
				"param1": "arg2",
			},
			want: []string{"value1", "value2"},
		},
		{
			name:      "different types",
			arguments: `{"num": 42, "bool": true, "str": "text"}`,
			parameters: map[string]string{
				"param0": "num",
				"param1": "bool",
				"param2": "str",
			},
			want: []string{"42", "true", "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildArgList(tt.arguments, tt.parameters)

			// For empty arguments, expect empty slice
			if tt.name == "empty arguments" {
				assert.Empty(t, got, "expected empty slice for empty arguments")
				return
			}

			// Convert reflect.Values to strings for comparison
			gotStrings := make([]string, 0, len(got))
			for _, g := range got {
				if g.IsValid() {
					gotStrings = append(gotStrings, fmt.Sprintf("%v", g.Interface()))
				}
			}

			assert.Equal(t, tt.want, gotStrings, "test case: %s", tt.name)
		})
	}
}

func TestCallFunction(t *testing.T) {
	tests := []struct {
		name        string
		fn          interface{}
		args        []interface{}
		contextVars types.ContextVars
		wantValue   string
		wantErr     bool
	}{
		{
			name: "string return",
			fn: func() string {
				return "test"
			},
			wantValue: "test",
		},
		{
			name: "int return",
			fn: func() int {
				return 42
			},
			wantValue: "42",
		},
		{
			name: "error return",
			fn: func() error {
				return assert.AnError
			},
			wantErr: true,
		},
		{
			name: "with context vars",
			fn: func(cv types.ContextVars) string {
				return cv["key"].(string)
			},
			contextVars: types.ContextVars{"key": "value"},
			wantValue:   "value",
		},
		{
			name: "time return",
			fn: func() time.Time {
				return time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			},
			wantValue: "2023-01-01T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert args to reflect.Value slice
			var args []reflect.Value
			if tt.args != nil {
				args = make([]reflect.Value, len(tt.args))
				for i, arg := range tt.args {
					args[i] = reflect.ValueOf(arg)
				}
			}
			result, err := callFunction(tt.fn, args, tt.contextVars)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, result.Value)
		})
	}
}

type testModel struct {
	provider provider.Provider
}

func (m testModel) Provider() provider.Provider { return m.provider }
func (m testModel) String() string              { return "test_model" }
func (m testModel) Name() string                { return "test_model" }

type testAgent struct {
	bubo.Agent
	testName           string
	testModel          bubo.Model
	testInstructions   string
	testTools          []bubo.AgentToolDefinition
	testToolChoice     string
	testParallel       bool
	renderInstructions func(cv types.ContextVars) (string, error)
}

func (t *testAgent) Name() string                      { return t.testName }
func (t *testAgent) Model() bubo.Model                 { return t.testModel }
func (t *testAgent) Instructions() string              { return t.testInstructions }
func (t *testAgent) Tools() []bubo.AgentToolDefinition { return t.testTools }
func (t *testAgent) ToolChoice() string                { return t.testToolChoice }
func (t *testAgent) ParallelToolCalls() bool           { return t.testParallel }
func (t *testAgent) RenderInstructions(cv types.ContextVars) (string, error) {
	if t.renderInstructions != nil {
		return t.renderInstructions(cv)
	}
	return t.testInstructions, nil
}

func (t *testAgent) CallTool(ctx context.Context, msg *messages.ToolCallMessage) {
	// The executor handles publishing the tool response after calling the tool
	// We just need to verify the tool exists and can be called
	for _, tool := range t.testTools {
		if tool.Name == msg.ToolCalls[0].Name {
			tool.Function.(func() string)()
			return
		}
	}
}

func newTestAgent() *testAgent {
	return &testAgent{
		testName:         "test_agent",
		testModel:        testModel{provider: &mockProvider{}},
		testInstructions: "test instructions",
		testTools: []bubo.AgentToolDefinition{
			{
				Name:     "test_tool",
				Function: func() string { return "result" },
			},
		},
	}
}

func TestWrapErr(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	sender := "test_sender"

	t.Run("nil error", func(t *testing.T) {
		_, ok := wrapErr(runID, turnID, sender, nil)
		assert.False(t, ok)
	})

	t.Run("pubsub.Error", func(t *testing.T) {
		originalErr := pubsub.Error{
			RunID:  runID,
			TurnID: turnID,
			Sender: sender,
			Err:    assert.AnError,
		}
		wrappedErr, ok := wrapErr(runID, turnID, sender, originalErr)
		assert.True(t, ok)
		assert.Equal(t, originalErr, wrappedErr)
	})

	t.Run("regular error", func(t *testing.T) {
		err := fmt.Errorf("test error")
		wrappedErr, ok := wrapErr(runID, turnID, sender, err)
		assert.True(t, ok)
		assert.Equal(t, runID, wrappedErr.RunID)
		assert.Equal(t, turnID, wrappedErr.TurnID)
		assert.Equal(t, sender, wrappedErr.Sender)
		assert.Equal(t, err, wrappedErr.Err)
	})
}

func TestCallFunctionExtended(t *testing.T) {
	tests := []struct {
		name        string
		fn          interface{}
		args        []interface{}
		contextVars types.ContextVars
		wantValue   string
		wantErr     bool
	}{
		{
			name: "uint return",
			fn: func() uint {
				return 42
			},
			wantValue: "42",
		},
		{
			name: "float return",
			fn: func() float64 {
				return 3.14
			},
			wantValue: "3.14",
		},
		{
			name: "stringer return",
			fn: func() fmt.Stringer {
				return uuid.MustParse("00000000-0000-0000-0000-000000000000")
			},
			wantValue: "00000000-0000-0000-0000-000000000000",
		},
		{
			name: "struct return",
			fn: func() struct{ Name string } {
				return struct{ Name string }{Name: "test"}
			},
			wantValue: `{"Name":"test"}`,
		},
		{
			name: "agent return",
			fn: func() bubo.Agent {
				return newTestAgent()
			},
			wantValue: `{"assistant":"test_agent"}`,
		},
		{
			name: "no return",
			fn: func() {
			},
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []reflect.Value
			if tt.args != nil {
				args = make([]reflect.Value, len(tt.args))
				for i, arg := range tt.args {
					args[i] = reflect.ValueOf(arg)
				}
			}
			result, err := callFunction(tt.fn, args, tt.contextVars)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, result.Value)
		})
	}
}

func TestHandleToolCalls(t *testing.T) {
	t.Run("basic tool call", func(t *testing.T) {
		broker := newMockBroker[any]()
		l := NewLocal[any](broker)
		agent := newTestAgent()

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID:       runID,
			agent:       agent,
			contextVars: types.ContextVars{},
			mem:         runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "test_tool",
						Arguments: "{}",
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)
	})

	t.Run("agent transfer before regular tools", func(t *testing.T) {
		broker := newMockBroker[any]()
		l := NewLocal[any](broker)

		nextTestAgent := newTestAgent()
		nextTestAgent.testName = "next_agent"

		executionOrder := []string{}
		agent := &testAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []bubo.AgentToolDefinition{
				{
					Name: "regular_tool",
					Function: func() string {
						executionOrder = append(executionOrder, "regular_tool")
						return "regular result"
					},
				},
				{
					Name: "agent_tool",
					Function: func() bubo.Agent {
						executionOrder = append(executionOrder, "agent_tool")
						return nextTestAgent
					},
				},
			},
		}

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "regular_tool",
						Arguments: "{}",
					},
					{
						Name:      "agent_tool",
						Arguments: "{}",
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Equal(t, nextTestAgent, nextAgent)
		assert.Equal(t, []string{"agent_tool"}, executionOrder, "agent tool should execute first and prevent regular tool execution")
	})

	t.Run("context variable propagation", func(t *testing.T) {
		broker := newMockBroker[any]()
		l := NewLocal[any](broker)

		var toolContextVars types.ContextVars
		agent := &testAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []bubo.AgentToolDefinition{
				{
					Name: "first_tool",
					Function: func(cv types.ContextVars) types.ContextVars {
						if cv == nil {
							cv = types.ContextVars{}
						}
						cv["key"] = "value1"
						return cv
					},
					Parameters: map[string]string{
						"param0": "cv",
					},
				},
				{
					Name: "second_tool",
					Function: func(cv types.ContextVars) string {
						toolContextVars = cv
						if cv == nil {
							return "no value"
						}
						val, ok := cv["key"]
						if !ok {
							return "no value"
						}
						return val.(string)
					},
					Parameters: map[string]string{
						"param0": "cv",
					},
				},
			},
		}

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "first_tool",
						Arguments: `{"cv": {}}`,
					},
					{
						Name:      "second_tool",
						Arguments: `{"cv": {}}`,
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)
		assert.Equal(t, "value1", toolContextVars["key"], "context variables should propagate between tool calls")
	})

	t.Run("memory state preservation", func(t *testing.T) {
		broker := newMockBroker[any]()
		l := NewLocal[any](broker)

		agent := &testAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []bubo.AgentToolDefinition{
				{
					Name: "tool1",
					Function: func() string {
						return "result1"
					},
				},
				{
					Name: "tool2",
					Function: func() string {
						return "result2"
					},
				},
			},
		}

		runID := uuidx.New()
		mem := runstate.NewAggregator()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   mem,
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "tool1",
						Arguments: "{}",
						ID:        "1",
					},
					{
						Name:      "tool2",
						Arguments: "{}",
						ID:        "2",
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)

		// Wait for first tool response
		event1, err := broker.waitForEvent(runID.String(), time.Second, func(e pubsub.Event) bool {
			if resp, ok := e.(pubsub.Request[messages.ToolResponse]); ok {
				return resp.Message.ToolName == "tool1"
			}
			return false
		})
		require.NoError(t, err)
		require.NotNil(t, event1)
		resp1 := event1.(pubsub.Request[messages.ToolResponse])
		assert.Equal(t, "result1", resp1.Message.Content)

		// Wait for second tool response
		event2, err := broker.waitForEvent(runID.String(), time.Second, func(e pubsub.Event) bool {
			if resp, ok := e.(pubsub.Request[messages.ToolResponse]); ok {
				return resp.Message.ToolName == "tool2"
			}
			return false
		})
		require.NoError(t, err)
		require.NotNil(t, event2)
		resp2 := event2.(pubsub.Request[messages.ToolResponse])
		assert.Equal(t, "result2", resp2.Message.Content)
	})

	t.Run("multiple agent transfers", func(t *testing.T) {
		broker := newMockBroker[any]()
		l := NewLocal[any](broker)

		nextTestAgent1 := newTestAgent()
		nextTestAgent1.testName = "next_agent1"
		nextTestAgent2 := newTestAgent()
		nextTestAgent2.testName = "next_agent2"

		executionOrder := []string{}
		agent := &testAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []bubo.AgentToolDefinition{
				{
					Name: "agent_tool1",
					Function: func() bubo.Agent {
						executionOrder = append(executionOrder, "agent_tool1")
						return nextTestAgent1
					},
				},
				{
					Name: "agent_tool2",
					Function: func() bubo.Agent {
						executionOrder = append(executionOrder, "agent_tool2")
						return nextTestAgent2
					},
				},
			},
		}

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "agent_tool1",
						Arguments: "{}",
					},
					{
						Name:      "agent_tool2",
						Arguments: "{}",
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Equal(t, nextTestAgent1, nextAgent, "should return first successful agent transfer")
		assert.Equal(t, []string{"agent_tool1"}, executionOrder, "should only execute first agent transfer")
	})
}

type mockSubscription struct {
	pubsub.Subscription
}

func (m *mockSubscription) Unsubscribe() {}

type mockTopic[T any] struct {
	pubsub.Topic[T]
	mu         sync.RWMutex
	published  []pubsub.Event
	hook       pubsub.Hook[T]
	eventsChan chan pubsub.Event
	subscribe  func(ctx context.Context, hook pubsub.Hook[T]) (pubsub.Subscription, error)
}

func (m *mockTopic[T]) Publish(ctx context.Context, event pubsub.Event) error {
	m.mu.Lock()
	m.published = append(m.published, event)
	m.mu.Unlock()

	// Send to channel if it exists - no lock needed as channels are thread-safe
	if m.eventsChan != nil {
		m.eventsChan <- event
	}
	return nil
}

func (m *mockTopic[T]) Subscribe(ctx context.Context, hook pubsub.Hook[T]) (pubsub.Subscription, error) {
	if m.subscribe != nil {
		return m.subscribe(ctx, hook)
	}
	m.mu.Lock()
	m.hook = hook
	m.mu.Unlock()
	return &mockSubscription{}, nil
}

// waitForEvent waits for an event that matches the given predicate
func (m *mockTopic[T]) waitForEvent(timeout time.Duration, predicate func(pubsub.Event) bool) (pubsub.Event, error) {
	// Initialize channel if needed
	m.mu.Lock()
	if m.eventsChan == nil {
		m.eventsChan = make(chan pubsub.Event, 100)
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
			return nil, fmt.Errorf("timeout waiting for event")
		}
	}
}

type mockBroker[T any] struct {
	pubsub.Broker[T]
	mu     sync.RWMutex
	topics map[string]*mockTopic[T]
}

func newMockBroker[T any]() *mockBroker[T] {
	return &mockBroker[T]{
		topics: make(map[string]*mockTopic[T]),
	}
}

func (m *mockBroker[T]) Topic(_ context.Context, id string) pubsub.Topic[T] {
	m.mu.RLock()
	t, ok := m.topics[id]
	m.mu.RUnlock()

	if ok {
		return t
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check in case another goroutine created the topic
	if t, ok := m.topics[id]; ok {
		return t
	}

	t = &mockTopic[T]{
		eventsChan: make(chan pubsub.Event, 100),
	}
	m.topics[id] = t
	return t
}

// Helper function to wait for a specific event type and optionally validate its content
func (m *mockBroker[T]) waitForEvent(id string, timeout time.Duration, predicate func(pubsub.Event) bool) (pubsub.Event, error) {
	topic, ok := m.topics[id]
	if !ok {
		return nil, fmt.Errorf("topic %s not found", id)
	}
	return topic.waitForEvent(timeout, predicate)
}

type mockProvider struct {
	provider.Provider
	responses []provider.StreamEvent
	err       error
}

func (m *mockProvider) ChatCompletion(ctx context.Context, params provider.CompletionParams) (<-chan provider.StreamEvent, error) {
	if m.err != nil {
		return nil, m.err
	}

	ch := make(chan provider.StreamEvent, len(m.responses))
	for _, resp := range m.responses {
		ch <- resp
	}
	close(ch)
	return ch, nil
}

func TestRun(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		mockResp := provider.Response[messages.AssistantMessage]{
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: `test result`, // Remove the quotes since we're expecting a string type
				},
			},
		}

		prov := &mockProvider{
			responses: []provider.StreamEvent{mockResp},
		}
		agent.testModel = testModel{provider: prov}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Wait for final response
		event, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			_, ok := e.(pubsub.Result[string])
			return ok
		})
		require.NoError(t, err)

		resp := event.(pubsub.Result[string])
		assert.Equal(t, "test result", resp.Result)
	})

	t.Run("provider error", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		prov := &mockProvider{
			err: fmt.Errorf("provider error"),
		}
		agent.testModel = testModel{provider: prov}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Wait for error event
		event, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			_, ok := e.(pubsub.Error)
			return ok
		})
		require.NoError(t, err)

		errEvent := event.(pubsub.Error)
		assert.Contains(t, errEvent.Err.Error(), "provider error")
	})

	t.Run("render instructions error", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := &testAgent{
			testName:         "test_agent",
			testModel:        testModel{provider: &mockProvider{}},
			testInstructions: "test instructions",
			testTools: []bubo.AgentToolDefinition{
				{
					Name:     "test_tool",
					Function: func() string { return "result" },
				},
			},
			renderInstructions: func(cv types.ContextVars) (string, error) {
				return "", fmt.Errorf("failed to render instructions")
			},
		}
		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Wait for error event
		event, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			_, ok := e.(pubsub.Error)
			return ok
		})
		require.NoError(t, err)

		errEvent := event.(pubsub.Error)
		assert.Contains(t, errEvent.Err.Error(), "failed to render instructions")
	})

	t.Run("tool call handling", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		toolCallResp := provider.Response[messages.ToolCallMessage]{
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "test_tool",
						Arguments: "{}",
					},
				},
			},
		}

		assistantResp := provider.Response[messages.AssistantMessage]{
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: `test result`,
				},
			},
		}

		prov := &mockProvider{
			responses: []provider.StreamEvent{toolCallResp, assistantResp},
		}
		agent.testModel = testModel{provider: prov}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Wait for tool call event
		toolCallEvent, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			_, ok := e.(pubsub.Response[messages.ToolCallMessage])
			return ok
		})
		require.NoError(t, err)
		require.NotNil(t, toolCallEvent)

		// Wait for tool response
		toolResponseEvent, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			if resp, ok := e.(pubsub.Request[messages.ToolResponse]); ok {
				return resp.Message.Content == "result"
			}
			return false
		})
		require.NoError(t, err)
		require.NotNil(t, toolResponseEvent)

		// Wait for final response
		finalEvent, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			if resp, ok := e.(pubsub.Result[string]); ok {
				return resp.Result == "test result"
			}
			return false
		})
		require.NoError(t, err)
		require.NotNil(t, finalEvent)
	})

	t.Run("context cancellation", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)

		err = local.Run(ctx, cmd)
		require.NoError(t, err)

		// Verify no events were published due to cancellation
		topic := broker.topics[cmd.ID.String()]
		require.NotNil(t, topic)
		assert.Empty(t, topic.published)
	})

	t.Run("parallel tool calls with error", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		agent.testParallel = true
		agent.testTools = []bubo.AgentToolDefinition{
			{
				Name: "error_tool",
				Function: func() error {
					return fmt.Errorf("tool error")
				},
			},
			{
				Name: "success_tool",
				Function: func() string {
					return "success"
				},
			},
		}

		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		toolCallResp := provider.Response[messages.ToolCallMessage]{
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{Name: "error_tool", Arguments: "{}"},
					{Name: "success_tool", Arguments: "{}"},
				},
			},
		}

		prov := &mockProvider{
			responses: []provider.StreamEvent{toolCallResp},
		}
		agent.testModel = testModel{provider: prov}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Wait for error event
		event, err := broker.waitForEvent(cmd.ID.String(), time.Second, func(e pubsub.Event) bool {
			if errEvent, ok := e.(pubsub.Error); ok {
				return strings.Contains(errEvent.Err.Error(), "tool error")
			}
			return false
		})
		require.NoError(t, err)
		require.NotNil(t, event)
	})

	t.Run("nil subscription error", func(t *testing.T) {
		broker := newMockBroker[string]()
		topic := &mockTopic[string]{
			eventsChan: make(chan pubsub.Event, 100),
			subscribe: func(ctx context.Context, hook pubsub.Hook[string]) (pubsub.Subscription, error) {
				return nil, nil
			},
		}

		cmd, err := NewRunCommand[string](newTestAgent(), runstate.NewAggregator(), &mockHook[string]{})
		require.NoError(t, err)

		broker.topics = map[string]*mockTopic[string]{
			cmd.ID.String(): topic,
		}
		local := NewLocal[string](broker)

		err = local.Run(context.Background(), cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "received nil subscription")
	})

	t.Run("multiple tool calls with context vars", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		agent.testTools = []bubo.AgentToolDefinition{
			{
				Name: "set_var",
				Function: func(cv types.ContextVars) types.ContextVars {
					if cv == nil {
						cv = types.ContextVars{}
					}
					cv["key"] = "value"
					return cv
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
			{
				Name: "use_var",
				Function: func(cv types.ContextVars) string {
					if cv == nil {
						return "no value"
					}
					if val, ok := cv["key"]; ok {
						return val.(string)
					}
					return "no value"
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
		}

		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		toolCallResp := provider.Response[messages.ToolCallMessage]{
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{Name: "set_var", Arguments: `{"cv": {}}`},
					{Name: "use_var", Arguments: `{"cv": {}}`},
				},
			},
		}

		prov := &mockProvider{
			responses: []provider.StreamEvent{toolCallResp},
		}
		agent.testModel = testModel{provider: prov}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)
		cmd.ContextVariables = types.ContextVars{}

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Wait for tool responses
		var foundValue bool
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) && !foundValue {
			event, err := broker.waitForEvent(cmd.ID.String(), 100*time.Millisecond, func(e pubsub.Event) bool {
				if resp, ok := e.(pubsub.Request[messages.ToolResponse]); ok {
					if resp.Message.Content == "value" {
						foundValue = true
						return true
					}
				}
				return false
			})
			if err == nil && event != nil {
				continue
			}
		}
		assert.True(t, foundValue, "use_var tool should have completed with correct value")
	})

	t.Run("max turns limit", func(t *testing.T) {
		broker := newMockBroker[string]()
		local := NewLocal[string](broker)

		agent := newTestAgent()
		thread := runstate.NewAggregator()
		hook := &mockHook[string]{}

		cmd, err := NewRunCommand[string](agent, thread, hook)
		require.NoError(t, err)
		cmd = cmd.WithMaxTurns(0) // Set max turns to 0 to trigger immediate return

		err = local.Run(context.Background(), cmd)
		require.NoError(t, err)

		// Verify no events were published due to max turns limit
		topic := broker.topics[cmd.ID.String()]
		require.NotNil(t, topic)
		assert.Empty(t, topic.published)
	})
}

func TestNewLocal(t *testing.T) {
	t.Run("nil broker", func(t *testing.T) {
		assert.Panics(t, func() {
			NewLocal[any](nil)
		})
	})

	t.Run("valid broker", func(t *testing.T) {
		broker := newMockBroker[any]()
		local := NewLocal[any](broker)
		assert.NotNil(t, local)
	})
}

func TestHandleToolCallsErrors(t *testing.T) {
	broker := newMockBroker[any]()
	l := NewLocal[any](broker)

	runID := uuidx.New()
	params := toolCallParams[any]{
		runID: runID,
		agent: newTestAgent(),
		mem:   runstate.NewAggregator(),
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "unknown_tool",
					Arguments: "{}",
				},
			},
		},
		topic: broker.Topic(context.Background(), runID.String()),
	}

	_, err := l.handleToolCalls(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestHandleToolCallsWithContextVars(t *testing.T) {
	broker := newMockBroker[any]()
	l := NewLocal[any](broker)

	contextVars := types.ContextVars{"test": "value"}
	agent := newTestAgent()
	agent.testTools = []bubo.AgentToolDefinition{
		{
			Name: "context_tool",
			Function: func(cv types.ContextVars) string {
				return cv["test"].(string)
			},
		},
	}

	runID := uuidx.New()
	params := toolCallParams[any]{
		runID:       runID,
		agent:       agent,
		contextVars: contextVars,
		mem:         runstate.NewAggregator(),
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "context_tool",
					Arguments: "{}",
				},
			},
		},
		topic: broker.Topic(context.Background(), runID.String()),
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Nil(t, nextAgent)

	// Wait for tool response
	event, err := broker.waitForEvent(runID.String(), time.Second, func(e pubsub.Event) bool {
		if resp, ok := e.(pubsub.Request[messages.ToolResponse]); ok {
			return resp.Message.Content == "value"
		}
		return false
	})
	require.NoError(t, err)
	require.NotNil(t, event)
}

func TestHandleToolCallsWithAgentReturn(t *testing.T) {
	broker := newMockBroker[any]()
	l := NewLocal[any](broker)

	nextTestAgent := newTestAgent()
	nextTestAgent.testName = "next_agent"

	agent := newTestAgent()
	agent.testTools = []bubo.AgentToolDefinition{
		{
			Name: "agent_tool",
			Function: func() bubo.Agent {
				return nextTestAgent
			},
		},
	}

	runID := uuidx.New()
	params := toolCallParams[any]{
		runID: runID,
		agent: agent,
		mem:   runstate.NewAggregator(),
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "agent_tool",
					Arguments: "{}",
				},
			},
		},
		topic: broker.Topic(context.Background(), runID.String()),
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, nextTestAgent, nextAgent)
}

func TestHandleToolCallsWithInvalidJSON(t *testing.T) {
	broker := newMockBroker[any]()
	l := NewLocal[any](broker)

	agent := newTestAgent()
	runID := uuidx.New()
	params := toolCallParams[any]{
		runID: runID,
		agent: agent,
		mem:   runstate.NewAggregator(),
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "test_tool",
					Arguments: "invalid json",
				},
			},
		},
		topic: broker.Topic(context.Background(), runID.String()),
	}

	_, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err) // Should not error as buildArgList handles invalid JSON gracefully
}

type textMarshaler struct {
	shouldError bool
}

func (t textMarshaler) MarshalText() ([]byte, error) {
	if t.shouldError {
		return nil, fmt.Errorf("marshal error")
	}
	return []byte("marshaled text"), nil
}

func TestHandleToolCallsWithMixedTools(t *testing.T) {
	broker := newMockBroker[any]()
	l := NewLocal[any](broker)

	var executionOrder []string
	var contextValue string

	agent := &testAgent{
		testName:  "test_agent",
		testModel: testModel{provider: &mockProvider{}},
		testTools: []bubo.AgentToolDefinition{
			{
				Name: "b_agent_tool", // Deliberately named to test order preservation
				Function: func() bubo.Agent {
					executionOrder = append(executionOrder, "b_agent_tool")
					return newTestAgent()
				},
			},
			{
				Name: "a_agent_tool", // Deliberately named to test order preservation
				Function: func() bubo.Agent {
					executionOrder = append(executionOrder, "a_agent_tool")
					return newTestAgent()
				},
			},
			{
				Name: "b_regular_tool", // Deliberately named to test order preservation
				Function: func() types.ContextVars {
					executionOrder = append(executionOrder, "b_regular_tool")
					cv := types.ContextVars{}
					cv["key"] = "test_value"
					return cv
				},
			},
			{
				Name: "a_regular_tool", // Deliberately named to test order preservation
				Function: func(cv types.ContextVars) string {
					executionOrder = append(executionOrder, "a_regular_tool")
					contextValue = cv["key"].(string)
					return cv["key"].(string)
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
		},
	}

	t.Run("preserves order within agent transfers partition", func(t *testing.T) {
		executionOrder = []string{}
		contextValue = ""

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "b_agent_tool",
						Arguments: "{}",
					},
					{
						Name:      "a_agent_tool",
						Arguments: "{}",
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.NotNil(t, nextAgent)
		assert.Equal(t, []string{"b_agent_tool"}, executionOrder,
			"should execute first agent tool in received order regardless of name")
	})

	t.Run("preserves order within regular tools partition", func(t *testing.T) {
		executionOrder = []string{}
		contextValue = ""

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "b_regular_tool",
						Arguments: "{}",
					},
					{
						Name:      "a_regular_tool",
						Arguments: `{"cv": {}}`,
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)
		assert.Equal(t, []string{"b_regular_tool", "a_regular_tool"}, executionOrder,
			"should execute regular tools in received order regardless of name")
		assert.Equal(t, "test_value", contextValue)
	})

	t.Run("agent transfers partition executes before regular tools partition", func(t *testing.T) {
		executionOrder = []string{}
		contextValue = ""

		runID := uuidx.New()
		params := toolCallParams[any]{
			runID: runID,
			agent: agent,
			mem:   runstate.NewAggregator(),
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "b_regular_tool",
						Arguments: "{}",
					},
					{
						Name:      "b_agent_tool",
						Arguments: "{}",
					},
					{
						Name:      "a_regular_tool",
						Arguments: `{"cv": {}}`,
					},
					{
						Name:      "a_agent_tool",
						Arguments: "{}",
					},
				},
			},
			topic: broker.Topic(context.Background(), runID.String()),
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.NotNil(t, nextAgent)
		assert.Equal(t, []string{"b_agent_tool"}, executionOrder,
			"should execute first agent tool in received order and stop")
		assert.Empty(t, contextValue)
	})
}

func TestHandleToolCallsContextPropagation(t *testing.T) {
	broker := newMockBroker[any]()
	l := NewLocal(broker)

	var toolValues []string
	agent := &testAgent{
		testName:  "test_agent",
		testModel: testModel{provider: &mockProvider{}},
		testTools: []bubo.AgentToolDefinition{
			{
				Name: "tool1",
				Function: func() types.ContextVars {
					cv := types.ContextVars{}
					cv["key1"] = "value1"
					cv["key2"] = "value2"
					return cv
				},
			},
			{
				Name: "tool2",
				Function: func(cv types.ContextVars) string {
					toolValues = append(toolValues, cv["key1"].(string))
					cv["key1"] = "updated"
					return "ok"
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
			{
				Name: "tool3",
				Function: func(cv types.ContextVars) string {
					toolValues = append(toolValues, cv["key1"].(string))
					toolValues = append(toolValues, cv["key2"].(string))
					return "ok"
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
		},
	}

	runID := uuidx.New()
	params := toolCallParams[any]{
		runID: runID,
		agent: agent,
		mem:   runstate.NewAggregator(),
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "tool1",
					Arguments: "{}",
				},
				{
					Name:      "tool2",
					Arguments: `{"cv": {}}`,
				},
				{
					Name:      "tool3",
					Arguments: `{"cv": {}}`,
				},
			},
		},
		topic: broker.Topic(context.Background(), runID.String()),
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Nil(t, nextAgent)
	assert.Equal(t, []string{"value1", "updated", "value2"}, toolValues, "context variables should propagate and update correctly")
}

func TestCallFunctionWithComplexTypes(t *testing.T) {
	tests := []struct {
		name      string
		fn        interface{}
		args      []interface{}
		wantValue string
		wantErr   bool
	}{
		{
			name: "text marshaler success",
			fn: func() encoding.TextMarshaler {
				return textMarshaler{shouldError: false}
			},
			wantValue: "marshaled text",
		},
		{
			name: "text marshaler error",
			fn: func() encoding.TextMarshaler {
				return textMarshaler{shouldError: true}
			},
			wantErr: true,
		},
		{
			name: "complex struct",
			fn: func() struct {
				Name   string
				Age    int
				Nested struct{ Value bool }
			} {
				return struct {
					Name   string
					Age    int
					Nested struct{ Value bool }
				}{
					Name:   "test",
					Age:    30,
					Nested: struct{ Value bool }{Value: true},
				}
			},
			wantValue: `{"Name":"test","Age":30,"Nested":{"Value":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []reflect.Value
			if tt.args != nil {
				args = make([]reflect.Value, len(tt.args))
				for i, arg := range tt.args {
					args[i] = reflect.ValueOf(arg)
				}
			}
			result, err := callFunction(tt.fn, args, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, result.Value)
		})
	}
}
