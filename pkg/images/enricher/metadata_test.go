package enricher

import (
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	metadataHashToVersion = map[uint64]int{
		// initial hash of the metadata maps to 0
		14694942439820752696: 0,
	}
)

func TestMetadataVersionUpdated(t *testing.T) {
	metadata := &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{
				{}, // This is necessary for hashstructure to consider changes to ImageLayer
			},
		},
	}
	version, err := hashstructure.Hash(metadata, &hashstructure.HashOptions{ZeroNil: true})
	require.NoError(t, err)

	_, exists := metadataHashToVersion[version]
	assert.Truef(t, exists, "the metadata map to version above needs to include the metadata hash %v to the metadata version. Update the metadata version if metadata will need to be re-pulled", version)

	var found bool
	for _, v := range metadataHashToVersion {
		if v == metadataVersion {
			found = true
			break
		}
	}
	assert.Truef(t, found, "the current version number %d must be in the metadataHashToVersionMap", metadataVersion)
}
