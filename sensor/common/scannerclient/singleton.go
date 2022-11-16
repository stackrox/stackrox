package scannerclient

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once          sync.Once
	scannerClient *Client
)

// GRPCClientSingleton returns a gRPC Client to a local Scanner.
// Only one Client per Sensor is required.
func GRPCClientSingleton() *Client {
	once.Do(func() {
		if !env.LocalImageScanningEnabled.BooleanSetting() {
			log.Info("Local scanning disabled, will not attempt to connect to a local scanner.")
			return
		}

		var err error
		scannerClient, err = dial(env.ScannerSlimGRPCEndpoint.Setting())
		// If err is not nil, then there was a configuration error.
		utils.Should(err)
	})
	return scannerClient
}
