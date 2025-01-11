package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	// Ensure API Key is loaded
	_ "github.com/joho/godotenv/autoload"

	"github.com/casualjim/bubo/agent"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/examples/internal/repl"
	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/casualjim/bubo/provider/openai"
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

// Refund an item. Refund an item. Make sure you have the item_id of the form item_... Ask for user confirmation before processing the refund.
//
// bubo:agentTool
func processRefund(itemID string, reason string) string {
	if reason == "" {
		reason = "NOT SPECIFIED"
	}

	fmt.Printf("[mock] Refunding item %s for because %s...\n", itemID, reason)
	// Process refund
	return "Success!"
}

// Apply a discount to the user's cart.
//
// bubo:agentTool
func applyDiscount() string {
	fmt.Printf("[mock] Applying discount...\n")
	return "Discount applied!"
}

// Call this function if a user is asking about a topic that is not handled by the current agent.
//
// bubo:agentTool
func transferBackToTriageAgent() api.Agent { return triageAgent }

// Transfer the conversation to the sales agent.
//
// bubo:agentTool
func transferToSales() api.Agent { return salesAgent }

// Transfer the conversation to the refunds agent.
//
// bubo:agentTool
func transferToRefunds() api.Agent { return refundsAgent }

var (
	triageAgent  api.Agent
	salesAgent   api.Agent
	refundsAgent api.Agent
)

func init() {
	triageAgent = agent.New(
		agent.Name("Triage Agent"),
		agent.Model(openai.GPT4oMini()),
		agent.Instructions("Determine which agent is best suited to handle the user's request, and transfer the conversation to that agent."),
		agent.Tools(transferToSalesTool, transferToRefundsTool),
	)

	salesAgent = agent.New(
		agent.Name("Sales Agent"),
		agent.Model(openai.GPT4oMini()),
		agent.Instructions(`Be super enthusiastic about selling bees and beekeeping equipment.
        Handle all sales-related queries including:
        - Live bees
        - Beekeeping kits and equipment
        - Starter packages
        Only transfer back to triage if the query is completely unrelated to sales or purchasing.
        Always try to help customers with their purchase before considering a transfer.`),
		agent.Tools(transferBackToTriageAgentTool),
	)

	refundsAgent = agent.New(
		agent.Name("Refunds Agent"),
		agent.Model(openai.GPT4oMini()),
		agent.Instructions("Help the user with a refund. If the reason is that it was too expensive, offer the user a refund code. If they insist, then process the refund."),
		agent.Tools(processRefundTool, applyDiscountTool, transferBackToTriageAgentTool),
	)
}

func main() {
	slog.Info("running basic/function-calling example")
	ctx := context.Background()

	if err := repl.Run(ctx, triageAgent); err != nil {
		slog.Error("repl.Run", slogx.Error(err))
	}
}
