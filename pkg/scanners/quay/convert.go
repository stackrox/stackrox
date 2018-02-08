package quay

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clair"
)

func convertScanToImageScan(image *v1.Image, s *scanResult) *v1.ImageScan {
	components := clair.ConvertFeatures(s.Data.Layer.Features)
	return &v1.ImageScan{
		Sha:        image.GetSha(),
		Registry:   image.GetRegistry(),
		Remote:     image.GetRemote(),
		Tag:        image.GetTag(),
		State:      v1.ImageScanState_COMPLETED,
		Components: components,
	}
}
