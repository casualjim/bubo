package provider

import (
	"context"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/pkg/messages"
	"github.com/casualjim/bubo/pkg/runstate"
)

type Model interface {
	Name() string
}

type Provider[Request any] interface {
	BuildCompletionRequest(bubo.Agent, *runstate.Aggregator) Request
	ChatCompletion(context.Context) (messages.ModelMessage, error)
}
