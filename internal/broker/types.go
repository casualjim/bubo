package broker

import (
	"context"

	"github.com/casualjim/bubo/events"
)

type Broker[T any] interface {
	Topic(context.Context, string) Topic[T]
}

type Topic[T any] interface {
	Publish(context.Context, events.Event) error
	Subscribe(context.Context, events.Hook[T]) (Subscription, error)
}

type Subscription interface {
	ID() string
	Unsubscribe()
}
