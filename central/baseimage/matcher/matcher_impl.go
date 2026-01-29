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
	var maxLayers int
	var matchedLayerDigests []string
	var baseImages []*storage.BaseImageInfo

	defer func() {
		log.Debugw("MatchWithBaseImages execution complete",
			"duration", time.Since(start),
			"layer_count", len(layers),
			"best_match_depth", maxLayers)
	}()

	if len(layers) == 0 {
		return nil, nil
	}

	firstLayer := layers[0]
	candidates, err := m.datastore.ListCandidateBaseImages(ctx, firstLayer)
	if err != nil {
		return nil, fmt.Errorf("listing candidates for layer %s: %w", firstLayer, err)
	}

	for _, c := range candidates {
		candidateLayers := c.GetLayers()
		slices.SortFunc(candidateLayers, func(a, b *storage.BaseImageLayer) int {
			return int(a.GetIndex() - b.GetIndex())
		})

		// A base image cannot have more layers than the target image.
		if len(layers) < len(candidateLayers) {
			continue
		}

		match := true
		current := make([]string, 0, len(candidateLayers))
		for i, l := range candidateLayers {
			if layers[i] != l.GetLayerDigest() {
				match = false
				break
			}
			current = append(current, l.GetLayerDigest())
		}

		if match {
			n := len(candidateLayers)

			// Found a deeper match: clear previous shallow matches
			if n > maxLayers {
				maxLayers = n
				baseImages = baseImages[:0]
				matchedLayerDigests = current
			}

			// Only add if it matches the current maximum depth
			if n == maxLayers {
				baseImages = append(baseImages, &storage.BaseImageInfo{
					BaseImageId:       c.GetId(),
					BaseImageFullName: fmt.Sprintf("%s:%s", c.GetRepository(), c.GetTag()),
					BaseImageDigest:   c.GetManifestDigest(),
					Created:           c.GetCreated(),
					Layers:            matchedLayerDigests,
				})
			}
		}
	}

	return baseImages, nil
}
