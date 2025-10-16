package clair

import (
	"github.com/stackrox/rox/generated/storage"
	clairConv "github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/stringutils"
	clairV1 "github.com/stackrox/scanner/api/v1"
)

func convertLayerToImageScan(image *storage.Image, layerEnvelope *clairV1.LayerEnvelope) *storage.ImageScan {
	os := stringutils.OrDefault(layerEnvelope.Layer.NamespaceName, "unknown")
	imageScan := &storage.ImageScan{}
	imageScan.SetOperatingSystem(stringutils.OrDefault(layerEnvelope.Layer.NamespaceName, "unknown"))
	imageScan.SetComponents(clairConv.ConvertFeatures(image, layerEnvelope.Layer.Features, os))
	return imageScan
}
