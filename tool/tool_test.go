package tool

import (
	"reflect"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestMustAgentFunction(t *testing.T) {
	testFunc := func() {}

	t.Run("valid function", func(t *testing.T) {
		assert.NotPanics(t, func() {
			def := Must(testFunc)
			assert.Equal(t, reflect.ValueOf(testFunc).Pointer(), reflect.ValueOf(def.Function).Pointer())
		})
	})

	t.Run("invalid function", func(t *testing.T) {
		assert.Panics(t, func() {
			Must("not a function")
		})
	})
}

func TestName(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "simple name",
			toolName: "test_tool",
		},
		{
			name:     "empty name",
			toolName: "testFunc", // Empty name should fall back to function name
		},
		{
			name:     "name with spaces",
			toolName: "test tool name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFunc := func() {}
			def, err := New(testFunc, Name(tt.toolName))
			if err != nil {
				t.Errorf("AgentTool() error = %v", err)
				return
			}
			if def.Name != tt.toolName {
				t.Errorf("AgentTool() with Name() got = %q, want %q", def.Name, tt.toolName)
			}
		})
	}
}

func TestDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "simple description",
			description: "A test tool",
		},
		{
			name:        "empty description",
			description: "",
		},
		{
			name:        "multiline description",
			description: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFunc := func() {}
			def, err := New(testFunc, Description(tt.description))
			if err != nil {
				t.Errorf("AgentTool() error = %v", err)
				return
			}
			if def.Description != tt.description {
				t.Errorf("AgentTool() with Description() got = %v, want %v", def.Description, tt.description)
			}
		})
	}
}

func TestParameters(t *testing.T) {
	tests := []struct {
		name       string
		parameters []string
		want       map[string]string
	}{
		{
			name:       "no parameters",
			parameters: []string{},
			want:       map[string]string{},
		},
		{
			name:       "single parameter",
			parameters: []string{"param1"},
			want: map[string]string{
				"param0": "param1",
			},
		},
		{
			name:       "multiple parameters",
			parameters: []string{"param1", "param2", "param3"},
			want: map[string]string{
				"param0": "param1",
				"param1": "param2",
				"param2": "param3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFunc := func() {}
			def, err := New(testFunc, Parameters(tt.parameters...))
			if err != nil {
				t.Errorf("AgentTool() error = %v", err)
				return
			}

			if len(def.Parameters) != len(tt.want) {
				t.Errorf("AgentTool() with Parameters() got len = %v, want len = %v", len(def.Parameters), len(tt.want))
				return
			}

			for k, v := range tt.want {
				if got := def.Parameters[k]; got != v {
					t.Errorf("AgentTool() with Parameters() got[%s] = %v, want[%s] = %v", k, got, k, v)
				}
			}
		})
	}
}

func TestWithToolCombined(t *testing.T) {
	testFunc := func() {}
	def, err := New(testFunc,
		Name("test_tool"),
		Description("A test tool"),
		Parameters("param1", "param2"),
	)
	if err != nil {
		t.Errorf("AgentTool() error = %v", err)
		return
	}

	// Verify all fields are set correctly
	if def.Name != "test_tool" {
		t.Errorf("Name got = %v, want %v", def.Name, "test_tool")
	}

	if def.Description != "A test tool" {
		t.Errorf("Description got = %v, want %v", def.Description, "A test tool")
	}

	wantParams := map[string]string{
		"param0": "param1",
		"param1": "param2",
	}

	if len(def.Parameters) != len(wantParams) {
		t.Errorf("Parameters got len = %v, want len = %v", len(def.Parameters), len(wantParams))
		return
	}

	for k, v := range wantParams {
		if got := def.Parameters[k]; got != v {
			t.Errorf("Parameters got[%s] = %v, want[%s] = %v", k, got, k, v)
		}
	}
}

func TestToolDefinition_ToNameAndSchema(t *testing.T) {
	om := orderedmap.New[string, *jsonschema.Schema]()
	om.Set("value1", &jsonschema.Schema{
		Type: "string",
	})
	tests := []struct {
		name       string
		tool       Definition
		wantName   string
		wantSchema *jsonschema.Schema
	}{
		{
			name: "basic tool",
			tool: Definition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  map[string]string{"param0": "value1"},

				Function: func(s string) string { return s },
			},
			wantName: "test_tool",
			wantSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: om,
				Required:   []string{"value1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotSchema := tt.tool.ToNameAndSchema()
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantSchema, gotSchema)
		})
	}
}
