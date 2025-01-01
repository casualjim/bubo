package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/examples/internal/msgfmt"
	pubsub "github.com/casualjim/bubo/internal/broker"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/provider/openai"
	"github.com/hokaccha/go-prettyjson"
	"github.com/joho/godotenv"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log = zerolog.New(output).With().Timestamp().Logger()
}

func init() {
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelDebug}),
	))
}

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", slogx.Error(err))
	}
	slog.Info("running basic/minimal example")
	ctx := context.Background()

	agent := bubo.NewAgent("minimal-agent", openai.GPT4oMini(), "You are a helpful assistant")
	exec := executor.NewLocal(pubsub.Local[string]())
	memory := shorttermmemory.NewAggregator()
	stream, hook := newChannelHook[string]()

	memory.AddUserPrompt(messages.New().WithSender("user").UserPrompt("Hello, world!"))

	cmd, err := executor.NewRunCommand(agent, memory, hook)
	if err != nil {
		slog.Error("failed to run completion", slogx.Error(err))
	}

	if err := exec.Run(ctx, cmd); err != nil {
		slog.Error("failed to run completion", slogx.Error(err))
	}
	if err := msgfmt.ConsolePretty(ctx, os.Stdout, stream); err != nil {
		slog.Error("failed to format output", slogx.Error(err))
	}

	jb, _ := prettyjson.Marshal(memory)
	fmt.Println(string(jb))
}

func newChannelHook[T any]() (<-chan events.Event, events.Hook[T]) {
	ch := make(chan events.Event, 100)
	return ch, &channelHook[T]{ch: ch}
}

type channelHook[T any] struct {
	ch chan<- events.Event
}

func (c *channelHook[T]) OnUserPrompt(ctx context.Context, msg messages.Message[messages.UserMessage]) {
	slog.InfoContext(ctx, "user prompt", slog.Any("message", msg))
	c.ch <- events.Request[messages.UserMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Message:   msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *channelHook[T]) OnAssistantChunk(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	slog.InfoContext(ctx, "assistant chunk", slog.Any("message", msg))
	c.ch <- events.Chunk[messages.AssistantMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Chunk:     msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *channelHook[T]) OnToolCallChunk(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	slog.InfoContext(ctx, "tool call chunk", slog.Any("message", msg))
	c.ch <- events.Chunk[messages.ToolCallMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Chunk:     msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *channelHook[T]) OnAssistantMessage(ctx context.Context, msg messages.Message[messages.AssistantMessage]) {
	slog.InfoContext(ctx, "assistant message", slog.Any("message", msg))
	c.ch <- events.Response[messages.AssistantMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Response:  msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *channelHook[T]) OnToolCallMessage(ctx context.Context, msg messages.Message[messages.ToolCallMessage]) {
	slog.InfoContext(ctx, "tool call message", slog.Any("message", msg))
	c.ch <- events.Response[messages.ToolCallMessage]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Response:  msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *channelHook[T]) OnToolCallResponse(ctx context.Context, msg messages.Message[messages.ToolResponse]) {
	slog.InfoContext(ctx, "tool call respons", slog.Any("message", msg))
	c.ch <- events.Request[messages.ToolResponse]{
		RunID:     msg.RunID,
		TurnID:    msg.TurnID,
		Message:   msg.Payload,
		Sender:    msg.Sender,
		Timestamp: msg.Timestamp,
		Meta:      msg.Meta,
	}
}

func (c *channelHook[T]) OnResult(ctx context.Context, result T) {
	slog.InfoContext(ctx, "completion result", slog.Any("result", result))
	close(c.ch)
}

func (c *channelHook[T]) OnError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "completion error", slogx.Error(err))
	close(c.ch)
}
