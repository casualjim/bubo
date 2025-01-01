package msgfmt

import (
	"context"
	"fmt"
	"io"
	"strings"

	buboevents "github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/fatih/color"
)

type Formatter interface {
	Format(context.Context, io.Writer, *shorttermmemory.Aggregator) error
}

type FormatterFunc func(context.Context, io.Writer, *shorttermmemory.Aggregator) error

func ConsolePretty(ctx context.Context, w io.Writer, events <-chan buboevents.Event) error {
	return printStreamingMessages(ctx, w, events)
}

func (fn FormatterFunc) Format(ctx context.Context, w io.Writer, s *shorttermmemory.Aggregator) error {
	return fn(ctx, w, s)
}

func printStreamingMessages(ctx context.Context, w io.Writer, events <-chan buboevents.Event) error {
	var content string
	var lastSender string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}

			switch e := event.(type) {
			case buboevents.Delim:
				if e.Delim == "end" && content != "" {
					fmt.Fprintln(w)
					content = ""
				}
			case buboevents.Chunk[messages.AssistantMessage]:
				if e.Sender != "" {
					lastSender = e.Sender
				}

				if e.Chunk.Content.Content != "" {
					if content == "" && lastSender != "" {
						fmt.Fprint(w, color.MagentaString(lastSender)+": ")
						lastSender = ""
					}

					fmt.Fprint(w, e.Chunk.Content.Content)
					content += e.Chunk.Content.Content
				}

			case buboevents.Chunk[messages.ToolCallMessage]:
				if e.Sender != "" {
					lastSender = e.Sender
				}

				if len(e.Chunk.ToolCalls) > 0 {
					for _, tc := range e.Chunk.ToolCalls {
						if tc.Name == "" {
							continue
						}
						args := strings.ReplaceAll(tc.Arguments, ": ", "=")
						fmt.Fprintf(w, "%s%s\n", color.YellowString(tc.Name), args)
					}
				}
			case buboevents.Error:
				fmt.Fprintf(w, "Error: %v\n", e.Err)
			}
		}
	}
}
