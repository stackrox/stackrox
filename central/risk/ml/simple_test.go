package ml

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestConfig_BasicStructure(t *testing.T) {
	config := &Config{
		Endpoint:   "localhost:8080",
		TLSEnabled: false,
		Timeout:    30 * time.Second,
	}

	assert.Equal(t, "localhost:8080", config.Endpoint)
	assert.False(t, config.TLSEnabled)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "ml-risk-service:8080", config.Endpoint)
	assert.False(t, config.TLSEnabled)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

func TestMLRiskResponse_BasicStructure(t *testing.T) {
	response := &MLRiskResponse{
		DeploymentID: "test-deployment",
		RiskScore:    7.5,
		ModelVersion: "v1.2.3",
		Timestamp:    time.Now().Unix(),
	}

	assert.Equal(t, "test-deployment", response.DeploymentID)
	assert.Equal(t, float32(7.5), response.RiskScore)
	assert.Equal(t, "v1.2.3", response.ModelVersion)
	assert.Greater(t, response.Timestamp, int64(0))
}

func TestDeploymentRiskRequest_Creation(t *testing.T) {
	request := &DeploymentRiskRequest{
		DeploymentID: "test-deployment",
		DeploymentFeatures: &DeploymentFeatures{
			Namespace:    "test-ns",
			ReplicaCount: 3,
		},
	}

	assert.Equal(t, "test-deployment", request.DeploymentID)
	assert.NotNil(t, request.DeploymentFeatures)
	assert.Equal(t, "test-ns", request.DeploymentFeatures.Namespace)
	assert.Equal(t, int32(3), request.DeploymentFeatures.ReplicaCount)
}

func TestTrainingExample_Creation(t *testing.T) {
	example := &TrainingExample{
		DeploymentID: "test-deployment",
		DeploymentFeatures: &DeploymentFeatures{
			Namespace:    "test-ns",
			ReplicaCount: 3,
		},
		CurrentRiskScore: 7.8,
	}

	assert.Equal(t, "test-deployment", example.DeploymentID)
	assert.NotNil(t, example.DeploymentFeatures)
	assert.Equal(t, float32(7.8), example.CurrentRiskScore)
}

func TestModelHealthResponse_Creation(t *testing.T) {
	response := &ModelHealthResponse{
		Healthy:      true,
		ModelVersion: "v1.0.0",
	}

	assert.True(t, response.Healthy)
	assert.Equal(t, "v1.0.0", response.ModelVersion)
}

func TestFeatureImportance_Creation(t *testing.T) {
	feature := &FeatureImportance{
		FeatureName:     "test_feature",
		ImportanceScore: 0.85,
		FeatureCategory: "security",
		Description:     "Test feature for security assessment",
	}

	assert.Equal(t, "test_feature", feature.FeatureName)
	assert.Equal(t, float32(0.85), feature.ImportanceScore)
	assert.Equal(t, "security", feature.FeatureCategory)
	assert.Equal(t, "Test feature for security assessment", feature.Description)
}

func TestDeploymentFeatures_Creation(t *testing.T) {
	features := &DeploymentFeatures{
		PolicyViolationCount:         5,
		PolicyViolationSeverityScore: 7.5,
		HostNetwork:                  false,
		HostPID:                      false,
		HostIPC:                      false,
		PrivilegedContainerCount:     0,
		AutomountServiceAccountToken: true,
		ExposedPortCount:             2,
		HasExternalExposure:          true,
		ReplicaCount:                 3,
		IsOrchestratorComponent:      false,
		IsPlatformComponent:          false,
		Namespace:                    "test-ns",
		IsInactive:                   false,
	}

	assert.Equal(t, int32(5), features.PolicyViolationCount)
	assert.Equal(t, float32(7.5), features.PolicyViolationSeverityScore)
	assert.False(t, features.HostNetwork)
	assert.True(t, features.AutomountServiceAccountToken)
	assert.Equal(t, int32(3), features.ReplicaCount)
	assert.Equal(t, "test-ns", features.Namespace)
}

func TestImageFeatures_Creation(t *testing.T) {
	features := &ImageFeatures{
		ImageID:             "test-image-id",
		ImageName:           "test-image:latest",
		CriticalVulnCount:   2,
		HighVulnCount:       5,
		MediumVulnCount:     10,
		LowVulnCount:        20,
		AvgCVSSScore:        6.5,
		MaxCVSSScore:        9.2,
		TotalComponentCount: 100,
		RiskyComponentCount: 15,
		IsClusterLocal:      false,
		BaseImage:           "ubuntu:20.04",
		LayerCount:          5,
	}

	assert.Equal(t, "test-image-id", features.ImageID)
	assert.Equal(t, "test-image:latest", features.ImageName)
	assert.Equal(t, int32(2), features.CriticalVulnCount)
	assert.Equal(t, float32(6.5), features.AvgCVSSScore)
	assert.Equal(t, int32(100), features.TotalComponentCount)
}

func TestTrainingResponse_Creation(t *testing.T) {
	metrics := &TrainingMetrics{
		ValidationNDCG:  0.85,
		ValidationAUC:   0.92,
		TrainingLoss:    0.15,
		EpochsCompleted: 50,
	}

	response := &TrainingResponse{
		Success:      true,
		ModelVersion: "v2.0.0",
		Metrics:      metrics,
		ErrorMessage: "",
	}

	assert.True(t, response.Success)
	assert.Equal(t, "v2.0.0", response.ModelVersion)
	assert.NotNil(t, response.Metrics)
	assert.Equal(t, float32(0.85), response.Metrics.ValidationNDCG)
	assert.Empty(t, response.ErrorMessage)
}

func TestNoOpClient_Interface(t *testing.T) {
	// Test that noOpClient implements MLRiskClient interface
	var _ MLRiskClient = &noOpClient{}
}

func TestNoOpClient_BasicOperations(t *testing.T) {
	client := &noOpClient{}
	ctx := context.Background()

	deployment := &storage.Deployment{
		Id:        "test-deployment",
		Namespace: "test-ns",
	}

	// Test GetDeploymentRisk
	resp, err := client.GetDeploymentRisk(ctx, deployment, nil)
	assert.NoError(t, err)
	assert.Nil(t, resp)

	// Test GetModelHealth
	health, err := client.GetModelHealth(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.False(t, health.Healthy)

	// Test Close
	err = client.Close()
	assert.NoError(t, err)
}

func TestSingleton_BasicFunctionality(t *testing.T) {
	// Test that Singleton returns something without error
	client := Singleton()
	assert.NotNil(t, client)

	// Test IsEnabled
	enabled := IsEnabled()
	assert.IsType(t, false, enabled) // Just test it returns a boolean

	// Test Reset doesn't panic
	Reset()
}

func TestIsEnabled_EnvironmentBased(t *testing.T) {
	// Test with disabled
	t.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "false")
	Reset()
	assert.False(t, IsEnabled())

	// Test with enabled
	t.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "true")
	Reset()
	assert.True(t, IsEnabled())
}
