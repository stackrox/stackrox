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

// Close does nothing
func (n *noOpClient) Close() error {
	return nil
}