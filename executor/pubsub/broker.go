package pubsub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo/pkg/messages"
	"github.com/casualjim/bubo/pkg/uuidx"
)

const defaultSlowSubscriberTimeout = 100 * time.Millisecond

type broker struct {
	topics                *haxmap.Map[string, *topic]
	slowSubscriberTimeout time.Duration
}

func LocalBroker() Broker {
	return &broker{
		topics:                haxmap.New[string, *topic](),
		slowSubscriberTimeout: defaultSlowSubscriberTimeout,
	}
}

// WithSlowSubscriberTimeout configures the timeout for detecting slow subscribers
func (b *broker) WithSlowSubscriberTimeout(timeout time.Duration) *broker {
	b.slowSubscriberTimeout = timeout
	return b
}

func (b *broker) Topic(ctx context.Context, id string) Topic {
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

func (t *topic) Publish(ctx context.Context, event Event) error {
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

func (t *topic) Subscribe(ctx context.Context, hook Hook) (Subscription, error) {
	if hook == nil {
		return nil, fmt.Errorf("hook is required")
	}
	sub := t.newSubscription(ctx, hook)
	return sub, nil
}

func (t *topic) newSubscription(ctx context.Context, hook Hook) *subscription {
	id := uuidx.NewString()
	sub := &subscription{
		id:        id, // Use the same ID for both the subscription and map key
		ctx:       ctx,
		channel:   make(chan Event, 50), // Buffer size optimized for typical usage
		closeOnce: sync.Once{},
		onClose:   func() { t.subscriptions.Del(id) },
		hook:      hook,
	}
	t.subscriptions.Set(id, sub)
	go sub.forwardToHook()
	return sub
}

type subscription struct {
	id        string
	ctx       context.Context
	channel   chan Event
	closeOnce sync.Once
	onClose   func()
	hook      Hook
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

func (s *subscription) forwardToHook() {
	for {
		select {
		case event, ok := <-s.channel:
			if !ok {
				return
			}
			switch event := event.(type) {
			case Delim:
				// Delim events are used for stream control and don't need to be forwarded to hooks
			case Request[messages.UserMessage]:
				s.hook.OnUserPrompt(s.ctx, messages.Message[messages.UserMessage]{
					Payload:   event.Message,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case Chunk[messages.AssistantMessage]:
				s.hook.OnAssistantChunk(s.ctx, messages.Message[messages.AssistantMessage]{
					Payload:   event.Chunk,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case Chunk[messages.ToolCallMessage]:
				s.hook.OnToolCallChunk(s.ctx, messages.Message[messages.ToolCallMessage]{
					Payload:   event.Chunk,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case Request[messages.ToolResponse]:
				s.hook.OnToolCallResponse(s.ctx, messages.Message[messages.ToolResponse]{
					Payload:   event.Message,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case Response[messages.ToolCallMessage]:
				s.hook.OnToolCallMessage(s.ctx, messages.Message[messages.ToolCallMessage]{
					Payload:   event.Response,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case Response[messages.AssistantMessage]:
				s.hook.OnAssistantMessage(s.ctx, messages.Message[messages.AssistantMessage]{
					Payload:   event.Response,
					Sender:    event.Sender,
					Timestamp: event.Timestamp,
					Meta:      event.Meta,
				})
			case Error:
				s.hook.OnError(s.ctx, event.Err)
			default:
				panic(fmt.Sprintf("unknown event type: %T", event))
			}
		case <-s.ctx.Done():
			return
		}
	}
}
