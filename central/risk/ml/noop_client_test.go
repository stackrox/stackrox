package ml

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoOpClient_GetDeploymentRisk(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	deployment := &storage.Deployment{
		Id:        "test-deployment",
		Namespace: "test-ns",
	}

	images := []*storage.Image{
		{Id: "test-image"},
	}

	resp, err := client.GetDeploymentRisk(ctx, deployment, images)

	assert.NoError(t, err)
	assert.Nil(t, resp, "NoOp client should return nil response")
}

func TestNoOpClient_GetBatchDeploymentRisk(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	requests := []*DeploymentRiskRequest{
		{
			DeploymentID: "deployment-1",
			DeploymentFeatures: &DeploymentFeatures{
				Namespace: "test-ns",
			},
		},
		{
			DeploymentID: "deployment-2",
			DeploymentFeatures: &DeploymentFeatures{
				Namespace: "test-ns",
			},
		},
	}

	responses, err := client.GetBatchDeploymentRisk(ctx, requests)

	assert.NoError(t, err)
	assert.NotNil(t, responses)
	assert.Empty(t, responses, "NoOp client should return empty slice")
}

func TestNoOpClient_TrainModel(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	trainingData := []*TrainingExample{
		{
			DeploymentID:     "test-deployment",
			CurrentRiskScore: 7.5,
		},
	}

	resp, err := client.TrainModel(ctx, trainingData)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Equal(t, "ML Risk Service is disabled", resp.ErrorMessage)
}

func TestNoOpClient_GetModelHealth(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	resp, err := client.GetModelHealth(ctx)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Healthy, "NoOp client should report unhealthy status")
}

func TestNoOpClient_GetDetailedHealth(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	tests := []struct {
		name          string
		includeTrends bool
		trendHours    int
	}{
		{
			name:          "without trends",
			includeTrends: false,
			trendHours:    0,
		},
		{
			name:          "with trends",
			includeTrends: true,
			trendHours:    24,
		},
		{
			name:          "with custom trend hours",
			includeTrends: true,
			trendHours:    48,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.GetDetailedHealth(ctx, tt.includeTrends, tt.trendHours)

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, "error", resp.OverallStatus)
			assert.Equal(t, float32(0.0), resp.OverallScore)
			assert.Empty(t, resp.HealthChecks)
			assert.Contains(t, resp.Recommendations, "ML Risk Service is disabled")
			assert.NotNil(t, resp.Trends)
			assert.Empty(t, resp.Trends)
		})
	}
}

func TestNoOpClient_ReloadModel(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	tests := []struct {
		name        string
		modelID     string
		version     string
		forceReload bool
	}{
		{
			name:        "basic reload",
			modelID:     "test-model",
			version:     "v1.0.0",
			forceReload: false,
		},
		{
			name:        "force reload",
			modelID:     "test-model",
			version:     "v2.0.0",
			forceReload: true,
		},
		{
			name:        "empty model ID",
			modelID:     "",
			version:     "v1.0.0",
			forceReload: false,
		},
		{
			name:        "empty version",
			modelID:     "test-model",
			version:     "",
			forceReload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.ReloadModel(ctx, tt.modelID, tt.version, tt.forceReload)

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.False(t, resp.Success)
			assert.Equal(t, "ML Risk Service is disabled", resp.Message)
		})
	}
}

func TestNoOpClient_ListModels(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	tests := []struct {
		name    string
		modelID string
	}{
		{
			name:    "with model ID",
			modelID: "test-model",
		},
		{
			name:    "without model ID",
			modelID: "",
		},
		{
			name:    "with long model ID",
			modelID: "very-long-model-id-that-should-still-work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.ListModels(ctx, tt.modelID)

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.NotNil(t, resp.Models)
			assert.Empty(t, resp.Models, "NoOp client should return empty models list")
		})
	}
}

func TestNoOpClient_Close(t *testing.T) {
	client := &noOpClient{}

	err := client.Close()

	assert.NoError(t, err, "NoOp client Close should never return an error")
}

