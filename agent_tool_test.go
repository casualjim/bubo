package bubo

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustAgentFunction(t *testing.T) {
	testFunc := func() {}

	t.Run("valid function", func(t *testing.T) {
		assert.NotPanics(t, func() {
			def := MustAgentTool(testFunc)
			assert.Equal(t, reflect.ValueOf(testFunc).Pointer(), reflect.ValueOf(def.Function).Pointer())
		})
	})

	t.Run("invalid function", func(t *testing.T) {
		assert.Panics(t, func() {
			MustAgentTool("not a function")
		})
	})
}

/*
func TestAgentFunctionDefinitionToOpenAI(t *testing.T) {
	def := AgentFunctionDefinition{
		Name:        "test",
		Description: "test description",
		Parameters:  map[string]string{"param0": "value"},
		Function:    func(value string) {},
	}

	openAIDef := def.ToOpenAI(nil)
	assert.Equal(t, openai.ChatCompletionToolTypeFunction, openAIDef.Type.Value)
	assert.Equal(t, def.Name, openAIDef.Function.Value.Name.Value)
	assert.Equal(t, def.Description, openAIDef.Function.Value.Description.Value)
	assert.True(t, openAIDef.Function.Value.Strict.Value)
}
*/
