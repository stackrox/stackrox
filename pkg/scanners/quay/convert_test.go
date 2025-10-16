package quay

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stretchr/testify/assert"
)

func getTestScan() (*scanResult, *storage.ImageScan, *storage.Image) {
	imageName := &storage.ImageName{}
	imageName.SetRegistry("quay.io")
	imageName.SetRemote("integration/nginx")
	imageName.SetTag("1.10")
	image := &storage.Image{}
	image.SetName(imageName)
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
	protoScan := &storage.ImageScan{}
	protoScan.SetComponents(protoComponents)
	return quayScan, protoScan, image
}

func TestConvertScanToImageScan(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	quayScan, protoScan, image := getTestScan()
	actualScan := convertScanToImageScan(image, quayScan)
	// Ignore Scan time in the test, as it is defined as the time we retrieve the scan.
	protoassert.Equal(t, protoScan.GetDataSource(), actualScan.GetDataSource())
	assert.Equal(t, "unknown", actualScan.GetOperatingSystem())
	protoassert.SlicesEqual(t, protoScan.GetComponents(), actualScan.GetComponents())
}
