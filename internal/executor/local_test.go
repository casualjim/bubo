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
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
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
			runID: runID,
			agent: agent,
			mem:   shorttermmemory.New(),
			hook:  &mockHook{},
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
		assert.Equal(t, []string{"agent_tool"}, executionOrder, "agent tool should execute first and prevent regular tool execution")
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
			runID: runID,
			agent: agent,
			mem:   shorttermmemory.New(),
			hook:  &mockHook{},
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
	responseReceived := make(chan string, 1)
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
			runID: runID,
			agent: agent,
			mem:   shorttermmemory.New(),
			hook:  &mockHook{},
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
			runID: runID,
			agent: agent,
			mem:   shorttermmemory.New(),
			hook:  &mockHook{},
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
			runID: runID,
			agent: agent,
			mem:   shorttermmemory.New(),
			hook:  &mockHook{},
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
