package manager

import (
	"context"

	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/scorer/deployment"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// managerWithML extends the risk manager with ML capabilities
type managerWithML struct {
	*managerImpl
	mlIntegration *MLIntegration
}

// NewManagerWithML creates a risk manager with ML integration
func NewManagerWithML(riskDataStore datastore.DataStore, deploymentScorer deployment.Scorer) Manager {
	baseManager := &managerImpl{
		riskDataStore:    riskDataStore,
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

		// Get image risks first (needed for both traditional and ML scoring)
		imageRisks := make([]*storage.Risk, 0, len(images))
		for _, image := range images {
			if imageRisk := m.CalculateImageRisk(image); imageRisk != nil {
				imageRisks = append(imageRisks, imageRisk)
			}
		}

		// Use ML integration to score deployment
		deploymentRisk := m.mlIntegration.ScoreDeployment(ctx, deployment, imageRisks)
		if deploymentRisk == nil {
			log.Debugf("No risk calculated for deployment %s", deployment.GetId())
			return
		}

		// Upsert the risk
		if err := m.riskDataStore.UpsertRisk(ctx, deploymentRisk); err != nil {
			log.Errorf("Failed to upsert risk for deployment %s: %v", deployment.GetId(), err)
		} else {
			log.Debugf("Risk calculated and stored for deployment %s: score=%.2f",
				deployment.GetId(), deploymentRisk.GetScore())
		}
	}()
}

// CalculateRiskAndUpsert calculates risk using ML integration and upserts synchronously
func (m *managerWithML) CalculateRiskAndUpsert(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) error {
	// Get image risks first
	imageRisks := make([]*storage.Risk, 0, len(images))
	for _, image := range images {
		if imageRisk := m.CalculateImageRisk(image); imageRisk != nil {
			imageRisks = append(imageRisks, imageRisk)
		}
	}

	// Use ML integration to score deployment
	deploymentRisk := m.mlIntegration.ScoreDeployment(ctx, deployment, imageRisks)
	if deploymentRisk == nil {
		log.Debugf("No risk calculated for deployment %s", deployment.GetId())
		return nil
	}

	// Upsert the risk
	if err := m.riskDataStore.UpsertRisk(ctx, deploymentRisk); err != nil {
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
func CreateManagerBasedOnConfig(riskDataStore datastore.DataStore, deploymentScorer deployment.Scorer) Manager {
	// Check if ML is enabled via environment variables
	if ml.IsEnabled() {
		log.Info("Creating risk manager with ML integration")
		return NewManagerWithML(riskDataStore, deploymentScorer)
	}

	log.Info("Creating traditional risk manager (ML disabled)")
	return &managerImpl{
		riskDataStore:    riskDataStore,
		deploymentScorer: deploymentScorer,
	}
}