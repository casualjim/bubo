package executor

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type textMarshaler struct {
	shouldError bool
}

func (t textMarshaler) MarshalText() ([]byte, error) {
	if t.shouldError {
		return nil, fmt.Errorf("marshal error")
	}
	return []byte("marshaled text"), nil
}

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
		l := NewLocal()
		agent := newTestAgent()

		runID := uuidx.New()
		params := toolCallParams{
			runID:       runID,
			agent:       agent,
			contextVars: types.ContextVars{},
			mem:         shorttermmemory.New(),
			hook:        &mockHook{},
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						Name:      "test_tool",
						Arguments: "{}",
					},
				},
			},
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)
	})

	t.Run("agent transfer before regular tools", func(t *testing.T) {
		l := NewLocal()

		nextTestAgent := newTestAgent()
		nextTestAgent.testName = "next_agent"

		executionOrder := []string{}
		agent := &mockAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []tool.Definition{
				{
					Name: "regular_tool",
					Function: func() string {
						executionOrder = append(executionOrder, "regular_tool")
						return "regular result"
					},
				},
				{
					Name: "agent_tool",
					Function: func() api.Owl {
						executionOrder = append(executionOrder, "agent_tool")
						return nextTestAgent
					},
				},
			},
		}

		runID := uuidx.New()
		params := toolCallParams{
			runID:       runID,
			agent:       agent,
			mem:         shorttermmemory.New(),
			hook:        &mockHook{},
			contextVars: make(types.ContextVars),
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
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Equal(t, nextTestAgent, nextAgent)
		assert.Equal(t, []string{"agent_tool"}, executionOrder, "tools should execute in order and return agent from agent tool")
	})

	t.Run("context variable propagation", func(t *testing.T) {
		l := NewLocal()

		var toolContextVars types.ContextVars
		agent := &mockAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []tool.Definition{
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
		params := toolCallParams{
			runID:       runID,
			agent:       agent,
			mem:         shorttermmemory.New(),
			hook:        &mockHook{},
			contextVars: make(types.ContextVars),
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
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)
		assert.Equal(t, "value1", toolContextVars["key"], "context variables should propagate between tool calls")
	})
}

func TestHandleToolCallsErrors(t *testing.T) {
	l := NewLocal()

	runID := uuidx.New()
	params := toolCallParams{
		runID: runID,
		agent: newTestAgent(),
		mem:   shorttermmemory.New(),
		hook:  &mockHook{},
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "unknown_tool",
					Arguments: "{}",
				},
			},
		},
	}

	_, err := l.handleToolCalls(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestHandleToolCallsWithContextVars(t *testing.T) {
	l := NewLocal()

	contextVars := types.ContextVars{"test": "value"}
	agent := newTestAgent()
	agent.testTools = []tool.Definition{
		{
			Name: "context_tool",
			Function: func(cv types.ContextVars) string {
				return cv["test"].(string)
			},
		},
	}

	// Create a channel to capture the tool response
	responseReceived := make(chan string, 10)
	hook := &mockHook{
		onToolCallResponse: func(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
			responseReceived <- msg.Payload.Content
		},
	}

	runID := uuidx.New()
	params := toolCallParams{
		runID:       runID,
		agent:       agent,
		contextVars: contextVars,
		mem:         shorttermmemory.New(),
		hook:        hook,
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "context_tool",
					Arguments: "{}",
				},
			},
		},
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Nil(t, nextAgent)

	// Wait for the response with timeout
	select {
	case response := <-responseReceived:
		assert.Equal(t, "value", response)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for tool response")
	}
}

func TestHandleToolCallsWithAgentReturn(t *testing.T) {
	l := NewLocal()

	nextTestAgent := newTestAgent()
	nextTestAgent.testName = "next_agent"

	agent := newTestAgent()
	agent.testTools = []tool.Definition{
		{
			Name: "agent_tool",
			Function: func() api.Owl {
				return nextTestAgent
			},
		},
	}

	runID := uuidx.New()
	params := toolCallParams{
		runID: runID,
		agent: agent,
		mem:   shorttermmemory.New(),
		hook:  &mockHook{},
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "agent_tool",
					Arguments: "{}",
				},
			},
		},
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, nextTestAgent, nextAgent)
}

