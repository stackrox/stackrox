package matcher

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/stackrox/rox/central/baseimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

type matcherImpl struct {
	datastore datastore.DataStore
}

var (
	log = logging.LoggerForModule()
)

// New creates a new base image watcher.
func New(
	datastore datastore.DataStore,
) Matcher {
	return &matcherImpl{
		datastore: datastore,
	}
}

func (m matcherImpl) MatchWithBaseImages(ctx context.Context, layers []string) ([]*storage.BaseImageInfo, error) {
	start := time.Now()
	var maxLayers int // Track the max layers found

	defer func() {
		log.Debugw("MatchWithBaseImages execution complete",
			"duration", time.Since(start),
			"layer_count", len(layers),
			"max_base_layers", maxLayers)
	}()

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

		// Ensure layers are in the correct order for comparison
		slices.SortFunc(candidateLayers, func(a, b *storage.BaseImageLayer) int {
			return int(a.GetIndex() - b.GetIndex())
		})

		// A base image cannot have more layers than the image being matched
		if len(layers) < len(candidateLayers) {
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
			n := len(candidateLayers)

			// We found a better (deeper) match
			if n > maxLayers {
				maxLayers = n
				// Clear previous results as they are no longer the "max"
				baseImages = baseImages[:0]
			}

			// This match is at the current max level
			if n == maxLayers {
				baseImages = append(baseImages, &storage.BaseImageInfo{
					BaseImageId:          c.GetId(),
					BaseImageFullName:    fmt.Sprintf("%s:%s", c.GetRepository(), c.GetTag()),
					BaseImageDigest:      c.GetManifestDigest(),
					Created:              c.GetCreated(),
					BaseImageTotalLayers: int32(n),
				})
			}
		}
	}

	return baseImages, nil
}
