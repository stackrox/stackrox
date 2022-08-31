package enrichment

import (
	"context"

	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Enricher enriches images with data from registries and scanners.
//
//go:generate mockgen-wrapper
type Enricher interface {
	// EnrichDeployment enriches the deployment and images only if they have IDs
	// It was enriched along with image indexes that were updated
	EnrichDeployment(ctx context.Context, enrichCtx enricher.EnrichmentContext, deployment *storage.Deployment) (images []*storage.Image, updatedIndices []int, pendingEnrichment bool, err error)
}

// New creates and returns a new Enricher.
func New(imageDatastore datastore.DataStore, imageEnricher enricher.ImageEnricher) Enricher {
	return &enricherImpl{
		images:        imageDatastore,
		imageEnricher: imageEnricher,
	}
}
