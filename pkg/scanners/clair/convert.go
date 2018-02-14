package clair

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	clairConv "bitbucket.org/stack-rox/apollo/pkg/clair"
	clairV1 "github.com/coreos/clair/api/v1"
)

func convertLayerToImageScan(image *v1.Image, layerEnvelope *clairV1.LayerEnvelope) *v1.ImageScan {
	return &v1.ImageScan{
		Name: &v1.ImageName{
			Sha:      image.GetName().GetSha(),
			Registry: image.GetName().GetRegistry(),
			Remote:   image.GetName().GetRemote(),
			Tag:      image.GetName().GetTag(),
		},
		State:      v1.ImageScanState_COMPLETED,
		Components: clairConv.ConvertFeatures(layerEnvelope.Layer.Features),
	}
}
