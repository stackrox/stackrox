package enrichment

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// UpdateMultiplier upserts a multiplier into the scorer
func (e *Enricher) UpdateMultiplier(multiplier *v1.Multiplier) {
	e.scorer.UpdateUserDefinedMultiplier(multiplier)
	go func() {
		if err := e.ReprocessRisk(); err != nil {
			logger.Errorf("Error reprocessing risk: %s", err)
		}
	}()
}

// RemoveMultiplier removes a multiplier from the scorer
func (e *Enricher) RemoveMultiplier(id string) {
	e.scorer.RemoveUserDefinedMultiplier(id)
	go e.ReprocessRisk()
}

// ReprocessRisk iterates over all of the deployments and reprocesses the risk for them
func (e *Enricher) ReprocessRisk() error {
	deployments, err := e.storage.GetDeployments()
	if err != nil {
		return err
	}
	for _, deployment := range deployments {
		deployment.Risk = e.scorer.Score(deployment)
		if err := e.storage.UpdateDeployment(deployment); err != nil {
			return err
		}
	}
	return nil
}
