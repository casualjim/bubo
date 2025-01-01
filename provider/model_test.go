package provider

import (
	"testing"

	"github.com/casualjim/bubo/tool"
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
		tool       tool.Definition
		wantName   string
		wantSchema *jsonschema.Schema
	}{
		{
			name: "basic tool",
			tool: tool.Definition{
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
