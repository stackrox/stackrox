package scannerclient

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once          sync.Once
	scannerClient ScannerClient

	isScannerV4Enabled = features.ScannerV4.Enabled()
)

// GRPCClientSingleton returns a gRPC ScannerClient to a local Scanner.
// Only one ScannerClient per Sensor is required.
func GRPCClientSingleton() ScannerClient {
	once.Do(func() {
		if !env.LocalImageScanningEnabled.BooleanSetting() {
			log.Infof("scanner disabled: %s is false, will not attempt to connect to a local scanner",
				env.LocalImageScanningEnabled.EnvVar())
			return
		}
		endpoint, err := getScannerEndpoint()
		utils.Should(err)
		if err != nil {
			return
		}
		if isScannerV4Enabled {
			scannerClient, err = dialV4(endpoint)
		} else {
			scannerClient, err = dialV2(endpoint)
		}
		utils.Should(err)
	})
	return scannerClient
}
