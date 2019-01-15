package enrichment

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Enricher enriches images with data from registries and scanners.
//go:generate mockgen-wrapper Enricher
type Enricher interface {
	// EnrichDeployment enriches the deployment and images only if they have IDs
	// It was enriched along with image indexes that were updated
	EnrichDeployment(deployment *storage.Deployment) ([]*storage.Image, bool, error)
	// EnrichDeploymentWithEmptyImages enriches the deployment and images even if they don't have IDs
	// It was enriched along with image indexes that were updated
	EnrichDeploymentWithEmptyImages(deployment *storage.Deployment) ([]*storage.Image, bool, error)
}

// New creates and returns a new Enricher.
func New(imageEnricher enricher.ImageEnricher) Enricher {
	return &enricherImpl{
		imageEnricher: imageEnricher,
	}
}
