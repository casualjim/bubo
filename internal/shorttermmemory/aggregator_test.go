package shorttermmemory

import (
	"testing"
	"time"

	"github.com/casualjim/bubo/messages"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator(t *testing.T) {
	// Helper function to create an aggregator with valid ID
	newAggregator := func() *Aggregator {
		return &Aggregator{
			id: uuid.New(),
		}
	}

	t.Run("NewAggregator", func(t *testing.T) {
		agg := New()
		assert.NotEqual(t, uuid.Nil, agg.ID(), "should have valid ID")
		assert.Empty(t, agg.messages, "should have empty messages")
		assert.Equal(t, Usage{}, agg.usage, "should have zero usage")
		assert.Equal(t, 0, agg.initLen, "should have zero initLen")
	})

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
			toolCall := messages.New().ToolCall([]messages.ToolCallData{
				{
					ID:        "test-id",
					Name:      "test-tool",
					Arguments: `{"arg": "value"}`,
				},
			})
			agg.AddToolCall(toolCall)

			assert.Equal(t, 1, agg.Len())
			msgs := agg.Messages()
			assert.Equal(t, "test-id", msgs[0].Payload.(messages.ToolCallMessage).ToolCalls[0].ID)
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
			toolCall := messages.New().ToolCall([]messages.ToolCallData{
				{
					ID:        "test-id",
					Name:      "test-tool",
					Arguments: `{"arg": "value"}`,
				},
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

	t.Run("turn length tracking", func(t *testing.T) {
		t.Run("TurnLen returns correct count after fork", func(t *testing.T) {
			agg := newAggregator()

			// Add initial messages
			msg1 := messages.New().UserPrompt("message 1")
			msg2 := messages.New().UserPrompt("message 2")
			agg.AddUserPrompt(msg1)
			agg.AddUserPrompt(msg2)

			// Fork and add more messages
			forked := agg.Fork()
			assert.Equal(t, 0, forked.TurnLen(), "forked aggregator should start with 0 turn length")

			msg3 := messages.New().UserPrompt("message 3")
			msg4 := messages.New().UserPrompt("message 4")
			forked.AddUserPrompt(msg3)
			forked.AddUserPrompt(msg4)

			assert.Equal(t, 2, forked.TurnLen(), "should count messages added after fork")
		})
	})

	t.Run("checkpoint operations", func(t *testing.T) {
		t.Run("Checkpoint creates accurate snapshot", func(t *testing.T) {
			agg := newAggregator()

			// Add different types of messages
			userMsg := messages.New().UserPrompt("user message")
			assistantMsg := messages.New().AssistantMessage("assistant message")
			toolCall := messages.New().ToolCall([]messages.ToolCallData{
				{
					ID:        "test-id",
					Name:      "test-tool",
					Arguments: `{"arg": "value"}`,
				},
			})

			agg.AddUserPrompt(userMsg)
			agg.AddAssistantMessage(assistantMsg)
			agg.AddToolCall(toolCall)

			// Set some usage data
			agg.usage = Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
			}

			// Create checkpoint
			checkpoint := agg.Checkpoint()

			// Verify checkpoint data
			assert.Equal(t, agg.ID(), checkpoint.ID(), "checkpoint should preserve aggregator ID")
			assert.Equal(t, agg.Len(), len(checkpoint.Messages()), "checkpoint should have same number of messages")
			assert.Equal(t, agg.Usage(), checkpoint.Usage(), "checkpoint should preserve usage data")

			// Verify message contents and types
			msgs := checkpoint.Messages()
			assert.IsType(t, messages.UserMessage{}, msgs[0].Payload)
			assert.IsType(t, messages.AssistantMessage{}, msgs[1].Payload)
			assert.IsType(t, messages.ToolCallMessage{}, msgs[2].Payload)

			// Verify checkpoint is immutable by modifying returned messages
			msgs = append(msgs, eraseType(messages.New().UserPrompt("new message")))
			checkpointMsgs := checkpoint.Messages()
			assert.Equal(t, 3, len(checkpointMsgs), "checkpoint messages should remain unchanged")
			assert.Equal(t, 4, len(msgs), "modified slice should have new message")
		})

		t.Run("Checkpoint preserves message order and content", func(t *testing.T) {
			agg := newAggregator()

			// Add messages in specific order
			msg1 := messages.New().UserPrompt("first")
			msg2 := messages.New().AssistantMessage("second")
			msg3 := messages.New().ToolResponse("test-id", "test-tool", "third")

			agg.AddUserPrompt(msg1)
			agg.AddAssistantMessage(msg2)
			agg.AddToolResponse(msg3)

			checkpoint := agg.Checkpoint()
			msgs := checkpoint.Messages()

			// Verify order and content
			assert.Equal(t, "first", msgs[0].Payload.(messages.UserMessage).Content.Content)
			assert.Equal(t, "second", msgs[1].Payload.(messages.AssistantMessage).Content.Content)
			assert.Equal(t, "third", msgs[2].Payload.(messages.ToolResponse).Content)
		})
	})

	t.Run("message iteration", func(t *testing.T) {
		t.Run("MessagesIter handles mixed message types", func(t *testing.T) {
			agg := newAggregator()

			// Add different types of messages
			messages := []struct {
				msg messages.Message[messages.ModelMessage]
				typ interface{}
			}{
				{eraseType(messages.New().UserPrompt("user")), messages.UserMessage{}},
				{eraseType(messages.New().AssistantMessage("assistant")), messages.AssistantMessage{}},
				{eraseType(messages.New().ToolCall([]messages.ToolCallData{{ID: "id", Name: "tool"}})), messages.ToolCallMessage{}},
				{eraseType(messages.New().ToolResponse("id", "tool", "response")), messages.ToolResponse{}},
			}

			for _, m := range messages {
				agg.add(m.msg)
			}

			// Verify iterator returns messages in correct order and type
			i := 0
			for msg := range agg.MessagesIter() {
				assert.IsType(t, messages[i].typ, msg.Payload)
				i++
			}
			assert.Equal(t, len(messages), i, "iterator should return all messages")
		})
	})

	t.Run("type erasure", func(t *testing.T) {
		t.Run("eraseType preserves message fields", func(t *testing.T) {
			// Create message with all fields populated
			original := messages.New().UserPrompt("test content")
			original.Sender = "test-sender"
			original.Timestamp = strfmt.DateTime(time.Now())

			// Erase type
			erased := eraseType(original)

			// Verify all fields are preserved
			assert.Equal(t, original.Sender, erased.Sender)
			assert.Equal(t, original.Timestamp, erased.Timestamp)
			assert.Equal(t, "test content", erased.Payload.(messages.UserMessage).Content.Content)
		})

		t.Run("eraseType handles different message types", func(t *testing.T) {
			testCases := []struct {
				name  string
				msg   messages.Message[messages.ModelMessage]
				check func(t *testing.T, msg messages.Message[messages.ModelMessage])
			}{
				{
					name: "UserMessage",
					msg:  eraseType(messages.New().UserPrompt("user content")),
					check: func(t *testing.T, msg messages.Message[messages.ModelMessage]) {
						payload, ok := msg.Payload.(messages.UserMessage)
						assert.True(t, ok)
						assert.Equal(t, "user content", payload.Content.Content)
					},
				},
				{
					name: "AssistantMessage",
					msg:  eraseType(messages.New().AssistantMessage("assistant content")),
					check: func(t *testing.T, msg messages.Message[messages.ModelMessage]) {
						payload, ok := msg.Payload.(messages.AssistantMessage)
						assert.True(t, ok)
						assert.Equal(t, "assistant content", payload.Content.Content)
					},
				},
				{
					name: "ToolCallMessage",
					msg:  eraseType(messages.New().ToolCall([]messages.ToolCallData{{ID: "test-id", Name: "test-tool"}})),
					check: func(t *testing.T, msg messages.Message[messages.ModelMessage]) {
						payload, ok := msg.Payload.(messages.ToolCallMessage)
						assert.True(t, ok)
						assert.Equal(t, "test-id", payload.ToolCalls[0].ID)
					},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					tc.check(t, tc.msg)
				})
			}
		})

		t.Run("MergeInto combines checkpoint state correctly", func(t *testing.T) {
			// Create source aggregator with messages and usage
			source := newAggregator()
			source.AddUserPrompt(messages.New().UserPrompt("source message"))
			source.usage = Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
			}

			// Create checkpoint from source
			checkpoint := source.Checkpoint()

			// Create target aggregator with different state
			target := newAggregator()
			target.AddUserPrompt(messages.New().UserPrompt("target message"))
			target.usage = Usage{
				CompletionTokens: 5,
				PromptTokens:     10,
				TotalTokens:      15,
			}

			// Merge checkpoint into target
			checkpoint.MergeInto(target)

			// Verify messages are combined correctly
			msgs := target.Messages()
			assert.Equal(t, 2, len(msgs), "should have messages from both aggregators")
			assert.Equal(t, "target message", msgs[0].Payload.(messages.UserMessage).Content.Content)
			assert.Equal(t, "source message", msgs[1].Payload.(messages.UserMessage).Content.Content)

			// Verify usage is combined
			usage := target.Usage()
			assert.Equal(t, int64(15), usage.CompletionTokens)
			assert.Equal(t, int64(30), usage.PromptTokens)
			assert.Equal(t, int64(45), usage.TotalTokens)
		})

		t.Run("checkpoint preserves usage details", func(t *testing.T) {
			source := newAggregator()
			source.usage = Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					ReasoningTokens:          5,
					AcceptedPredictionTokens: 3,
					RejectedPredictionTokens: 2,
					AudioTokens:              1,
				},
				PromptTokensDetails: PromptTokensDetails{
					CachedTokens: 8,
					AudioTokens:  7,
				},
			}

			checkpoint := source.Checkpoint()
			usage := checkpoint.Usage()

			// Verify all usage details are preserved
			assert.Equal(t, source.usage.CompletionTokens, usage.CompletionTokens)
			assert.Equal(t, source.usage.PromptTokens, usage.PromptTokens)
			assert.Equal(t, source.usage.TotalTokens, usage.TotalTokens)
			assert.Equal(t, source.usage.CompletionTokensDetails, usage.CompletionTokensDetails)
			assert.Equal(t, source.usage.PromptTokensDetails, usage.PromptTokensDetails)
		})
	})
}
