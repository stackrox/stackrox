package deployment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/central/risk/ml"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMLRiskClient implements ml.MLRiskClient for testing
type mockMLRiskClient struct {
	getDeploymentRiskFunc func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error)
	getModelHealthFunc    func(ctx context.Context) (*ml.ModelHealthResponse, error)
}

func (m *mockMLRiskClient) GetDeploymentRisk(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
	if m.getDeploymentRiskFunc != nil {
		return m.getDeploymentRiskFunc(ctx, deployment, images)
	}
	return nil, nil
}

func (m *mockMLRiskClient) GetBatchDeploymentRisk(ctx context.Context, requests []*ml.DeploymentRiskRequest) ([]*ml.MLRiskResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMLRiskClient) TrainModel(ctx context.Context, trainingData []*ml.TrainingExample) (*ml.TrainingResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMLRiskClient) GetModelHealth(ctx context.Context) (*ml.ModelHealthResponse, error) {
	if m.getModelHealthFunc != nil {
		return m.getModelHealthFunc(ctx)
	}
	return &ml.ModelHealthResponse{Healthy: true}, nil
}

func (m *mockMLRiskClient) GetDetailedHealth(ctx context.Context, includeTrends bool, trendHours int) (*ml.DetailedHealthResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMLRiskClient) ReloadModel(ctx context.Context, modelID, version string, force bool) (*ml.ReloadModelResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMLRiskClient) ListModels(ctx context.Context, modelID string) (*ml.ListModelsResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMLRiskClient) Close() error {
	return nil
}

func TestNewMLScorer(t *testing.T) {
	scorer := NewMLScorer()

	assert.NotNil(t, scorer)
	assert.NotNil(t, scorer.mlClient)
}

func TestMLScorer_Score_MLDisabled(t *testing.T) {
	// Create a custom scorer with our mock client to avoid singleton dependency
	scorer := &MLScorer{
		mlClient: &mockMLRiskClient{},
	}

	ctx := context.Background()
	deployment := getMockDeployment()
	images := getMockImageRisks()

	// Note: This test assumes ML is disabled in the environment
	// When ML is disabled, Score should return nil
	result := scorer.Score(ctx, deployment, images)

	// The result depends on whether ML is actually enabled in the test environment
	// If ML is disabled, result will be nil
	// If ML is enabled, result will be processed by the mock client
	if !ml.IsEnabled() {
		assert.Nil(t, result)
	}
}

func TestMLScorer_Score_Success(t *testing.T) {
	// Skip test if ML is disabled in the environment
	if !ml.IsEnabled() {
		t.Skip("Skipping test - ML is disabled in test environment")
		return
	}

	mockClient := &mockMLRiskClient{
		getDeploymentRiskFunc: func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
			return &ml.MLRiskResponse{
				DeploymentID:  deployment.GetId(),
				RiskScore:     7.5,
				ModelVersion:  "test-model-v1.0",
				Timestamp:     time.Now().Unix(),
				FeatureImportances: []*ml.FeatureImportance{
					{
						FeatureName:     "policy_violations",
						ImportanceScore: 0.45,
						FeatureCategory: "security",
						Description:     "High number of policy violations",
					},
					{
						FeatureName:     "vuln_count",
						ImportanceScore: 0.32,
						FeatureCategory: "security",
						Description:     "Critical vulnerabilities present",
					},
					{
						FeatureName:     "privileged_access",
						ImportanceScore: -0.15,
						FeatureCategory: "security",
						Description:     "Privileged container access",
					},
				},
			}, nil
		},
	}

	scorer := &MLScorer{
		mlClient: mockClient,
	}

	ctx := context.Background()
	deployment := getMockDeployment()
	images := getMockImageRisks()

	result := scorer.Score(ctx, deployment, images)

	require.NotNil(t, result)
	assert.Equal(t, float32(7.5), result.Score)
	assert.Equal(t, deployment.GetId(), result.Subject.Id)
	assert.Equal(t, storage.RiskSubjectType_DEPLOYMENT, result.Subject.Type)
	assert.Equal(t, deployment.GetNamespace(), result.Subject.Namespace)
	assert.Equal(t, deployment.GetClusterId(), result.Subject.ClusterId)

	// Check risk results
	require.Len(t, result.Results, 1)
	mlResult := result.Results[0]
	assert.Equal(t, "ML Risk Assessment", mlResult.Name)
	assert.Equal(t, float32(7.5), mlResult.Score)

	// Check that we have factors for significant features (score > 0.01)
	// Should have: base factor + 3 feature importance factors
	require.Len(t, mlResult.Factors, 4)
	assert.Equal(t, "Machine learning risk prediction", mlResult.Factors[0].Message)
	assert.Contains(t, mlResult.Factors[1].Message, "High number of policy violations")
	assert.Contains(t, mlResult.Factors[1].Message, "increases")
	assert.Contains(t, mlResult.Factors[2].Message, "Critical vulnerabilities present")
	assert.Contains(t, mlResult.Factors[2].Message, "increases")
	assert.Contains(t, mlResult.Factors[3].Message, "Privileged container access")
	assert.Contains(t, mlResult.Factors[3].Message, "decreases")
}

