package bubo

import (
	"fmt"

	"github.com/fogfish/opts"
)

// WithFunctionName returns a function that sets the Name field of
// agentFunctionOptions to the provided name. This can be used to
// configure an agent function with a specific name.
//
// Parameters:
//   - name: A string representing the name to be assigned.
//
// Returns:
//   - A function that takes a pointer to agentFunctionOptions and sets its Name field.
var WithFunctionName = opts.ForName[AgentFunctionDefinition, string]("Name")

// WithFunctionDescription returns a function that sets the description of an agent function.
// It takes a string parameter 'description' and returns a function that modifies the
// 'Description' field of the provided 'agentFunctionOptions' struct.
var WithFunctionDescription = opts.ForName[AgentFunctionDefinition, string]("Description")

// WithFunctionParameters returns a function that sets the Parameters field
// of agentFunctionOptions to a map where each parameter is assigned a key
// in the format "paramN", where N is the index of the parameter in the input slice.
//
// Parameters:
//
//	parameters - a variadic string slice containing the parameters to be set.
//
// Returns:
//
//	A function that takes a pointer to agentFunctionOptions and sets its Parameters field.
func WithFunctionParameters(parameters ...string) opts.Option[AgentFunctionDefinition] {
	return opts.Type[AgentFunctionDefinition](func(o *AgentFunctionDefinition) error {
		o.Parameters = make(map[string]string, len(parameters))
		for i, p := range parameters {
			o.Parameters[fmt.Sprintf("param%d", i)] = p
		}
		return nil
	})
}
