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
	"github.com/casualjim/bubo/types"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log = zerolog.New(output).With().Timestamp().Logger()
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelError}),
	))
}

// Print account details for a user.
//
//bubo:agentTool
func printAccountDetails(ctx types.ContextVars) string {
	userID := ctx["user_id"].(int)
	name := ctx["name"].(string)
	fmt.Printf("Account Details: %s %d\n", name, userID)
	return "Success"
}

var accountDetailsAgent = agent.New(
	agent.Name("account-details"),
	agent.Model(openai.GPT4oMini()),
	agent.Instructions("You are a helpful agent. Greet the user by name ({{.name}})."),
	agent.Tools(printAccountDetailsTool),
)

func main() {
	slog.Info("running basic/context-variables example")

	ctx := context.Background()
	contextVars := map[string]any{"user_id": 123, "name": "James"}

	hook, result := msgfmt.Console[string](ctx, os.Stdout)

	p := bubo.New(
		bubo.Agents(accountDetailsAgent),
		bubo.Steps(
			bubo.Step(accountDetailsAgent.Name(), "Hi!"),
			bubo.Step(accountDetailsAgent.Name(), "Print my account details"),
		),
	)

	if err := p.Run(ctx, bubo.Local(hook, bubo.WithContextVars(contextVars))); err != nil {
		slog.Error("error running agent", "error", err)
		return
	}

	<-result
}
