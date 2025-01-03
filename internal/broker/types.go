package broker

import (
	"context"

	"github.com/casualjim/bubo/events"
)

type Broker interface {
	Topic(context.Context, string) Topic
}

type Topic interface {
	Publish(context.Context, events.Event) error
	Subscribe(context.Context, events.Hook) (Subscription, error)
}

type Subscription interface {
	ID() string
	Unsubscribe()
}
