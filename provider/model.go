package provider

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/pkg/reflectx"
	"github.com/casualjim/bubo/types"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Provider interface {
	ChatCompletion(context.Context, CompletionParams) (<-chan StreamEvent, error)
}

type CompletionParams struct {
	RunID          uuid.UUID
	Instructions   string
	Thread         *shorttermmemory.Aggregator
	Stream         bool
	ResponseSchema *jsonschema.Schema
	Model          interface {
		Name() string
		Provider() Provider
	}
	Tools []ToolDefinition
	_     struct{}
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]string
	Function    any
}

var functionReflector = jsonschema.Reflector{
	AllowAdditionalProperties: true,
	DoNotReference:            true,
}

func (td ToolDefinition) ToNameAndSchema() (string, *jsonschema.Schema) {
	return functionDefinitionJSON(&functionReflector, td)
}

func functionDefinitionJSON(reflector *jsonschema.Reflector, f ToolDefinition) (string, *jsonschema.Schema) {
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

func ToDynamicJSON(val any) (map[string]any, error) {
	result := make(map[string]any)
	b, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}
