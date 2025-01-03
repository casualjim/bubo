package executor

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type testResponse struct {
	Message string `json:"message"`
}

func TestNewRunCommand(t *testing.T) {
	t.Run("creates command with valid inputs", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.New()
		hook := &mockHook{}

		cmd, err := NewRunCommand(agent, thread, hook)
		require.NoError(t, err)
		assert.NotNil(t, cmd.ID())
		assert.Equal(t, agent, cmd.Agent)
		assert.Equal(t, thread, cmd.Thread)
		assert.Equal(t, hook, cmd.Hook)
	})

	t.Run("creates command with gjson.Result type", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set up test components
		responseCh := make(chan provider.StreamEvent, 1)
		prov := &mockProvider{
			streamCh: responseCh,
		}

		agent := &mockAgent{
			testModel: testModel{provider: prov},
		}
		thread := shorttermmemory.New()
		hook := &mockHook{}

		cmd, err := NewRunCommand(agent, thread, hook)
		require.NoError(t, err)
		assert.NotNil(t, cmd.ID())
		assert.Equal(t, agent, cmd.Agent)
		assert.Equal(t, thread, cmd.Thread)
		assert.Equal(t, hook, cmd.Hook)

		local := NewLocal()
		promise := NewFuture[gjson.Result](DefaultUnmarshal[gjson.Result]())

		// Start the execution in a goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- local.Run(ctx, cmd, promise)
		}()

		// Send a valid JSON response
		responseCh <- provider.Response[messages.AssistantMessage]{
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: `{"result": "test"}`,
				},
			},
			Checkpoint: shorttermmemory.New().Checkpoint(),
		}
		close(responseCh)

		// Check for Run errors
		select {
		case err := <-errCh:
			require.NoError(t, err, "unexpected error from Run")
		case <-ctx.Done():
			t.Fatal("timeout waiting for Run completion")
		}

		// Wait for the promise result
		result, err := promise.Get()
		require.NoError(t, err)
		assert.True(t, result.Get("result").Exists())
		assert.Equal(t, "test", result.Get("result").String())
	})

	t.Run("fails with nil agent", func(t *testing.T) {
		thread := shorttermmemory.New()
		hook := &mockHook{}

		_, err := NewRunCommand(nil, thread, hook)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("fails with nil thread", func(t *testing.T) {
		agent := &mockAgent{}
		hook := &mockHook{}

		_, err := NewRunCommand(agent, nil, hook)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "thread is required")
	})

	t.Run("fails with nil hook", func(t *testing.T) {
		agent := &mockAgent{}
		thread := shorttermmemory.New()

		_, err := NewRunCommand(agent, thread, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook is required")
	})

	t.Run("unmarshaler works with regular struct", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set up test components
		responseCh := make(chan provider.StreamEvent, 1)
		prov := &mockProvider{
			streamCh: responseCh,
		}

		agent := &mockAgent{
			testModel: testModel{provider: prov},
		}
		thread := shorttermmemory.New()
		hook := &mockHook{}

		cmd, err := NewRunCommand(agent, thread, hook)
		require.NoError(t, err)

		local := NewLocal()
		promise := NewFuture[testResponse](DefaultUnmarshal[testResponse]())

		// Start the execution in a goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- local.Run(ctx, cmd, promise)
		}()

		// Send a valid JSON response
		responseCh <- provider.Response[messages.AssistantMessage]{
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: `{"message": "test"}`,
				},
			},
			Checkpoint: shorttermmemory.New().Checkpoint(),
		}
		close(responseCh)

		// Check for Run errors
		select {
		case err := <-errCh:
			require.NoError(t, err, "unexpected error from Run")
		case <-ctx.Done():
			t.Fatal("timeout waiting for Run completion")
		}

		// Wait for the promise result
		result, err := promise.Get()
		require.NoError(t, err)
		assert.Equal(t, testResponse{Message: "test"}, result)
	})

	t.Run("unmarshaler fails with invalid json for regular struct", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set up test components
		responseCh := make(chan provider.StreamEvent, 1)
		prov := &mockProvider{
			streamCh: responseCh,
		}

		agent := &mockAgent{
			testModel: testModel{provider: prov},
		}
		thread := shorttermmemory.New()
		hook := &mockHook{}

		cmd, err := NewRunCommand(agent, thread, hook)
		require.NoError(t, err)

		local := NewLocal()
		promise := NewFuture[testResponse](DefaultUnmarshal[testResponse]())

		// Start the execution in a goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- local.Run(ctx, cmd, promise)
		}()

		// Send an invalid JSON response
		responseCh <- provider.Response[messages.AssistantMessage]{
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: `{"invalid": json}`,
				},
			},
			Checkpoint: shorttermmemory.New().Checkpoint(),
		}
		close(responseCh)

		// Check for Run errors
		select {
		case err := <-errCh:
			require.NoError(t, err, "unexpected error from Run")
		case <-ctx.Done():
			t.Fatal("timeout waiting for Run completion")
		}

		// Wait for the promise result
		result, err := promise.Get()
		assert.Error(t, err, "expected error for invalid JSON")
		assert.Equal(t, testResponse{}, result)
	})
}

func TestRunCommandMethods(t *testing.T) {
	agent := &mockAgent{}
	thread := shorttermmemory.New()
	hook := &mockHook{}

	cmd, err := NewRunCommand(agent, thread, hook)
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
