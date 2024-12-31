package provider

import (
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestToolDefinition_ToNameAndSchema(t *testing.T) {
	om := orderedmap.New[string, *jsonschema.Schema]()
	om.Set("value1", &jsonschema.Schema{
		Type: "string",
	})
	tests := []struct {
		name       string
		tool       ToolDefinition
		wantName   string
		wantSchema *jsonschema.Schema
	}{
		{
			name: "basic tool",
			tool: ToolDefinition{
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

func TestToDynamicJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "simple struct",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{
				Name: "test",
				Age:  30,
			},
			want: map[string]interface{}{
				"name": "test",
				"age":  float64(30),
			},
			wantErr: false,
		},
		{
			name:    "invalid input",
			input:   make(chan int),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToDynamicJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
