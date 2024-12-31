package bubo

import (
	"os"
	"testing"

	"github.com/casualjim/bubo/types"
	openai "github.com/openai/openai-go"
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
		agent.DisableParallelToolCalls() // Start from disabled
		agent.EnableParallelToolCalls()
		assert.True(t, agent.ParallelToolCalls())
	})

	t.Run("DisableParallelToolCalls", func(t *testing.T) {
		agent.EnableParallelToolCalls() // Start from enabled
		agent.DisableParallelToolCalls()
		assert.False(t, agent.ParallelToolCalls())
	})

	t.Run("WithParallelToolCalls", func(t *testing.T) {
		agent.DisableParallelToolCalls() // Start from disabled
		result := agent.WithParallelToolCalls()
		assert.Same(t, agent, result)
		assert.True(t, agent.ParallelToolCalls())
	})

	t.Run("WithoutParallelToolCalls", func(t *testing.T) {
		agent.EnableParallelToolCalls() // Start from enabled
		result := agent.WithoutParallelToolCalls()
		assert.Same(t, agent, result)
		assert.False(t, agent.ParallelToolCalls())
	})
}

func TestRenderInstructions(t *testing.T) {
	t.Run("no template variables", func(t *testing.T) {
		agent := NewAgent("test", "gpt-4", "simple instructions")
		result, err := agent.RenderInstructions(types.ContextVars{})
		require.NoError(t, err)
		assert.Equal(t, "simple instructions", result)
	})

	t.Run("with template variables", func(t *testing.T) {
		agent := NewAgent("test", "gpt-4", "Hello {{.Name}}")
		result, err := agent.RenderInstructions(types.ContextVars{"Name": "World"})
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("with invalid template", func(t *testing.T) {
		agent := NewAgent("test", "gpt-4", "Hello {{.Name")
		_, err := agent.RenderInstructions(types.ContextVars{"Name": "World"})
		require.Error(t, err)
	})

	t.Run("with missing variable", func(t *testing.T) {
		agent := NewAgent("test", "gpt-4", "Hello {{.Name}}")
		_, err := agent.RenderInstructions(types.ContextVars{})
		require.Error(t, err)
	})
}

func TestNewAgentWithEmptyModel(t *testing.T) {
	t.Run("uses env var when set", func(t *testing.T) {
		oldModel := os.Getenv("OPENAI_DEFAULT_MODEL")
		defer os.Setenv("OPENAI_DEFAULT_MODEL", oldModel)

		os.Setenv("OPENAI_DEFAULT_MODEL", "test-model")
		agent := NewAgent("test", "", "instructions")
		assert.Equal(t, "test-model", agent.Model())
	})

	t.Run("uses default when env var not set", func(t *testing.T) {
		oldModel := os.Getenv("OPENAI_DEFAULT_MODEL")
		defer os.Setenv("OPENAI_DEFAULT_MODEL", oldModel)

		os.Unsetenv("OPENAI_DEFAULT_MODEL")
		agent := NewAgent("test", "", "instructions")
		assert.Equal(t, openai.ChatModelGPT4oMini, agent.Model())
	})
}
