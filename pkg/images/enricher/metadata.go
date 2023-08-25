package enricher

import (
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// metadataVersion needs to be incremented if there are changes to the enrichment code that requires that the metadata be re-pulled
	// or if the metadataHash is modified and the metadata should be re-pulled to populate the new fields
	metadataVersion = 0
)

var (
	// metadataHashToVersion needs to be updated in two separate circumstances. If there are changes to the metadata struct that
	// change the metadataHash then a new entry with that hash mapped to an incremented value of the metadataVersion needs to be added.
	// If there are changes to the enrichment code that require a re-pull of the metadata, then the value that the current hash points at
	// should be incremented to the new metadataVersion
	metadataHashToVersion = map[uint64]int{
		// initial hash of the metadata maps to 0
		// hash changed when mirror* fields were added to storage.DataSource.
		7410019116170870180: 0,
	}

	metadataHash uint64
)

func init() {
	metadata := &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{
				{}, // This is necessary for hashstructure to consider changes to ImageLayer
			},
		},
	}
	var err error
	metadataHash, err = hashstructure.Hash(metadata, hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
	utils.Must(err)

	if val, ok := metadataHashToVersion[metadataHash]; !ok || val != metadataVersion {
		panic("current metadata hash must be equal to current version in map")
	}
}

func metadataIsOutOfDate(meta *storage.ImageMetadata) bool {
	if meta == nil {
		return true
	}
	return meta.GetVersion() != metadataVersion
}
