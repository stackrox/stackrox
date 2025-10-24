package ml

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// noOpClient is a no-operation implementation of MLRiskClient
// Used when ML service is disabled or unavailable
type noOpClient struct{}

// GetDeploymentRisk returns nil (no ML risk assessment)
func (n *noOpClient) GetDeploymentRisk(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*MLRiskResponse, error) {
	return nil, nil
}

// GetBatchDeploymentRisk returns empty slice
func (n *noOpClient) GetBatchDeploymentRisk(ctx context.Context, requests []*DeploymentRiskRequest) ([]*MLRiskResponse, error) {
	return []*MLRiskResponse{}, nil
}

// TrainModel returns success without training
func (n *noOpClient) TrainModel(ctx context.Context, trainingData []*TrainingExample) (*TrainingResponse, error) {
	return &TrainingResponse{
		Success:      false,
		ErrorMessage: "ML Risk Service is disabled",
	}, nil
}

// GetModelHealth returns unhealthy status
func (n *noOpClient) GetModelHealth(ctx context.Context) (*ModelHealthResponse, error) {
	return &ModelHealthResponse{
		Healthy: false,
	}, nil
}

// GetDetailedHealth returns unhealthy status with no trends
func (n *noOpClient) GetDetailedHealth(ctx context.Context, includeTrends bool, trendHours int) (*DetailedHealthResponse, error) {
	return &DetailedHealthResponse{
		OverallStatus: "error",
		OverallScore:  0.0,
		HealthChecks:  []*HealthCheckDetail{},
		Recommendations: []string{"ML Risk Service is disabled"},
		Trends:        map[string]interface{}{},
	}, nil
}

// ReloadModel returns error indicating service is disabled
func (n *noOpClient) ReloadModel(ctx context.Context, modelID, version string, forceReload bool) (*ReloadModelResponse, error) {
	return &ReloadModelResponse{
		Success: false,
		Message: "ML Risk Service is disabled",
	}, nil
}

// ListModels returns empty models list
func (n *noOpClient) ListModels(ctx context.Context, modelID string) (*ListModelsResponse, error) {
	return &ListModelsResponse{
		Models: []*ModelInfo{},
	}, nil
}

// Close does nothing
func (n *noOpClient) Close() error {
	return nil
}