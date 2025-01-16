// Package types provides core type definitions used throughout the Bubo framework.
package types

import "encoding/json"

// ContextVars represents a key-value store of context variables used for template rendering.
// It maps string keys to values of any type. These variables can be used to customize
// agent instructions through template substitution.
//
// Common use cases include:
//   - Passing user information to agents
//   - Configuring agent behavior dynamically
//   - Sharing state between conversation turns
//   - Providing environment-specific settings
//
// Example usage:
//
//	vars := ContextVars{
//	    "user_id": "123",
//	    "preferences": map[string]string{
//	        "language": "en",
//	        "timezone": "UTC",
//	    },
//	    "features": []string{"chat", "search"},
//	    "settings": struct {
//	        Theme    string `json:"theme"`
//	        FontSize int    `json:"fontSize"`
//	    }{
//	        Theme:    "dark",
//	        FontSize: 14,
//	    },
//	}
//
// The variables can be used in agent instructions:
//
//	agent := agent.New(
//	    agent.Instructions(`
//	        User ID: {{.user_id}}
//	        Language: {{.preferences.language}}
//	        Features: {{range .features}}{{.}} {{end}}
//	    `),
//	)
//
// Thread Safety:
// ContextVars is a map type and is not safe for concurrent modification.
// If variables need to be modified during execution, proper synchronization
// should be implemented by the caller.
type ContextVars map[string]any

// String returns a JSON string representation of the ContextVars.
// If marshaling fails, it returns an empty string.
//
// This method is useful for:
//   - Debugging context variable content
//   - Logging variable state
//   - Serializing variables for storage
//   - Template rendering
//
// Example:
//
//	vars := ContextVars{"key": "value"}
//	fmt.Println(vars.String()) // Output: {"key":"value"}
func (cv ContextVars) String() string {
	jsonData, err := json.Marshal(cv)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
