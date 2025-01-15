package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/uuidx"
)

const defaultSlowSubscriberTimeout = 100 * time.Millisecond

type localBroker struct {
	topics                *haxmap.Map[string, *topic]
	slowSubscriberTimeout time.Duration
}

func Local() Broker {
	return &localBroker{
		topics:                haxmap.New[string, *topic](),
		slowSubscriberTimeout: defaultSlowSubscriberTimeout,
	}
}

// WithSlowSubscriberTimeout configures the timeout for detecting slow subscribers
func (b *localBroker) WithSlowSubscriberTimeout(timeout time.Duration) *localBroker {
	b.slowSubscriberTimeout = timeout
	return b
}

func (b *localBroker) Topic(ctx context.Context, id string) Topic {
	topic, _ := b.topics.GetOrCompute(id, func() *topic {
		return &topic{
			ID:                    id,
			subscriptions:         haxmap.New[string, *subscription](),
			slowSubscriberTimeout: b.slowSubscriberTimeout,
		}
	})
	return topic
}

type topic struct {
	ID                    string
	subscriptions         *haxmap.Map[string, *subscription]
	slowSubscriberTimeout time.Duration
}

func (t *topic) Publish(ctx context.Context, event events.Event) error {
	t.subscriptions.ForEach(func(id string, sub *subscription) bool {
		if sub == nil {
			return true
		}

		// Check if subscription is still active
		select {
		case <-ctx.Done():
			return false
		case <-sub.ctx.Done():
			sub.Unsubscribe()
			return true
		default:
		}

		// Try to send the event
		select {
		case <-ctx.Done():
			return false
		case <-sub.ctx.Done():
			sub.Unsubscribe()
		case sub.channel <- event:
			// Successfully sent
		case <-time.After(t.slowSubscriberTimeout):
			// Channel is full after timeout, unsubscribe
			sub.Unsubscribe()
		}
		return true
	})
	return nil
}

func (t *topic) Subscribe(ctx context.Context, hook events.Hook) (Subscription, error) {
	if hook == nil {
		return nil, fmt.Errorf("hook is required")
	}
	sub := t.newSubscription(ctx, hook)
	return sub, nil
}

func (t *topic) newSubscription(ctx context.Context, hook events.Hook) *subscription {
	id := uuidx.NewString()
	sub := &subscription{
		id:        id, // Use the same ID for both the subscription and map key
		ctx:       ctx,
		channel:   make(chan events.Event, 50),
		closeOnce: sync.Once{},
		onClose:   func() { t.subscriptions.Del(id) },
	}
	t.subscriptions.Set(id, sub)
	go forwardToHook(ctx, sub.channel, hook)
	return sub
}

type subscription struct {
	id        string
	ctx       context.Context
	channel   chan events.Event
	closeOnce sync.Once
	onClose   func()
	// hook      events.Hook
}

func (s *subscription) ID() string {
	return s.id
}

func (s *subscription) Unsubscribe() {
	s.closeOnce.Do(func() {
		if s.onClose != nil {
			s.onClose()
		}
		close(s.channel)
	})
}

func forwardToHook(ctx context.Context, from chan events.Event, to events.Hook) {
	for {
		select {
		case event, ok := <-from:
			if !ok {
				return
			}
			switch event := event.(type) {
			case events.Delim:
				// Delim events are used for stream control and don't need to be forwarded to hooks
			case events.Request[messages.UserMessage]:
				to.OnUserPrompt(ctx, messages.Message[messages.UserMessage]{
					Payload:   event.Message,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Chunk[messages.AssistantMessage]:
				to.OnAssistantChunk(ctx, messages.Message[messages.AssistantMessage]{
					Payload:   event.Chunk,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Chunk[messages.ToolCallMessage]:
				to.OnToolCallChunk(ctx, messages.Message[messages.ToolCallMessage]{
					Payload:   event.Chunk,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Request[messages.ToolResponse]:
				to.OnToolCallResponse(ctx, messages.Message[messages.ToolResponse]{
					Payload:   event.Message,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Response[messages.ToolCallMessage]:
				to.OnToolCallMessage(ctx, messages.Message[messages.ToolCallMessage]{
					Payload:   event.Response,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Response[messages.AssistantMessage]:
				to.OnAssistantMessage(ctx, messages.Message[messages.AssistantMessage]{
					Payload:   event.Response,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})

			case events.Error:
				to.OnError(ctx, event.Err)
			default:
				panic(fmt.Sprintf("unknown event type: %T", event))
			}
		case <-ctx.Done():
			return
		}
	}
}
