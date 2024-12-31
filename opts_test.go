package bubo

import (
	"testing"
)

func TestWithToolName(t *testing.T) {
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
			def, err := AgentTool(testFunc, WithToolName(tt.toolName))
			if err != nil {
				t.Errorf("AgentTool() error = %v", err)
				return
			}
			if def.Name != tt.toolName {
				t.Errorf("AgentTool() with WithToolName() got = %q, want %q", def.Name, tt.toolName)
			}
		})
	}
}

func TestWithToolDescription(t *testing.T) {
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
			def, err := AgentTool(testFunc, WithToolDescription(tt.description))
			if err != nil {
				t.Errorf("AgentTool() error = %v", err)
				return
			}
			if def.Description != tt.description {
				t.Errorf("AgentTool() with WithToolDescription() got = %v, want %v", def.Description, tt.description)
			}
		})
	}
}

func TestWithToolParameters(t *testing.T) {
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
			def, err := AgentTool(testFunc, WithToolParameters(tt.parameters...))
			if err != nil {
				t.Errorf("AgentTool() error = %v", err)
				return
			}

			if len(def.Parameters) != len(tt.want) {
				t.Errorf("AgentTool() with WithToolParameters() got len = %v, want len = %v", len(def.Parameters), len(tt.want))
				return
			}

			for k, v := range tt.want {
				if got := def.Parameters[k]; got != v {
					t.Errorf("AgentTool() with WithToolParameters() got[%s] = %v, want[%s] = %v", k, got, k, v)
				}
			}
		})
	}
}

func TestWithToolCombined(t *testing.T) {
	testFunc := func() {}
	def, err := AgentTool(testFunc,
		WithToolName("test_tool"),
		WithToolDescription("A test tool"),
		WithToolParameters("param1", "param2"),
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
