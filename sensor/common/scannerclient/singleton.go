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
	scannerClient      ScannerClient
	scannerClientMutex sync.Mutex

	isScannerV4Enabled = features.ScannerV4.Enabled()
)

// GRPCClientSingleton returns a gRPC ScannerClient to a local Scanner.
// Only one ScannerClient per Sensor is required.
func GRPCClientSingleton() ScannerClient {
	scannerClientMutex.Lock()
	defer scannerClientMutex.Unlock()

	if scannerClient != nil {
		return scannerClient
	}

	if !env.LocalImageScanningEnabled.BooleanSetting() {
		log.Infof("scanner disabled: %s is false, will not attempt to connect to a local scanner",
			env.LocalImageScanningEnabled.EnvVar())
		return nil
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

	return scannerClient
}

// resetGRPCClient resets the current scanner client so that it will be recreated
// on next retrieval.
func resetGRPCClient() {
	scannerClientMutex.Lock()
	defer scannerClientMutex.Unlock()

	if scannerClient == nil {
		return
	}

	utils.Should(scannerClient.Close())
	scannerClient = nil
}
