package enrichment

import (
	"context"

	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule(events.EnableAdministrationEvents())
)

// Enricher enriches images with data from registries and scanners.
//
//go:generate mockgen-wrapper
type Enricher interface {
	// EnrichDeployment enriches the deployment and images only if they have IDs
	// It was enriched along with image indexes that were updated
	EnrichDeployment(ctx context.Context, enrichCtx enricher.EnrichmentContext, deployment *storage.Deployment) (images []*storage.Image, updatedIndices []int, pendingEnrichment bool, err error)

	// EnrichDeploymentV2 enriches the deployment and images only if they have IDs
	// It was enriched along with image indexes that were updated
	EnrichDeploymentV2(ctx context.Context, enrichCtx enricher.EnrichmentContext, deployment *storage.Deployment) (images []*storage.ImageV2, updatedIndices []int, pendingEnrichment bool, err error)
}

// New creates and returns a new Enricher.
func New(imageDatastore datastore.DataStore, imageV2Datastore imageV2Datastore.DataStore, imageEnricher enricher.ImageEnricher, imageEnricherV2 enricher.ImageEnricherV2) Enricher {
	return &enricherImpl{
		images:          imageDatastore,
		imagesV2:        imageV2Datastore,
		imageEnricher:   imageEnricher,
		imageEnricherV2: imageEnricherV2,
	}
}
