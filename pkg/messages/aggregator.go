package messages

import "github.com/google/uuid"

type AggregatedMessages []Message

func (a AggregatedMessages) Len() int {
	return len(a)
}

type Aggregator struct {
	ID       uuid.UUID
	Messages []Message
}
