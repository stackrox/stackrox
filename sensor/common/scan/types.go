package scan

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
)

var _ types.ImageWithMetadata = (*imageWithMetadata)(nil)

type imageWithMetadata struct {
	id       string
	metadata *storage.ImageMetadata
}

//nolint:revive
func (i *imageWithMetadata) GetId() string {
	return i.id
}

func (i *imageWithMetadata) GetMetadata() *storage.ImageMetadata {
	return i.metadata
}
