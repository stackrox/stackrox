package enrichment

import (
	"fmt"
	"sync"

	alertDS "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDS "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDS "bitbucket.org/stack-rox/apollo/central/image/datastore"
	imageIntegrationDS "bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
	"bitbucket.org/stack-rox/apollo/central/imageintegration/enricher"
	multiplierDS "bitbucket.org/stack-rox/apollo/central/multiplier/store"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

var (
	logger = logging.LoggerForModule()
)

// Enricher enriches images with data from registries and scanners.
type Enricher struct {
	deploymentStorage       deploymentDS.DataStore
	imageStorage            imageDS.DataStore
	imageIntegrationStorage imageIntegrationDS.DataStore
	multiplierStorage       multiplierDS.Store
	alertStorage            alertDS.DataStore

	scorerMutex sync.Mutex
	scorer      *risk.Scorer
}

// New creates and returns a new Enricher.
func New(deploymentStorage deploymentDS.DataStore,
	imageStorage imageDS.DataStore,
	imageIntegrationStorage imageIntegrationDS.DataStore,
	multiplierStorage multiplierDS.Store,
	alertStorage alertDS.DataStore,
	scorer *risk.Scorer) (*Enricher, error) {
	e := &Enricher{
		deploymentStorage:       deploymentStorage,
		imageStorage:            imageStorage,
		imageIntegrationStorage: imageIntegrationStorage,
		multiplierStorage:       multiplierStorage,
		alertStorage:            alertStorage,
		scorer:                  scorer,
	}
	if err := e.initializeImageIntegrations(); err != nil {
		return nil, err
	}
	if err := e.initializeMultipliers(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Enricher) initializeImageIntegrations() error {
	protoImageIntegrations, err := e.imageIntegrationStorage.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
	if err != nil {
		return err
	}

	for _, protoImageIntegration := range protoImageIntegrations {
		integration, err := sources.NewImageIntegration(protoImageIntegration)
		if err != nil {
			return fmt.Errorf("error generating an image integration from a persisted image integration: %s", err)
		}
		enricher.ImageEnricher.UpdateImageIntegration(integration)
	}
	return nil
}

// UpdateImageIntegration updates the enricher's map of active image integratinos
func (e *Enricher) UpdateImageIntegration(integration *sources.ImageIntegration) {
	enricher.ImageEnricher.UpdateImageIntegration(integration)
}

// RemoveImageIntegration removes a image integration from the enricher's map of active image integrations
func (e *Enricher) RemoveImageIntegration(id string) {
	enricher.ImageEnricher.RemoveImageIntegration(id)
}

func (e *Enricher) initializeMultipliers() error {
	protoMultipliers, err := e.multiplierStorage.GetMultipliers()
	if err != nil {
		return err
	}
	for _, mult := range protoMultipliers {
		e.scorer.UpdateUserDefinedMultiplier(mult)
	}
	return nil
}

// Enrich enriches a deployment with data from registries and scanners.
func (e *Enricher) Enrich(deployment *v1.Deployment) (bool, error) {
	var deploymentUpdated bool
	for _, c := range deployment.GetContainers() {
		if updated := enricher.ImageEnricher.EnrichImage(c.Image); updated {
			if err := e.imageStorage.UpsertDedupeImage(c.Image); err != nil {
				return false, err
			}
			deploymentUpdated = true
		}
	}
	if deploymentUpdated {
		if err := e.deploymentStorage.UpdateDeployment(deployment); err != nil {
			return false, err
		}
	}
	return deploymentUpdated, nil
}
