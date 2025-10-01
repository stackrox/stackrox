package enricher

import (
	"context"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vmEnricherOnce     sync.Once
	vmEnricherInstance VirtualMachineEnricher
	vmLog              = logging.LoggerForModule()
)

func Singleton() VirtualMachineEnricher {
	vmEnricherOnce.Do(func() {
		scannerClient, err := createScannerV4Client()
		if err != nil {
			vmLog.Errorf("Failed to create Scanner V4 client for VM enricher: %v", err)
			// Return enricher with nil client - it will handle errors gracefully
			vmEnricherInstance = New(nil)
			return
		}
		vmEnricherInstance = New(scannerClient)
	})
	return vmEnricherInstance
}

func createScannerV4Client() (client.Scanner, error) {
	// Use same defaults as Scanner V4 node scanner but namespace-independent
	indexerEndpoint := scannerv4.DefaultIndexerEndpoint
	matcherEndpoint := scannerv4.DefaultMatcherEndpoint

	ctx := context.Background()
	return client.NewGRPCScanner(ctx,
		client.WithIndexerAddress(indexerEndpoint),
		client.WithMatcherAddress(matcherEndpoint),
	)
}
