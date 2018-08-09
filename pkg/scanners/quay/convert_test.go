package quay

import (
	"testing"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stretchr/testify/assert"
)

func getTestScan() (*scanResult, *v1.ImageScan, *v1.Image) {
	image := &v1.Image{
		Name: &v1.ImageName{
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
	protoScan := &v1.ImageScan{
		Components: protoComponents,
	}
	return quayScan, protoScan, image
}

func TestConvertScanToImageScan(t *testing.T) {
	quayScan, protoScan, image := getTestScan()
	assert.Equal(t, protoScan, convertScanToImageScan(image, quayScan))
}
