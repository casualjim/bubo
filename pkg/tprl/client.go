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

// NewClient creates a new Temporal client with the specified options.
// It initializes a logger and uses it to create a lazy Temporal client.
// The Temporal server address is obtained from the environment variable
// "TEMPORAL_ADDRESS", or defaults to the client's default host and port.
// Returns the created Temporal client or an error if the client creation fails.
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
