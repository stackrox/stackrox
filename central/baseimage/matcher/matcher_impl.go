package matcher

import (
	"context"
	"fmt"
	"slices"

	"github.com/stackrox/rox/central/baseimage/datastore"
	"github.com/stackrox/rox/generated/storage"
)

type matcherImpl struct {
	datastore datastore.DataStore
}

// New creates a new base image watcher.
func New(
	datastore datastore.DataStore,
) Matcher {
	return &matcherImpl{
		datastore: datastore,
	}
}

func (m matcherImpl) MatchWithBaseImages(ctx context.Context, layers []string) ([]*storage.BaseImageInfo, error) {
	if len(layers) == 0 {
		return nil, nil
	}
	firstLayer := layers[0]
	candidates, err := m.datastore.ListCandidateBaseImages(ctx, firstLayer)
	if err != nil {
		return nil, fmt.Errorf("listing candidates for layer %s: %w", firstLayer, err)
	}
	var baseImages []*storage.BaseImageInfo
	for _, c := range candidates {
		candidateLayers := c.GetLayers()
		slices.SortFunc(candidateLayers, func(a, b *storage.BaseImageLayer) int {
			return int(a.GetIndex() - b.GetIndex())
		})
		if len(layers) <= len(candidateLayers) {
			continue
		}
		match := true
		for i, l := range candidateLayers {
			if layers[i] != l.GetLayerDigest() {
				match = false
				break
			}
		}

		if match {
			baseImages = append(baseImages, &storage.BaseImageInfo{
				BaseImageId:       c.GetId(),
				BaseImageFullName: fmt.Sprintf("%s:%s", c.GetRepository(), c.GetTag()),
				BaseImageDigest:   c.GetManifestDigest(),
			})
		}
	}
	return baseImages, nil
}
