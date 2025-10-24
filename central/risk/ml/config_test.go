package ml

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelStorageConfig_Defaults(t *testing.T) {
	// Clear all relevant environment variables
	envVars := []string{
		"ROX_ML_MODEL_STORAGE_BACKEND",
		"ROX_ML_MODEL_STORAGE_BASE_PATH",
		"ROX_ML_MODEL_BACKUP_ENABLED",
		"ROX_ML_MODEL_BACKUP_FREQUENCY",
		"ROX_ML_MODEL_ENCRYPTION_ENABLED",
		"ROX_ML_MODEL_COMPRESSION_ENABLED",
		"ROX_ML_MODEL_RETENTION_DAYS",
		"ROX_ML_MODEL_VERSIONING_ENABLED",
		"ROX_ML_MAX_MODEL_VERSIONS",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	config := GetModelStorageConfig()

	assert.Equal(t, "local", config.Backend)
	assert.Equal(t, "/app/models", config.BasePath)
	assert.False(t, config.BackupEnabled)
	assert.Equal(t, "daily", config.BackupFrequency)
	assert.False(t, config.EncryptionEnabled)
	assert.True(t, config.CompressionEnabled)
	assert.Equal(t, 0, config.RetentionDays)
	assert.True(t, config.VersioningEnabled)
	assert.Equal(t, 10, config.MaxVersions)
	assert.Nil(t, config.GCSConfig)
}

func TestGetModelStorageConfig_CustomValues(t *testing.T) {
	os.Setenv("ROX_ML_MODEL_STORAGE_BACKEND", "gcs")
	os.Setenv("ROX_ML_MODEL_STORAGE_BASE_PATH", "/custom/path")
	os.Setenv("ROX_ML_MODEL_BACKUP_ENABLED", "true")
	os.Setenv("ROX_ML_MODEL_BACKUP_FREQUENCY", "weekly")
	os.Setenv("ROX_ML_MODEL_ENCRYPTION_ENABLED", "true")
	os.Setenv("ROX_ML_MODEL_COMPRESSION_ENABLED", "false")
	os.Setenv("ROX_ML_MODEL_RETENTION_DAYS", "30")
	os.Setenv("ROX_ML_MODEL_VERSIONING_ENABLED", "false")
	os.Setenv("ROX_ML_MAX_MODEL_VERSIONS", "5")

	defer func() {
		os.Unsetenv("ROX_ML_MODEL_STORAGE_BACKEND")
		os.Unsetenv("ROX_ML_MODEL_STORAGE_BASE_PATH")
		os.Unsetenv("ROX_ML_MODEL_BACKUP_ENABLED")
		os.Unsetenv("ROX_ML_MODEL_BACKUP_FREQUENCY")
		os.Unsetenv("ROX_ML_MODEL_ENCRYPTION_ENABLED")
		os.Unsetenv("ROX_ML_MODEL_COMPRESSION_ENABLED")
		os.Unsetenv("ROX_ML_MODEL_RETENTION_DAYS")
		os.Unsetenv("ROX_ML_MODEL_VERSIONING_ENABLED")
		os.Unsetenv("ROX_ML_MAX_MODEL_VERSIONS")
	}()

	config := GetModelStorageConfig()

	assert.Equal(t, "gcs", config.Backend)
	assert.Equal(t, "/custom/path", config.BasePath)
	assert.True(t, config.BackupEnabled)
	assert.Equal(t, "weekly", config.BackupFrequency)
	assert.True(t, config.EncryptionEnabled)
	assert.False(t, config.CompressionEnabled)
	assert.Equal(t, 30, config.RetentionDays)
	assert.False(t, config.VersioningEnabled)
	assert.Equal(t, 5, config.MaxVersions)
	assert.Nil(t, config.GCSConfig) // Should be nil since GCS env vars not set
}

func TestGetModelStorageConfig_GCSBackend(t *testing.T) {
	os.Setenv("ROX_ML_MODEL_STORAGE_BACKEND", "gcs")
	os.Setenv("ROX_ML_GCS_PROJECT_ID", "test-project")
	os.Setenv("ROX_ML_GCS_CREDENTIALS_PATH", "/path/to/credentials.json")
	os.Setenv("ROX_ML_GCS_BUCKET_NAME", "test-bucket")

	defer func() {
		os.Unsetenv("ROX_ML_MODEL_STORAGE_BACKEND")
		os.Unsetenv("ROX_ML_GCS_PROJECT_ID")
		os.Unsetenv("ROX_ML_GCS_CREDENTIALS_PATH")
		os.Unsetenv("ROX_ML_GCS_BUCKET_NAME")
	}()

	config := GetModelStorageConfig()

	assert.Equal(t, "gcs", config.Backend)
	require.NotNil(t, config.GCSConfig)
	assert.Equal(t, "test-project", config.GCSConfig.ProjectID)
	assert.Equal(t, "/path/to/credentials.json", config.GCSConfig.CredentialsPath)
	assert.Equal(t, "test-bucket", config.GCSConfig.BucketName)
}

func TestGetModelStorageConfig_NonGCSBackend(t *testing.T) {
	os.Setenv("ROX_ML_MODEL_STORAGE_BACKEND", "local")
	os.Setenv("ROX_ML_GCS_PROJECT_ID", "test-project")
	os.Setenv("ROX_ML_GCS_CREDENTIALS_PATH", "/path/to/credentials.json")
	os.Setenv("ROX_ML_GCS_BUCKET_NAME", "test-bucket")

	defer func() {
		os.Unsetenv("ROX_ML_MODEL_STORAGE_BACKEND")
		os.Unsetenv("ROX_ML_GCS_PROJECT_ID")
		os.Unsetenv("ROX_ML_GCS_CREDENTIALS_PATH")
		os.Unsetenv("ROX_ML_GCS_BUCKET_NAME")
	}()

	config := GetModelStorageConfig()

	assert.Equal(t, "local", config.Backend)
	assert.Nil(t, config.GCSConfig, "GCS config should be nil for non-GCS backends")
}

func TestGetModelDeploymentConfig_Defaults(t *testing.T) {
	// Clear all relevant environment variables
	envVars := []string{
		"ROX_ML_MODEL_AUTO_DEPLOY_ENABLED",
		"ROX_ML_MODEL_DEPLOYMENT_THRESHOLD",
		"ROX_ML_MODEL_HEALTH_CHECK_ENABLED",
		"ROX_ML_MODEL_HEALTH_CHECK_INTERVAL",
		"ROX_ML_MODEL_DRIFT_DETECTION_ENABLED",
		"ROX_ML_MODEL_DRIFT_THRESHOLD",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	config := GetModelDeploymentConfig()

	assert.False(t, config.AutoDeployEnabled)
	assert.Equal(t, 0.85, config.DeploymentThreshold)
	assert.True(t, config.HealthCheckEnabled)
	assert.Equal(t, 5*time.Minute, config.HealthCheckInterval)
	assert.False(t, config.DriftDetectionEnabled)
	assert.Equal(t, 0.1, config.DriftThreshold)
}

func TestGetModelDeploymentConfig_CustomValues(t *testing.T) {
	os.Setenv("ROX_ML_MODEL_AUTO_DEPLOY_ENABLED", "true")
	os.Setenv("ROX_ML_MODEL_DEPLOYMENT_THRESHOLD", "0.95")
	os.Setenv("ROX_ML_MODEL_HEALTH_CHECK_ENABLED", "false")
	os.Setenv("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL", "10m")
	os.Setenv("ROX_ML_MODEL_DRIFT_DETECTION_ENABLED", "true")
	os.Setenv("ROX_ML_MODEL_DRIFT_THRESHOLD", "0.2")

	defer func() {
		os.Unsetenv("ROX_ML_MODEL_AUTO_DEPLOY_ENABLED")
		os.Unsetenv("ROX_ML_MODEL_DEPLOYMENT_THRESHOLD")
		os.Unsetenv("ROX_ML_MODEL_HEALTH_CHECK_ENABLED")
		os.Unsetenv("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL")
		os.Unsetenv("ROX_ML_MODEL_DRIFT_DETECTION_ENABLED")
		os.Unsetenv("ROX_ML_MODEL_DRIFT_THRESHOLD")
	}()

	config := GetModelDeploymentConfig()

	assert.True(t, config.AutoDeployEnabled)
	assert.Equal(t, 0.95, config.DeploymentThreshold)
	assert.False(t, config.HealthCheckEnabled)
	assert.Equal(t, 10*time.Minute, config.HealthCheckInterval)
	assert.True(t, config.DriftDetectionEnabled)
	assert.Equal(t, 0.2, config.DriftThreshold)
}

func TestGetModelDeploymentConfig_InvalidHealthCheckInterval(t *testing.T) {
	os.Setenv("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL", "invalid-duration")
	defer os.Unsetenv("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL")

	config := GetModelDeploymentConfig()

	// Should fall back to default 5 minutes
	assert.Equal(t, 5*time.Minute, config.HealthCheckInterval)
}

func TestGetModelDeploymentConfig_InvalidFloatValues(t *testing.T) {
	os.Setenv("ROX_ML_MODEL_DEPLOYMENT_THRESHOLD", "invalid-float")
	os.Setenv("ROX_ML_MODEL_DRIFT_THRESHOLD", "not-a-number")

	defer func() {
		os.Unsetenv("ROX_ML_MODEL_DEPLOYMENT_THRESHOLD")
		os.Unsetenv("ROX_ML_MODEL_DRIFT_THRESHOLD")
	}()

	config := GetModelDeploymentConfig()

	// Should fall back to 0.0 when parsing fails
	assert.Equal(t, 0.0, config.DeploymentThreshold)
	assert.Equal(t, 0.0, config.DriftThreshold)
}

func TestModelStorageConfig_Validate_Local(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:  "local",
		BasePath: "/app/models",
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestModelStorageConfig_Validate_GCS_Valid(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:  "gcs",
		BasePath: "/models",
		GCSConfig: &GCSConfig{
			ProjectID:       "test-project",
			CredentialsPath: "/path/to/creds.json",
			BucketName:      "test-bucket",
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestModelStorageConfig_Validate_EmptyBackend(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:  "",
		BasePath: "/app/models",
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage backend is required")
}

func TestModelStorageConfig_Validate_EmptyBasePath(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:  "local",
		BasePath: "",
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base path is required")
}

func TestModelStorageConfig_Validate_UnsupportedBackend(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:  "unsupported",
		BasePath: "/app/models",
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported storage backend")
}

func TestModelStorageConfig_Validate_GCS_NoConfig(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:   "gcs",
		BasePath:  "/models",
		GCSConfig: nil,
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GCS configuration is required")
}

func TestModelStorageConfig_Validate_GCS_NoBucket(t *testing.T) {
	config := &ModelStorageConfig{
		Backend:  "gcs",
		BasePath: "/models",
		GCSConfig: &GCSConfig{
			ProjectID:       "test-project",
			CredentialsPath: "/path/to/creds.json",
			BucketName:      "",
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GCS bucket name is required")
}

func TestModelStorageConfig_ValidateSupportedBackends(t *testing.T) {
	supportedBackends := []string{"local", "gcs"}

	for _, backend := range supportedBackends {
		t.Run(backend, func(t *testing.T) {
			config := &ModelStorageConfig{
				Backend:  backend,
				BasePath: "/test/path",
			}

			if backend == "gcs" {
				config.GCSConfig = &GCSConfig{
					ProjectID:       "test-project",
					CredentialsPath: "/creds.json",
					BucketName:      "test-bucket",
				}
			}

			err := config.Validate()
			assert.NoError(t, err, "Backend %s should be supported", backend)
		})
	}
}

func TestEnvironmentVariableParsing(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		value    string
		testFunc func(*testing.T)
	}{
		{
			name:   "boolean true variations",
			envVar: "ROX_ML_MODEL_BACKUP_ENABLED",
			value:  "true",
			testFunc: func(t *testing.T) {
				config := GetModelStorageConfig()
				assert.True(t, config.BackupEnabled)
			},
		},
		{
			name:   "boolean false variations",
			envVar: "ROX_ML_MODEL_BACKUP_ENABLED",
			value:  "false",
			testFunc: func(t *testing.T) {
				config := GetModelStorageConfig()
				assert.False(t, config.BackupEnabled)
			},
		},
		{
			name:   "integer parsing",
			envVar: "ROX_ML_MODEL_RETENTION_DAYS",
			value:  "45",
			testFunc: func(t *testing.T) {
				config := GetModelStorageConfig()
				assert.Equal(t, 45, config.RetentionDays)
			},
		},
		{
			name:   "string values",
			envVar: "ROX_ML_MODEL_BACKUP_FREQUENCY",
			value:  "hourly",
			testFunc: func(t *testing.T) {
				config := GetModelStorageConfig()
				assert.Equal(t, "hourly", config.BackupFrequency)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.value)
			defer os.Unsetenv(tt.envVar)

			tt.testFunc(t)
		})
	}
}

func TestHealthCheckIntervalParsing(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{
			name:     "valid seconds",
			value:    "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "valid minutes",
			value:    "2m",
			expected: 2 * time.Minute,
		},
		{
			name:     "valid hours",
			value:    "1h",
			expected: 1 * time.Hour,
		},
		{
			name:     "invalid format",
			value:    "invalid",
			expected: 5 * time.Minute, // default
		},
		{
			name:     "empty string",
			value:    "",
			expected: 5 * time.Minute, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL", tt.value)
			defer os.Unsetenv("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL")

			config := GetModelDeploymentConfig()
			assert.Equal(t, tt.expected, config.HealthCheckInterval)
		})
	}
}

// Benchmark configuration loading
func BenchmarkGetModelStorageConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetModelStorageConfig()
	}
}

func BenchmarkGetModelDeploymentConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetModelDeploymentConfig()
	}
}

func TestConfigurationConsistency(t *testing.T) {
	// Test that multiple calls return consistent configurations
	config1 := GetModelStorageConfig()
	config2 := GetModelStorageConfig()

	assert.Equal(t, config1.Backend, config2.Backend)
	assert.Equal(t, config1.BasePath, config2.BasePath)
	assert.Equal(t, config1.EncryptionEnabled, config2.EncryptionEnabled)
	assert.Equal(t, config1.CompressionEnabled, config2.CompressionEnabled)

	deployConfig1 := GetModelDeploymentConfig()
	deployConfig2 := GetModelDeploymentConfig()

	assert.Equal(t, deployConfig1.AutoDeployEnabled, deployConfig2.AutoDeployEnabled)
	assert.Equal(t, deployConfig1.DeploymentThreshold, deployConfig2.DeploymentThreshold)
	assert.Equal(t, deployConfig1.HealthCheckEnabled, deployConfig2.HealthCheckEnabled)
}
