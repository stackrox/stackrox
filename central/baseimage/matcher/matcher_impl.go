package matcher

import (
	"context"
	"fmt"
	"slices"

	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/central/baseimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

type matcherImpl struct {
	datastore datastore.DataStore
}

var (
	log = logging.LoggerForModule(events.EnableAdministrationEvents())
)

// New creates a new base image watcher.
func New(
	datastore datastore.DataStore,
) Matcher {
	return &matcherImpl{
		datastore: datastore,
	}
}

func (m matcherImpl) MatchWithBaseImages(ctx context.Context, layers []string, imgName string, imgId string) []*storage.BaseImageInfo {
	if len(layers) == 0 {
		log.Infof("Base Image matching: not able to get image layers from %s", imgName)
		return nil
	}
	firstLayer := layers[0]
	candidates, err := m.datastore.ListCandidateBaseImages(ctx, firstLayer)
	if err != nil {
		log.Errorw("Matching image with base images",
			logging.FromContext(ctx),
			logging.ImageID(imgId),
			logging.Err(err),
			logging.String("request_image", imgName))
		return nil
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
		log.Infof(">>>> Getting base images candidates: %s, %s", c.GetRepository(), c.GetTag())
		match := true
		for i, l := range candidateLayers {
			log.Infof(">>>> Getting base image layer: %s, %s", layers[i], l.GetLayerDigest())
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
	return baseImages
}
