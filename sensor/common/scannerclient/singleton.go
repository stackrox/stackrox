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
		if !env.UseLocalScanner.BooleanSetting() {
			log.Info("No local Scanner connection desired")
			return
		}

		var err error
		scannerClient, err = dial(env.ScannerGRPCEndpoint.Setting())
		// If err is not nil, then there was a configuration error.
		_ = utils.Should(err)
	})
	return scannerClient
}
