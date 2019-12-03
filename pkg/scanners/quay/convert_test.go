package quay

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
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
	protoScan := &storage.ImageScan{
		Components: protoComponents,
	}
	return quayScan, protoScan, image
}

func TestConvertScanToImageScan(t *testing.T) {
	quayScan, protoScan, image := getTestScan()
	assert.Equal(t, protoScan, convertScanToImageScan(image, quayScan))
}
