package manager

import (
	"context"

	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/ml"
	"github.com/stackrox/rox/central/risk/scorer/deployment"
	"github.com/stackrox/rox/generated/storage"
)

// managerWithML extends the risk manager with ML capabilities
type managerWithML struct {
	*managerImpl
	mlIntegration *MLIntegration
}

// NewManagerWithML creates a risk manager with ML integration
func NewManagerWithML(riskStorage datastore.DataStore, deploymentScorer deployment.Scorer) Manager {
	baseManager := &managerImpl{
		riskStorage:      riskStorage,
		deploymentScorer: deploymentScorer,
	}

	mlIntegration := NewMLIntegration(deploymentScorer)

	return &managerWithML{
		managerImpl:   baseManager,
		mlIntegration: mlIntegration,
	}
}

// CalculateRiskAndUpsertAsync calculates risk using ML integration and upserts asynchronously
func (m *managerWithML) CalculateRiskAndUpsertAsync(deployment *storage.Deployment, images []*storage.Image) {
	go func() {
		ctx := context.Background()

		// Get image risks from storage (similar to ReprocessDeploymentRisk pattern)
		imageRisks := make([]*storage.Risk, 0, len(deployment.GetContainers()))
		for _, container := range deployment.GetContainers() {
			if imgID := container.GetImage().GetId(); imgID != "" {
				risk, exists, err := m.riskStorage.GetRisk(ctx, imgID, storage.RiskSubjectType_IMAGE)
				if err != nil {
					log.Errorf("error getting risk for image %s: %v", imgID, err)
					continue
				}
				if !exists {
					continue
				}
				imageRisks = append(imageRisks, risk)
			}
		}

		// Use ML integration to score deployment
		deploymentRisk := m.mlIntegration.ScoreDeployment(ctx, deployment, imageRisks)
		if deploymentRisk == nil {
			log.Debugf("No risk calculated for deployment %s", deployment.GetId())
			return
		}

		// Upsert the risk
		if err := m.riskStorage.UpsertRisk(ctx, deploymentRisk); err != nil {
			log.Errorf("Failed to upsert risk for deployment %s: %v", deployment.GetId(), err)
		} else {
			log.Debugf("Risk calculated and stored for deployment %s: score=%.2f",
				deployment.GetId(), deploymentRisk.GetScore())
		}
	}()
}

// CalculateRiskAndUpsert calculates risk using ML integration and upserts synchronously
func (m *managerWithML) CalculateRiskAndUpsert(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) error {
	// Get image risks from storage (similar to ReprocessDeploymentRisk pattern)
	imageRisks := make([]*storage.Risk, 0, len(deployment.GetContainers()))
	for _, container := range deployment.GetContainers() {
		if imgID := container.GetImage().GetId(); imgID != "" {
			risk, exists, err := m.riskStorage.GetRisk(ctx, imgID, storage.RiskSubjectType_IMAGE)
			if err != nil {
				log.Errorf("error getting risk for image %s: %v", imgID, err)
				continue
			}
			if !exists {
				continue
			}
			imageRisks = append(imageRisks, risk)
		}
	}

	// Use ML integration to score deployment
	deploymentRisk := m.mlIntegration.ScoreDeployment(ctx, deployment, imageRisks)
	if deploymentRisk == nil {
		log.Debugf("No risk calculated for deployment %s", deployment.GetId())
		return nil
	}

	// Upsert the risk
	if err := m.riskStorage.UpsertRisk(ctx, deploymentRisk); err != nil {
		log.Errorf("Failed to upsert risk for deployment %s: %v", deployment.GetId(), err)
		return err
	}

	log.Debugf("Risk calculated and stored for deployment %s: score=%.2f",
		deployment.GetId(), deploymentRisk.GetScore())
	return nil
}

// GetMLIntegration returns the ML integration instance
func (m *managerWithML) GetMLIntegration() *MLIntegration {
	return m.mlIntegration
}

// IsMLEnabled returns whether ML integration is enabled
func (m *managerWithML) IsMLEnabled() bool {
	return m.mlIntegration.IsMLEnabled()
}

// GetMLHealthStatus returns ML service health status
func (m *managerWithML) GetMLHealthStatus(ctx context.Context) (*ml.ModelHealthResponse, error) {
	return m.mlIntegration.GetMLHealthStatus(ctx)
}

// CreateManagerBasedOnConfig creates appropriate manager based on ML configuration
func CreateManagerBasedOnConfig(riskStorage datastore.DataStore, deploymentScorer deployment.Scorer) Manager {
	// Check if ML is enabled via environment variables
	if ml.IsEnabled() {
		log.Info("Creating risk manager with ML integration")
		return NewManagerWithML(riskStorage, deploymentScorer)
	}

	log.Info("Creating traditional risk manager (ML disabled)")
	return &managerImpl{
		riskStorage:      riskStorage,
		deploymentScorer: deploymentScorer,
	}
}
