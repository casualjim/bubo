package types

import "encoding/json"

// ContextVars represents a key-value store of context variables used for template rendering.
// It maps string keys to values of any type. These variables can be used to customize
// agent instructions through template substitution.
type ContextVars map[string]any

// String returns a JSON string representation of the ContextVars.
// If marshaling fails, it returns an empty string.
func (cv ContextVars) String() string {
	jsonData, err := json.Marshal(cv)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
