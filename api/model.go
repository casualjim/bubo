package api

import (
	"github.com/casualjim/bubo/provider"
)

// Model defines the interface that all AI models must implement. It serves as a bridge
// between the model's identity and its provider implementation, allowing for a clean
// separation between model configuration and provider functionality.
//
// The interface is intentionally minimal to make it easy to implement while providing
// all necessary information for the system to work with different models.
type Model interface {
	// Name returns the identifier for this model as expected by the provider.
	// For example, with OpenAI this might be "gpt-4" or "gpt-3.5-turbo".
	// The name should be consistent with what the provider's API expects.
	Name() string

	// Provider returns the provider implementation that can execute requests
	// for this model. The provider handles the actual communication with
	// the AI service (e.g., OpenAI, Anthropic).
	//
	// The returned provider must be safe for concurrent use, as it may be
	// accessed by multiple goroutines simultaneously.
	Provider() provider.Provider
}
