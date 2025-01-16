package provider

import (
	"context"

	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/tool"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
)

// Provider defines the interface for AI model providers (e.g., OpenAI, Anthropic).
// Implementations of this interface handle the specifics of communicating with
// different AI services while maintaining a consistent interface for the rest
// of the application.
type Provider interface {
	ChatCompletion(context.Context, CompletionParams) (<-chan StreamEvent, error)
}

// CompletionParams encapsulates all parameters needed for a chat completion request.
// It provides configuration for how the AI model should process the request and
// structure its response.
type CompletionParams struct {
	// RunID uniquely identifies this completion request for tracking and debugging
	RunID uuid.UUID

	// Instructions provide the system prompt or role instructions for the AI
	Instructions string

	// Thread contains the conversation history and context
	Thread *shorttermmemory.Aggregator

	// Stream indicates whether to receive responses as a stream of chunks
	// When true, responses come incrementally. When false, wait for complete response.
	Stream bool

	// ResponseSchema defines the structure for formatted output
	// When provided, the AI will attempt to format its response according to this schema
	ResponseSchema *StructuredOutput

	// Model specifies which AI model to use for this completion
	// It must provide its name and associated provider
	Model interface {
		Name() string
		Provider() Provider
	}

	// Tools defines the available functions/capabilities the AI can use
	Tools []tool.Definition

	// Prevents unkeyed literals
	_ struct{}
}

// StructuredOutput defines a schema for formatted AI responses.
// This allows requesting responses in specific formats for easier parsing
// and validation.
type StructuredOutput struct {
	// Name identifies this output format
	Name string

	// Description explains the purpose and usage of this format
	Description string

	// Schema defines the JSON structure that responses should follow
	Schema *jsonschema.Schema
}
