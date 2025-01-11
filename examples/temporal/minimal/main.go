package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"time"

	// Ensure API Key is loaded
	_ "github.com/joho/godotenv/autoload"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/agent"
	"github.com/casualjim/bubo/examples/internal/msgfmt"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/pkg/tprl"
	"github.com/casualjim/bubo/provider/openai"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
)

var log zerolog.Logger

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log = zerolog.New(output).With().Timestamp().Logger()
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelWarn}),
	))
}

var minimalAgent = agent.New(agent.Name("minimal-agent"), agent.Model(openai.GPT4oMini()), agent.Instructions("You are a helpful assistant"))

func main() {
	slog.Info("running temporal/minimal example")

	if err := mainE(context.Background()); err != nil {
		slog.Error("failed to run minimal example", slogx.Error(err))
		os.Exit(1)
	}
}

func mainE(ctx context.Context) error {
	tp, err := tprl.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create temporal client: %w", err)
	}

	_, err = tp.CheckHealth(ctx, &client.CheckHealthRequest{})
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}

	u, err := user.Current()
	buboName := "User"
	if err == nil && u != nil {
		buboName = u.Username
	}

	hook, result := msgfmt.Console[string](ctx, os.Stdout)
	msg := "What is the answer to the ultimate question of life, the universe, and everything?"
	prompt := messages.New().WithSender(buboName).UserPrompt(msg)
	p := bubo.New(
		bubo.Agents(minimalAgent),
		bubo.Steps(
			bubo.Step(minimalAgent.Name(), prompt),
		),
	)
	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
		return err
	}

	<-result
	return nil
}
