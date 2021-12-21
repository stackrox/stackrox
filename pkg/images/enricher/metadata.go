package enricher

import (
	"github.com/mitchellh/hashstructure"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

type hashableVersion struct {
	internalMetadataVersion, hashedMetadataVersion uint64
}

func (h hashableVersion) hash() uint64 {
	version, err := hashstructure.Hash(h, nil)
	utils.Must(err)
	return version
}

var (
	// currentMetadataVersion is the hash of both internalMetadataVersion and hashedMetadataVersion and is
	// used to determine if we need to re-pull the image metadata
	currentMetadataVersion = hashableVersion{
		// internalMetadataVersion is a developer incremented version in the case that we modify the logic
		// populating the ImageMetadata struct
		internalMetadataVersion: 0,

		// hashedMetadataVersion is a hash value computed based on the fields of the image metadata message
		hashedMetadataVersion: func() uint64 {
			metadata := &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{
						{}, // This is necessary for hashstructure to consider changes to ImageLayer
					},
				},
			}
			version, err := hashstructure.Hash(metadata, &hashstructure.HashOptions{ZeroNil: true})
			utils.Must(err)
			return version
		}(),
	}.hash()
)

func metadataIsOutOfDate(meta *storage.ImageMetadata) bool {
	if meta == nil {
		return true
	}
	// This provides backwards compatibility and avoids a migration as 0xF17CE73F93A5AF97 is the initial hash value
	if meta.GetVersion() == 0 && currentMetadataVersion == 0xF17CE73F93A5AF97 {
		return false
	}
	return meta.GetVersion() != currentMetadataVersion
}
