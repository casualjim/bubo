package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	// Ensure API Key is loaded
	_ "github.com/joho/godotenv/autoload"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/examples/internal/msgfmt"
	"github.com/casualjim/bubo/owl"
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
	englishOwl = owl.New(
		owl.Name("English Owl"),
		owl.Model(openai.GPT4oMini()),
		owl.Instructions("You only speak English, so you only reply in english."),
		owl.Tools(transferToSpanishAgentTool),
	)
	spanishOwl = owl.New(
		owl.Name("Spanish Owl"),
		owl.Model(openai.GPT4oMini()),
		owl.Instructions("You only speak Spanish, so you only reply in spanish."),
	)
)

// Transfer spanish speaking users immediately
//
// bubo:agentTool
func transferToSpanishAgent() api.Owl { return spanishOwl }

func main() {
	slog.Info("running basic/function-calling example")
	ctx := context.Background()

	hook, result := msgfmt.Console[string](ctx, os.Stdout)

	p := bubo.New(
		bubo.Owls(englishOwl),
		bubo.Steps(
			bubo.Step(englishOwl.Name(), "Hola. ¿Como estás?"),
		),
	)

	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
		slog.Error("error running agent", "error", err)
		return
	}

	<-result
}
