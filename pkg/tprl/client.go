package tprl

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/casualjim/bubo/pkg/slogx"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
)

func envStrOrDefault(key string, def string) string {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	return s
}

func NewClient() (client.Client, error) {
	lg := slog.Default().With(slogx.LoggerName("bubo.temporal"))

	cl, err := client.NewLazyClient(client.Options{
		HostPort: envStrOrDefault("TEMPORAL_ADDRESS", client.DefaultHostPort),
		Logger:   log.NewStructuredLogger(lg),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}
	return cl, nil
}
