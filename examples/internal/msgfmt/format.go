package msgfmt

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/casualjim/bubo"
	buboevents "github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
)

type Formatter interface {
	Format(context.Context, io.Writer, *shorttermmemory.Aggregator) error
}

type FormatterFunc func(context.Context, io.Writer, *shorttermmemory.Aggregator) error

var glam *glamour.TermRenderer

func init() {
	var err error
	glam, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	if err != nil {
		panic(err)
	}
}

func ConsolePretty[T any](ctx context.Context, w io.Writer, events <-chan buboevents.Event) error {
	return printStreamingMessages[T](ctx, w, events, make(chan T))
}

func Console[T any](ctx context.Context, w io.Writer) (bubo.Hook[T], <-chan T) {
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
		case _, ok := <-doneC:
			if !ok {
				return nil
			}
		case event, ok := <-events:

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
			case buboevents.Response[messages.AssistantMessage]:
				if e.Sender == "" {
					fmt.Fprint(w, color.MagentaString("Assistant")+": ")
				} else {
					fmt.Fprint(w, color.MagentaString(e.Sender)+": ")
				}
				out, _ := glam.Render(e.Response.Content.Content)
				fmt.Fprintln(w, out)
			case buboevents.Response[messages.ToolCallMessage]:
				if e.Sender == "" {
					fmt.Fprint(w, color.YellowString("Tool")+": ")
				} else {
					fmt.Fprint(w, color.YellowString(e.Sender)+": ")
				}
				if len(e.Response.ToolCalls) > 1 {
					fmt.Fprintln(w)
				}

				for tc := range slices.Values(e.Response.ToolCalls) {
					args := strings.ReplaceAll(tc.Arguments, ": ", "=")
					fmt.Fprintf(w, "%s%s\n", color.YellowString(tc.Name), args)
				}
			case buboevents.Result[T]:
				doneC <- e.Result
			case buboevents.Error:
				fmt.Fprintf(w, "Error: %v\n", e.Err)
			}

			if !ok {
				close(doneC)
				return nil
			}
		}
	}
}

func newConsoleHook[T any]() (<-chan buboevents.Event, bubo.Hook[T]) {
	ch := make(chan buboevents.Event, 100)
	return ch, &consoleHook[T]{ch: ch}
}

type consoleHook[T any] struct {
	ch chan<- buboevents.Event
}

func (c *consoleHook[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	slog.InfoContext(ctx, "user prompt", slog.Any("msg", msg))
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
	slog.InfoContext(ctx, "assistant chunk", slog.Any("msg", msg))
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
	slog.InfoContext(ctx, "tool call chunk", slog.Any("msg", msg))
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
	slog.Info("assistant message", slog.Any("msg", msg))
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
	slog.InfoContext(ctx, "tool call message", slog.Any("msg", msg))
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
	slog.InfoContext(ctx, "tool call response", slog.Any("msg", msg))
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
}

func (c *consoleHook[T]) OnError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "completion error", slogx.Error(err))
	c.ch <- buboevents.Error{Err: err}
}

func (c *consoleHook[T]) OnClose(ctx context.Context) {
	slog.InfoContext(ctx, "completion closed")
	close(c.ch)
}
