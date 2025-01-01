package tool

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/casualjim/bubo/pkg/reflectx"
	"github.com/casualjim/bubo/pkg/stdx"
	"github.com/casualjim/bubo/types"
	"github.com/fogfish/opts"
	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Definition represents the definition of an agent function.
// It includes the function's name, description, parameters, and the function itself.
type Definition struct {
	Name        string
	Description string
	Parameters  map[string]string
	Function    any
}

var functionReflector = jsonschema.Reflector{
	AllowAdditionalProperties: true,
	DoNotReference:            true,
}

func (td Definition) ToNameAndSchema() (string, *jsonschema.Schema) {
	return functionDefinitionJSON(&functionReflector, td)
}

func functionDefinitionJSON(reflector *jsonschema.Reflector, f Definition) (string, *jsonschema.Schema) {
	// Get the type and value using reflection
	val := reflect.ValueOf(f.Function)
	typ := val.Type()

	// Get function name
	name := f.Name
	if name == "" && typ.Kind() == reflect.Func {
		// For named types (like AgentFunction), use the type name
		if typ.Name() != "" {
			name = typ.String()
		} else {
			// For methods, use the method name
			if typ.NumIn() > 0 && typ.In(0).Kind() == reflect.Struct {
				if fn := runtime.FuncForPC(val.Pointer()); fn != nil {
					name = fn.Name()
					if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
						name = name[lastDot+1:]
					}
				}
			} else {
				// For anonymous functions, use the full signature
				if fn := runtime.FuncForPC(val.Pointer()); fn != nil {
					name = fn.Name()
					if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
						name = name[lastDot+1:]
					}
				} else {
					name = typ.String()
				}
			}
		}
	}

	// Create function parameters schema
	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: orderedmap.New[string, *jsonschema.Schema](),
	}

	// If it's a function type, analyze its signature
	if typ.Kind() == reflect.Func {
		// Get input parameters
		numIn := typ.NumIn()
		startIdx := 0
		// Skip receiver for methods
		if numIn > 0 && typ.In(0).Kind() == reflect.Struct {
			startIdx = 1
		}

		var required []string
		for i := startIdx; i < numIn; i++ {
			paramType := typ.In(i)
			if reflectx.IsRefinedType[types.ContextVars](paramType) {
				continue
			}

			paramName := fmt.Sprintf("param%d", i-startIdx)
			if f.Parameters != nil {
				if p, ok := f.Parameters[paramName]; ok {
					paramName = p
				}
			}

			propSchema := reflector.ReflectFromType(paramType)
			propSchema.Version = ""
			schema.Properties.Set(paramName, propSchema)
			required = append(required, paramName)
		}
		if len(required) > 0 {
			schema.Required = required
		}
	}

	return name, schema
}

// Option is a type alias for a function that modifies
// the configuration options of an agent tool. It allows for
// flexible and customizable configuration of agent tools by
// applying various options.
type Option = opts.Option[Definition]

// Must wraps the AgentTool call and ensures that any error
// returned by AgentTool is handled by panicking. It takes a function `f`
// and a variadic number of AgentToolOption `options` as arguments, and
// returns an AgentToolDefinition. If AgentTool returns an error,
// Must will panic.
//
// Parameters:
//   - f: The function to be wrapped.
//   - options: A variadic number of options to configure the agent tool.
//
// Returns:
//   - ToolDefinition: The definition of the agent tool.
func Must(f any, options ...Option) Definition {
	return stdx.Must1(New(f, options...))
}

// New creates an AgentToolDefinition from the provided function and options.
// The function is assigned to the Function field of the resulting AgentToolDefinition.
//
// Parameters:
//   - f: The function to be assigned to the AgentToolDefinition.
//   - options: A variadic list of AgentToolOption to configure the AgentToolDefinition.
//
// Returns:
//
//	A ToolDefinition with the provided function and applied options.
func New(f any, options ...Option) (Definition, error) {
	// validate that f is a function
	if !reflectx.IsFunction(f) {
		return Definition{}, fmt.Errorf("provided value is not a function")
	}

	var def Definition
	if err := opts.Apply(&def, options); err != nil {
		return Definition{}, err
	}
	if def.Name == "" {
		def.Name = reflectx.FunctionName(f)
	}

	def.Function = f
	return def, nil
}

// Name returns a function that sets the Name field of
// agentFunctionOptions to the provided name. This can be used to
// configure an agent function with a specific name.
//
// Parameters:
//   - name: A string representing the name to be assigned.
//
// Returns:
//   - A function that takes a pointer to agentFunctionOptions and sets its Name field.
var Name = opts.ForName[Definition, string]("Name")

// Description returns a function that sets the description of an agent function.
// It takes a string parameter 'description' and returns a function that modifies the
// 'Description' field of the provided 'agentFunctionOptions' struct.
var Description = opts.ForName[Definition, string]("Description")

// Parameters returns a function that sets the Parameters field
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
func Parameters(parameters ...string) opts.Option[Definition] {
	return opts.Type[Definition](func(o *Definition) error {
		o.Parameters = make(map[string]string, len(parameters))
		for i, p := range parameters {
			o.Parameters[fmt.Sprintf("param%d", i)] = p
		}
		return nil
	})
}
