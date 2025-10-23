package deployment

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/ml"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// MLScorer uses machine learning to score deployment risk
type MLScorer struct {
	mlClient ml.MLRiskClient
}

// NewMLScorer creates a new ML-based deployment scorer
func NewMLScorer() *MLScorer {
	return &MLScorer{
		mlClient: ml.Singleton(),
	}
}

// Score takes a deployment and evaluates its risk using ML
func (s *MLScorer) Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	if !ml.IsEnabled() {
		log.Debug("ML Risk Service is disabled, skipping ML scoring")
		return nil
	}

	// Convert Risk objects to Image objects for ML client
	imageObjects := make([]*storage.Image, 0, len(images))
	for _, imageRisk := range images {
		// In practice, we'd need to fetch the actual Image objects
		// For now, create minimal Image objects from Risk data
		imageObj := &storage.Image{
			Id: imageRisk.GetSubject().GetId(),
			// Add other required fields...
		}
		imageObjects = append(imageObjects, imageObj)
	}

	// Get ML risk assessment
	mlResponse, err := s.mlClient.GetDeploymentRisk(ctx, deployment, imageObjects)
	if err != nil {
		log.Errorf("Failed to get ML risk assessment for deployment %s: %v", deployment.GetId(), err)
		return nil
	}

	if mlResponse == nil {
		log.Debug("No ML risk response for deployment %s", deployment.GetId())
		return nil
	}

	// Convert ML response to StackRox Risk format
	riskResults := make([]*storage.Risk_Result, 0, len(mlResponse.FeatureImportances))

	// Create risk result for ML prediction
	mlRiskResult := &storage.Risk_Result{
		Name:  "ML Risk Assessment",
		Score: mlResponse.RiskScore,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: "Machine learning risk prediction",
			},
		},
	}
	riskResults = append(riskResults, mlRiskResult)

	// Add feature importance as additional risk factors
	for _, importance := range mlResponse.FeatureImportances {
		if importance.ImportanceScore > 0.01 { // Only include significant features
			factor := &storage.Risk_Result_Factor{
				Message: s.formatFeatureImportance(importance),
			}
			mlRiskResult.Factors = append(mlRiskResult.Factors, factor)
		}
	}

	// Create overall risk object
	risk := &storage.Risk{
		Score:   mlResponse.RiskScore,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:        deployment.GetId(),
			Type:      storage.RiskSubjectType_DEPLOYMENT,
			Namespace: deployment.GetNamespace(),
			ClusterId: deployment.GetClusterId(),
		},
	}

	// Generate risk ID
	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Error(err)
		return nil
	}
	risk.Id = riskID

	log.Infof("ML risk assessment for deployment %s: score=%.2f, model=%s",
		deployment.GetId(), mlResponse.RiskScore, mlResponse.ModelVersion)

	return risk
}

// formatFeatureImportance formats feature importance for display
func (s *MLScorer) formatFeatureImportance(importance *ml.FeatureImportance) string {
	direction := "increases"
	if importance.ImportanceScore < 0 {
		direction = "decreases"
	}

	return fmt.Sprintf("%s %s risk by %.3f",
		importance.Description,
		direction,
		abs(importance.ImportanceScore))
}

// abs returns the absolute value of a float32
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// IsMLEnabled returns whether ML scoring is enabled
func IsMLEnabled() bool {
	return ml.IsEnabled()
}

// GetMLHealthStatus returns the health status of the ML service
func GetMLHealthStatus(ctx context.Context) (*ml.ModelHealthResponse, error) {
	client := ml.Singleton()
	return client.GetModelHealth(ctx)
}