package clair

import (
	"github.com/stackrox/rox/generated/storage"
	clairConv "github.com/stackrox/rox/pkg/clair"
	clairV1 "github.com/stackrox/scanner/api/v1"
)

func convertLayerToImageScan(image *storage.Image, layerEnvelope *clairV1.LayerEnvelope) *storage.ImageScan {
	return &storage.ImageScan{
		Components: clairConv.ConvertFeatures(image, layerEnvelope.Layer.Features),
	}
}
