package relay

import (
	"context"
	"net"

	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// Handler defines the interface for handling individual connections
type Handler interface {
	// HandleConnection processes a single connection and returns data
	HandleConnection(ctx context.Context, conn net.Conn) (interface{}, error)
}

// DefaultVsockPort is the default port the vsock service listens on
const DefaultVsockPort uint32 = 1234
