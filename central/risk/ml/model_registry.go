package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

// ModelStatus represents the status of a model
type ModelStatus string

const (
	ModelStatusDraft     ModelStatus = "draft"
	ModelStatusTraining  ModelStatus = "training"
	ModelStatusReady     ModelStatus = "ready"
	ModelStatusDeployed  ModelStatus = "deployed"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusFailed    ModelStatus = "failed"
)

// ModelMetadata represents ML model metadata stored in Central
type ModelMetadata struct {
	ID                   string                 `json:"id" db:"id"`
	ModelID              string                 `json:"model_id" db:"model_id"`
	Version              string                 `json:"version" db:"version"`
	Algorithm            string                 `json:"algorithm" db:"algorithm"`
	FeatureCount         int                    `json:"feature_count" db:"feature_count"`
	TrainingTimestamp    time.Time              `json:"training_timestamp" db:"training_timestamp"`
	ModelSizeBytes       int64                  `json:"model_size_bytes" db:"model_size_bytes"`
	Checksum             string                 `json:"checksum" db:"checksum"`
	PerformanceMetrics   map[string]interface{} `json:"performance_metrics" db:"performance_metrics"`
	Config               map[string]interface{} `json:"config" db:"config"`
	Tags                 map[string]string      `json:"tags" db:"tags"`
	Description          string                 `json:"description" db:"description"`
	CreatedBy            string                 `json:"created_by" db:"created_by"`
	Status               ModelStatus            `json:"status" db:"status"`
	StorageBackend       string                 `json:"storage_backend" db:"storage_backend"`
	StoragePath          string                 `json:"storage_path" db:"storage_path"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
	DeployedAt           *time.Time             `json:"deployed_at,omitempty" db:"deployed_at"`
	LastUsedAt           *time.Time             `json:"last_used_at,omitempty" db:"last_used_at"`
	UsageCount           int64                  `json:"usage_count" db:"usage_count"`
}

// ModelRegistryStore provides database operations for model metadata
type ModelRegistryStore interface {
	RegisterModel(ctx context.Context, metadata *ModelMetadata) error
	GetModel(ctx context.Context, modelID, version string) (*ModelMetadata, bool, error)
	GetLatestModel(ctx context.Context, modelID string) (*ModelMetadata, bool, error)
	ListModels(ctx context.Context, modelID string) ([]*ModelMetadata, error)
	ListAllModels(ctx context.Context) ([]*ModelMetadata, error)
	UpdateModelStatus(ctx context.Context, modelID, version string, status ModelStatus) error
	DeleteModel(ctx context.Context, modelID, version string) error
	GetDeployedModel(ctx context.Context) (*ModelMetadata, bool, error)
	SetDeployedModel(ctx context.Context, modelID, version string) error
	UpdateUsageStats(ctx context.Context, modelID, version string) error
	GetModelStats(ctx context.Context) (*ModelRegistryStats, error)
}

// ModelRegistryStats provides statistics about the model registry
type ModelRegistryStats struct {
	TotalModels        int       `json:"total_models"`
	TotalVersions      int       `json:"total_versions"`
	DeployedModel      string    `json:"deployed_model,omitempty"`
	DeployedVersion    string    `json:"deployed_version,omitempty"`
	LastTrainingTime   time.Time `json:"last_training_time"`
	TotalStorageBytes  int64     `json:"total_storage_bytes"`
	ModelsbyAlgorithm  map[string]int `json:"models_by_algorithm"`
	ModelsByStatus     map[string]int `json:"models_by_status"`
}

// modelRegistryStoreImpl implements ModelRegistryStore using PostgreSQL
type modelRegistryStoreImpl struct {
	db postgres.DB
}

// NewModelRegistryStore creates a new model registry store
func NewModelRegistryStore(db postgres.DB) ModelRegistryStore {
	return &modelRegistryStoreImpl{
		db: db,
	}
}

// RegisterModel registers a new model or model version
func (s *modelRegistryStoreImpl) RegisterModel(ctx context.Context, metadata *ModelMetadata) error {
	if metadata.ID == "" {
		metadata.ID = uuid.NewV4().String()
	}

	now := time.Now()
	metadata.CreatedAt = now
	metadata.UpdatedAt = now

	if metadata.Status == "" {
		metadata.Status = ModelStatusReady
	}

	// Convert maps to JSON for storage
	performanceMetricsJSON, err := json.Marshal(metadata.PerformanceMetrics)
	if err != nil {
		return errors.Wrap(err, "failed to marshal performance metrics")
	}

	configJSON, err := json.Marshal(metadata.Config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	tagsJSON, err := json.Marshal(metadata.Tags)
	if err != nil {
		return errors.Wrap(err, "failed to marshal tags")
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO ml_model_registry
		(id, model_id, version, algorithm, feature_count, training_timestamp,
		 model_size_bytes, checksum, performance_metrics, config, tags,
		 description, created_by, status, storage_backend, storage_path,
		 created_at, updated_at, usage_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (model_id, version) DO UPDATE SET
			algorithm = EXCLUDED.algorithm,
			feature_count = EXCLUDED.feature_count,
			training_timestamp = EXCLUDED.training_timestamp,
			model_size_bytes = EXCLUDED.model_size_bytes,
			checksum = EXCLUDED.checksum,
			performance_metrics = EXCLUDED.performance_metrics,
			config = EXCLUDED.config,
			tags = EXCLUDED.tags,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			storage_backend = EXCLUDED.storage_backend,
			storage_path = EXCLUDED.storage_path,
			updated_at = $18`,
		metadata.ID, metadata.ModelID, metadata.Version, metadata.Algorithm,
		metadata.FeatureCount, metadata.TrainingTimestamp, metadata.ModelSizeBytes,
		metadata.Checksum, string(performanceMetricsJSON), string(configJSON),
		string(tagsJSON), metadata.Description, metadata.CreatedBy, metadata.Status,
		metadata.StorageBackend, metadata.StoragePath, metadata.CreatedAt,
		metadata.UpdatedAt, metadata.UsageCount)

	if err != nil {
		return errors.Wrapf(err, "failed to register model %s version %s", metadata.ModelID, metadata.Version)
	}

	log.Infof("Registered model %s version %s in registry", metadata.ModelID, metadata.Version)
	return nil
}