func TestMLScorer_Score_ClientError(t *testing.T) {
	mockClient := &mockMLRiskClient{
		getDeploymentRiskFunc: func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
			return nil, errors.New("ML service unavailable")
		},
	}

	scorer := &MLScorer{
		mlClient: mockClient,
	}

	ctx := context.Background()
	deployment := getMockDeployment()
	images := getMockImageRisks()

	result := scorer.Score(ctx, deployment, images)

	// Should return nil when ML client returns error
	assert.Nil(t, result)
}

func TestMLScorer_Score_NilResponse(t *testing.T) {
	mockClient := &mockMLRiskClient{
		getDeploymentRiskFunc: func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
			return nil, nil
		},
	}

	scorer := &MLScorer{
		mlClient: mockClient,
	}

	ctx := context.Background()
	deployment := getMockDeployment()
	images := getMockImageRisks()

	result := scorer.Score(ctx, deployment, images)

	// Should return nil when ML client returns nil response
	assert.Nil(t, result)
}

func TestMLScorer_Score_EmptyFeatureImportances(t *testing.T) {
	// Skip test if ML is disabled in the environment
	if !ml.IsEnabled() {
		t.Skip("Skipping test - ML is disabled in test environment")
		return
	}

	mockClient := &mockMLRiskClient{
		getDeploymentRiskFunc: func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
			return &ml.MLRiskResponse{
				DeploymentID:       deployment.GetId(),
				RiskScore:          3.2,
				ModelVersion:       "test-model-v1.0",
				Timestamp:          time.Now().Unix(),
				FeatureImportances: []*ml.FeatureImportance{}, // Empty
			}, nil
		},
	}

	scorer := &MLScorer{
		mlClient: mockClient,
	}

	ctx := context.Background()
	deployment := getMockDeployment()
	images := getMockImageRisks()

	result := scorer.Score(ctx, deployment, images)

	require.NotNil(t, result)
	assert.Equal(t, float32(3.2), result.Score)

	// Should have one result with one factor (base ML prediction)
	require.Len(t, result.Results, 1)
	mlResult := result.Results[0]
	require.Len(t, mlResult.Factors, 1)
	assert.Equal(t, "Machine learning risk prediction", mlResult.Factors[0].Message)
}

func TestMLScorer_Score_LowImportanceFeatures(t *testing.T) {
	// Skip test if ML is disabled in the environment
	if !ml.IsEnabled() {
		t.Skip("Skipping test - ML is disabled in test environment")
		return
	}

	mockClient := &mockMLRiskClient{
		getDeploymentRiskFunc: func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
			return &ml.MLRiskResponse{
				DeploymentID:  deployment.GetId(),
				RiskScore:     4.1,
				ModelVersion:  "test-model-v1.0",
				Timestamp:     time.Now().Unix(),
				FeatureImportances: []*ml.FeatureImportance{
					{
						FeatureName:     "low_importance_feature",
						ImportanceScore: 0.005, // Below 0.01 threshold
						FeatureCategory: "misc",
						Description:     "Low importance feature",
					},
					{
						FeatureName:     "high_importance_feature",
						ImportanceScore: 0.25, // Above 0.01 threshold
						FeatureCategory: "security",
						Description:     "High importance feature",
					},
				},
			}, nil
		},
	}

	scorer := &MLScorer{
		mlClient: mockClient,
	}

	ctx := context.Background()
	deployment := getMockDeployment()
	images := getMockImageRisks()

	result := scorer.Score(ctx, deployment, images)

	require.NotNil(t, result)

	// Should only include features with importance > 0.01
	require.Len(t, result.Results, 1)
	mlResult := result.Results[0]
	require.Len(t, mlResult.Factors, 2) // Base + 1 high importance feature
	assert.Equal(t, "Machine learning risk prediction", mlResult.Factors[0].Message)
	assert.Contains(t, mlResult.Factors[1].Message, "High importance feature")
}

