package image

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoChartPanic(t *testing.T) {
	// Verify at runtime that this won't panic
	GetCentralChart(nil)
	GetMonitoringChart()
	GetScannerChart()
}

func TestTLSSecretFiles(t *testing.T) {
	for _, chartAndFiles := range []struct {
		chartPrefix     string
		files           set.FrozenStringSet
		knownExceptions set.FrozenStringSet
	}{
		{chartPrefix: sensorChartPrefix, files: SensorMTLSFiles},
		{chartPrefix: centralChartPrefix, files: CentralMTLSFiles, knownExceptions: set.NewFrozenStringSet("default-tls-cert-secret.yaml")},
		{chartPrefix: scannerChartPrefix, files: ScannerMTLSFiles},
	} {
		c := chartAndFiles
		t.Run(c.chartPrefix, func(t *testing.T) {
			var actualFilesWithSecrets []string
			for _, fileName := range K8sBox.List() {
				base := filepath.Base(fileName)
				if !strings.HasPrefix(fileName, c.chartPrefix) {
					continue
				}
				if c.knownExceptions.Contains(base) {
					continue
				}
				contents, err := K8sBox.Find(fileName)
				require.NoError(t, err)
				contentsStr := string(contents)
				if strings.Contains(contentsStr, "-tls") && strings.Contains(contentsStr, "kind: Secret") {
					actualFilesWithSecrets = append(actualFilesWithSecrets, base)
				}
			}
			assert.ElementsMatch(t, actualFilesWithSecrets, c.files.AsSlice(),
				"If you have added or removed a new TLS secret to %s, please update the relevant constant")
		})
	}

}
