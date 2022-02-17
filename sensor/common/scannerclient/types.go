package scannerclient

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var _ types.ImageWithMetadata = (*imageData)(nil)

type imageData struct {
	id       string
	metadata *storage.ImageMetadata
	*scannerV1.GetImageComponentsResponse
}

func (i *imageData) GetId() string {
	return i.id
}

func (i *imageData) GetMetadata() *storage.ImageMetadata {
	return i.metadata
}
