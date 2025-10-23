package ml

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/env"
)

// Storage configuration environment variables
var (
	// Model storage backend (local, s3, gcs, azure)
	ModelStorageBackend = env.RegisterSetting("ROX_ML_MODEL_STORAGE_BACKEND", env.WithDefault("local"))

	// Base path for model storage
	ModelStorageBasePath = env.RegisterSetting("ROX_ML_MODEL_STORAGE_BASE_PATH", env.WithDefault("/app/models"))

	// Enable model backup
	ModelBackupEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_BACKUP_ENABLED", false)

	// Model backup frequency (hourly, daily, weekly)
	ModelBackupFrequency = env.RegisterSetting("ROX_ML_MODEL_BACKUP_FREQUENCY", env.WithDefault("daily"))

	// Enable model encryption at rest
	ModelEncryptionEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_ENCRYPTION_ENABLED", false)

	// Enable model compression
	ModelCompressionEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_COMPRESSION_ENABLED", true)

	// Model retention policy (in days, 0 = keep forever)
	ModelRetentionDays = env.RegisterIntegerSetting("ROX_ML_MODEL_RETENTION_DAYS", 0)

	// GCS configuration
	GCSProjectID = env.RegisterSetting("ROX_ML_GCS_PROJECT_ID")
	GCSCredentialsPath = env.RegisterSetting("ROX_ML_GCS_CREDENTIALS_PATH")
	GCSBucketName = env.RegisterSetting("ROX_ML_GCS_BUCKET_NAME")

	// Model versioning settings
	ModelVersioningEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_VERSIONING_ENABLED", true)
	MaxModelVersions = env.RegisterIntegerSetting("ROX_ML_MAX_MODEL_VERSIONS", 10)

	// Auto-deployment settings
	ModelAutoDeployEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_AUTO_DEPLOY_ENABLED", false)
	ModelDeploymentThreshold = env.RegisterFloatSetting("ROX_ML_MODEL_DEPLOYMENT_THRESHOLD", 0.85)

	// Model health check settings
	ModelHealthCheckEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_HEALTH_CHECK_ENABLED", true)
	ModelHealthCheckInterval = env.RegisterSetting("ROX_ML_MODEL_HEALTH_CHECK_INTERVAL", env.WithDefault("5m"))

	// Model monitoring settings
	ModelDriftDetectionEnabled = env.RegisterBooleanSetting("ROX_ML_MODEL_DRIFT_DETECTION_ENABLED", false)
	ModelDriftThreshold = env.RegisterFloatSetting("ROX_ML_MODEL_DRIFT_THRESHOLD", 0.1)
)

// ModelStorageConfig provides configuration for model storage
type ModelStorageConfig struct {
	Backend             string
	BasePath            string
	EncryptionEnabled   bool
	CompressionEnabled  bool
	BackupEnabled       bool
	BackupFrequency     string
	RetentionDays       int
	VersioningEnabled   bool
	MaxVersions         int

	// Cloud-specific configurations
	GCSConfig   *GCSConfig
}

// GCSConfig provides Google Cloud Storage configuration
type GCSConfig struct {
	ProjectID       string
	CredentialsPath string
	BucketName      string
}

// ModelDeploymentConfig provides configuration for model deployment
type ModelDeploymentConfig struct {
	AutoDeployEnabled    bool
	DeploymentThreshold  float64
	HealthCheckEnabled   bool
	HealthCheckInterval  time.Duration
	DriftDetectionEnabled bool
	DriftThreshold       float64
}

// GetModelStorageConfig creates a ModelStorageConfig from environment variables
func GetModelStorageConfig() *ModelStorageConfig {
	config := &ModelStorageConfig{
		Backend:             ModelStorageBackend.Setting(),
		BasePath:            ModelStorageBasePath.Setting(),
		EncryptionEnabled:   ModelEncryptionEnabled.BooleanSetting(),
		CompressionEnabled:  ModelCompressionEnabled.BooleanSetting(),
		BackupEnabled:       ModelBackupEnabled.BooleanSetting(),
		BackupFrequency:     ModelBackupFrequency.Setting(),
		RetentionDays:       ModelRetentionDays.IntegerSetting(),
		VersioningEnabled:   ModelVersioningEnabled.BooleanSetting(),
		MaxVersions:         MaxModelVersions.IntegerSetting(),
	}

	// Configure cloud storage based on backend
	if config.Backend == "gcs" {
		config.GCSConfig = &GCSConfig{
			ProjectID:       GCSProjectID.Setting(),
			CredentialsPath: GCSCredentialsPath.Setting(),
			BucketName:      GCSBucketName.Setting(),
		}
	}

	return config
}

// GetModelDeploymentConfig creates a ModelDeploymentConfig from environment variables
func GetModelDeploymentConfig() *ModelDeploymentConfig {
	healthCheckInterval, err := time.ParseDuration(ModelHealthCheckInterval.Setting())
	if err != nil {
		healthCheckInterval = 5 * time.Minute
	}

	return &ModelDeploymentConfig{
		AutoDeployEnabled:     ModelAutoDeployEnabled.BooleanSetting(),
		DeploymentThreshold:   ModelDeploymentThreshold.FloatSetting(),
		HealthCheckEnabled:    ModelHealthCheckEnabled.BooleanSetting(),
		HealthCheckInterval:   healthCheckInterval,
		DriftDetectionEnabled: ModelDriftDetectionEnabled.BooleanSetting(),
		DriftThreshold:        ModelDriftThreshold.FloatSetting(),
	}
}

// Validate validates the storage configuration
func (c *ModelStorageConfig) Validate() error {
	if c.Backend == "" {
		return fmt.Errorf("storage backend is required")
	}

	if c.BasePath == "" {
		return fmt.Errorf("base path is required")
	}

	switch c.Backend {
	case "local":
		// No additional validation needed for local storage
	case "gcs":
		if c.GCSConfig == nil {
			return fmt.Errorf("GCS configuration is required for GCS backend")
		}
		if c.GCSConfig.BucketName == "" {
			return fmt.Errorf("GCS bucket name is required")
		}
	default:
		return fmt.Errorf("unsupported storage backend: %s (supported: local, gcs)", c.Backend)
	}

	return nil
}