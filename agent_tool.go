package bubo

import (
	"fmt"

	"github.com/casualjim/bubo/pkg/reflectx"
	"github.com/casualjim/bubo/pkg/stdx"
	"github.com/fogfish/opts"
)

// AgentToolOption is a type alias for a function that modifies
// the configuration options of an agent tool. It allows for
// flexible and customizable configuration of agent tools by
// applying various options.
type AgentToolOption = opts.Option[AgentToolDefinition]

// MustAgentTool wraps the AgentTool call and ensures that any error
// returned by AgentTool is handled by panicking. It takes a function `f`
// and a variadic number of AgentToolOption `options` as arguments, and
// returns an AgentToolDefinition. If AgentTool returns an error,
// MustAgentTool will panic.
//
// Parameters:
//   - f: The function to be wrapped.
//   - options: A variadic number of options to configure the agent tool.
//
// Returns:
//   - AgentToolDefinition: The definition of the agent tool.
func MustAgentTool(f any, options ...AgentToolOption) AgentToolDefinition {
	return stdx.Must1(AgentTool(f, options...))
}

// AgentTool creates an AgentToolDefinition from the provided function and options.
// The function is assigned to the Function field of the resulting AgentToolDefinition.
//
// Parameters:
//   - f: The function to be assigned to the AgentToolDefinition.
//   - options: A variadic list of AgentToolOption to configure the AgentToolDefinition.
//
// Returns:
//
//	An AgentToolDefinition with the provided function and applied options.
func AgentTool(f any, options ...AgentToolOption) (AgentToolDefinition, error) {
	// validate that f is a function
	if !reflectx.IsFunction(f) {
		return AgentToolDefinition{}, fmt.Errorf("provided value is not a function")
	}

	var def AgentToolDefinition
	if err := opts.Apply(&def, options); err != nil {
		return AgentToolDefinition{}, err
	}
	if def.Name == "" {
		def.Name = reflectx.FunctionName(f)
	}

	def.Function = f
	return def, nil
}

// AgentToolDefinition represents the definition of an agent function.
// It includes the function's name, description, parameters, and the function itself.
type AgentToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]string
	Function    any
}
