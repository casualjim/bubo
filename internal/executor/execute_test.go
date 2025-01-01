package executor

import (
	"math"
	"testing"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type testResponse struct {
	Message string `json:"message"`
}

type mockAgent struct {
	bubo.Owl
}

func TestNewRunCommand(t *testing.T) {
	t.Run("creates command with valid inputs", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.NewAggregator()
		hook := &mockHook[testResponse]{}

		cmd, err := NewRunCommand[testResponse](agent, thread, hook)
		require.NoError(t, err)
		assert.NotNil(t, cmd.ID)
		assert.Equal(t, agent, cmd.Agent)
		assert.Equal(t, thread, cmd.Thread)
		assert.Equal(t, hook, cmd.Hook)
		assert.NotNil(t, cmd.ResponseSchema)
		assert.NotNil(t, cmd.UnmarshalResponse)
	})

	t.Run("creates command with gjson.Result type", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.NewAggregator()
		hook := &mockHook[gjson.Result]{}

		cmd, err := NewRunCommand[gjson.Result](agent, thread, hook)
		require.NoError(t, err)
		assert.NotNil(t, cmd.ID)
		assert.Equal(t, agent, cmd.Agent)
		assert.Equal(t, thread, cmd.Thread)
		assert.Equal(t, hook, cmd.Hook)
		assert.Nil(t, cmd.ResponseSchema) // Schema should be nil for gjson.Result
		assert.NotNil(t, cmd.UnmarshalResponse)

		// Test gjson unmarshaler
		result, err := cmd.UnmarshalResponse([]byte(`{"test": "value"}`))
		require.NoError(t, err)
		assert.Equal(t, "value", result.Get("test").String())
	})

	t.Run("fails with nil agent", func(t *testing.T) {
		thread := shorttermmemory.NewAggregator()
		hook := &mockHook[testResponse]{}

		_, err := NewRunCommand[testResponse](nil, thread, hook)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("fails with nil thread", func(t *testing.T) {
		agent := &mockAgent{}
		hook := &mockHook[testResponse]{}

		_, err := NewRunCommand[testResponse](agent, nil, hook)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "thread is required")
	})

	t.Run("fails with nil hook", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.NewAggregator()

		_, err := NewRunCommand[testResponse](agent, thread, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook is required")
	})

	t.Run("unmarshaler works with regular struct", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.NewAggregator()
		hook := &mockHook[testResponse]{}

		cmd, err := NewRunCommand[testResponse](agent, thread, hook)
		require.NoError(t, err)

		result, err := cmd.UnmarshalResponse([]byte(`{"message": "test"}`))
		require.NoError(t, err)
		assert.Equal(t, "test", result.Message)
	})

	t.Run("unmarshaler fails with invalid json for regular struct", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.NewAggregator()
		hook := &mockHook[testResponse]{}

		cmd, err := NewRunCommand[testResponse](agent, thread, hook)
		require.NoError(t, err)

		_, err = cmd.UnmarshalResponse([]byte(`invalid json`))
		require.Error(t, err)
	})
}

func TestRunCommandMethods(t *testing.T) {
	agent := &mockAgent{}
	thread := shorttermmemory.NewAggregator()
	hook := &mockHook[testResponse]{}

	cmd, err := NewRunCommand[testResponse](agent, thread, hook)
	require.NoError(t, err)

	t.Run("WithStream", func(t *testing.T) {
		modified := cmd.WithStream(true)
		assert.True(t, modified.Stream)
		assert.False(t, cmd.Stream) // Original should be unchanged

		modified = modified.WithStream(false)
		assert.False(t, modified.Stream)
	})

	t.Run("WithMaxTurns", func(t *testing.T) {
		modified := cmd.WithMaxTurns(5)
		assert.Equal(t, 5, modified.MaxTurns)
		assert.Equal(t, math.MaxInt, cmd.MaxTurns) // Original should be unchanged

		modified = modified.WithMaxTurns(10)
		assert.Equal(t, 10, modified.MaxTurns)
	})

	t.Run("WithContextVariables", func(t *testing.T) {
		vars := types.ContextVars{"key": "value"}
		modified := cmd.WithContextVariables(vars)
		assert.Equal(t, vars, modified.ContextVariables)
		assert.Nil(t, cmd.ContextVariables) // Original should be unchanged

		newVars := types.ContextVars{"new": "value"}
		modified = modified.WithContextVariables(newVars)
		assert.Equal(t, newVars, modified.ContextVariables)
	})
}
