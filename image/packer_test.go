package image

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoChartPanic(t *testing.T) {
	// Verify at runtime that this won't panic
	GetCentralChart(nil)
	GetScannerChart()
}

var (
	nameRegexp = regexp.MustCompile(`(?: +|\t)name: (.*)`)
)

func TestSensorTLSGVKs(t *testing.T) {
	var actualNames []string
	for _, fileName := range K8sBox.List() {
		if !strings.HasPrefix(fileName, sensorChartPrefix) {
			continue
		}
		base := filepath.Base(fileName)
		if !SensorMTLSFiles.Contains(base) {
			continue
		}
		contents, err := K8sBox.Find(fileName)
		require.NoError(t, err)

		match := nameRegexp.FindSubmatch(contents)
		if len(match) < 2 {
			t.Fatalf("Contents %s didn't match name regexp", string(contents))
		}
		actualNames = append(actualNames, string(match[1]))
	}
	namesFromConst := make([]string, 0, len(SensorCertObjectRefs))
	for ref := range SensorCertObjectRefs {
		namesFromConst = append(namesFromConst, ref.Name)
	}
	assert.ElementsMatch(t, actualNames, namesFromConst)
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
			assert.ElementsMatchf(t, actualFilesWithSecrets, c.files.AsSlice(),
				"If you have added or removed a new TLS secret to %s, please update the relevant constant", c.chartPrefix)
		})
	}

}
