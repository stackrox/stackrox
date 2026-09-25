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
	loggedNoCentralV4  bool

	isScannerV4Enabled = features.ScannerV4.Enabled()
)

// GRPCClientSingleton returns a gRPC ScannerClient to a local Scanner.
// Only one ScannerClient per Sensor is required. A nil client may
// be returned.
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

	// Scanner V4 is the only local scanner Sensor connects to. Using it requires
	// both Scanner V4 to be installed locally and Central to advertise support
	// for Scanner V4.
	// When either is missing there is no local scanner to talk to, so we return a nil
	// client.
	if !isScannerV4Enabled {
		log.Warn("Local image scanning is enabled but Scanner V4 is disabled in Sensor; no local scanner will be used")
		return nil
	}
	if !centralcaps.Has(centralsensor.ScannerV4Supported) {
		if !loggedNoCentralV4 {
			log.Info("Local image scanning is enabled but Central does not support Scanner V4; no local scanner will be used")
			loggedNoCentralV4 = true
		}
		return nil
	}

	log.Info("Creating Scanner V4 client")
	var err error
	scannerClient, err = dialV4()
	utils.Should(err)

	return scannerClient
}

// resetGRPCClient resets the current scanner client so that it will be recreated
// on next retrieval.
func resetGRPCClient() {
	scannerClientMutex.Lock()
	defer scannerClientMutex.Unlock()

	loggedNoCentralV4 = false
	if scannerClient == nil {
		return
	}

	utils.Should(scannerClient.Close())
	scannerClient = nil
}
