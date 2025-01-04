package provider

import (
	"context"

	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/tool"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
)

type Provider interface {
	ChatCompletion(context.Context, CompletionParams) (<-chan StreamEvent, error)
}

type CompletionParams struct {
	RunID          uuid.UUID
	Instructions   string
	Thread         *shorttermmemory.Aggregator
	Stream         bool
	ResponseSchema *StructuredOutput
	Model          interface {
		Name() string
		Provider() Provider
	}
	Tools []tool.Definition
	_     struct{}
}

type StructuredOutput struct {
	Name        string
	Description string
	Schema      *jsonschema.Schema
}
