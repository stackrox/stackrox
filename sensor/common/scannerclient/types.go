package scannerclient

import (
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

type imageData struct {
	Metadata *storage.ImageMetadata
	*scannerV1.GetImageComponentsResponse
}
