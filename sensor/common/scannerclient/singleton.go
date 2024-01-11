package scannerclient

import (
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/centralcaps"
)

var (
	scannerClient        ScannerClient
	scannerClientOnce    sync.Once
	scannerClientRWMutex sync.RWMutex

	isScannerV4Enabled = features.ScannerV4Enabled.Enabled()
)

// GRPCClientSingleton returns a gRPC ScannerClient to a local Scanner.
// Only one ScannerClient per Sensor is required.
func GRPCClientSingleton() ScannerClient {
	scannerClientRWMutex.RLock()
	defer scannerClientRWMutex.RUnlock()

	scannerClientOnce.Do(func() {
		if !env.LocalImageScanningEnabled.BooleanSetting() {
			log.Infof("scanner disabled: %s is false, will not attempt to connect to a local scanner",
				env.LocalImageScanningEnabled.EnvVar())
			return
		}
		var err error
		if isScannerV4Enabled && centralcaps.Has(centralsensor.ScannerV4Supported) {
			log.Info("Creating Scanner V4 client")
			scannerClient, err = dialV4()
		} else {
			log.Info("Creating Scanner V2 client")
			scannerClient, err = dialV2()
		}
		utils.Should(err)
	})
	return scannerClient
}

// Reset will close the current scanner client and setup the singleton so that the
// next invocation triggers a recreate of the client.
func resetGRPCClient() {
	scannerClientRWMutex.Lock()
	defer scannerClientRWMutex.Unlock()

	if scannerClient != nil {
		utils.Should(scannerClient.Close())
	}
	scannerClientOnce = sync.Once{}
}
