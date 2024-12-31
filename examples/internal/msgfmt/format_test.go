package msgfmt

import (
	"context"
	"strings"
	"testing"

	"github.com/casualjim/bubo/executor/pubsub"
	"github.com/casualjim/bubo/pkg/messages"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsolePrettyStreaming(t *testing.T) {
	ctx := context.Background()
	events := make(chan pubsub.Event)

	// Start streaming in a goroutine
	go func() {
		defer close(events)

		// Send some test events
		events <- pubsub.Chunk[messages.AssistantMessage]{
			RunID: uuid.New(),
			Chunk: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "test streaming message",
				},
			},
			Sender: "assistant",
		}

		events <- pubsub.Chunk[messages.ToolCallMessage]{
			RunID: uuid.New(),
			Chunk: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "test-id",
						Name:      "test_tool",
						Arguments: `{"arg": "value"}`,
					},
				},
			},
			Sender: "tool",
		}

		events <- pubsub.Error{
			RunID:  uuid.New(),
			Err:    assert.AnError,
			Sender: "system",
		}

		events <- pubsub.Delim{
			RunID: uuid.New(),
			Delim: "end",
		}
	}()

	var buf strings.Builder
	err := ConsolePretty(ctx, &buf, events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, color.MagentaString("assistant")+": test streaming message")
	assert.Contains(t, output, color.YellowString("test_tool")+"{\"arg\"=\"value\"}")
	assert.Contains(t, output, "Error: assert.AnError")
}
