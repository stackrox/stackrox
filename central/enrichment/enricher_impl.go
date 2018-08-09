package enrichment

import (
	"github.com/gogo/protobuf/proto"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageIntegrationDS "github.com/stackrox/rox/central/imageintegration/datastore"
	multiplierDS "github.com/stackrox/rox/central/multiplier/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/enricher"
)

// enricherImpl enriches images with data from registries and scanners.
type enricherImpl struct {
	deploymentStorage       deploymentDS.DataStore
	imageStorage            imageDS.DataStore
	imageIntegrationStorage imageIntegrationDS.DataStore
	multiplierStorage       multiplierDS.Store

	imageEnricher enricher.ImageEnricher
	scorer        risk.Scorer
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
		if err := e.addRiskToDeployment(deployment); err != nil {
			logger.Errorf("Error reprocessing deployment risk: %s", err)
			return
		}
	}
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *enricherImpl) ReprocessDeploymentRisk(deployment *v1.Deployment) error {
	deployment = proto.Clone(deployment).(*v1.Deployment)
	return e.addRiskToDeployment(deployment)
}

// addRiskToDeployment will add the risk
func (e *enricherImpl) addRiskToDeployment(deployment *v1.Deployment) error {
	deployment.Risk = e.scorer.Score(deployment)
	return e.deploymentStorage.UpdateDeployment(deployment)
}