func TestHandleToolCallsWithInvalidJSON(t *testing.T) {
	l := NewLocal()

	agent := newTestAgent()
	runID := uuidx.New()
	params := toolCallParams{
		runID: runID,
		agent: agent,
		mem:   shorttermmemory.New(),
		hook:  &mockHook{},
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					Name:      "test_tool",
					Arguments: "invalid json",
				},
			},
		},
	}

	_, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err) // Should not error as buildArgList handles invalid JSON gracefully
}

func TestHandleToolCallsSessionContent(t *testing.T) {
	l := NewLocal()

	// Create an agent with tools that manipulate context and return values
	agent := &mockAgent{
		testName:  "test_agent",
		testModel: testModel{provider: &mockProvider{}},
		testTools: []tool.Definition{
			{
				Name: "tool1",
				Function: func() types.ContextVars {
					cv := types.ContextVars{}
					cv["key1"] = "value1"
					return cv
				},
			},
			{
				Name: "tool2",
				Function: func(cv types.ContextVars) string {
					return fmt.Sprintf("got value: %v", cv["key1"])
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
		},
	}

	runID := uuidx.New()
	mem := shorttermmemory.New()
	initialLen := mem.Len()

	params := toolCallParams{
		runID:       runID,
		agent:       agent,
		mem:         mem,
		hook:        &mockHook{},
		contextVars: make(types.ContextVars),
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					ID:        "call1",
					Name:      "tool1",
					Arguments: "{}",
				},
				{
					ID:        "call2",
					Name:      "tool2",
					Arguments: `{"cv": {}}`,
				},
			},
		},
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Nil(t, nextAgent)

	// Verify session content
	msgs := mem.Messages()
	assert.Equal(t, initialLen+2, len(msgs), "Should have added 2 tool response messages")

	// Verify first tool response
	msg1 := msgs[initialLen]
	toolResp1, ok := msg1.Payload.(messages.ToolResponse)
	require.True(t, ok)
	assert.Equal(t, "tool1", toolResp1.ToolName)
	assert.Equal(t, "call1", toolResp1.ToolCallID)
	assert.Equal(t, "", toolResp1.Content) // Context vars don't produce content
	assert.Equal(t, runID, msg1.RunID)
	assert.Equal(t, mem.ID(), msg1.TurnID)
	assert.Equal(t, "test_agent", msg1.Sender)

	// Verify second tool response
	msg2 := msgs[initialLen+1]
	toolResp2, ok := msg2.Payload.(messages.ToolResponse)
	require.True(t, ok)
	assert.Equal(t, "tool2", toolResp2.ToolName)
	assert.Equal(t, "call2", toolResp2.ToolCallID)
	assert.Equal(t, "got value: value1", toolResp2.Content)
	assert.Equal(t, runID, msg2.RunID)
	assert.Equal(t, mem.ID(), msg2.TurnID)
	assert.Equal(t, "test_agent", msg2.Sender)
}

