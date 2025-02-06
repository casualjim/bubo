package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	// Ensure API Key is loaded
	_ "github.com/casualjim/bubo/provider/openai"
	_ "github.com/joho/godotenv/autoload"

	"github.com/casualjim/bubo/internal/broker"
	"github.com/casualjim/bubo/internal/executor"
	"github.com/casualjim/bubo/pkg/natsx"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/pkg/tprl"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
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

	nt, err := natsx.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create nats client: %w", err)
	}

	tpa := executor.NewTemporalAgentWorker(tp, broker.NATS(nt))

	if err := tpa.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("failed to run temporal agent worker: %w", err)
	}
	return nil
}
