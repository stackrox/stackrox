package scannerclient

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type infoCountingLogger struct {
	logging.Logger
	infoCalls int
}

func (l *infoCountingLogger) Info(_ ...interface{}) {
	l.infoCalls++
}

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

func TestGRPCClientSingletonLogsMissingCentralCapabilityOncePerReset(t *testing.T) {
	t.Setenv(env.LocalImageScanningEnabled.EnvVar(), "true")
	centralcaps.Set(nil)
	t.Cleanup(func() { centralcaps.Set(nil) })

	origScannerV4Enabled := isScannerV4Enabled
	isScannerV4Enabled = true
	t.Cleanup(func() { isScannerV4Enabled = origScannerV4Enabled })

	resetGRPCClient()
	t.Cleanup(resetGRPCClient)

	origLog := log
	countingLog := &infoCountingLogger{Logger: origLog}
	log = countingLog
	t.Cleanup(func() { log = origLog })

	assert.Nil(t, GRPCClientSingleton())
	assert.Nil(t, GRPCClientSingleton())
	require.Equal(t, 1, countingLog.infoCalls)

	resetGRPCClient() // The client is still nil, but the log should be enabled again.
	assert.Nil(t, GRPCClientSingleton())
	assert.Equal(t, 2, countingLog.infoCalls)
}
