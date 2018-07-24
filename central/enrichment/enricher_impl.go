package enrichment

import (
	"fmt"

	deploymentDS "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDS "bitbucket.org/stack-rox/apollo/central/image/datastore"
	imageIntegrationDS "bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
	multiplierDS "bitbucket.org/stack-rox/apollo/central/multiplier/store"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/imageenricher"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

// enricherImpl enriches images with data from registries and scanners.
type enricherImpl struct {
	deploymentStorage       deploymentDS.DataStore
	imageStorage            imageDS.DataStore
	imageIntegrationStorage imageIntegrationDS.DataStore
	multiplierStorage       multiplierDS.Store

	imageEnricher imageenricher.ImageEnricher
	scorer        risk.Scorer
}

func (e *enricherImpl) initializeImageIntegrations() error {
	protoImageIntegrations, err := e.imageIntegrationStorage.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
	if err != nil {
		return err
	}

	for _, protoImageIntegration := range protoImageIntegrations {
		integration, err := sources.NewImageIntegration(protoImageIntegration)
		if err != nil {
			return fmt.Errorf("error generating an image integration from a persisted image integration: %s", err)
		}
		e.imageEnricher.IntegrationSet().UpdateImageIntegration(integration)
	}
	return nil
}

func (e *enricherImpl) initializeMultipliers() error {
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
func (e *enricherImpl) Enrich(deployment *v1.Deployment) (bool, error) {
	var deploymentUpdated bool
	for _, c := range deployment.GetContainers() {
		if updated := e.imageEnricher.EnrichImage(c.Image); updated {
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

// UpdateImageIntegration updates the enricher's map of active image integratinos
func (e *enricherImpl) UpdateImageIntegration(integration *sources.ImageIntegration) {
	e.imageEnricher.IntegrationSet().UpdateImageIntegration(integration)
}

// RemoveImageIntegration removes a image integration from the enricher's map of active image integrations
func (e *enricherImpl) RemoveImageIntegration(id string) {
	e.imageEnricher.IntegrationSet().RemoveImageIntegration(id)
}

// UpdateMultiplier upserts a multiplier into the scorer
func (e *enricherImpl) UpdateMultiplier(multiplier *v1.Multiplier) {
	e.scorer.UpdateUserDefinedMultiplier(multiplier)
	go e.ReprocessRisk()
}

// RemoveMultiplier removes a multiplier from the scorer
func (e *enricherImpl) RemoveMultiplier(id string) {
	e.scorer.RemoveUserDefinedMultiplier(id)
	go e.ReprocessRisk()
}

// ReprocessRisk iterates over all of the deployments and reprocesses the risk for them
func (e *enricherImpl) ReprocessRisk() {
	deployments, err := e.deploymentStorage.GetDeployments()
	if err != nil {
		logger.Errorf("Error reprocessing risk: %s", err)
		return
	}

	for _, deployment := range deployments {
		if err := e.ReprocessDeploymentRisk(deployment); err != nil {
			logger.Errorf("Error reprocessing deployment risk: %s", err)
			return
		}
	}
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *enricherImpl) ReprocessDeploymentRisk(deployment *v1.Deployment) error {
	deployment.Risk = e.scorer.Score(deployment)

	return e.deploymentStorage.UpdateDeployment(deployment)
}
