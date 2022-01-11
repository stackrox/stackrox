package enricher

import (
	"github.com/stackrox/rox/generated/storage"
)

const (
	metadataVersion = 0
)

func metadataIsOutOfDate(meta *storage.ImageMetadata) bool {
	if meta == nil {
		return true
	}
	return meta.GetVersion() != metadataVersion
}