// GetModel retrieves a specific model version
func (s *modelRegistryStoreImpl) GetModel(ctx context.Context, modelID, version string) (*ModelMetadata, bool, error) {
	var metadata ModelMetadata
	var performanceMetricsJSON, configJSON, tagsJSON string

	err := s.db.QueryRow(ctx, `
		SELECT id, model_id, version, algorithm, feature_count, training_timestamp,
		       model_size_bytes, checksum, performance_metrics, config, tags,
		       description, created_by, status, storage_backend, storage_path,
		       created_at, updated_at, deployed_at, last_used_at, usage_count
		FROM ml_model_registry
		WHERE model_id = $1 AND version = $2`,
		modelID, version).Scan(
		&metadata.ID, &metadata.ModelID, &metadata.Version, &metadata.Algorithm,
		&metadata.FeatureCount, &metadata.TrainingTimestamp, &metadata.ModelSizeBytes,
		&metadata.Checksum, &performanceMetricsJSON, &configJSON, &tagsJSON,
		&metadata.Description, &metadata.CreatedBy, &metadata.Status,
		&metadata.StorageBackend, &metadata.StoragePath, &metadata.CreatedAt,
		&metadata.UpdatedAt, &metadata.DeployedAt, &metadata.LastUsedAt, &metadata.UsageCount)

	if err != nil {
		if postgres.IsNoResultsErr(err) {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "failed to get model %s version %s", modelID, version)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal([]byte(performanceMetricsJSON), &metadata.PerformanceMetrics); err != nil {
		log.Warnf("Failed to unmarshal performance metrics for model %s version %s: %v", modelID, version, err)
		metadata.PerformanceMetrics = make(map[string]interface{})
	}

	if err := json.Unmarshal([]byte(configJSON), &metadata.Config); err != nil {
		log.Warnf("Failed to unmarshal config for model %s version %s: %v", modelID, version, err)
		metadata.Config = make(map[string]interface{})
	}

	if err := json.Unmarshal([]byte(tagsJSON), &metadata.Tags); err != nil {
		log.Warnf("Failed to unmarshal tags for model %s version %s: %v", modelID, version, err)
		metadata.Tags = make(map[string]string)
	}

	return &metadata, true, nil
}

// GetLatestModel retrieves the latest version of a model
func (s *modelRegistryStoreImpl) GetLatestModel(ctx context.Context, modelID string) (*ModelMetadata, bool, error) {
	var metadata ModelMetadata
	var performanceMetricsJSON, configJSON, tagsJSON string

	err := s.db.QueryRow(ctx, `
		SELECT id, model_id, version, algorithm, feature_count, training_timestamp,
		       model_size_bytes, checksum, performance_metrics, config, tags,
		       description, created_by, status, storage_backend, storage_path,
		       created_at, updated_at, deployed_at, last_used_at, usage_count
		FROM ml_model_registry
		WHERE model_id = $1
		ORDER BY training_timestamp DESC
		LIMIT 1`,
		modelID).Scan(
		&metadata.ID, &metadata.ModelID, &metadata.Version, &metadata.Algorithm,
		&metadata.FeatureCount, &metadata.TrainingTimestamp, &metadata.ModelSizeBytes,
		&metadata.Checksum, &performanceMetricsJSON, &configJSON, &tagsJSON,
		&metadata.Description, &metadata.CreatedBy, &metadata.Status,
		&metadata.StorageBackend, &metadata.StoragePath, &metadata.CreatedAt,
		&metadata.UpdatedAt, &metadata.DeployedAt, &metadata.LastUsedAt, &metadata.UsageCount)

	if err != nil {
		if postgres.IsNoResultsErr(err) {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "failed to get latest model %s", modelID)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal([]byte(performanceMetricsJSON), &metadata.PerformanceMetrics); err != nil {
		metadata.PerformanceMetrics = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(configJSON), &metadata.Config); err != nil {
		metadata.Config = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(tagsJSON), &metadata.Tags); err != nil {
		metadata.Tags = make(map[string]string)
	}

	return &metadata, true, nil
}

// ListModels lists all versions of a specific model
func (s *modelRegistryStoreImpl) ListModels(ctx context.Context, modelID string) ([]*ModelMetadata, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, model_id, version, algorithm, feature_count, training_timestamp,
		       model_size_bytes, checksum, performance_metrics, config, tags,
		       description, created_by, status, storage_backend, storage_path,
		       created_at, updated_at, deployed_at, last_used_at, usage_count
		FROM ml_model_registry
		WHERE model_id = $1
		ORDER BY training_timestamp DESC`,
		modelID)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to list models for %s", modelID)
	}
	defer rows.Close()

	var models []*ModelMetadata
	for rows.Next() {
		var metadata ModelMetadata
		var performanceMetricsJSON, configJSON, tagsJSON string

		err := rows.Scan(
			&metadata.ID, &metadata.ModelID, &metadata.Version, &metadata.Algorithm,
			&metadata.FeatureCount, &metadata.TrainingTimestamp, &metadata.ModelSizeBytes,
			&metadata.Checksum, &performanceMetricsJSON, &configJSON, &tagsJSON,
			&metadata.Description, &metadata.CreatedBy, &metadata.Status,
			&metadata.StorageBackend, &metadata.StoragePath, &metadata.CreatedAt,
			&metadata.UpdatedAt, &metadata.DeployedAt, &metadata.LastUsedAt, &metadata.UsageCount)

		if err != nil {
			log.Warnf("Failed to scan model row: %v", err)
			continue
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(performanceMetricsJSON), &metadata.PerformanceMetrics)
		json.Unmarshal([]byte(configJSON), &metadata.Config)
		json.Unmarshal([]byte(tagsJSON), &metadata.Tags)

		models = append(models, &metadata)
	}

	return models, nil
}

// ListAllModels lists all models in the registry
func (s *modelRegistryStoreImpl) ListAllModels(ctx context.Context) ([]*ModelMetadata, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, model_id, version, algorithm, feature_count, training_timestamp,
		       model_size_bytes, checksum, performance_metrics, config, tags,
		       description, created_by, status, storage_backend, storage_path,
		       created_at, updated_at, deployed_at, last_used_at, usage_count
		FROM ml_model_registry
		ORDER BY model_id, training_timestamp DESC`)

	if err != nil {
		return nil, errors.Wrap(err, "failed to list all models")
	}
	defer rows.Close()

	var models []*ModelMetadata
	for rows.Next() {
		var metadata ModelMetadata
		var performanceMetricsJSON, configJSON, tagsJSON string

		err := rows.Scan(
			&metadata.ID, &metadata.ModelID, &metadata.Version, &metadata.Algorithm,
			&metadata.FeatureCount, &metadata.TrainingTimestamp, &metadata.ModelSizeBytes,
			&metadata.Checksum, &performanceMetricsJSON, &configJSON, &tagsJSON,
			&metadata.Description, &metadata.CreatedBy, &metadata.Status,
			&metadata.StorageBackend, &metadata.StoragePath, &metadata.CreatedAt,
			&metadata.UpdatedAt, &metadata.DeployedAt, &metadata.LastUsedAt, &metadata.UsageCount)

		if err != nil {
			log.Warnf("Failed to scan model row: %v", err)
			continue
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(performanceMetricsJSON), &metadata.PerformanceMetrics)
		json.Unmarshal([]byte(configJSON), &metadata.Config)
		json.Unmarshal([]byte(tagsJSON), &metadata.Tags)

		models = append(models, &metadata)
	}

	return models, nil
}

// UpdateModelStatus updates the status of a model
func (s *modelRegistryStoreImpl) UpdateModelStatus(ctx context.Context, modelID, version string, status ModelStatus) error {
	result, err := s.db.Exec(ctx, `
		UPDATE ml_model_registry
		SET status = $1, updated_at = $2
		WHERE model_id = $3 AND version = $4`,
		status, time.Now(), modelID, version)

	if err != nil {
		return errors.Wrapf(err, "failed to update status for model %s version %s", modelID, version)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("model %s version %s not found", modelID, version)
	}

	log.Infof("Updated model %s version %s status to %s", modelID, version, status)
	return nil
}

// DeleteModel deletes a model version from the registry
func (s *modelRegistryStoreImpl) DeleteModel(ctx context.Context, modelID, version string) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM ml_model_registry
		WHERE model_id = $1 AND version = $2`,
		modelID, version)

	if err != nil {
		return errors.Wrapf(err, "failed to delete model %s version %s", modelID, version)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("model %s version %s not found", modelID, version)
	}

	log.Infof("Deleted model %s version %s from registry", modelID, version)
	return nil
}

// GetDeployedModel gets the currently deployed model
func (s *modelRegistryStoreImpl) GetDeployedModel(ctx context.Context) (*ModelMetadata, bool, error) {
	var metadata ModelMetadata
	var performanceMetricsJSON, configJSON, tagsJSON string

	err := s.db.QueryRow(ctx, `
		SELECT id, model_id, version, algorithm, feature_count, training_timestamp,
		       model_size_bytes, checksum, performance_metrics, config, tags,
		       description, created_by, status, storage_backend, storage_path,
		       created_at, updated_at, deployed_at, last_used_at, usage_count
		FROM ml_model_registry
		WHERE status = $1
		ORDER BY deployed_at DESC
		LIMIT 1`,
		ModelStatusDeployed).Scan(
		&metadata.ID, &metadata.ModelID, &metadata.Version, &metadata.Algorithm,
		&metadata.FeatureCount, &metadata.TrainingTimestamp, &metadata.ModelSizeBytes,
		&metadata.Checksum, &performanceMetricsJSON, &configJSON, &tagsJSON,
		&metadata.Description, &metadata.CreatedBy, &metadata.Status,
		&metadata.StorageBackend, &metadata.StoragePath, &metadata.CreatedAt,
		&metadata.UpdatedAt, &metadata.DeployedAt, &metadata.LastUsedAt, &metadata.UsageCount)

	if err != nil {
		if postgres.IsNoResultsErr(err) {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "failed to get deployed model")
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(performanceMetricsJSON), &metadata.PerformanceMetrics)
	json.Unmarshal([]byte(configJSON), &metadata.Config)
	json.Unmarshal([]byte(tagsJSON), &metadata.Tags)

	return &metadata, true, nil
}

// SetDeployedModel sets a model as the deployed model
func (s *modelRegistryStoreImpl) SetDeployedModel(ctx context.Context, modelID, version string) error {
	now := time.Now()

	// First, unset any currently deployed models
	_, err := s.db.Exec(ctx, `
		UPDATE ml_model_registry
		SET status = $1, updated_at = $2
		WHERE status = $3`,
		ModelStatusReady, now, ModelStatusDeployed)

	if err != nil {
		return errors.Wrap(err, "failed to unset current deployed model")
	}

	// Set the new deployed model
	result, err := s.db.Exec(ctx, `
		UPDATE ml_model_registry
		SET status = $1, deployed_at = $2, updated_at = $3
		WHERE model_id = $4 AND version = $5`,
		ModelStatusDeployed, now, now, modelID, version)

	if err != nil {
		return errors.Wrapf(err, "failed to set model %s version %s as deployed", modelID, version)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("model %s version %s not found", modelID, version)
	}

	log.Infof("Set model %s version %s as deployed", modelID, version)
	return nil
}

// UpdateUsageStats updates usage statistics for a model
func (s *modelRegistryStoreImpl) UpdateUsageStats(ctx context.Context, modelID, version string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE ml_model_registry
		SET usage_count = usage_count + 1, last_used_at = $1, updated_at = $2
		WHERE model_id = $3 AND version = $4`,
		time.Now(), time.Now(), modelID, version)

	if err != nil {
		return errors.Wrapf(err, "failed to update usage stats for model %s version %s", modelID, version)
	}

	return nil
}

