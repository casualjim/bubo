package agent

import (
	"testing"

	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testModel struct{}

func (m *testModel) Name() string {
	return "test-model"
}

func (m *testModel) Provider() provider.Provider {
	return nil
}

func TestDefaultAgent(t *testing.T) {
	t.Run("basic properties", func(t *testing.T) {
		agent := &defaultAgent{
			name:         "test-agent",
			model:        &testModel{},
			instructions: "test instructions",
		}

		assert.Equal(t, "test-agent", agent.Name())
		assert.Equal(t, &testModel{}, agent.Model())
		assert.False(t, agent.ParallelToolCalls())
		assert.Empty(t, agent.Tools())
	})
}

func TestNewAgent(t *testing.T) {
	agent := New(Name("test"), Model(&testModel{}), Instructions("instructions"))

	assert.Equal(t, "test", agent.Name())
	assert.Equal(t, &testModel{}, agent.Model())
	assert.True(t, agent.ParallelToolCalls())
	assert.Empty(t, agent.Tools())
}

func TestRenderInstructions(t *testing.T) {
	t.Run("no template variables", func(t *testing.T) {
		agent := New(Name("test"), Model(&testModel{}), Instructions("simple instructions"))
		result, err := agent.RenderInstructions(types.ContextVars{})
		require.NoError(t, err)
		assert.Equal(t, "simple instructions", result)
	})

	t.Run("with template variables", func(t *testing.T) {
		agent := New(Name("test"), Model(&testModel{}), Instructions("Hello {{.Name}}"))
		result, err := agent.RenderInstructions(types.ContextVars{"Name": "World"})
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("with invalid template", func(t *testing.T) {
		agent := New(Name("test"), Model(&testModel{}), Instructions("Hello {{.Name"))
		_, err := agent.RenderInstructions(types.ContextVars{"Name": "World"})
		require.Error(t, err)
	})

	t.Run("with missing variable", func(t *testing.T) {
		agent := New(Name("test"), Model(&testModel{}), Instructions("Hello {{.Name}}"))
		_, err := agent.RenderInstructions(types.ContextVars{})
		require.Error(t, err)
	})
}
