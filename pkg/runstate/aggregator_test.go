package runstate

import (
	"testing"

	"github.com/casualjim/bubo/pkg/messages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator(t *testing.T) {
	// Helper function to create an aggregator with a valid ID
	newAggregator := func() *Aggregator {
		return &Aggregator{
			id: uuid.New(),
		}
	}

	t.Run("basic operations", func(t *testing.T) {
		t.Run("new aggregator has valid ID", func(t *testing.T) {
			agg := newAggregator()
			assert.NotEqual(t, uuid.Nil, agg.ID())
		})

		t.Run("empty aggregator has length 0", func(t *testing.T) {
			agg := newAggregator()
			assert.Equal(t, 0, agg.Len())
		})

		t.Run("Messages returns copy of messages", func(t *testing.T) {
			agg := newAggregator()
			msg1 := messages.New().UserPrompt("message 1")
			msg2 := messages.New().UserPrompt("message 2")
			agg.AddUserPrompt(msg1)
			agg.AddUserPrompt(msg2)

			msgs := agg.Messages()
			assert.Equal(t, 2, len(msgs))

			// Verify it's a copy by modifying the returned slice
			msgs = append(msgs, eraseType(messages.New().UserPrompt("message 3")))
			assert.Equal(t, 2, agg.Len(), "original aggregator should be unchanged")
			assert.Equal(t, 3, len(msgs), "returned slice should be modified")
		})

		t.Run("MessagesIter provides iterator over messages", func(t *testing.T) {
			agg := newAggregator()
			msg1 := messages.New().UserPrompt("message 1")
			msg2 := messages.New().UserPrompt("message 2")
			agg.AddUserPrompt(msg1)
			agg.AddUserPrompt(msg2)

			count := 0
			for m := range agg.MessagesIter() {
				require.NotNil(t, m)
				count++
			}
			assert.Equal(t, 2, count)
		})
	})

	t.Run("message type handling", func(t *testing.T) {
		t.Run("AddMessage handles any model message type", func(t *testing.T) {
			agg := newAggregator()
			userMsg := messages.New().UserPrompt("user message")
			assistantMsg := messages.New().AssistantMessage("assistant message")

			AddMessage(agg, userMsg)
			AddMessage(agg, assistantMsg)

			assert.Equal(t, 2, agg.Len())
			msgs := agg.Messages()
			assert.Equal(t, "user message", msgs[0].Payload.(messages.UserMessage).Content.Content)
			assert.Equal(t, "assistant message", msgs[1].Payload.(messages.AssistantMessage).Content.Content)
		})

		t.Run("AddUserPrompt adds user messages", func(t *testing.T) {
			agg := newAggregator()
			msg := messages.New().UserPrompt("test message")
			agg.AddUserPrompt(msg)

			assert.Equal(t, 1, agg.Len())
			msgs := agg.Messages()
			assert.Equal(t, "test message", msgs[0].Payload.(messages.UserMessage).Content.Content)
		})

		t.Run("AddAssistantMessage adds assistant messages", func(t *testing.T) {
			agg := newAggregator()
			msg := messages.New().AssistantMessage("test response")
			agg.AddAssistantMessage(msg)

			assert.Equal(t, 1, agg.Len())
			msgs := agg.Messages()
			assert.Equal(t, "test response", msgs[0].Payload.(messages.AssistantMessage).Content.Content)
		})

		t.Run("AddToolCall adds tool call messages", func(t *testing.T) {
			agg := newAggregator()
			toolCall := messages.New().ToolCall("test-id", messages.ToolCallData{
				Name: "test-tool", Arguments: `{"arg": "value"}`,
			})
			agg.AddToolCall(toolCall)

			assert.Equal(t, 1, agg.Len())
			msgs := agg.Messages()
			assert.Equal(t, "test-id", msgs[0].Payload.(messages.ToolCallMessage).ID)
		})

		t.Run("AddToolResponse adds tool response messages", func(t *testing.T) {
			agg := newAggregator()
			response := messages.New().ToolResponse("call-id", "test-tool", "result")
			agg.AddToolResponse(response)

			assert.Equal(t, 1, agg.Len())
			msgs := agg.Messages()
			assert.Equal(t, "result", msgs[0].Payload.(messages.ToolResponse).Content)
		})
	})

	t.Run("fork and join operations", func(t *testing.T) {
		t.Run("Fork creates new aggregator with copied messages", func(t *testing.T) {
			original := newAggregator()
			msg1 := messages.New().UserPrompt("message 1")
			msg2 := messages.New().UserPrompt("message 2")
			original.AddUserPrompt(msg1)
			original.AddUserPrompt(msg2)

			forked := original.Fork()

			// Verify IDs are different
			assert.NotEqual(t, original.ID(), forked.ID())

			// Verify messages are copied
			assert.Equal(t, original.Len(), forked.Len())

			// Verify modifications don't affect original
			msg3 := messages.New().UserPrompt("message 3")
			forked.AddUserPrompt(msg3)
			assert.Equal(t, 2, original.Len())
			assert.Equal(t, 3, forked.Len())
		})

		t.Run("Join combines messages correctly after fork", func(t *testing.T) {
			agg1 := newAggregator()
			msg1 := messages.New().UserPrompt("message 1")
			msg2 := messages.New().UserPrompt("message 2")
			agg1.AddUserPrompt(msg1)
			agg1.AddUserPrompt(msg2)

			agg2 := agg1.Fork() // agg2 has msg1, msg2 and initLen=2

			// Add new message to original
			msg3 := messages.New().UserPrompt("message 3")
			agg1.AddUserPrompt(msg3)

			// Add new message to fork
			msg4 := messages.New().UserPrompt("message 4")
			agg2.AddUserPrompt(msg4)

			// Join should only append new messages from fork
			agg1.Join(agg2)

			msgs := agg1.Messages()
			assert.Equal(t, 4, msgs.Len(), "should have 4 messages total")

			// Verify message order
			assert.Equal(t, "message 1", msgs[0].Payload.(messages.UserMessage).Content.Content, "first message")
			assert.Equal(t, "message 2", msgs[1].Payload.(messages.UserMessage).Content.Content, "second message")
			assert.Equal(t, "message 3", msgs[2].Payload.(messages.UserMessage).Content.Content, "third message")
			assert.Equal(t, "message 4", msgs[3].Payload.(messages.UserMessage).Content.Content, "fourth message")
		})

		t.Run("Join respects message ordering with mixed types", func(t *testing.T) {
			agg1 := newAggregator()

			// Add initial messages
			userMsg := messages.New().UserPrompt("user message")
			assistantMsg := messages.New().AssistantMessage("assistant message")
			agg1.AddUserPrompt(userMsg)
			agg1.AddAssistantMessage(assistantMsg)

			// Fork and add different types of messages
			agg2 := agg1.Fork()

			// Add to original
			toolCall := messages.New().ToolCall("test-id", messages.ToolCallData{
				Name: "test-tool", Arguments: `{"arg": "value"}`,
			})
			agg1.AddToolCall(toolCall)

			// Add to fork
			toolResponse := messages.New().ToolResponse("test-id", "test-tool", "result")
			agg2.AddToolResponse(toolResponse)

			// Join and verify
			agg1.Join(agg2)

			msgs := agg1.Messages()
			assert.Equal(t, 4, msgs.Len(), "should have all messages")

			// Verify correct types and order
			assert.IsType(t, messages.UserMessage{}, msgs[0].Payload)
			assert.IsType(t, messages.AssistantMessage{}, msgs[1].Payload)
			assert.IsType(t, messages.ToolCallMessage{}, msgs[2].Payload)
			assert.IsType(t, messages.ToolResponse{}, msgs[3].Payload)
		})
	})

	t.Run("usage tracking", func(t *testing.T) {
		agg1 := &Aggregator{
			id: uuid.New(),
			usage: Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					ReasoningTokens: 5,
				},
				PromptTokensDetails: PromptTokensDetails{
					CachedTokens: 8,
				},
			},
		}

		agg2 := &Aggregator{
			id: uuid.New(),
			usage: Usage{
				CompletionTokens: 15,
				PromptTokens:     25,
				TotalTokens:      40,
				CompletionTokensDetails: CompletionTokensDetails{
					ReasoningTokens: 7,
				},
				PromptTokensDetails: PromptTokensDetails{
					CachedTokens: 12,
				},
			},
		}

		agg1.Join(agg2)

		usage := agg1.Usage()
		assert.Equal(t, int64(25), usage.CompletionTokens)
		assert.Equal(t, int64(45), usage.PromptTokens)
		assert.Equal(t, int64(70), usage.TotalTokens)
		assert.Equal(t, int64(12), usage.CompletionTokensDetails.ReasoningTokens)
		assert.Equal(t, int64(20), usage.PromptTokensDetails.CachedTokens)
	})

	t.Run("Join respects initLen for message appending", func(t *testing.T) {
		// Create first aggregator with initial messages
		agg1 := &Aggregator{
			id: uuid.New(),
		}
		msg1 := messages.New().UserPrompt("message 1")
		msg2 := messages.New().UserPrompt("message 2")
		agg1.add(eraseType(msg1))
		agg1.add(eraseType(msg2))

		// Fork when length is 2
		agg2 := agg1.Fork()
		assert.Equal(t, 2, agg2.initLen, "fork should have initLen of 2")

		// Add messages to both aggregators
		msg3 := messages.New().UserPrompt("message 3")
		agg1.add(eraseType(msg3)) // Add to original

		msg4 := messages.New().UserPrompt("message 4")
		msg5 := messages.New().UserPrompt("message 5")
		agg2.add(eraseType(msg4)) // Add to fork
		agg2.add(eraseType(msg5)) // Add to fork

		// Join should only append messages after initLen from fork
		agg1.Join(agg2)

		msgs := agg1.Messages()
		assert.Equal(t, 5, msgs.Len(), "should have 5 messages total")

		// Verify final message order:
		// - msg1, msg2 (original)
		// - msg3 (added to original after fork)
		// - msg4, msg5 (added to fork after initLen)
		assert.Equal(t, "message 1", msgs[0].Payload.(messages.UserMessage).Content.Content, "first message")
		assert.Equal(t, "message 2", msgs[1].Payload.(messages.UserMessage).Content.Content, "second message")
		assert.Equal(t, "message 3", msgs[2].Payload.(messages.UserMessage).Content.Content, "third message")
		assert.Equal(t, "message 4", msgs[3].Payload.(messages.UserMessage).Content.Content, "fourth message")
		assert.Equal(t, "message 5", msgs[4].Payload.(messages.UserMessage).Content.Content, "fifth message")
	})
}
