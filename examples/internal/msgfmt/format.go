package msgfmt

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	buboevents "github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/fatih/color"
)

type Formatter interface {
	Format(context.Context, io.Writer, *shorttermmemory.Aggregator) error
}

type FormatterFunc func(context.Context, io.Writer, *shorttermmemory.Aggregator) error

func ConsolePretty[T any](ctx context.Context, w io.Writer, events <-chan buboevents.Event) error {
	return printStreamingMessages[T](ctx, w, events, make(chan T))
}

func Console[T any](ctx context.Context, w io.Writer) (buboevents.Hook, <-chan T) {
	ch, hook := newConsoleHook[T]()
	doneC := make(chan T)
	go func() {
		if err := printStreamingMessages[T](ctx, w, ch, doneC); err != nil {
			slog.ErrorContext(ctx, "failed to format output", slogx.Error(err))
		}
	}()
	return hook, doneC
}

func (fn FormatterFunc) Format(ctx context.Context, w io.Writer, s *shorttermmemory.Aggregator) error {
	return fn(ctx, w, s)
}

func printStreamingMessages[T any](ctx context.Context, w io.Writer, events <-chan buboevents.Event, doneC chan T) error {
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
				close(doneC)
			}
		}
	}
}

func newConsoleHook[T any]() (<-chan buboevents.Event, buboevents.Hook) {
	ch := make(chan buboevents.Event, 100)
	return ch, &consoleHook[T]{ch: ch}
}

type consoleHook[T any] struct {
	ch chan<- buboevents.Event
}

func (c *consoleHook[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	slog.InfoContext(ctx, "user prompt", slog.Any("message", msg))
	c.ch <- buboevents.Request[messages.UserMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Message:   msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	slog.InfoContext(ctx, "assistant chunk", slog.Any("message", msg))
	c.ch <- buboevents.Chunk[messages.AssistantMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Chunk:     msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	slog.InfoContext(ctx, "tool call chunk", slog.Any("message", msg))
	c.ch <- buboevents.Chunk[messages.ToolCallMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Chunk:     msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	slog.InfoContext(ctx, "assistant message", slog.Any("message", msg))
	c.ch <- buboevents.Response[messages.AssistantMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Response:  msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	slog.InfoContext(ctx, "tool call message", slog.Any("message", msg))
	c.ch <- buboevents.Response[messages.ToolCallMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Response:  msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	slog.InfoContext(ctx, "tool call respons", slog.Any("message", msg))
	c.ch <- buboevents.Request[messages.ToolResponse]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Message:   msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnResult(ctx context.Context, result T) {
	slog.InfoContext(ctx, "completion result", slog.Any("result", result))
	c.ch <- buboevents.Result[T]{Result: result}
	close(c.ch)
}

func (c *consoleHook[T]) OnError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "completion error", slogx.Error(err))
	c.ch <- buboevents.Error{Err: err}
	close(c.ch)
}