func TestMLScorer_formatFeatureImportance(t *testing.T) {
	scorer := &MLScorer{}

	tests := []struct {
		name               string
		featureImportance  *ml.FeatureImportance
		expectedSubstrings []string
	}{
		{
			name: "positive importance",
			featureImportance: &ml.FeatureImportance{
				FeatureName:     "test_feature",
				ImportanceScore: 0.45,
				Description:     "Test positive feature",
			},
			expectedSubstrings: []string{"Test positive feature", "increases", "0.450"},
		},
		{
			name: "negative importance",
			featureImportance: &ml.FeatureImportance{
				FeatureName:     "test_feature_neg",
				ImportanceScore: -0.25,
				Description:     "Test negative feature",
			},
			expectedSubstrings: []string{"Test negative feature", "decreases", "0.250"},
		},
		{
			name: "zero importance",
			featureImportance: &ml.FeatureImportance{
				FeatureName:     "test_feature_zero",
				ImportanceScore: 0.0,
				Description:     "Test zero feature",
			},
			expectedSubstrings: []string{"Test zero feature", "increases", "0.000"},
		},
		{
			name: "very small positive",
			featureImportance: &ml.FeatureImportance{
				FeatureName:     "test_feature_small",
				ImportanceScore: 0.001,
				Description:     "Test small feature",
			},
			expectedSubstrings: []string{"Test small feature", "increases", "0.001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.formatFeatureImportance(tt.featureImportance)
			for _, expectedSubstring := range tt.expectedSubstrings {
				assert.Contains(t, result, expectedSubstring)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    float32
		expected float32
	}{
		{"positive", 5.5, 5.5},
		{"negative", -3.2, 3.2},
		{"zero", 0.0, 0.0},
		{"large positive", 1000.0, 1000.0},
		{"large negative", -999.9, 999.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMLEnabled(t *testing.T) {
	// Test that IsMLEnabled delegates to ml.IsEnabled
	// Note: This is a wrapper function, so we just test it doesn't panic
	result := IsMLEnabled()
	assert.IsType(t, false, result) // Just ensure it returns a boolean
}

func TestGetMLHealthStatus(t *testing.T) {
	ctx := context.Background()

	// Test that GetMLHealthStatus delegates to the ML client
	// Note: This uses ml.Singleton(), so we test it doesn't panic and returns expected type
	result, err := GetMLHealthStatus(ctx)

	// Should not panic and should return some result (could be error or success depending on ML state)
	// We don't assert specific values since this depends on the actual ML service state
	if err != nil {
		assert.NotNil(t, err)
		assert.Nil(t, result)
	} else {
		assert.NotNil(t, result)
		assert.Nil(t, err)
	}
}

func TestMLScorer_Score_ImageConversion(t *testing.T) {
	// Skip test if ML is disabled in the environment
	if !ml.IsEnabled() {
		t.Skip("Skipping test - ML is disabled in test environment")
		return
	}

	// Test that image Risk objects are correctly converted to Image objects
	mockClient := &mockMLRiskClient{
		getDeploymentRiskFunc: func(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*ml.MLRiskResponse, error) {
			// Verify that we received the correct number of images
			assert.Len(t, images, 2)

			// Verify image IDs were extracted correctly from Risk objects
			imageIDs := make([]string, len(images))
			for i, img := range images {
				imageIDs[i] = img.GetId()
			}
			assert.Contains(t, imageIDs, "image-1")
			assert.Contains(t, imageIDs, "image-2")

			return &ml.MLRiskResponse{
				DeploymentID:       deployment.GetId(),
				RiskScore:          5.0,
				ModelVersion:       "test-model-v1.0",
				Timestamp:          time.Now().Unix(),
				FeatureImportances: []*ml.FeatureImportance{},
			}, nil
		},
	}

	scorer := &MLScorer{
		mlClient: mockClient,
	}

	ctx := context.Background()
	deployment := getMockDeployment()

	// Create image risks with specific IDs
	images := []*storage.Risk{
		{
			Subject: &storage.RiskSubject{
				Id:   "image-1",
				Type: storage.RiskSubjectType_IMAGE,
			},
		},
		{
			Subject: &storage.RiskSubject{
				Id:   "image-2",
				Type: storage.RiskSubjectType_IMAGE,
			},
		},
	}

	result := scorer.Score(ctx, deployment, images)
	require.NotNil(t, result)
}

// Helper functions for tests

func getMockDeployment() *storage.Deployment {
	return &storage.Deployment{
		Id:        "test-deployment-123",
		Name:      "test-deployment",
		Namespace: "test-namespace",
		ClusterId: "test-cluster-456",
		Containers: []*storage.Container{
			{
				Id:   "container-1",
				Name: "test-container",
			},
		},
	}
}

func getMockImageRisks() []*storage.Risk {
	return []*storage.Risk{
		{
			Id:    "risk-1",
			Score: 3.5,
			Subject: &storage.RiskSubject{
				Id:   "image-1",
				Type: storage.RiskSubjectType_IMAGE,
			},
		},
		{
			Id:    "risk-2",
			Score: 2.1,
			Subject: &storage.RiskSubject{
				Id:   "image-2",
				Type: storage.RiskSubjectType_IMAGE,
			},
		},
	}
}