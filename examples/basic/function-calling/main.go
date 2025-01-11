package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	// Ensure API Key is loaded
	_ "github.com/joho/godotenv/autoload"

	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/agent"
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
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelDebug}),
	))
}

// getWeather returns the weather for a location at a given time.
//
//bubo:agentTool
func getWeather(location string, date string) (string, error) {
	return `{"temp":67,"unit":"F"}`, nil
}

var weatherAgent = agent.New(
	agent.Name("simple-weather-agent"),
	agent.Model(openai.GPT4oMini()),
	agent.Instructions("You are a helpful agent, always call the tool when you need to get the weather."),
	agent.Tools(getWeatherTool),
)

func main() {
	slog.Info("running basic/function-calling example")
	ctx := context.Background()

	hook, result := msgfmt.Console[string](ctx, os.Stdout)

	p := bubo.New(
		bubo.Agents(weatherAgent),
		bubo.Steps(
			bubo.Step(weatherAgent.Name(), "What's the weather in NYC?"),
		),
	)

	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
		slog.Error("error running agent", "error", err)
		return
	}

	fmt.Println(<-result)
}
