package manager

import (
	"context"

	"github.com/stackrox/rox/central/risk/ml"
	"github.com/stackrox/rox/central/risk/scorer/deployment"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// Environment variable to enable ML risk mode
	mlRiskModeEnabled = env.RegisterBooleanSetting("ROX_ML_RISK_MODE_ENABLED", false)
)

// MLRiskMode represents the mode of ML risk assessment
type MLRiskMode int

const (
	// MLRiskModeDisabled - ML risk assessment is disabled (default)
	MLRiskModeDisabled MLRiskMode = iota
	// MLRiskModeAugmented - ML risk augments traditional risk scoring
	MLRiskModeAugmented
	// MLRiskModeReplacement - ML risk replaces traditional risk scoring
	MLRiskModeReplacement
)

// MLIntegration handles integration between traditional and ML risk scoring
type MLIntegration struct {
	mode              MLRiskMode
	mlScorer          *deployment.MLScorer
	traditionalScorer deployment.Scorer
}

// NewMLIntegration creates a new ML integration
func NewMLIntegration(traditionalScorer deployment.Scorer) *MLIntegration {
	mode := MLRiskModeDisabled
	if mlRiskModeEnabled.BooleanSetting() && ml.IsEnabled() {
		mode = MLRiskModeAugmented // Default to augmented mode when enabled
	}

	return &MLIntegration{
		mode:              mode,
		mlScorer:          deployment.NewMLScorer(),
		traditionalScorer: traditionalScorer,
	}
}

// ScoreDeployment scores a deployment using the configured ML integration mode
func (m *MLIntegration) ScoreDeployment(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	switch m.mode {
	case MLRiskModeDisabled:
		// Use only traditional scoring
		return m.traditionalScorer.Score(ctx, deployment, images)

	case MLRiskModeAugmented:
		// Use both traditional and ML scoring, combine results
		return m.scoreAugmented(ctx, deployment, images)

	case MLRiskModeReplacement:
		// Use only ML scoring
		mlRisk := m.mlScorer.Score(ctx, deployment, images)
		if mlRisk != nil {
			return mlRisk
		}
		// Fall back to traditional if ML fails
		log.Warnf("ML scoring failed for deployment %s, falling back to traditional scoring", deployment.GetId())
		return m.traditionalScorer.Score(ctx, deployment, images)

	default:
		log.Errorf("Unknown ML risk mode: %d", m.mode)
		return m.traditionalScorer.Score(ctx, deployment, images)
	}
}

// scoreAugmented combines traditional and ML risk scoring
func (m *MLIntegration) scoreAugmented(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	// Get traditional risk assessment
	traditionalRisk := m.traditionalScorer.Score(ctx, deployment, images)

	// Get ML risk assessment
	mlRisk := m.mlScorer.Score(ctx, deployment, images)

	// If ML is not available, return traditional risk
	if mlRisk == nil {
		if traditionalRisk != nil {
			log.Debugf("ML scoring not available for deployment %s, using traditional scoring", deployment.GetId())
		}
		return traditionalRisk
	}

	// If traditional is not available, return ML risk
	if traditionalRisk == nil {
		log.Debugf("Traditional scoring not available for deployment %s, using ML scoring", deployment.GetId())
		return mlRisk
	}

	// Combine both assessments
	return m.combineRiskAssessments(traditionalRisk, mlRisk, deployment)
}

// combineRiskAssessments combines traditional and ML risk assessments
func (m *MLIntegration) combineRiskAssessments(traditionalRisk, mlRisk *storage.Risk, deployment *storage.Deployment) *storage.Risk {
	// Weighted combination of scores (configurable weights)
	traditionalWeight := float32(0.6) // Traditional scoring weight
	mlWeight := float32(0.4)          // ML scoring weight

	combinedScore := traditionalRisk.GetScore()*traditionalWeight + mlRisk.GetScore()*mlWeight

	// Combine risk results
	combinedResults := make([]*storage.Risk_Result, 0, len(traditionalRisk.GetResults())+len(mlRisk.GetResults())+1)

	// Add traditional results
	for _, result := range traditionalRisk.GetResults() {
		combinedResults = append(combinedResults, result)
	}

	// Add ML results
	for _, result := range mlRisk.GetResults() {
		// Prefix ML results to distinguish them
		mlResult := &storage.Risk_Result{
			Name:    "ML: " + result.GetName(),
			Score:   result.GetScore(),
			Factors: result.GetFactors(),
		}
		combinedResults = append(combinedResults, mlResult)
	}

	// Add combined assessment result
	combinedResult := &storage.Risk_Result{
		Name:  "Combined Risk Assessment",
		Score: combinedScore,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("Traditional score: %.2f (weight: %.1f)", traditionalRisk.GetScore(), traditionalWeight),
			},
			{
				Message: fmt.Sprintf("ML score: %.2f (weight: %.1f)", mlRisk.GetScore(), mlWeight),
			},
			{
				Message: fmt.Sprintf("Combined score: %.2f", combinedScore),
			},
		},
	}
	combinedResults = append(combinedResults, combinedResult)

	// Create combined risk object
	combinedRisk := &storage.Risk{
		Id:      traditionalRisk.GetId(), // Use traditional risk ID
		Score:   combinedScore,
		Results: combinedResults,
		Subject: traditionalRisk.GetSubject(),
	}

	log.Infof("Combined risk assessment for deployment %s: traditional=%.2f, ml=%.2f, combined=%.2f",
		deployment.GetId(), traditionalRisk.GetScore(), mlRisk.GetScore(), combinedScore)

	return combinedRisk
}

// GetMode returns the current ML integration mode
func (m *MLIntegration) GetMode() MLRiskMode {
	return m.mode
}

// SetMode sets the ML integration mode (for testing/configuration)
func (m *MLIntegration) SetMode(mode MLRiskMode) {
	m.mode = mode
	log.Infof("ML integration mode set to: %d", mode)
}

// IsMLEnabled returns whether ML integration is enabled
func (m *MLIntegration) IsMLEnabled() bool {
	return m.mode != MLRiskModeDisabled && ml.IsEnabled()
}

// GetMLHealthStatus returns ML service health status
func (m *MLIntegration) GetMLHealthStatus(ctx context.Context) (*ml.ModelHealthResponse, error) {
	if !m.IsMLEnabled() {
		return &ml.ModelHealthResponse{
			Healthy: false,
		}, nil
	}

	return deployment.GetMLHealthStatus(ctx)
}

// TriggerMLTraining triggers ML model training with recent risk data
func (m *MLIntegration) TriggerMLTraining(ctx context.Context, trainingData []*ml.TrainingExample) (*ml.TrainingResponse, error) {
	if !m.IsMLEnabled() {
		return &ml.TrainingResponse{
			Success:      false,
			ErrorMessage: "ML integration is disabled",
		}, nil
	}

	client := ml.Singleton()
	return client.TrainModel(ctx, trainingData)
}

// GetMLModeString returns a string representation of the ML mode
func GetMLModeString(mode MLRiskMode) string {
	switch mode {
	case MLRiskModeDisabled:
		return "disabled"
	case MLRiskModeAugmented:
		return "augmented"
	case MLRiskModeReplacement:
		return "replacement"
	default:
		return "unknown"
	}
}

// ParseMLMode parses a string into an ML mode
func ParseMLMode(modeStr string) MLRiskMode {
	switch modeStr {
	case "disabled":
		return MLRiskModeDisabled
	case "augmented":
		return MLRiskModeAugmented
	case "replacement":
		return MLRiskModeReplacement
	default:
		return MLRiskModeDisabled
	}
}