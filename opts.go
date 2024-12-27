package bubo

import (
	"fmt"

	"github.com/fogfish/opts"
)

// WithToolName returns a function that sets the Name field of
// agentFunctionOptions to the provided name. This can be used to
// configure an agent function with a specific name.
//
// Parameters:
//   - name: A string representing the name to be assigned.
//
// Returns:
//   - A function that takes a pointer to agentFunctionOptions and sets its Name field.
var WithToolName = opts.ForName[AgentToolDefinition, string]("Name")

// WithToolDescription returns a function that sets the description of an agent function.
// It takes a string parameter 'description' and returns a function that modifies the
// 'Description' field of the provided 'agentFunctionOptions' struct.
var WithToolDescription = opts.ForName[AgentToolDefinition, string]("Description")

// WithToolParameters returns a function that sets the Parameters field
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
func WithToolParameters(parameters ...string) opts.Option[AgentToolDefinition] {
	return opts.Type[AgentToolDefinition](func(o *AgentToolDefinition) error {
		o.Parameters = make(map[string]string, len(parameters))
		for i, p := range parameters {
			o.Parameters[fmt.Sprintf("param%d", i)] = p
		}
		return nil
	})
}
