package scannerclient

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once          sync.Once
	scannerClient ScannerClient

	isScannerV4Enabled = env.ScannerV4Enabled.BooleanSetting()
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
		var err error
		if isScannerV4Enabled {
			scannerClient, err = dialV4()
		} else {
			scannerClient, err = dialV2()
		}
		utils.Should(err)
	})
	return scannerClient
}
