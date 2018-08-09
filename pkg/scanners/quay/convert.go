package quay

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clair"
)

func convertScanToImageScan(image *v1.Image, s *scanResult) *v1.ImageScan {
	components := clair.ConvertFeatures(s.Data.Layer.Features)
	return &v1.ImageScan{
		Components: components,
	}
}