func TestHandleToolCallsWithMixedTools(t *testing.T) {
	l := NewLocal()

	var executionOrder []string
	var contextValue string

	agent := &mockAgent{
		testName:  "test_agent",
		testModel: testModel{provider: &mockProvider{}},
		testTools: []tool.Definition{
			{
				Name: "b_agent_tool", // Deliberately named to test order preservation
				Function: func() api.Owl {
					executionOrder = append(executionOrder, "b_agent_tool")
					return newTestAgent()
				},
			},
			{
				Name: "a_agent_tool", // Deliberately named to test order preservation
				Function: func() api.Owl {
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
		params := toolCallParams{
			runID:       runID,
			agent:       agent,
			mem:         shorttermmemory.New(),
			hook:        &mockHook{},
			contextVars: make(types.ContextVars),
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
		params := toolCallParams{
			runID:       runID,
			agent:       agent,
			mem:         shorttermmemory.New(),
			hook:        &mockHook{},
			contextVars: make(types.ContextVars),
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
		params := toolCallParams{
			runID:       runID,
			agent:       agent,
			mem:         shorttermmemory.New(),
			hook:        &mockHook{},
			contextVars: make(types.ContextVars),
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
	l := NewLocal()

	var toolValues []string
	agent := &mockAgent{
		testName:  "test_agent",
		testModel: testModel{provider: &mockProvider{}},
		testTools: []tool.Definition{
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
	params := toolCallParams{
		runID: runID,
		agent: agent,
		mem:   shorttermmemory.New(),
		hook:  &mockHook{},
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
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Nil(t, nextAgent)
	assert.Equal(t, []string{"value1", "updated", "value2"}, toolValues, "context variables should propagate and update correctly")
}

func TestHandleToolCallsSessionFork(t *testing.T) {
	l := NewLocal()

	// Create an agent with a tool that modifies the session
	agent := &mockAgent{
		testName:  "test_agent",
		testModel: testModel{provider: &mockProvider{}},
		testTools: []tool.Definition{
			{
				Name: "modify_session",
				Function: func() string {
					return "test response"
				},
			},
		},
	}

	// Create a session with initial content
	mem := shorttermmemory.New()
	mem.AddUserPrompt(messages.Message[messages.UserMessage]{
		RunID:     uuidx.New(),
		TurnID:    mem.ID(),
		Sender:    "user",
		Timestamp: strfmt.DateTime(time.Now()),
		Payload:   messages.UserMessage{Content: messages.ContentOrParts{Content: "initial message"}},
	})
	initialLen := mem.Len()

	runID := uuidx.New()
	params := toolCallParams{
		runID: runID,
		agent: agent,
		mem:   mem,
		hook:  &mockHook{},
		toolCalls: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					ID:        "call1",
					Name:      "modify_session",
					Arguments: "{}",
				},
			},
		},
	}

	nextAgent, err := l.handleToolCalls(context.Background(), params)
	require.NoError(t, err)
	assert.Nil(t, nextAgent)

	// Verify original session
	msgs := mem.Messages()
	assert.Equal(t, initialLen+1, len(msgs), "Should have added 1 tool response message")

	// Verify the initial message is preserved
	msg1 := msgs[0]
	userMsg, ok := msg1.Payload.(messages.UserMessage)
	require.True(t, ok)
	assert.Equal(t, "initial message", userMsg.Content.Content)

	// Verify the tool response was added
	msg2 := msgs[1]
	toolResp, ok := msg2.Payload.(messages.ToolResponse)
	require.True(t, ok)
	assert.Equal(t, "modify_session", toolResp.ToolName)
	assert.Equal(t, "call1", toolResp.ToolCallID)
	assert.Equal(t, "test response", toolResp.Content)
}

func TestHandleToolCallsSessionEdgeCases(t *testing.T) {
	l := NewLocal()

	t.Run("empty tool response", func(t *testing.T) {
		agent := &mockAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []tool.Definition{
				{
					Name: "empty_tool",
					Function: func() string {
						return ""
					},
				},
			},
		}

		mem := shorttermmemory.New()
		initialLen := mem.Len()

		runID := uuidx.New()
		params := toolCallParams{
			runID: runID,
			agent: agent,
			mem:   mem,
			hook:  &mockHook{},
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "call1",
						Name:      "empty_tool",
						Arguments: "{}",
					},
				},
			},
		}

		nextAgent, err := l.handleToolCalls(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, nextAgent)

		msgs := mem.Messages()
		assert.Equal(t, initialLen+1, len(msgs), "Should have added 1 tool response message")

		toolResp, ok := msgs[0].Payload.(messages.ToolResponse)
		require.True(t, ok)
		assert.Equal(t, "", toolResp.Content, "Empty response should be preserved")
	})

	t.Run("concurrent tool calls", func(t *testing.T) {
		agent := &mockAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []tool.Definition{
				{
					Name: "concurrent_tool",
					Function: func() types.ContextVars {
						cv := types.ContextVars{}
						cv["key"] = "value"
						return cv
					},
				},
			},
		}

		mem := shorttermmemory.New()
		initialLen := mem.Len()

		// Create multiple concurrent tool calls
		const numCalls = 5
		var params []toolCallParams
		var forkedSessions []*shorttermmemory.Aggregator

		for i := 0; i < numCalls; i++ {
			forked := mem.Fork()
			forkedSessions = append(forkedSessions, forked)
			params = append(params, toolCallParams{
				runID: uuidx.New(),
				agent: agent,
				mem:   forked,
				hook:  &mockHook{},
				toolCalls: messages.ToolCallMessage{
					ToolCalls: []messages.ToolCallData{
						{
							ID:        fmt.Sprintf("call%d", i+1),
							Name:      "concurrent_tool",
							Arguments: "{}",
						},
					},
				},
			})
		}

		// Run tool calls concurrently
		errCh := make(chan error, numCalls)
		for i := 0; i < numCalls; i++ {
			go func(p toolCallParams) {
				nextAgent, err := l.handleToolCalls(context.Background(), p)
				if err != nil {
					errCh <- err
					return
				}
				if nextAgent != nil {
					errCh <- fmt.Errorf("unexpected next agent")
					return
				}
				errCh <- nil
			}(params[i])
		}

		// Wait for all calls to complete
		for i := 0; i < numCalls; i++ {
			err := <-errCh
			require.NoError(t, err)
		}

		// Join all forked sessions back to the main session
		for _, forked := range forkedSessions {
			mem.Join(forked)
		}

		// Verify final session state
		msgs := mem.Messages()
		assert.Equal(t, initialLen+numCalls, len(msgs), "Should have added messages for all tool calls")

		// Create a map of tool call IDs to verify all expected calls are present
		toolCallIDs := make(map[string]bool)
		for _, msg := range msgs {
			if toolResp, ok := msg.Payload.(messages.ToolResponse); ok {
				toolCallIDs[toolResp.ToolCallID] = true
			}
		}

		// Verify all expected tool calls are present
		for i := 1; i <= numCalls; i++ {
			callID := fmt.Sprintf("call%d", i)
			assert.True(t, toolCallIDs[callID], "Missing tool call response for %s", callID)
		}
	})

	t.Run("fork error handling", func(t *testing.T) {
		agent := &mockAgent{
			testName:  "test_agent",
			testModel: testModel{provider: &mockProvider{}},
			testTools: []tool.Definition{
				{
					Name: "error_tool",
					Function: func() error {
						return fmt.Errorf("test error")
					},
				},
			},
		}

		mem := shorttermmemory.New()
		initialLen := mem.Len()

		// Add some initial content
		mem.AddUserPrompt(messages.Message[messages.UserMessage]{
			RunID:     uuidx.New(),
			TurnID:    mem.ID(),
			Sender:    "user",
			Timestamp: strfmt.DateTime(time.Now()),
			Payload:   messages.UserMessage{Content: messages.ContentOrParts{Content: "initial message"}},
		})

		runID := uuidx.New()
		params := toolCallParams{
			runID: runID,
			agent: agent,
			mem:   mem,
			hook:  &mockHook{},
			toolCalls: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "call1",
						Name:      "error_tool",
						Arguments: "{}",
					},
				},
			},
		}

		_, err := l.handleToolCalls(context.Background(), params)
		require.Error(t, err)

		// Verify original session was not modified
		msgs := mem.Messages()
		assert.Equal(t, initialLen+1, len(msgs), "Original session should be unchanged")
		userMsg, ok := msgs[0].Payload.(messages.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "initial message", userMsg.Content.Content)
	})
}

