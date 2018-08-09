package clair

import (
	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/api/v1"
	clairConv "github.com/stackrox/rox/pkg/clair"
)

func convertLayerToImageScan(image *v1.Image, layerEnvelope *clairV1.LayerEnvelope) *v1.ImageScan {
	return &v1.ImageScan{
		Components: clairConv.ConvertFeatures(layerEnvelope.Layer.Features),
	}
}
