package enrichanddetect

import (
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
)

// EnricherAndDetector combines enrichment and detection into a single function call.
//go:generate mockgen-wrapper EnricherAndDetector
type EnricherAndDetector interface {
	EnrichAndDetect(deployment *storage.Deployment) error
}

// New returns a new instance of a EnricherAndDetector.
func New(enricher enrichment.Enricher, manager lifecycle.Manager, deploymentDataStore deploymentDatastore.DataStore, imageDataStore imageDatastore.DataStore) EnricherAndDetector {
	return &enricherAndDetectorImpl{
		enricher:            enricher,
		manager:             manager,
		deploymentDatastore: deploymentDataStore,
		imageDatastore:      imageDataStore,
	}
}
