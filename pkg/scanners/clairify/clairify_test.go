package clairify

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/features"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stretchr/testify/assert"
)

func getTestScan() (*clairV1.LayerEnvelope, *storage.ImageScan, *storage.Image) {
	scannerVersion := "2.22.0"

	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}
	clairFeatures, protoComponents := mock.GetTestFeatures()

	env := clairV1.LayerEnvelope{
		Layer: &clairV1.Layer{
			NamespaceName: "debian:8",
			Features:      clairFeatures,
		},
		ScannerVersion: scannerVersion,
	}

	protoScan := &storage.ImageScan{
		Components:      protoComponents,
		ScannerVersion:  scannerVersion,
		OperatingSystem: "debian:8",
		Notes: []storage.ImageScan_Note{
			storage.ImageScan_OS_CVES_STALE,
		},
	}
	return &env, protoScan, image
}

func TestConvertLayerToImageScan(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	layer, protoScan, image := getTestScan()
	actualScan := convertLayerToImageScan(image, layer)
	// Ignore Scan time in the test, as it is defined as the time we retrieve the scan.
	assert.Equal(t, protoScan.DataSource, actualScan.DataSource)
	assert.Equal(t, "debian:8", actualScan.OperatingSystem)
	assert.Equal(t, protoScan.Components, actualScan.Components)
	assert.Equal(t, protoScan.ScannerVersion, actualScan.ScannerVersion)
	assert.Len(t, protoScan.Notes, 1)
	assert.Contains(t, protoScan.Notes, convertNote(clairV1.OSCVEsStale))
}
