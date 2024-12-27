package bubo

import (
	"fmt"

	"github.com/casualjim/bubo/pkg/reflectx"
	"github.com/casualjim/bubo/pkg/stdx"
	"github.com/fogfish/opts"
)

// AgentFunctionOption is a type alias for a function that modifies
// the configuration options of an agent function. It allows for
// flexible and customizable configuration of agent functions by
// applying various options.
type AgentFunctionOption = opts.Option[AgentFunctionDefinition]

// MustAgentFunction wraps the AgentFunction call and ensures that any error
// returned by AgentFunction is handled by panicking. It takes a function `f`
// and a variadic number of AgentFunctionOption `options` as arguments, and
// returns an AgentFunctionDefinition. If AgentFunction returns an error,
// MustAgentFunction will panic.
//
// Parameters:
//   - f: The function to be wrapped.
//   - options: A variadic number of options to configure the agent function.
//
// Returns:
//   - AgentFunctionDefinition: The definition of the agent function.
func MustAgentFunction(f any, options ...AgentFunctionOption) AgentFunctionDefinition {
	return stdx.Must1(AgentFunction(f, options...))
}

// AgentFunction creates an AgentFunctionDefinition from the provided function and options.
// The function is assigned to the Function field of the resulting AgentFunctionDefinition.
//
// Parameters:
//   - f: The function to be assigned to the AgentFunctionDefinition.
//   - options: A variadic list of AgentFunctionOption to configure the AgentFunctionDefinition.
//
// Returns:
//
//	An AgentFunctionDefinition with the provided function and applied options.
func AgentFunction(f any, options ...AgentFunctionOption) (AgentFunctionDefinition, error) {
	// validate that f is a function
	if !reflectx.IsFunction(f) {
		return AgentFunctionDefinition{}, fmt.Errorf("provided value is not a function")
	}

	var def AgentFunctionDefinition
	if err := opts.Apply(&def, options); err != nil {
		return AgentFunctionDefinition{}, err
	}
	if def.Name == "" {
		def.Name = reflectx.FunctionName(f)
	}

	def.Function = f
	return def, nil
}

// AgentFunctionDefinition represents the definition of an agent function.
// It includes the function's name, description, parameters, and the function itself.
type AgentFunctionDefinition struct {
	Name        string
	Description string
	Parameters  map[string]string
	Function    any
}

/*
func (afd *AgentFunctionDefinition) ToOpenAI(reflector *jsonschema.Reflector) openai.ChatCompletionToolParam {
	if reflector == nil {
		reflector = &jsonschema.Reflector{
			AllowAdditionalProperties: true,
			DoNotReference:            true,
		}
	}
	return functionDefinitionJSON(reflector, *afd)
}
*/
