package enrichment

import (
	deploymentDS "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDS "bitbucket.org/stack-rox/apollo/central/image/datastore"
	imageIntegrationDS "bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
	multiplierDS "bitbucket.org/stack-rox/apollo/central/multiplier/store"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/imageenricher"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

var (
	logger = logging.LoggerForModule()
)

// Enricher enriches images with data from registries and scanners.
type Enricher interface {
	Enrich(deployment *v1.Deployment) (bool, error)

	UpdateImageIntegration(integration *sources.ImageIntegration)
	RemoveImageIntegration(id string)

	UpdateMultiplier(multiplier *v1.Multiplier)
	RemoveMultiplier(id string)

	ReprocessRisk()
	ReprocessDeploymentRisk(deployment *v1.Deployment) error
}

// New creates and returns a new Enricher.
func New(deploymentStorage deploymentDS.DataStore,
	imageStorage imageDS.DataStore,
	imageIntegrationStorage imageIntegrationDS.DataStore,
	multiplierStorage multiplierDS.Store,
	imageEnricher imageenricher.ImageEnricher,
	scorer risk.Scorer) (Enricher, error) {
	e := &enricherImpl{
		deploymentStorage:       deploymentStorage,
		imageStorage:            imageStorage,
		imageIntegrationStorage: imageIntegrationStorage,
		multiplierStorage:       multiplierStorage,
		imageEnricher:           imageEnricher,
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
