package types

import (
	"testing"
)

func TestContextVars_String(t *testing.T) {
	tests := []struct {
		name string
		cv   ContextVars
		want string
	}{
		{
			name: "empty map",
			cv:   ContextVars{},
			want: "{}",
		},
		{
			name: "simple key-value",
			cv:   ContextVars{"key": "value"},
			want: `{"key":"value"}`,
		},
		{
			name: "multiple types",
			cv: ContextVars{
				"string": "value",
				"number": 42,
				"bool":   true,
				"null":   nil,
			},
			want: `{"bool":true,"null":null,"number":42,"string":"value"}`,
		},
		{
			name: "nested structures",
			cv: ContextVars{
				"nested": map[string]interface{}{
					"array": []interface{}{1, 2, 3},
					"obj":   map[string]interface{}{"key": "value"},
				},
			},
			want: `{"nested":{"array":[1,2,3],"obj":{"key":"value"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cv.String(); got != tt.want {
				t.Errorf("ContextVars.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContextVars_MapOperations(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		cv := make(ContextVars)
		cv["key"] = "value"

		if got := cv["key"]; got != "value" {
			t.Errorf("Expected value %v, got %v", "value", got)
		}
	})

	t.Run("update", func(t *testing.T) {
		cv := ContextVars{"key": "old"}
		cv["key"] = "new"

		if got := cv["key"]; got != "new" {
			t.Errorf("Expected value %v, got %v", "new", got)
		}
	})

	t.Run("delete", func(t *testing.T) {
		cv := ContextVars{"key": "value"}
		delete(cv, "key")

		if _, exists := cv["key"]; exists {
			t.Error("Key should not exist after deletion")
		}
	})

	t.Run("length", func(t *testing.T) {
		cv := ContextVars{
			"key1": "value1",
			"key2": "value2",
		}

		if len(cv) != 2 {
			t.Errorf("Expected length 2, got %d", len(cv))
		}
	})
}