func TestRunWithStreamingAgentHandoff(t *testing.T) {
	l := NewLocal()

	nextAgent := &mockAgent{
		testName: "next_agent",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Response[messages.AssistantMessage]{
					Response: messages.AssistantMessage{
						Content: messages.AssistantContentOrParts{
							Content: "response from next agent",
						},
					},
				},
			},
		}},
	}

	agent := &mockAgent{
		testName: "test_agent",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Delim{Delim: "start"},
				provider.Chunk[messages.ToolCallMessage]{
					Chunk: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{
							{
								ID:        "tool1",
								Name:      "agent_tool",
								Arguments: "{}",
							},
						},
					},
				},
				provider.Delim{Delim: "end"},
				provider.Response[messages.ToolCallMessage]{
					Response: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{
							{
								ID:        "tool1",
								Name:      "agent_tool",
								Arguments: "{}",
							},
						},
					},
				},
			},
		}},
		testTools: []tool.Definition{
			{
				Name: "agent_tool",
				Function: func() api.Owl {
					return nextAgent
				},
			},
		},
	}

	thread := shorttermmemory.New()

	var toolCallChunks []messages.ToolCallMessage
	var toolCallResponses []messages.ToolCallMessage
	var assistantMessages []messages.AssistantMessage
	hook := &mockHook{
		onToolCallChunk: func(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
			toolCallChunks = append(toolCallChunks, msg.Payload)
		},
		onToolCallMessage: func(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
			toolCallResponses = append(toolCallResponses, msg.Payload)
		},
		onAssistantMessage: func(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
			assistantMessages = append(assistantMessages, msg.Payload)
		},
	}

	cmd, err := NewRunCommand(agent, thread, hook)
	require.NoError(t, err)
	cmd = cmd.WithStream(true)

	fut := NewFuture(DefaultUnmarshal[string]())
	err = l.Run(context.Background(), cmd, fut)
	require.NoError(t, err)

	// Verify streaming responses
	assert.Len(t, toolCallChunks, 1, "Should receive one tool call chunk")
	assert.Len(t, toolCallResponses, 1, "Should receive tool call response for agent transfer")
	assert.Len(t, assistantMessages, 1, "Should receive final response from next agent")
	assert.Equal(t, "response from next agent", assistantMessages[0].Content.Content)

	result, err := fut.Get()
	require.NoError(t, err)
	assert.Equal(t, "response from next agent", result)
}

