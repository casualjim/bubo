package msgfmt

import (
	"context"
	"strings"
	"testing"

	buboevents "github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/messages"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsolePrettyStreaming(t *testing.T) {
	ctx := context.Background()
	events := make(chan buboevents.Event)

	// Start streaming in a goroutine
	go func() {
		defer close(events)

		// Send some test events
		events <- buboevents.Chunk[messages.AssistantMessage]{
			RunID: uuid.New(),
			Chunk: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "test streaming message",
				},
			},
			Sender: "assistant",
		}

		events <- buboevents.Chunk[messages.ToolCallMessage]{
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

		events <- buboevents.Error{
			RunID:  uuid.New(),
			Err:    assert.AnError,
			Sender: "system",
		}

		events <- buboevents.Delim{
			RunID: uuid.New(),
			Delim: "end",
		}
	}()

	var buf strings.Builder
	err := ConsolePretty[any](ctx, &buf, events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, color.MagentaString("assistant")+": test streaming message")
	assert.Contains(t, output, color.YellowString("test_tool")+"{\"arg\"=\"value\"}")
	assert.Contains(t, output, "Error: assert.AnError")
}
