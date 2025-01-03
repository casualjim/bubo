package broker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/provider"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type recordingHook struct {
	mu                sync.Mutex
	wg                *sync.WaitGroup
	ready             chan struct{} // signals when hook is ready to receive events
	userPrompts       []messages.Message[messages.UserMessage]
	assistantChunks   []messages.Message[messages.AssistantMessage]
	toolCallChunks    []messages.Message[messages.ToolCallMessage]
	assistantMessages []messages.Message[messages.AssistantMessage]
	toolCallMessages  []messages.Message[messages.ToolCallMessage]
	toolCallResponses []messages.Message[messages.ToolResponse]
	errors            []error
}

func newRecordingHook() *recordingHook {
	return &recordingHook{
		ready: make(chan struct{}),
	}
}

func (r *recordingHook) signalReady() {
	close(r.ready)
}

func (r *recordingHook) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	r.mu.Lock()
	r.userPrompts = append(r.userPrompts, msg)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

func (r *recordingHook) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	r.mu.Lock()
	r.assistantChunks = append(r.assistantChunks, msg)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

func (r *recordingHook) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	r.mu.Lock()
	r.toolCallChunks = append(r.toolCallChunks, msg)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

func (r *recordingHook) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	r.mu.Lock()
	r.assistantMessages = append(r.assistantMessages, msg)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

func (r *recordingHook) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	r.mu.Lock()
	r.toolCallMessages = append(r.toolCallMessages, msg)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

func (r *recordingHook) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	r.mu.Lock()
	r.toolCallResponses = append(r.toolCallResponses, msg)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

func (r *recordingHook) OnResult(ctx context.Context, result interface{}) {
}

func (r *recordingHook) OnError(ctx context.Context, err error) {
	r.mu.Lock()
	r.errors = append(r.errors, err)
	r.mu.Unlock()
	if r.wg != nil {
		r.wg.Done()
	}
}

type overflowHook struct {
	*recordingHook
	processed chan struct{} // signals when processing is complete
	block     chan struct{} // controls when to process events
}

func (h *overflowHook) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	// Wait for signal to process events
	<-h.block
	h.recordingHook.OnAssistantMessage(ctx, msg)

	h.mu.Lock()
	if len(h.assistantMessages) == cap(h.block) { // we've processed as many events as our blocking channel capacity
		select {
		case <-h.processed: // Already closed
		default:
			close(h.processed)
		}
	}
	h.mu.Unlock()
}

func TestBroker(t *testing.T) {
	t.Run("creates unique topics", func(t *testing.T) {
		broker := Local()
		topic1 := broker.Topic(context.Background(), "test1")
		topic2 := broker.Topic(context.Background(), "test2")
		assert.NotEqual(t, topic1, topic2)
	})

	t.Run("reuses existing topics", func(t *testing.T) {
		broker := Local()
		topic1 := broker.Topic(context.Background(), "test")
		topic2 := broker.Topic(context.Background(), "test")
		assert.Equal(t, topic1, topic2)
	})
}