func TestRunWithStreamingToolCalls(t *testing.T) {
	l := NewLocal()

	agent := &mockAgent{
		testName: "test_agent",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Delim{Delim: "start"},
				provider.Chunk[messages.ToolCallMessage]{
					Chunk: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{
							{
								ID:        "tool1",
								Name:      "test_tool",
								Arguments: `{"arg": "value"}`,
							},
						},
					},
				},
				provider.Delim{Delim: "end"},
				provider.Response[messages.ToolCallMessage]{
					Response: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{
							{
								ID:        "tool1",
								Name:      "test_tool",
								Arguments: `{"arg": "value"}`,
							},
						},
					},
				},
				provider.Response[messages.AssistantMessage]{
					Response: messages.AssistantMessage{
						Content: messages.AssistantContentOrParts{
							Content: "tool result",
						},
					},
				},
			},
		}},
		testTools: []tool.Definition{
			{
				Name: "test_tool",
				Function: func() string {
					return "tool result"
				},
			},
		},
	}

	thread := shorttermmemory.New()

	var toolCallChunks []messages.ToolCallMessage
	var toolCallResponses []messages.ToolCallMessage
	hook := &mockHook{
		onToolCallChunk: func(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
			toolCallChunks = append(toolCallChunks, msg.Payload)
		},
		onToolCallMessage: func(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
			toolCallResponses = append(toolCallResponses, msg.Payload)
		},
	}

	cmd, err := NewRunCommand(agent, thread, hook)
	require.NoError(t, err)
	cmd = cmd.WithStream(true)

	fut := NewFuture(DefaultUnmarshal[string]())
	err = l.Run(context.Background(), cmd, fut)
	require.NoError(t, err)

	// Verify streaming responses
	assert.Len(t, toolCallChunks, 1, "Should receive one tool call chunk")
	assert.Len(t, toolCallResponses, 1, "Should receive tool call response")

	result, err := fut.Get()
	require.NoError(t, err)
	assert.Equal(t, "tool result", result)
}

func TestRunWithStreaming(t *testing.T) {
	l := NewLocal()

	agent := &mockAgent{
		testName: "test_agent",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Delim{Delim: "start"},
				provider.Chunk[messages.AssistantMessage]{
					Chunk: messages.AssistantMessage{
						Content: messages.AssistantContentOrParts{
							Content: "streaming chunk",
						},
					},
				},
				provider.Delim{Delim: "end"},
				provider.Response[messages.AssistantMessage]{
					Response: messages.AssistantMessage{
						Content: messages.AssistantContentOrParts{
							Content: "streaming chunk", // Same content as chunk
						},
					},
				},
			},
		}},
	}

	thread := shorttermmemory.New()

	var streamingResponses []string
	hook := &mockHook{
		onAssistantChunk: func(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
			streamingResponses = append(streamingResponses, msg.Payload.Content.Content)
		},
		onAssistantMessage: func(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
			streamingResponses = append(streamingResponses, msg.Payload.Content.Content)
		},
	}

	cmd, err := NewRunCommand(agent, thread, hook)
	require.NoError(t, err)
	cmd = cmd.WithStream(true)

	fut := NewFuture(DefaultUnmarshal[string]())
	err = l.Run(context.Background(), cmd, fut)
	require.NoError(t, err)

	// Verify streaming responses
	assert.Equal(t, []string{"streaming chunk", "streaming chunk"}, streamingResponses,
		"Should receive both streaming chunk and final response")

	result, err := fut.Get()
	require.NoError(t, err)
	assert.Equal(t, "streaming chunk", result)
}

