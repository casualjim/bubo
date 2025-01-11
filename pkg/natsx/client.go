package natsx

import (
	"os"

	"github.com/nats-io/nats.go"
)

// NewClient creates a new connection to a NATS server using the URL specified
// in the NATS_URL environment variable. The connection is configured with a
// client name "bubo" and compression enabled.
//
// Returns:
//   - *nats.Conn: A pointer to the established NATS connection.
//   - error: An error if the connection could not be established.
func NewClient(opts ...nats.Option) (*nats.Conn, error) {
	if len(opts) == 0 {
		opts = append(opts, nats.Name("bubo"), nats.Compression(true))
	}
	return nats.Connect(os.Getenv("NATS_URL"), opts...)
}
