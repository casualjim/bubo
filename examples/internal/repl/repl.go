package repl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
)

type noopPromise struct{}

func (noopPromise) Complete(string) {}
func (noopPromise) Error(error)     {}

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

func Run(ctx context.Context, startingOwl api.Owl) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)
	agent := startingOwl
	history := shorttermmemory.New()

	for {
		fmt.Printf("%s: ", color.CyanString("User"))
		if !scanner.Scan() {
			fmt.Println("Exiting...")
			break
		}

		input := scanner.Text()
		if strings.EqualFold(input, "exit") {
			break
		}

		finished, hook := newConsoleHook[string]()
		cmd, err := executor.NewRunCommand(agent, history, hook)
		if err != nil {
			return err
		}

		umsg := messages.New().WithSender("User").UserPrompt(input)
		history.AddUserPrompt(umsg)
		hook.OnUserPrompt(ctx, umsg)
		exec := executor.NewLocal()

		go func() {
			defer hook.OnClose(ctx)
			if err := exec.Run(ctx, cmd.WithStream(true), noopPromise{}); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				return
			}
		}()

		var content string
		var streaming bool
		var lastSender string
		for msg := range finished {
			switch m := msg.(type) {
			case events.Request[messages.UserMessage]:
				fmt.Fprintln(os.Stdout)
			case events.Chunk[messages.AssistantMessage]:
				if !streaming {
					streaming = true
					fmt.Fprintln(os.Stdout)
				}
				if m.Sender != "" {
					lastSender = m.Sender
				}

				if m.Chunk.Content.Content != "" {
					if content == "" && lastSender != "" {
						fmt.Fprint(os.Stdout, color.MagentaString(lastSender)+": ")
						lastSender = ""
					}

					fmt.Fprint(os.Stdout, m.Chunk.Content.Content)
					content += m.Chunk.Content.Content
				}
			case events.Chunk[messages.ToolCallMessage]:
				if !streaming {
					streaming = true
				}
				if m.Sender != "" {
					lastSender = m.Sender
				}

				if len(m.Chunk.ToolCalls) > 0 {
					for _, tc := range m.Chunk.ToolCalls {
						if tc.Name == "" {
							continue
						}
						args := strings.ReplaceAll(tc.Arguments, ": ", "=")
						fmt.Fprintf(os.Stdout, "%s%s\n", color.YellowString(tc.Name), args)
					}
				}
			case events.Response[messages.ToolCallMessage]:
				if streaming || len(content) > 0 {
					content = ""
					fmt.Fprintln(os.Stdout)
					streaming = false
					continue
				}
				if m.Sender == "" {
					fmt.Fprint(os.Stdout, color.YellowString("Tool")+": ")
				} else {
					fmt.Fprint(os.Stdout, color.YellowString(m.Sender)+": ")
				}
				if len(m.Response.ToolCalls) > 1 {
					fmt.Fprintln(os.Stdout)
				}

				for tc := range slices.Values(m.Response.ToolCalls) {
					args := strings.ReplaceAll(tc.Arguments, ": ", "=")
					fmt.Fprintf(os.Stdout, "%s%s\n", color.YellowString(tc.Name), args)
				}
			case events.Response[messages.AssistantMessage]:
				if streaming || len(content) > 0 {
					content = ""
					fmt.Fprintln(os.Stdout)
					streaming = false
					continue
				}
				if m.Sender == "" {
					fmt.Fprint(os.Stdout, color.MagentaString("Assistant")+": ")
				} else {
					fmt.Fprint(os.Stdout, color.MagentaString(m.Sender)+": ")
				}
				out, _ := glam.Render(m.Response.Content.Content)
				fmt.Fprintln(os.Stdout, out)
			case events.Request[messages.ToolResponse]:
				if m.Sender == "" {
					fmt.Fprint(os.Stdout, color.YellowString("Tool")+": ")
				} else {
					fmt.Fprint(os.Stdout, color.YellowString(m.Sender)+": ")
				}
				fmt.Fprint(os.Stdout, m.Message.Content)
			case events.Error:
				fmt.Fprintf(os.Stdout, "Error: %v\n", m.Err)
			default:
				fmt.Fprintf(os.Stdout, "unknown message type: %T\n", m)
			}
		}
		fmt.Fprintln(os.Stdout)
	}
	return nil
}

func newConsoleHook[T any]() (chan events.Event, bubo.Hook[T]) {
	ch := make(chan events.Event, 100)
	return ch, &consoleHook[T]{ch: ch}
}

type consoleHook[T any] struct {
	ch chan<- events.Event
}

func (c *consoleHook[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	c.ch <- events.Request[messages.UserMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Message:   msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	c.ch <- events.Chunk[messages.AssistantMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Chunk:     msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	c.ch <- events.Chunk[messages.ToolCallMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Chunk:     msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	c.ch <- events.Response[messages.AssistantMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Response:  msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	c.ch <- events.Response[messages.ToolCallMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Response:  msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	c.ch <- events.Request[messages.ToolResponse]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Message:   msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *consoleHook[T]) OnResult(ctx context.Context, result T) {
	c.ch <- events.Result[T]{Result: result}
}

func (c *consoleHook[T]) OnError(ctx context.Context, err error) {
	c.ch <- events.Error{Err: err}
}

func (c *consoleHook[T]) OnClose(ctx context.Context) {
	close(c.ch)
}