// GetModelStats gets overall registry statistics
func (s *modelRegistryStoreImpl) GetModelStats(ctx context.Context) (*ModelRegistryStats, error) {
	stats := &ModelRegistryStats{
		ModelsbyAlgorithm: make(map[string]int),
		ModelsByStatus:    make(map[string]int),
	}

	// Get basic counts
	err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT model_id) as total_models,
			COUNT(*) as total_versions,
			COALESCE(SUM(model_size_bytes), 0) as total_storage_bytes,
			COALESCE(MAX(training_timestamp), '1970-01-01'::timestamp) as last_training_time
		FROM ml_model_registry`).Scan(
		&stats.TotalModels, &stats.TotalVersions, &stats.TotalStorageBytes, &stats.LastTrainingTime)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get basic model stats")
	}

	// Get deployed model info
	deployedModel, exists, err := s.GetDeployedModel(ctx)
	if err == nil && exists {
		stats.DeployedModel = deployedModel.ModelID
		stats.DeployedVersion = deployedModel.Version
	}

	// Get counts by algorithm
	rows, err := s.db.Query(ctx, `
		SELECT algorithm, COUNT(*)
		FROM ml_model_registry
		GROUP BY algorithm`)

	if err != nil {
		log.Warnf("Failed to get algorithm stats: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var algorithm string
			var count int
			if err := rows.Scan(&algorithm, &count); err == nil {
				stats.ModelsbyAlgorithm[algorithm] = count
			}
		}
	}

	// Get counts by status
	rows, err = s.db.Query(ctx, `
		SELECT status, COUNT(*)
		FROM ml_model_registry
		GROUP BY status`)

	if err != nil {
		log.Warnf("Failed to get status stats: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err == nil {
				stats.ModelsByStatus[status] = count
			}
		}
	}

	return stats, nil
}

// ModelRegistry provides high-level model registry operations
type ModelRegistry struct {
	store ModelRegistryStore
}

// NewModelRegistry creates a new model registry
func NewModelRegistry(store ModelRegistryStore) *ModelRegistry {
	return &ModelRegistry{
		store: store,
	}
}

// RegisterModel registers a new model with validation
func (r *ModelRegistry) RegisterModel(ctx context.Context, metadata *ModelMetadata) error {
	ctx = sac.WithAllAccess(ctx)

	// Validate required fields
	if metadata.ModelID == "" {
		return fmt.Errorf("model_id is required")
	}
	if metadata.Version == "" {
		return fmt.Errorf("version is required")
	}
	if metadata.Algorithm == "" {
		return fmt.Errorf("algorithm is required")
	}

	return r.store.RegisterModel(ctx, metadata)
}

// GetModel gets a specific model version
func (r *ModelRegistry) GetModel(ctx context.Context, modelID, version string) (*ModelMetadata, bool, error) {
	ctx = sac.WithAllAccess(ctx)
	return r.store.GetModel(ctx, modelID, version)
}

// GetLatestModel gets the latest version of a model
func (r *ModelRegistry) GetLatestModel(ctx context.Context, modelID string) (*ModelMetadata, bool, error) {
	ctx = sac.WithAllAccess(ctx)
	return r.store.GetLatestModel(ctx, modelID)
}

// ListModels lists all versions of a model
func (r *ModelRegistry) ListModels(ctx context.Context, modelID string) ([]*ModelMetadata, error) {
	ctx = sac.WithAllAccess(ctx)
	return r.store.ListModels(ctx, modelID)
}

// ListAllModels lists all models
func (r *ModelRegistry) ListAllModels(ctx context.Context) ([]*ModelMetadata, error) {
	ctx = sac.WithAllAccess(ctx)
	return r.store.ListAllModels(ctx)
}

// SetDeployedModel sets the active deployed model
func (r *ModelRegistry) SetDeployedModel(ctx context.Context, modelID, version string) error {
	ctx = sac.WithAllAccess(ctx)
	return r.store.SetDeployedModel(ctx, modelID, version)
}

// GetDeployedModel gets the currently deployed model
func (r *ModelRegistry) GetDeployedModel(ctx context.Context) (*ModelMetadata, bool, error) {
	ctx = sac.WithAllAccess(ctx)
	return r.store.GetDeployedModel(ctx)
}

// UpdateUsageStats updates model usage statistics
func (r *ModelRegistry) UpdateUsageStats(ctx context.Context, modelID, version string) error {
	ctx = sac.WithAllAccess(ctx)
	return r.store.UpdateUsageStats(ctx, modelID, version)
}

// GetStats gets registry statistics
func (r *ModelRegistry) GetStats(ctx context.Context) (*ModelRegistryStats, error) {
	ctx = sac.WithAllAccess(ctx)
	return r.store.GetModelStats(ctx)
}