func TestRunWithAgentChain(t *testing.T) {
	l := NewLocal()

	// Create agents in reverse order since each needs to know about the next
	agent3 := &mockAgent{
		testName: "agent3",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Response[messages.ToolCallMessage]{
					Response: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{{
							ID:        "final",
							Name:      "final_tool",
							Arguments: `{"cv": {}}`,
						}},
					},
				},
				provider.Response[messages.AssistantMessage]{
					Response: messages.AssistantMessage{
						Content: messages.AssistantContentOrParts{
							Content: "Final response after all tools",
						},
					},
				},
			},
		}},
		testTools: []tool.Definition{{
			Name: "final_tool",
			Function: func(cv types.ContextVars) string {
				return fmt.Sprintf("final result with context: %v, %v, %v",
					cv["key1"], cv["key2"], cv["key3"])
			},
			Parameters: map[string]string{
				"param0": "cv",
			},
		}},
	}

	agent2 := &mockAgent{
		testName: "agent2",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Response[messages.ToolCallMessage]{
					Response: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{
							{
								ID:        "intermediate",
								Name:      "intermediate_tool",
								Arguments: `{"cv": {}}`,
							},
							{
								ID:        "transfer2",
								Name:      "transfer_to_agent3",
								Arguments: "{}",
							},
						},
					},
				},
			},
		}},
		testTools: []tool.Definition{
			{
				Name: "intermediate_tool",
				Function: func(cv types.ContextVars) types.ContextVars {
					cv["key3"] = "value3"
					return cv
				},
				Parameters: map[string]string{
					"param0": "cv",
				},
			},
			{
				Name: "transfer_to_agent3",
				Function: func() api.Owl {
					return agent3
				},
			},
		},
	}

	agent1 := &mockAgent{
		testName: "agent1",
		testModel: testModel{provider: &mockProvider{
			responses: []provider.StreamEvent{
				provider.Response[messages.ToolCallMessage]{
					Response: messages.ToolCallMessage{
						ToolCalls: []messages.ToolCallData{
							{
								ID:        "setup",
								Name:      "setup_tool",
								Arguments: "{}",
							},
							{
								ID:        "transfer1",
								Name:      "transfer_to_agent2",
								Arguments: "{}",
							},
						},
					},
				},
			},
		}},
		testTools: []tool.Definition{
			{
				Name: "setup_tool",
				Function: func() types.ContextVars {
					return types.ContextVars{
						"key1": "value1",
						"key2": "value2",
					}
				},
			},
			{
				Name: "transfer_to_agent2",
				Function: func() api.Owl {
					return agent2
				},
			},
		},
	}

	thread := shorttermmemory.New()

	var toolResponses []messages.Message[messages.ToolResponse]
	var toolCallMessages []messages.Message[messages.ToolCallMessage]
	var assistantMessages []messages.Message[messages.AssistantMessage]
	hook := &mockHook{
		onToolCallMessage: func(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
			toolCallMessages = append(toolCallMessages, msg)
		},
		onToolCallResponse: func(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
			toolResponses = append(toolResponses, msg)
		},
		onAssistantMessage: func(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
			assistantMessages = append(assistantMessages, msg)
		},
	}

	cmd, err := NewRunCommand(agent1, thread, hook)
	require.NoError(t, err)

	fut := NewFuture(DefaultUnmarshal[string]())
	err = l.Run(context.Background(), cmd, fut)
	require.NoError(t, err)

	msgs := thread.Messages()
	assert.Equal(t, 1, len(msgs), "Should have final message in chain")

	assert.Len(t, toolResponses, 1, "Should have final tool response")
	assert.Len(t, assistantMessages, 1, "Should have final assistant message")

	// Only verify senders if we have enough messages
	if assert.Len(t, toolCallMessages, 3) {
		var senders []string
		for _, msg := range toolCallMessages {
			senders = append(senders, msg.Sender)
		}
		expectedSenders := []string{
			"agent1", // transfer_to_agent2
			"agent2", // transfer_to_agent3
			"agent3", // final_tool
		}
		assert.Equal(t, expectedSenders, senders[:len(expectedSenders)])
	}
	if assert.Len(t, toolResponses, 1) {
		var senders []string
		for _, msg := range toolResponses {
			senders = append(senders, msg.Sender)
		}
		expectedSenders := []string{
			"agent3", // final assistant message
		}
		assert.Equal(t, expectedSenders, senders[:len(expectedSenders)])
	}
	if assert.Len(t, assistantMessages, 1) {
		var senders []string
		for _, msg := range assistantMessages {
			senders = append(senders, msg.Sender)
		}
		expectedSenders := []string{
			"agent3", // final assistant message
		}
		assert.Equal(t, expectedSenders, senders[:len(expectedSenders)])
	}

	result, err := fut.Get()
	require.NoError(t, err)
	assert.Equal(t, "Final response after all tools", result)
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