func TestTopic(t *testing.T) {
	t.Run("publishes events to all subscribers", func(t *testing.T) {
		broker := Local().(*localBroker)
		broker = broker.WithSlowSubscriberTimeout(1 * time.Millisecond) // Very short timeout for testing
		topic := broker.Topic(context.Background(), "test")

		var wg sync.WaitGroup
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
		recorder1.wg = &wg
		recorder2.wg = &wg

		// Test AssistantMessage
		wg.Add(4) // 2 recorders * 2 messages
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

	t.Run("handles channel overflow", func(t *testing.T) {
		broker := Local().(*localBroker)
		broker = broker.WithSlowSubscriberTimeout(50 * time.Millisecond) // Give enough time for test setup
		topic := broker.Topic(context.Background(), "test")
		ctx := context.Background()

		const bufferSize = 10 // Small buffer to ensure overflow
		processed := make(chan struct{})
		block := make(chan struct{}, bufferSize) // Control channel with same capacity as expected processed events

		recorder := &overflowHook{
			recordingHook: newRecordingHook(),
			processed:     processed,
			block:         block,
		}

		sub, err := topic.Subscribe(ctx, recorder)
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Signal hook is ready
		recorder.signalReady()
		<-recorder.ready

		// Publish more events than the buffer can handle
		const numEvents = bufferSize * 2 // Double the buffer size to ensure overflow
		for i := 0; i < numEvents; i++ {
			msg := messages.New().AssistantMessage(fmt.Sprintf("message-%d", i))
			event := events.Response[messages.AssistantMessage]{
				RunID:    uuid.New(),
				TurnID:   uuid.New(),
				Response: msg.Payload,
			}
			err := topic.Publish(ctx, event)
			require.NoError(t, err)
		}

		// Allow processing of exactly bufferSize events
		for i := 0; i < bufferSize; i++ {
			block <- struct{}{} // Release one event at a time
		}

		// Wait for processing to complete
		<-processed

		// Verify exactly bufferSize events were processed
		recorder.mu.Lock()
		messagesLen := len(recorder.assistantMessages)
		recorder.mu.Unlock()

		assert.Equal(t, bufferSize, messagesLen, "Should process exactly bufferSize events")
	})

	t.Run("respects publish context cancellation", func(t *testing.T) {
		broker := Local()
		topic := broker.Topic(context.Background(), "test")

		// Create a subscriber
		recorder := newRecordingHook()
		sub, err := topic.Subscribe(context.Background(), recorder)
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Signal hook is ready
		recorder.signalReady()

		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Publish event with cancelled context
		msg := messages.New().AssistantMessage("test message")
		event := events.Response[messages.AssistantMessage]{
			RunID:    uuid.New(),
			TurnID:   uuid.New(),
			Response: msg.Payload,
		}
		err = topic.Publish(ctx, event)
		require.NoError(t, err) // Publish still succeeds but returns early

		// Give a short time for any unexpected processing
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		<-ctx.Done()
		recorder.mu.Lock()
		assert.Len(t, recorder.assistantMessages, 0)
		recorder.mu.Unlock()
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		broker := Local()
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
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		<-ctx.Done()

		// Publish event after cancellation
		msg := messages.New().AssistantMessage("test message")
		event := events.Response[messages.AssistantMessage]{
			RunID:    uuid.New(),
			TurnID:   uuid.New(),
			Response: msg.Payload,
		}
		err = topic.Publish(context.Background(), event)
		require.NoError(t, err)

		// Verify event wasn't processed
		recorder.mu.Lock()
		assert.Len(t, recorder.assistantMessages, 0)
		recorder.mu.Unlock()
	})

	t.Run("handles unsubscribe", func(t *testing.T) {
		broker := Local()
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

	t.Run("handles concurrent operations", func(t *testing.T) {
		broker := Local()
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
		}
		defer func() {
			for _, sub := range subs {
				sub.Unsubscribe()
			}
		}()

		// Signal all hooks are ready
		for _, recorder := range recorders {
			recorder.signalReady()
		}

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

func TestFromStreamEvent(t *testing.T) {
	runID := uuid.New()
	turnID := uuid.New()
	timestamp := strfmt.DateTime(time.Now())
	meta := gjson.Parse("{}")

	t.Run("converts Delim", func(t *testing.T) {
		streamEvent := provider.Delim{
			RunID:  runID,
			TurnID: turnID,
			Delim:  "test",
		}
		event := events.FromStreamEvent(streamEvent, "")
		delim, ok := event.(events.Delim)
		require.True(t, ok)
		assert.Equal(t, streamEvent.RunID, delim.RunID)
		assert.Equal(t, streamEvent.TurnID, delim.TurnID)
		assert.Equal(t, streamEvent.Delim, delim.Delim)
	})

	t.Run("converts AssistantMessage chunk", func(t *testing.T) {
		msg := messages.New().AssistantMessage("test")
		streamEvent := provider.Chunk[messages.AssistantMessage]{
			RunID:     runID,
			TurnID:    turnID,
			Chunk:     msg.Payload,
			Timestamp: timestamp,
			Meta:      meta,
		}
		event := events.FromStreamEvent(streamEvent, "test")
		chunk, ok := event.(events.Chunk[messages.AssistantMessage])
		require.True(t, ok)
		assert.Equal(t, streamEvent.RunID, chunk.RunID)
		assert.Equal(t, streamEvent.TurnID, chunk.TurnID)
		assert.Equal(t, streamEvent.Chunk, chunk.Chunk)
		assert.Equal(t, "test", chunk.Sender)
		assert.Equal(t, streamEvent.Timestamp, chunk.Timestamp)
		assert.Equal(t, streamEvent.Meta, chunk.Meta)
	})

	t.Run("converts ToolCallMessage chunk", func(t *testing.T) {
		msg := messages.New().ToolCall([]messages.ToolCallData{{
			ID:        "test-id",
			Name:      "test-tool",
			Arguments: `{"arg":"value"}`,
		}})
		streamEvent := provider.Chunk[messages.ToolCallMessage]{
			RunID:     runID,
			TurnID:    turnID,
			Chunk:     msg.Payload,
			Timestamp: timestamp,
			Meta:      meta,
		}
		event := events.FromStreamEvent(streamEvent, "test")
		chunk, ok := event.(events.Chunk[messages.ToolCallMessage])
		require.True(t, ok)
		assert.Equal(t, streamEvent.RunID, chunk.RunID)
		assert.Equal(t, streamEvent.TurnID, chunk.TurnID)
		assert.Equal(t, streamEvent.Chunk, chunk.Chunk)
		assert.Equal(t, "test", chunk.Sender)
		assert.Equal(t, streamEvent.Timestamp, chunk.Timestamp)
		assert.Equal(t, streamEvent.Meta, chunk.Meta)
	})

	t.Run("converts AssistantMessage response", func(t *testing.T) {
		msg := messages.New().AssistantMessage("test")
		streamEvent := provider.Response[messages.AssistantMessage]{
			RunID:     runID,
			TurnID:    turnID,
			Response:  msg.Payload,
			Timestamp: timestamp,
			Meta:      meta,
		}
		event := events.FromStreamEvent(streamEvent, "test")
		response, ok := event.(events.Response[messages.AssistantMessage])
		require.True(t, ok)
		assert.Equal(t, streamEvent.RunID, response.RunID)
		assert.Equal(t, streamEvent.TurnID, response.TurnID)
		assert.Equal(t, streamEvent.Response, response.Response)
		assert.Equal(t, "test", response.Sender)
		assert.Equal(t, streamEvent.Timestamp, response.Timestamp)
		assert.Equal(t, streamEvent.Meta, response.Meta)
	})

	t.Run("converts ToolCallMessage response", func(t *testing.T) {
		msg := messages.New().ToolCall([]messages.ToolCallData{{
			ID:        "test-id",
			Name:      "test-tool",
			Arguments: `{"arg":"value"}`,
		}})
		streamEvent := provider.Response[messages.ToolCallMessage]{
			RunID:     runID,
			TurnID:    turnID,
			Response:  msg.Payload,
			Timestamp: timestamp,
			Meta:      meta,
		}
		event := events.FromStreamEvent(streamEvent, "test")
		response, ok := event.(events.Response[messages.ToolCallMessage])
		require.True(t, ok)
		assert.Equal(t, streamEvent.RunID, response.RunID)
		assert.Equal(t, streamEvent.TurnID, response.TurnID)
		assert.Equal(t, streamEvent.Response, response.Response)
		assert.Equal(t, "test", response.Sender)
		assert.Equal(t, streamEvent.Timestamp, response.Timestamp)
		assert.Equal(t, streamEvent.Meta, response.Meta)
	})

	t.Run("converts Error", func(t *testing.T) {
		err := errors.New("test error")
		streamEvent := provider.Error{
			RunID:     runID,
			TurnID:    turnID,
			Err:       err,
			Timestamp: timestamp,
			Meta:      meta,
		}
		event := events.FromStreamEvent(streamEvent, "test")
		errorEvent, ok := event.(events.Error)
		require.True(t, ok)
		assert.Equal(t, streamEvent.RunID, errorEvent.RunID)
		assert.Equal(t, streamEvent.TurnID, errorEvent.TurnID)
		assert.Equal(t, streamEvent.Err, errorEvent.Err)
		assert.Equal(t, "test", errorEvent.Sender)
		assert.Equal(t, streamEvent.Timestamp, errorEvent.Timestamp)
		assert.Equal(t, streamEvent.Meta, errorEvent.Meta)
	})

	t.Run("panics on unknown event type", func(t *testing.T) {
		type unknownEvent struct {
			provider.Delim        // embed a known type but modify it to make it unknown
			Extra          string // add an extra field to make it a different type
		}
		assert.Panics(t, func() {
			events.FromStreamEvent(unknownEvent{}, "")
		})
	})
}
