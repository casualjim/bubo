package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	// Ensure API Key is loaded
	_ "github.com/joho/godotenv/autoload"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/agent"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/examples/internal/msgfmt"
	"github.com/casualjim/bubo/provider/openai"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log = zerolog.New(output).With().Timestamp().Logger()
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelWarn}),
	))
}

var (
	englishAgent = agent.New(
		agent.Name("English Agent"),
		agent.Model(openai.GPT4oMini()),
		agent.Instructions("You only speak English, so you only reply in english."),
		agent.Tools(transferToSpanishAgentTool),
	)
	spanishAgent = agent.New(
		agent.Name("Spanish Agent"),
		agent.Model(openai.GPT4oMini()),
		agent.Instructions("You only speak Spanish, so you only reply in spanish."),
	)
)

// Transfer spanish speaking users immediately
//
// bubo:agentTool
func transferToSpanishAgent() api.Agent { return spanishAgent }

func main() {
	slog.Info("running basic/function-calling example")
	ctx := context.Background()

	hook, result := msgfmt.Console[string](ctx, os.Stdout)

	p := bubo.New(
		bubo.Agents(englishAgent),
		bubo.Steps(
			bubo.Step(englishAgent.Name(), "Hola. ¿Como estás?"),
		),
	)

	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
		slog.Error("error running agent", "error", err)
		return
	}

	<-result
}
