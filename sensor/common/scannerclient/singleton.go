package scannerclient

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once          sync.Once
	scannerClient *client
)

// GRPCClientSingleton returns a gRPC client to a local Scanner.
// Only one client per Sensor is required.
func GRPCClientSingleton() *client {
	once.Do(func() {
		var err error
		scannerClient, err = newGRPCClient(env.ScannerEndpoint.Setting())
		_ = utils.Should(err)
	})
	return scannerClient
}
