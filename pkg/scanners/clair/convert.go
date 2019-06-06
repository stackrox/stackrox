package clair

import (
	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
	clairConv "github.com/stackrox/rox/pkg/clair"
)

func convertLayerToImageScan(image *storage.Image, layerEnvelope *clairV1.LayerEnvelope) *storage.ImageScan {
	return &storage.ImageScan{
		Components: clairConv.ConvertFeatures(image, layerEnvelope.Layer.Features),
	}
}
