package bubo

import (
	"context"

	"github.com/casualjim/bubo/events"
)

type Hook[T any] interface {
	events.Hook
	OnResult(context.Context, T)
	OnClose(context.Context)
}
