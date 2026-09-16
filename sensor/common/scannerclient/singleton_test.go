package scannerclient

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
)

func TestGRPCClientSingleton(t *testing.T) {
	tests := map[string]struct {
		localImageScanningEnabled bool
		scannerV4Enabled          bool
		centralCaps               []centralsensor.CentralCapability
	}{
		"local image scanning disabled": {
			localImageScanningEnabled: false,
			scannerV4Enabled:          true,
			centralCaps:               []centralsensor.CentralCapability{centralsensor.ScannerV4Supported},
		},
		"scanner v4 disabled in sensor": {
			localImageScanningEnabled: true,
			scannerV4Enabled:          false,
			centralCaps:               []centralsensor.CentralCapability{centralsensor.ScannerV4Supported},
		},
		"central does not support scanner v4": {
			localImageScanningEnabled: true,
			scannerV4Enabled:          true,
			centralCaps:               nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.LocalImageScanningEnabled.EnvVar(), strconv.FormatBool(tt.localImageScanningEnabled))

			origScannerV4Enabled := isScannerV4Enabled
			isScannerV4Enabled = tt.scannerV4Enabled
			t.Cleanup(func() { isScannerV4Enabled = origScannerV4Enabled })

			centralcaps.Set(tt.centralCaps)
			t.Cleanup(func() { centralcaps.Set(nil) })

			scannerClient = nil
			t.Cleanup(func() { scannerClient = nil })

			assert.Nil(t, GRPCClientSingleton())
		})
	}
}
