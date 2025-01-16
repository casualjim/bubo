// Package bubo provides a framework for building conversational AI agents that can interact
// in a structured manner. It supports multi-agent conversations, structured output,
// and flexible execution contexts.
//
// The execution package is a core component that handles:
//   - Conversation workflow execution
//   - Context and state management
//   - Structured output handling
//   - Event hooks and promises
//   - Resource cleanup
//
// Key concepts:
//   - ExecutionContext: Manages the execution environment and configuration
//   - Promises: Handles asynchronous result processing
//   - Hooks: Provides extension points for monitoring and control
//   - Structured Output: Enables type-safe response handling
package bubo

import (
	"context"
	"reflect"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/types"
	"github.com/fogfish/opts"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"
)

// Local creates a new ExecutionContext configured for local execution.
// It sets up a future-based promise system with the provided hook for handling results
// of type T. The context can be further customized using the provided options.
//
// The function is generic over type T, which represents the expected result type
// of the conversation. This enables type-safe handling of conversation outputs.
//
// Example usage:
//
//	hook := &MyHook[string]{} // Custom hook implementation
//	ctx := Local[string](hook,
//	    WithContextVars(types.ContextVars{"user": "alice"}),
//	    Streaming(true),
//	    WithMaxTurns(5),
//	    StructuredOutput[MyResponse]("response", "Structured response format"),
//	)
//
// Parameters:
//   - hook: Implementation of Hook[T] for handling results and lifecycle events
//   - options: Optional configuration parameters for the execution context
//
// The created context includes:
//   - Local executor for running commands
//   - Future-based promise system
//   - Event hook integration
//   - Automatic cleanup on completion
func Local[T any](hook Hook[T], options ...opts.Option[ExecutionContext]) ExecutionContext {
	fut := executor.NewFuture(executor.DefaultUnmarshal[T]())
	dp := &deferredPromise[T]{
		promise: fut,
		hook:    hook,
	}

	execCtx := ExecutionContext{
		executor: executor.NewLocal(),
		hook:     hook,
		promise:  dp,
		onClose: func(ctx context.Context) {
			dp.Forward(ctx)
			hook.OnClose(ctx)
		},
	}

	if err := opts.Apply(&execCtx, options); err != nil {
		panic(err)
	}

	return execCtx
}

// ExecutionContext holds the configuration and state for executing conversation steps.
// It manages the execution environment, event hooks, promises for handling results,
// and various execution parameters.
//
// The context is responsible for:
//   - Managing the execution environment
//   - Handling event hooks and promises
//   - Configuring response formats
//   - Managing conversation state
//   - Enforcing execution limits
//   - Coordinating cleanup
//
// Thread Safety:
// ExecutionContext is designed to be used by a single conversation workflow.
// While its components (hooks, promises) may be thread-safe, the context itself
// should not be shared across multiple concurrent conversations.
type ExecutionContext struct {
	executor       executor.Executor          // The executor responsible for running commands
	hook           events.Hook                // Hook for handling conversation events
	promise        executor.Promise           // Promise for handling command results
	responseSchema *provider.StructuredOutput // Schema for structured output responses
	contextVars    types.ContextVars          // Variables available in the execution context
	onClose        func(context.Context)      // Cleanup function called when execution completes
	stream         bool                       // Whether to stream responses
	maxTurns       int                        // Maximum number of conversation turns
}

