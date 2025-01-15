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
	"github.com/casualjim/bubo/examples/internal/msgfmt"
	"github.com/casualjim/bubo/provider/openai"
	"github.com/k0kubun/pp/v3"
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

// Get the current weather in a given location. Location MUST be a city.
//
// bubo:agentTool
func getWeather(location string) string {
	return "Sunny, 25°C"
}

var weatherAgent = agent.New(
	agent.Name("simple-weather-agent"),
	agent.Model(openai.GPT4oMini()),
	agent.Instructions("You are a helpful agent, always call the tool when you need to get the weather."),
	agent.Tools(getWeatherTool),
)

func main() {
	slog.Info("running basic/structured-output example")
	ctx := context.Background()

	hook, result := msgfmt.Console[Response](ctx, os.Stdout)

	p := bubo.New(
		bubo.Agents(weatherAgent),
		bubo.Steps(
			bubo.Step(weatherAgent.Name(), "Begin a very brief introduction of Greece, then incorporate the local weather of a few towns"),
		),
	)

	if err := p.Run(ctx, bubo.Local(hook)); err != nil {
		slog.Error("error running agent", "error", err)
		return
	}

	pp.Println(<-result)
}

type WeatherResponse struct {
	City string `json:"city"`
	Temp int    `json:"temp"`
	Unit string `json:"unit"`
}

type Response struct {
	Weather []WeatherResponse `json:"weather"`
	Story   string            `json:"story"`
}
