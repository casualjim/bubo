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

type localBroker[T any] struct {
	topics                *haxmap.Map[string, *topic[T]]
	slowSubscriberTimeout time.Duration
}

func Local[T any]() Broker[T] {
	return &localBroker[T]{
		topics:                haxmap.New[string, *topic[T]](),
		slowSubscriberTimeout: defaultSlowSubscriberTimeout,
	}
}

// WithSlowSubscriberTimeout configures the timeout for detecting slow subscribers
func (b *localBroker[T]) WithSlowSubscriberTimeout(timeout time.Duration) *localBroker[T] {
	b.slowSubscriberTimeout = timeout
	return b
}

func (b *localBroker[T]) Topic(ctx context.Context, id string) Topic[T] {
	topic, _ := b.topics.GetOrCompute(id, func() *topic[T] {
		return &topic[T]{
			ID:                    id,
			subscriptions:         haxmap.New[string, *subscription[T]](),
			slowSubscriberTimeout: b.slowSubscriberTimeout,
		}
	})
	return topic
}

type topic[T any] struct {
	ID                    string
	subscriptions         *haxmap.Map[string, *subscription[T]]
	slowSubscriberTimeout time.Duration
}

func (t *topic[T]) Publish(ctx context.Context, event events.Event) error {
	t.subscriptions.ForEach(func(id string, sub *subscription[T]) bool {
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

func (t *topic[T]) Subscribe(ctx context.Context, hook events.Hook[T]) (Subscription, error) {
	if hook == nil {
		return nil, fmt.Errorf("hook is required")
	}
	sub := t.newSubscription(ctx, hook)
	return sub, nil
}

func (t *topic[T]) newSubscription(ctx context.Context, hook events.Hook[T]) *subscription[T] {
	id := uuidx.NewString()
	sub := &subscription[T]{
		id:        id, // Use the same ID for both the subscription and map key
		ctx:       ctx,
		channel:   make(chan events.Event, 50), // Buffer size optimized for typical usage
		closeOnce: sync.Once{},
		onClose:   func() { t.subscriptions.Del(id) },
		hook:      hook,
	}
	t.subscriptions.Set(id, sub)
	go sub.forwardToHook()
	return sub
}

type subscription[T any] struct {
	id        string
	ctx       context.Context
	channel   chan events.Event
	closeOnce sync.Once
	onClose   func()
	hook      events.Hook[T]
}

func (s *subscription[T]) ID() string {
	return s.id
}

func (s *subscription[T]) Unsubscribe() {
	s.closeOnce.Do(func() {
		if s.onClose != nil {
			s.onClose()
		}
		close(s.channel)
	})
}

func (s *subscription[T]) forwardToHook() {
	for {
		select {
		case event, ok := <-s.channel:
			if !ok {
				return
			}
			switch event := event.(type) {
			case events.Delim:
				// Delim events are used for stream control and don't need to be forwarded to hooks
			case events.Request[messages.UserMessage]:
				s.hook.OnUserPrompt(s.ctx, messages.Message[messages.UserMessage]{
					Payload:   event.Message,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Chunk[messages.AssistantMessage]:
				s.hook.OnAssistantChunk(s.ctx, messages.Message[messages.AssistantMessage]{
					Payload:   event.Chunk,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Chunk[messages.ToolCallMessage]:
				s.hook.OnToolCallChunk(s.ctx, messages.Message[messages.ToolCallMessage]{
					Payload:   event.Chunk,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Request[messages.ToolResponse]:
				s.hook.OnToolCallResponse(s.ctx, messages.Message[messages.ToolResponse]{
					Payload:   event.Message,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Response[messages.ToolCallMessage]:
				s.hook.OnToolCallMessage(s.ctx, messages.Message[messages.ToolCallMessage]{
					Payload:   event.Response,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Response[messages.AssistantMessage]:
				s.hook.OnAssistantMessage(s.ctx, messages.Message[messages.AssistantMessage]{
					Payload:   event.Response,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case events.Result[T]:
				s.hook.OnResult(s.ctx, event.Result)
			case events.Error:
				s.hook.OnError(s.ctx, event.Err)
			default:
				panic(fmt.Sprintf("unknown event type: %T", event))
			}
		case <-s.ctx.Done():
			return
		}
	}
}