// createCommand builds a RunCommand for the given agent using the current execution context.
// It applies context variables, structured output schema, streaming settings, and turn limits
// to the command configuration.
//
// The function combines all execution parameters into a single command:
//   - Agent configuration and memory
//   - Context variables for dynamic behavior
//   - Structured output schema for response validation
//   - Streaming settings for real-time responses
//   - Turn limits for conversation control
//
// Parameters:
//   - agent: The AI agent that will execute the command
//   - mem: Memory aggregator for conversation history
//
// Returns:
//   - RunCommand: Configured command ready for execution
//   - error: Any error encountered during command creation
func (e *ExecutionContext) createCommand(agent api.Agent, mem *shorttermmemory.Aggregator) (executor.RunCommand, error) {
	cmd, err := executor.NewRunCommand(agent, mem, e.hook)
	if err != nil {
		return executor.RunCommand{}, err
	}
	if len(e.contextVars) > 0 {
		cmd = cmd.WithContextVariables(e.contextVars)
	}
	if e.responseSchema != nil {
		cmd = cmd.WithStructuredOutput(e.responseSchema)
	}
	if e.stream {
		cmd = cmd.WithStream(e.stream)
	}
	if e.maxTurns > 0 {
		cmd = cmd.WithMaxTurns(e.maxTurns)
	}
	return cmd, nil
}

// jsonSchema generates a JSON schema for type T unless T is a gjson.Result or string.
// This is used for structured output validation in conversation responses.
//
// The function uses reflection to:
//   - Generate a JSON schema for complex types
//   - Skip schema generation for gjson.Result and string types
//   - Support custom type validation through jsonschema tags
//
// Type Parameters:
//   - T: The type to generate a schema for
//
// Returns:
//   - *jsonschema.Schema: Generated schema or nil for skipped types
func jsonSchema[T any]() *jsonschema.Schema {
	var schema *jsonschema.Schema
	var isGjsonResult bool
	var t T
	_, isGjsonResult = any(t).(gjson.Result)
	isString := reflect.TypeFor[T]().Kind() == reflect.String

	if !isGjsonResult && !isString {
		schema = executor.ToJSONSchema[T]()
	}

	return schema
}

var (
	// WithContextVars is an option to set context variables for the execution context.
	// These variables are made available to agents during execution and can be used
	// to pass dynamic configuration or state information.
	//
	// Example:
	//  Local(hook, WithContextVars(types.ContextVars{
	//      "user_id": "123",
	//      "preferences": map[string]string{"lang": "en"},
	//  }))
	WithContextVars = opts.ForName[ExecutionContext, types.ContextVars]("contextVars")

	// Streaming is an option to enable/disable response streaming.
	// When enabled, responses are sent incrementally as they become available.
	// When disabled, responses are sent only after completion.
	//
	// Example:
	//  Local(hook, Streaming(true))
	Streaming = opts.ForName[ExecutionContext, bool]("stream")

	// WithMaxTurns is an option to set the maximum number of conversation turns.
	// This helps prevent infinite loops and control resource usage.
	// A turn consists of one complete agent interaction cycle.
	//
	// Example:
	//  Local(hook, WithMaxTurns(5))
	WithMaxTurns = opts.ForName[ExecutionContext, int]("maxTurns")
)

// StructuredOutput creates an option to configure structured output for responses.
// It generates a JSON schema for type T and associates it with the given name and description.
// The schema is used to validate and structure the conversation output.
//
// This option enables type-safe handling of conversation results by:
//   - Generating a JSON schema from the Go type
//   - Validating responses against the schema
//   - Providing structured data to result handlers
//
// Example usage:
//
//	type Response struct {
//	    Status  string `json:"status"`
//	    Message string `json:"message"`
//	}
//
//	ctx := Local[Response](hook,
//	    StructuredOutput[Response](
//	        "status_response",
//	        "Response with status and message",
//	    ),
//	)
//
// Parameters:
//   - name: Identifier for this output format
//   - description: Human-readable explanation of the format
//
// Type Parameters:
//   - T: The Go type to generate a schema for
func StructuredOutput[T any](name, description string) opts.Option[ExecutionContext] {
	return opts.Type[ExecutionContext](func(s *ExecutionContext) error {
		schema := jsonSchema[T]()
		if schema != nil {
			s.responseSchema = &provider.StructuredOutput{
				Name:        name,
				Description: description,
				Schema:      schema,
			}
		}
		return nil
	})
}