func TestNoOpClient_ConcurrentAccess(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	// Test concurrent access to ensure thread safety
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test various methods concurrently
			_, err1 := client.GetDeploymentRisk(ctx, &storage.Deployment{Id: "test"}, nil)
			assert.NoError(t, err1)

			_, err2 := client.GetModelHealth(ctx)
			assert.NoError(t, err2)

			_, err3 := client.GetDetailedHealth(ctx, true, 24)
			assert.NoError(t, err3)

			_, err4 := client.ReloadModel(ctx, "model", "v1", false)
			assert.NoError(t, err4)

			_, err5 := client.ListModels(ctx, "model")
			assert.NoError(t, err5)

			err6 := client.Close()
			assert.NoError(t, err6)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestNoOpClient_WithNilContext(t *testing.T) {
	client := &noOpClient{}

	// Test all methods with nil context (should not panic)
	_, err1 := client.GetDeploymentRisk(nil, &storage.Deployment{Id: "test"}, nil)
	assert.NoError(t, err1)

	_, err2 := client.GetBatchDeploymentRisk(nil, nil)
	assert.NoError(t, err2)

	_, err3 := client.TrainModel(nil, nil)
	assert.NoError(t, err3)

	_, err4 := client.GetModelHealth(nil)
	assert.NoError(t, err4)

	_, err5 := client.GetDetailedHealth(nil, false, 0)
	assert.NoError(t, err5)

	_, err6 := client.ReloadModel(nil, "", "", false)
	assert.NoError(t, err6)

	_, err7 := client.ListModels(nil, "")
	assert.NoError(t, err7)
}

func TestNoOpClient_WithCancelledContext(t *testing.T) {
	client := &noOpClient{}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test all methods with cancelled context (should not return context errors)
	_, err1 := client.GetDeploymentRisk(ctx, &storage.Deployment{Id: "test"}, nil)
	assert.NoError(t, err1)

	_, err2 := client.GetBatchDeploymentRisk(ctx, nil)
	assert.NoError(t, err2)

	_, err3 := client.TrainModel(ctx, nil)
	assert.NoError(t, err3)

	_, err4 := client.GetModelHealth(ctx)
	assert.NoError(t, err4)

	_, err5 := client.GetDetailedHealth(ctx, false, 0)
	assert.NoError(t, err5)

	_, err6 := client.ReloadModel(ctx, "", "", false)
	assert.NoError(t, err6)

	_, err7 := client.ListModels(ctx, "")
	assert.NoError(t, err7)
}

// Benchmark to ensure NoOp operations are fast
func BenchmarkNoOpClient_GetDeploymentRisk(b *testing.B) {
	client := &noOpClient{}
	ctx := context.Background()
	deployment := &storage.Deployment{Id: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetDeploymentRisk(ctx, deployment, nil)
	}
}

func BenchmarkNoOpClient_GetModelHealth(b *testing.B) {
	client := &noOpClient{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetModelHealth(ctx)
	}
}

func TestNoOpClient_ResponseConsistency(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	// Test that responses are consistent across multiple calls
	health1, _ := client.GetModelHealth(ctx)
	health2, _ := client.GetModelHealth(ctx)

	assert.Equal(t, health1.Healthy, health2.Healthy)

	detailedHealth1, _ := client.GetDetailedHealth(ctx, true, 24)
	detailedHealth2, _ := client.GetDetailedHealth(ctx, true, 24)

	assert.Equal(t, detailedHealth1.OverallStatus, detailedHealth2.OverallStatus)
	assert.Equal(t, detailedHealth1.OverallScore, detailedHealth2.OverallScore)

	training1, _ := client.TrainModel(ctx, nil)
	training2, _ := client.TrainModel(ctx, nil)

	assert.Equal(t, training1.Success, training2.Success)
	assert.Equal(t, training1.ErrorMessage, training2.ErrorMessage)
}
