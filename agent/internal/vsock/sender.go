package vsock

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

// SendIndexReport sends an IndexReport via VSOCK to the host
func SendIndexReport(ctx context.Context, indexReport *v1.IndexReport) error {
	log := logging.LoggerForModule()

	log.Debug("Connecting to VSOCK host...")
	client, err := Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to VSOCK: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Warnf("Failed to close VSOCK client: %v", err)
		}
	}()

	log.Debug("Sending IndexReport via VSOCK...")
	if err := client.SendProtobuf(indexReport); err != nil {
		return fmt.Errorf("failed to send IndexReport: %w", err)
	}

	log.Debug("Successfully sent IndexReport via VSOCK")
	return nil
}
