// Package shorttermmemory provides functionality for managing the runtime state of message processing,
// including message aggregation, forking, and joining of message streams, as well as usage tracking.
package shorttermmemory

import (
	"iter"
	"slices"

	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// AggregatedMessages represents a collection of model messages that can be processed together.
// It provides a type-safe way to handle multiple messages while maintaining their order.
type AggregatedMessages []messages.Message[messages.ModelMessage]

// Len returns the number of messages in the collection.
func (a AggregatedMessages) Len() int {
	return len(a)
}

// New creates and initializes a new Aggregator instance.
// It sets up:
// - A new unique identifier
// - An empty message collection
// - Zero-initialized usage statistics
//
// Example:
//
//	agg := New()
//	// agg is ready to accept messages and track usage
func New() *Aggregator {
	return &Aggregator{
		id:       uuidx.New(),
		messages: make(AggregatedMessages, 0),
		usage:    Usage{},
	}
}

// Aggregator manages a collection of messages and their associated usage statistics.
// It supports fork-join operations to allow parallel processing of message streams
// while maintaining message order and proper usage tracking.
type Aggregator struct {
	id       uuid.UUID          // Unique identifier for this aggregator
	messages AggregatedMessages // Collection of messages being managed
	initLen  int                // Initial length at fork time, used for joining
	usage    Usage              // Usage statistics for token consumption
}

// ID returns the unique identifier of this aggregator.
// This ID is generated when the aggregator is created or forked.
func (a *Aggregator) ID() uuid.UUID {
	return a.id
}

// Len returns the total number of messages currently held by the aggregator.
func (a *Aggregator) Len() int {
	return a.messages.Len()
}

// TurnLen returns the number of messages added to the aggregator since it was forked.
func (a *Aggregator) TurnLen() int {
	return len(a.messages) - a.initLen
}

// Messages returns a copy of all messages in the aggregator.
// The returned slice is a deep copy, so modifications to it won't affect
// the original messages in the aggregator.
func (a *Aggregator) Messages() AggregatedMessages {
	return slices.Clone(a.messages)
}

// MessagesIter returns an iterator over all messages in the aggregator.
// This provides a memory-efficient way to process messages sequentially
// without creating a copy of the entire message slice.
func (a *Aggregator) MessagesIter() iter.Seq[messages.Message[messages.ModelMessage]] {
	return slices.Values(a.messages)
}

// eraseType converts a Message[T] to Message[ModelMessage] while preserving all fields.
// This is used internally to store messages of different specific types in the aggregator
// while maintaining type safety. The conversion is safe because T is constrained to ModelMessage.
func eraseType[T messages.ModelMessage](m messages.Message[T]) messages.Message[messages.ModelMessage] {
	return messages.Message[messages.ModelMessage]{
		RunID:     m.RunID,
		TurnID:    m.TurnID,
		Payload:   m.Payload,
		Sender:    m.Sender,
		Timestamp: m.Timestamp,
		Meta:      m.Meta,
	}
}

// AddMessage adds any message type that implements ModelMessage to the aggregator.
// This is a generic function that can handle any valid message type in the system.
// For common message types, prefer using the specific Add methods (AddUserPrompt,
// AddAssistantMessage, etc.) as they provide better type safety and clarity.
//
// Example:
//
//	agg := &Aggregator{...}
//	msg := messages.New().UserPrompt("hello")
//	AddMessage(agg, msg)
func AddMessage[T messages.ModelMessage](a *Aggregator, m messages.Message[T]) {
	a.add(eraseType(m))
}

// AddUserPrompt adds a user message to the aggregator.
// This is typically used for adding messages that represent user input or queries.
//
// Example:
//
//	msg := messages.New().UserPrompt("What's the weather?")
//	agg.AddUserPrompt(msg)
func (a *Aggregator) AddUserPrompt(m messages.Message[messages.UserMessage]) {
	a.add(eraseType(m))
}

// AddAssistantMessage adds an assistant's response message to the aggregator.
// This is used for messages that represent responses or outputs from the assistant.
//
// Example:
//
//	msg := messages.New().AssistantMessage("The weather is sunny.")
//	agg.AddAssistantMessage(msg)
func (a *Aggregator) AddAssistantMessage(m messages.Message[messages.AssistantMessage]) {
	a.add(eraseType(m))
}

// AddToolCall adds a tool call message to the aggregator.
// This is used when the assistant needs to invoke an external tool or service.
//
// Example:
//
//	toolCall := messages.New().ToolCall("weather-api", []ToolCallData{...})
//	agg.AddToolCall(toolCall)
func (a *Aggregator) AddToolCall(m messages.Message[messages.ToolCallMessage]) {
	a.add(eraseType(m))
}

// AddToolResponse adds a tool's response message to the aggregator.
// This is used to store the results returned from external tool invocations.
//
// Example:
//
//	response := messages.New().ToolResponse("call-id", "weather-api", "Temperature: 72°F")
//	agg.AddToolResponse(response)
func (a *Aggregator) AddToolResponse(m messages.Message[messages.ToolResponse]) {
	a.add(eraseType(m))
}

// add is an internal method that appends a new message to the aggregator's message collection.
// It's used by the public Add* methods after they've converted their specific message types
// to the generic ModelMessage type. Messages are added in order, maintaining the sequence
// of the conversation.
func (a *Aggregator) add(m messages.Message[messages.ModelMessage]) {
	a.messages = append(a.messages, m)
}

// Usage returns the current usage statistics for this aggregator.
// This includes token counts for prompts and completions, as well as
// detailed breakdowns of token usage by category.
func (a *Aggregator) Usage() Usage {
	return a.usage
}

func (a *Aggregator) AddUsage(u *Usage) {
	a.usage.AddUsage(u)
}

// Fork creates a new aggregator that starts with a copy of the current messages.
// The new aggregator gets:
// - A new unique ID
// - A copy of all current messages
// - An initLen set to the current message count
// This allows for parallel processing of message streams that can be joined later.
func (a *Aggregator) Fork() *Aggregator {
	return &Aggregator{
		id:       uuid.New(),
		messages: slices.Clone(a.messages),
		initLen:  a.Len(),
	}
}

// Join combines messages from a forked aggregator back into this one.
// It:
//   - Appends only the messages that were added to the forked aggregator after it was forked
//     (determined using b.initLen)
//   - Combines usage statistics from both aggregators
//
// The join operation maintains message order by:
// 1. Keeping all original messages
// 2. Keeping any messages added to this aggregator after the fork
// 3. Appending only new messages from the forked aggregator (those after b.initLen)
//
// Example:
//
//	original := &Aggregator{...}  // Has messages [1,2]
//	forked := original.Fork()     // forked has [1,2] and initLen=2
//	original.Add(msg3)            // original now has [1,2,3]
//	forked.Add(msg4)             // forked now has [1,2,4]
//	original.Join(forked)         // original ends with [1,2,3,4]
func (a *Aggregator) Join(b *Aggregator) {
	// When we join, we want to append only the messages that were added to b
	// after it was forked. b.initLen represents the number of messages b had
	// when it was forked, so any messages after that index are new.
	a.messages = append(a.messages, b.messages[b.initLen:]...)
	a.usage.AddUsage(&b.usage)
}

// Checkpoint creates a snapshot of the current aggregator state.
// This allows saving the current state of messages and usage statistics
// for later reference or restoration. The checkpoint includes:
// - The aggregator's unique ID
// - A deep copy of all current messages
// - The current usage statistics
//
// Example:
//
//	agg := &Aggregator{...}
//	checkpoint := agg.Checkpoint()  // Save current state
//	// ... make changes to agg ...
//	// checkpoint still holds the original state
func (a *Aggregator) Checkpoint() Checkpoint {
	return Checkpoint{
		id:       a.id,
		messages: slices.Clone(a.messages),
		usage:    a.usage,
		initLen:  a.initLen,
	}
}

// Checkpoint represents a snapshot of an aggregator's state at a specific point in time.
// It contains an immutable copy of the aggregator's state, including:
// - The unique identifier of the source aggregator
// - A snapshot of all messages at checkpoint time
// - The usage statistics at checkpoint time
//
// Checkpoints are useful for:
// - Creating save points in long-running operations
// - Comparing states at different points in time
// - Rolling back to previous states if needed
type Checkpoint struct {
	id       uuid.UUID          // Unique identifier for this aggregator
	messages AggregatedMessages // Collection of messages being managed
	usage    Usage              // Usage statistics for token consumption
	initLen  int                // Initial length at fork time, used for joining
}

// ID returns the unique identifier of the aggregator that created this checkpoint.
// This ID matches the source aggregator's ID at the time the checkpoint was created.
func (c *Checkpoint) ID() uuid.UUID {
	return c.id
}

// Messages returns a copy of all messages that were present in the aggregator
// at the time this checkpoint was created. The returned slice is a deep copy,
// so modifications won't affect the checkpoint's stored messages.
func (c *Checkpoint) Messages() AggregatedMessages {
	return slices.Clone(c.messages)
}

// Usage returns the usage statistics that were recorded in the aggregator
// at the time this checkpoint was created. This includes all token counts
// and usage metrics up to the checkpoint time.
func (c *Checkpoint) Usage() Usage {
	return c.usage
}

// MergeInto merges the checkpoint's state into another aggregator.
// This operation:
// - Appends messages from the checkpoint that were added after its fork point
// - Combines the checkpoint's usage statistics with the target aggregator's
//
// This is useful when you want to apply a saved state to a different or
// new aggregator instance.
//
// Example:
//
//	checkpoint := sourceAgg.Checkpoint()
//	targetAgg := NewAggregator()
//	checkpoint.MergeInto(targetAgg)  // targetAgg now contains checkpoint's state
func (c *Checkpoint) MergeInto(other *Aggregator) {
	other.messages = append(other.messages, c.messages[c.initLen:]...)
	other.usage.AddUsage(&c.usage)
	if other.id == uuid.Nil {
		other.id = c.id
	}
}

func (c Checkpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID       string                                     `json:"id"`
		Messages []*messages.Message[messages.ModelMessage] `json:"messages"`
		Usage    Usage                                      `json:"usage"`
		InitLen  int                                        `json:"init_len"`
	}{
		ID:       c.id.String(),
		Messages: ptrSlice(c.messages),
		Usage:    c.usage,
		InitLen:  c.initLen,
	})
}

func ptrSlice[T any](in []T) (out []*T) {
	out = make([]*T, len(in))
	for i, v := range in {
		out[i] = &v
	}
	return
}

func (c *Checkpoint) UnmarshalJSON(data []byte) error {
	var tmp struct {
		ID       string                                    `json:"id"`
		Messages []messages.Message[messages.ModelMessage] `json:"messages"`
		Usage    Usage                                     `json:"usage"`
		InitLen  int                                       `json:"init_len"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if id, err := uuid.Parse(tmp.ID); err != nil {
		return err
	} else {
		c.id = id
	}
	c.messages = tmp.Messages
	c.usage = tmp.Usage
	c.initLen = tmp.InitLen
	return nil
}
