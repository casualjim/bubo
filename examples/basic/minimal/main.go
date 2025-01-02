package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/examples/internal/msgfmt"
	"github.com/casualjim/bubo/owl"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/provider/openai"
	_ "github.com/joho/godotenv/autoload"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log = zerolog.New(output).With().Timestamp().Logger()
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelDebug}),
	))
}

func main() {
	slog.Info("running basic/minimal example")
	ctx := context.Background()

	hook, _ := msgfmt.Console[string](ctx, os.Stdout)
	p := bubo.New(
		bubo.WithOwls(
			owl.New(owl.Name("minimal-agent"), owl.Model(openai.GPT4oMini()), owl.Instructions("You are a helpful assistant")),
		),
	)
	fut, config := bubo.Local(hook)
	if err := p.Run(ctx, "Hello, world!", config); err != nil {
		slog.Error("failed to run bubo", slogx.Error(err))
		return
	}

	success, err := fut.Get()
	if err != nil {
		slog.Error("failed to get result", slogx.Error(err))
		return
	}
	fmt.Println(success)
}
