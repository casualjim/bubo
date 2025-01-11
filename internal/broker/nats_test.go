package broker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/messages"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func setupNATS(t *testing.T) *nats.Conn {
	nc, err := nats.Connect(nats.DefaultURL)
	require.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})
	return nc
}

func TestNATSBroker(t *testing.T) {
	t.Run("creates broker instance", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		require.NotNil(t, broker)
	})

	t.Run("creates unique topics", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic1 := broker.Topic(context.Background(), "test1")
		topic2 := broker.Topic(context.Background(), "test2")
		assert.NotEqual(t, topic1, topic2)
	})

	t.Run("reuses existing topics", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic1 := broker.Topic(context.Background(), "test")
		topic2 := broker.Topic(context.Background(), "test")
		assert.Equal(t, topic1, topic2)
	})
}

func TestNATSTopic(t *testing.T) {
	t.Run("publishes events to all subscribers", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic := broker.Topic(context.Background(), "test")

		recorder1 := newRecordingHook()
		recorder2 := newRecordingHook()

		ctx := context.Background()
		sub1, err := topic.Subscribe(ctx, recorder1)
		require.NoError(t, err)
		sub2, err := topic.Subscribe(ctx, recorder2)
		require.NoError(t, err)
		defer sub1.Unsubscribe()
		defer sub2.Unsubscribe()

		// Signal hooks are ready
		recorder1.signalReady()
		recorder2.signalReady()

		// Test different event types
		runID := uuid.New()
		turnID := uuid.New()
		timestamp := strfmt.DateTime(time.Now())

		// Set up WaitGroup for both recorders
		var wg sync.WaitGroup
		wg.Add(4) // 2 recorders * 2 messages
		recorder1.wg = &wg
		recorder2.wg = &wg

		// Test AssistantMessage
		msg := messages.New().AssistantMessage("test message")
		event1 := events.Response[messages.AssistantMessage]{
			RunID:     runID,
			TurnID:    turnID,
			Response:  msg.Payload,
			Sender:    "test",
			Timestamp: timestamp,
			Meta:      gjson.Parse("{}"),
		}
		err1 := topic.Publish(ctx, event1)
		require.NoError(t, err1)

		// Test ToolCallMessage
		msg2 := messages.New().ToolCall([]messages.ToolCallData{{
			ID:        "test-id",
			Name:      "test-tool",
			Arguments: `{"arg":"value"}`,
		}})
		event2 := events.Response[messages.ToolCallMessage]{
			RunID:     runID,
			TurnID:    turnID,
			Response:  msg2.Payload,
			Sender:    "test",
			Timestamp: timestamp,
			Meta:      gjson.Parse("{}"),
		}
		err2 := topic.Publish(ctx, event2)
		require.NoError(t, err2)

		// Wait for all messages to be processed
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for messages to be processed")
		}

		// Verify both hooks received the events
		recorder1.mu.Lock()
		assert.Len(t, recorder1.assistantMessages, 1)
		assert.Len(t, recorder1.toolCallMessages, 1)
		recorder1.mu.Unlock()

		recorder2.mu.Lock()
		assert.Len(t, recorder2.assistantMessages, 1)
		assert.Len(t, recorder2.toolCallMessages, 1)
		recorder2.mu.Unlock()
	})

	t.Run("handles unsubscribe", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic := broker.Topic(context.Background(), "test")

		ctx := context.Background()
		recorder := newRecordingHook()
		sub, err := topic.Subscribe(ctx, recorder)
		require.NoError(t, err)

		// Signal hook is ready
		recorder.signalReady()

		// Unsubscribe and wait a moment for unsubscribe to propagate
		sub.Unsubscribe()
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer timeoutCancel()
		<-timeoutCtx.Done()

		// Publish event after unsubscribe
		msg := messages.New().AssistantMessage("test message")
		event := events.Response[messages.AssistantMessage]{
			RunID:    uuid.New(),
			TurnID:   uuid.New(),
			Response: msg.Payload,
		}
		err = topic.Publish(ctx, event)
		require.NoError(t, err)

		// Verify event wasn't processed
		recorder.mu.Lock()
		assert.Len(t, recorder.assistantMessages, 0)
		recorder.mu.Unlock()
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic := broker.Topic(context.Background(), "test")

		ctx, cancel := context.WithCancel(context.Background())
		recorder := newRecordingHook()
		sub, err := topic.Subscribe(ctx, recorder)
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Signal hook is ready
		recorder.signalReady()

		// Cancel context and wait a moment for cancellation to propagate
		cancel()
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer timeoutCancel()
		<-timeoutCtx.Done()

		// Publish event after cancellation
		msg := messages.New().AssistantMessage("test message")
		event := events.Response[messages.AssistantMessage]{
			RunID:    uuid.New(),
			TurnID:   uuid.New(),
			Response: msg.Payload,
		}
		err = topic.Publish(ctx, event)
		require.NoError(t, err)

		// Verify event wasn't processed
		recorder.mu.Lock()
		assert.Len(t, recorder.assistantMessages, 0)
		recorder.mu.Unlock()
	})

	t.Run("handles invalid message", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic := broker.Topic(context.Background(), "test")

		ctx := context.Background()
		recorder := newRecordingHook()
		sub, err := topic.Subscribe(ctx, recorder)
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Signal hook is ready
		recorder.signalReady()

		// Publish invalid JSON data directly using NATS client
		err = nc.Publish("test", []byte("invalid json"))
		require.NoError(t, err)

		// Wait a bit to ensure no messages are processed
		time.Sleep(100 * time.Millisecond)
		recorder.mu.Lock()
		assert.Empty(t, recorder.assistantMessages)
		assert.Empty(t, recorder.userPrompts)
		assert.Empty(t, recorder.toolCallMessages)
		recorder.mu.Unlock()
	})

	t.Run("handles concurrent operations", func(t *testing.T) {
		nc := setupNATS(t)
		broker := NATS(nc)
		topic := broker.Topic(context.Background(), "test")
		ctx := context.Background()

		// Create multiple subscribers
		const numSubscribers = 10
		recorders := make([]*recordingHook, numSubscribers)
		subs := make([]Subscription, numSubscribers)
		var processWg sync.WaitGroup        // WaitGroup for event processing
		processWg.Add(numSubscribers * 100) // Each subscriber will process 100 events

		for i := 0; i < numSubscribers; i++ {
			recorders[i] = newRecordingHook()
			recorders[i].wg = &processWg // Pass WaitGroup to recorder
			sub, err := topic.Subscribe(ctx, recorders[i])
			require.NoError(t, err)
			subs[i] = sub
			recorders[i].signalReady()
		}
		defer func() {
			for _, sub := range subs {
				sub.Unsubscribe()
			}
		}()

		// Publish multiple events concurrently
		const numEvents = 100
		var publishWg sync.WaitGroup
		publishWg.Add(numEvents)
		for i := 0; i < numEvents; i++ {
			go func(i int) {
				defer publishWg.Done()
				msg := messages.New().AssistantMessage(fmt.Sprintf("message-%d", i))
				event := events.Response[messages.AssistantMessage]{
					RunID:    uuid.New(),
					TurnID:   uuid.New(),
					Response: msg.Payload,
				}
				err := topic.Publish(ctx, event)
				require.NoError(t, err)
			}(i)
		}

		// Wait for all events to be published and processed
		publishWg.Wait()
		processWg.Wait()

		// Verify all subscribers received all events
		for _, recorder := range recorders {
			recorder.mu.Lock()
			assert.Len(t, recorder.assistantMessages, numEvents)
			recorder.mu.Unlock()
		}
	})
}
