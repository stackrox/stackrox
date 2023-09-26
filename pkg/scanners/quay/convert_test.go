package quay

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/features"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stretchr/testify/assert"
)

func getTestScan() (*scanResult, *storage.ImageScan, *storage.Image) {
	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}
	quayFeatures, protoComponents := mock.GetTestFeatures()

	quayScan := &scanResult{
		Status: "scanned",
		Data: clairV1.LayerEnvelope{
			Layer: &clairV1.Layer{
				Features: quayFeatures,
			},
		},
	}
	// Leaving OperatingSystem blank, so we can make sure it says 'unknown'
	protoScan := &storage.ImageScan{
		Components: protoComponents,
	}
	return quayScan, protoScan, image
}

func TestConvertScanToImageScan(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	quayScan, protoScan, image := getTestScan()
	actualScan := convertScanToImageScan(image, quayScan)
	// Ignore Scan time in the test, as it is defined as the time we retrieve the scan.
	assert.Equal(t, protoScan.DataSource, actualScan.DataSource)
	assert.Equal(t, "unknown", actualScan.OperatingSystem)
	assert.Equal(t, protoScan.Components, actualScan.Components)
}
