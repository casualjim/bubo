package bubo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAgent(t *testing.T) {
	t.Run("basic properties", func(t *testing.T) {
		agent := &DefaultAgent{
			name:         "test-agent",
			model:        "gpt-4",
			instructions: "test instructions",
			toolChoice:   "auto",
		}

		assert.Equal(t, "test-agent", agent.Name())
		assert.Equal(t, "gpt-4", agent.Model())
		assert.Equal(t, "test instructions", agent.Instructions())
		assert.Equal(t, "auto", agent.ToolChoice())
		assert.False(t, agent.ParallelToolCalls())
		assert.Empty(t, agent.Tools())
	})
}

func TestNewAgent(t *testing.T) {
	agent := NewAgent("test", "gpt-4", "instructions")

	assert.Equal(t, "test", agent.Name())
	assert.Equal(t, "gpt-4", agent.Model())
	assert.Equal(t, "instructions", agent.Instructions())
	assert.Empty(t, agent.ToolChoice())
	assert.True(t, agent.ParallelToolCalls())
	assert.Empty(t, agent.Tools())
}

func TestDefaultAgentToolManagement(t *testing.T) {
	agent := NewAgent("test", "gpt-4", "instructions")

	testFunc := func() {}
	def1, err := AgentTool(testFunc, WithToolName("func1"))
	require.NoError(t, err)

	def2, err := AgentTool(testFunc, WithToolName("func2"))
	require.NoError(t, err)

	t.Run("AddTool", func(t *testing.T) {
		agent.AddTool(def1)
		assert.Len(t, agent.Tools(), 1)
		assert.Equal(t, "func1", agent.Tools()[0].Name)

		agent.AddTool(def2)
		assert.Len(t, agent.Tools(), 2)
	})

	t.Run("WithTool", func(t *testing.T) {
		agent := NewAgent("test", "gpt-4", "instructions")

		result := agent.WithTool(def1)
		assert.Same(t, agent, result)
		assert.Len(t, agent.Tools(), 1)

		agent.WithTool(def2)
		assert.Len(t, agent.Tools(), 2)
	})
}

func TestDefaultAgentToolChoice(t *testing.T) {
	agent := NewAgent("test", "gpt-4", "instructions")

	t.Run("SetToolChoice", func(t *testing.T) {
		agent.SetToolChoice("none")
		assert.Equal(t, "none", agent.ToolChoice())
	})

	t.Run("WithToolChoice", func(t *testing.T) {
		result := agent.WithToolChoice("auto")
		assert.Same(t, agent, result)
		assert.Equal(t, "auto", agent.ToolChoice())
	})
}

func TestDefaultAgentParallelToolCalls(t *testing.T) {
	agent := NewAgent("test", "gpt-4", "instructions")

	t.Run("EnableParallelToolCalls", func(t *testing.T) {
		agent.EnableParallelToolCalls()
		assert.True(t, agent.ParallelToolCalls())
	})

	t.Run("DisableParallelToolCalls", func(t *testing.T) {
		agent.DisableParallelToolCalls()
		assert.True(t, agent.ParallelToolCalls())
	})

	t.Run("WithParallelToolCalls", func(t *testing.T) {
		result := agent.WithParallelToolCalls()
		assert.Same(t, agent, result)
		assert.True(t, agent.ParallelToolCalls())
	})

	t.Run("WithoutParallelToolCalls", func(t *testing.T) {
		result := agent.WithoutParallelToolCalls()
		assert.Same(t, agent, result)
		assert.False(t, agent.ParallelToolCalls())
	})
}
