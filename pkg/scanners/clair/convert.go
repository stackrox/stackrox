package clair

import (
	"github.com/stackrox/stackrox/generated/storage"
	clairConv "github.com/stackrox/stackrox/pkg/clair"
	"github.com/stackrox/stackrox/pkg/stringutils"
	clairV1 "github.com/stackrox/scanner/api/v1"
)

func convertLayerToImageScan(image *storage.Image, layerEnvelope *clairV1.LayerEnvelope) *storage.ImageScan {
	return &storage.ImageScan{
		OperatingSystem: stringutils.OrDefault(layerEnvelope.Layer.NamespaceName, "unknown"),
		Components:      clairConv.ConvertFeatures(image, layerEnvelope.Layer.Features),
	}
}
