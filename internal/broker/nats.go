package broker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/nats-io/nats.go"
)

type Event struct {
	CompletionChunk events.Chunk[messages.Response]
	Response        events.Result[any]
	Message         *shorttermmemory.Checkpoint
}

type natsBroker struct {
	client *nats.Conn
	topics *haxmap.Map[string, *natsTopic]
}

func NATS(client *nats.Conn) *natsBroker {
	return &natsBroker{
		client: client,
		topics: haxmap.New[string, *natsTopic](),
	}
}

func (b *natsBroker) Topic(ctx context.Context, id string) Topic {
	top, _ := b.topics.GetOrCompute(id, func() *natsTopic {
		return &natsTopic{
			subject: id,
			client:  b.client,
		}
	})
	return top
}

type natsTopic struct {
	client  *nats.Conn
	subject string
}

func (t *natsTopic) Publish(ctx context.Context, event events.Event) error {
	eb, err := events.ToJSON(event)
	if err != nil {
		return err
	}
	return t.client.Publish(t.subject, eb)
}

func (t *natsTopic) Subscribe(ctx context.Context, hook events.Hook) (Subscription, error) {
	if hook == nil {
		return nil, fmt.Errorf("hook is required")
	}
	sub := make(chan events.Event, 50)
	nsub, err := t.client.Subscribe(t.subject, func(msg *nats.Msg) {
		event, err := events.FromJSON(msg.Data)
		if err != nil {
			slog.Error("failed to unmarshal event", slogx.Error(err))
			return
		}

		sub <- event

		if msg.Reply != "" {
			if nerr := msg.Ack(); nerr != nil {
				slog.Error("failed to ack message", slogx.Error(nerr))
				return
			}
		}
	})

	nsub.SetClosedHandler(func(_ string) { close(sub) })
	if err != nil {
		return nil, err
	}

	go forwardToHook(ctx, sub, hook)
	return &natsSubscription{
		id:  uuidx.NewString(),
		sub: nsub,
	}, nil
}

type natsSubscription struct {
	id  string
	sub *nats.Subscription
}

func (n *natsSubscription) ID() string {
	return n.id
}

func (n *natsSubscription) Unsubscribe() {
	if err := n.sub.Unsubscribe(); err != nil {
		slog.Error("failed to unsubscribe", slogx.Error(err), slog.String("subscription", n.id))
	}
}
