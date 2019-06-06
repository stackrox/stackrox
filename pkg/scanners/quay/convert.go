package quay

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
)

func convertScanToImageScan(image *storage.Image, s *scanResult) *storage.ImageScan {
	components := clair.ConvertFeatures(image, s.Data.Layer.Features)
	return &storage.ImageScan{
		Components: components,
	}
}
