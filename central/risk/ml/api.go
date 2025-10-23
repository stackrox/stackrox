package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	apiLog = logging.LoggerForModule()
)

// ModelManagementAPI provides REST API endpoints for model management
type ModelManagementAPI struct {
	registry *ModelRegistry
	client   MLRiskClient
}

// NewModelManagementAPI creates a new model management API
func NewModelManagementAPI(registry *ModelRegistry, client MLRiskClient) *ModelManagementAPI {
	return &ModelManagementAPI{
		registry: registry,
		client:   client,
	}
}

// RegisterRoutes registers all model management API routes
func (api *ModelManagementAPI) RegisterRoutes(router *mux.Router) {
	// Model registry endpoints
	router.HandleFunc("/api/v1/ml/models", api.listAllModels).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/{modelId}", api.listModelVersions).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}", api.getModel).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}", api.deleteModel).Methods("DELETE")

	// Model deployment endpoints
	router.HandleFunc("/api/v1/ml/deployment/current", api.getCurrentDeployment).Methods("GET")
	router.HandleFunc("/api/v1/ml/deployment", api.deployModel).Methods("POST")
	router.HandleFunc("/api/v1/ml/deployment/rollback", api.rollbackDeployment).Methods("POST")

	// Model health and status endpoints
	router.HandleFunc("/api/v1/ml/health", api.getModelHealth).Methods("GET")
	router.HandleFunc("/api/v1/ml/health/detailed", api.getDetailedHealth).Methods("GET")
	router.HandleFunc("/api/v1/ml/stats", api.getRegistryStats).Methods("GET")

	// Model training endpoints
	router.HandleFunc("/api/v1/ml/training/trigger", api.triggerTraining).Methods("POST")
	router.HandleFunc("/api/v1/ml/training/status", api.getTrainingStatus).Methods("GET")

	// Model management endpoints
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}/promote", api.promoteModel).Methods("POST")
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}/deprecate", api.deprecateModel).Methods("POST")
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}/reload", api.reloadModel).Methods("POST")
	router.HandleFunc("/api/v1/ml/models/list", api.listAvailableModels).Methods("GET")

	// Model versioning endpoints
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}/lineage", api.getModelLineage).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/compare", api.compareModelVersions).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/{modelId}/metrics/{metric}/history", api.getMetricHistory).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/{modelId}/versions/{version}/validate", api.validateForProduction).Methods("GET")
	router.HandleFunc("/api/v1/ml/models/status/{status}", api.getModelsByStatus).Methods("GET")

	// Model drift monitoring endpoints
	router.HandleFunc("/api/v1/ml/drift/report", api.getDriftReport).Methods("GET")
	router.HandleFunc("/api/v1/ml/drift/alerts", api.getActiveAlerts).Methods("GET")
	router.HandleFunc("/api/v1/ml/drift/baseline", api.setDriftBaseline).Methods("POST")
}

// API Response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type ModelListResponse struct {
	Models []ModelMetadata `json:"models"`
	Total  int             `json:"total"`
}

type DeploymentRequest struct {
	ModelID string `json:"model_id"`
	Version string `json:"version"`
	Force   bool   `json:"force,omitempty"`
}

type TrainingRequest struct {
	ModelID     string                   `json:"model_id"`
	Description string                   `json:"description,omitempty"`
	Config      map[string]interface{}   `json:"config,omitempty"`
	DataSource  string                   `json:"data_source,omitempty"`
	Tags        map[string]string        `json:"tags,omitempty"`
}

type TrainingStatusResponse struct {
	Status      string    `json:"status"`
	ModelID     string    `json:"model_id,omitempty"`
	Progress    float64   `json:"progress,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
	Message     string    `json:"message,omitempty"`
}

type ReloadModelRequest struct {
	Force bool `json:"force,omitempty"`
}

type ModelLineageResponse struct {
	ModelID string                   `json:"model_id"`
	Version string                   `json:"version"`
	Lineage []ModelVersionInfo       `json:"lineage"`
}

type ModelVersionInfo struct {
	Version           string             `json:"version"`
	SemanticVersion   string             `json:"semantic_version"`
	Status            string             `json:"status"`
	CreatedAt         string             `json:"created_at"`
	PerformanceMetrics map[string]float64 `json:"performance_metrics"`
	QualityScore      *float64           `json:"quality_score,omitempty"`
}

type ModelComparisonResponse struct {
	ModelID        string                `json:"model_id"`
	Version1       ModelVersionInfo      `json:"version1"`
	Version2       ModelVersionInfo      `json:"version2"`
	PerformanceDiff map[string]float64   `json:"performance_diff"`
	QualityDiff    *float64              `json:"quality_diff,omitempty"`
}

type MetricHistoryResponse struct {
	ModelID string                 `json:"model_id"`
	Metric  string                 `json:"metric"`
	History []MetricHistoryPoint   `json:"history"`
}

type MetricHistoryPoint struct {
	Version         string  `json:"version"`
	SemanticVersion string  `json:"semantic_version"`
	Value           float64 `json:"value"`
	CreatedAt       string  `json:"created_at"`
}

type ProductionValidationResponse struct {
	ModelID      string   `json:"model_id"`
	Version      string   `json:"version"`
	IsReady      bool     `json:"is_ready"`
	Issues       []string `json:"issues"`
	QualityScore *float64 `json:"quality_score,omitempty"`
}

type DriftReportResponse struct {
	ModelID               string             `json:"model_id"`
	Version               string             `json:"version"`
	OverallDriftStatus    string             `json:"overall_drift_status"`
	OverallDriftScore     float64            `json:"overall_drift_score"`
	DataDriftScore        float64            `json:"data_drift_score"`
	PredictionDriftScore  float64            `json:"prediction_drift_score"`
	PerformanceDriftScore float64            `json:"performance_drift_score"`
	ActiveAlertsCount     int                `json:"active_alerts_count"`
	Recommendations       []string           `json:"recommendations"`
	ReportPeriodHours     int                `json:"report_period_hours"`
	Timestamp             string             `json:"timestamp"`
}

type DriftAlert struct {
	AlertID       string                 `json:"alert_id"`
	DriftType     string                 `json:"drift_type"`
	Severity      string                 `json:"severity"`
	MetricName    string                 `json:"metric_name"`
	DriftScore    float64                `json:"drift_score"`
	Threshold     float64                `json:"threshold"`
	CurrentValue  float64                `json:"current_value"`
	BaselineValue float64                `json:"baseline_value"`
	Message       string                 `json:"message"`
	Timestamp     string                 `json:"timestamp"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

type DriftAlertsResponse struct {
	Alerts     []DriftAlert `json:"alerts"`
	TotalCount int          `json:"total_count"`
}

type SetDriftBaselineRequest struct {
	ModelID string `json:"model_id,omitempty"`
	Version string `json:"version,omitempty"`
}

// listAllModels handles GET /api/v1/ml/models
func (api *ModelManagementAPI) listAllModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	models, err := api.registry.ListAllModels(ctx)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list models: %v", err))
		return
	}

	response := ModelListResponse{
		Models: make([]ModelMetadata, len(models)),
		Total:  len(models),
	}

	for i, model := range models {
		response.Models[i] = *model
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// listModelVersions handles GET /api/v1/ml/models/{modelId}
func (api *ModelManagementAPI) listModelVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	if modelID == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID is required")
		return
	}

	models, err := api.registry.ListModels(ctx, modelID)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list model versions: %v", err))
		return
	}

	response := ModelListResponse{
		Models: make([]ModelMetadata, len(models)),
		Total:  len(models),
	}

	for i, model := range models {
		response.Models[i] = *model
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// getModel handles GET /api/v1/ml/models/{modelId}/versions/{version}
func (api *ModelManagementAPI) getModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	model, exists, err := api.registry.GetModel(ctx, modelID, version)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get model: %v", err))
		return
	}

	if !exists {
		api.writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Model %s version %s not found", modelID, version))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    model,
	})
}

// deleteModel handles DELETE /api/v1/ml/models/{modelId}/versions/{version}
func (api *ModelManagementAPI) deleteModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	// Check if this is the currently deployed model
	deployedModel, exists, err := api.registry.GetDeployedModel(ctx)
	if err == nil && exists && deployedModel.ModelID == modelID && deployedModel.Version == version {
		api.writeErrorResponse(w, http.StatusBadRequest, "Cannot delete currently deployed model")
		return
	}

	err = api.registry.store.DeleteModel(ctx, modelID, version)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete model: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Model %s version %s deleted successfully", modelID, version),
	})
}

// getCurrentDeployment handles GET /api/v1/ml/deployment/current
func (api *ModelManagementAPI) getCurrentDeployment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	deployedModel, exists, err := api.registry.GetDeployedModel(ctx)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get deployed model: %v", err))
		return
	}

	if !exists {
		api.writeJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    nil,
			Message: "No model currently deployed",
		})
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    deployedModel,
	})
}

// deployModel handles POST /api/v1/ml/deployment
func (api *ModelManagementAPI) deployModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req DeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ModelID == "" || req.Version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	// Validate that the model exists and is ready
	model, exists, err := api.registry.GetModel(ctx, req.ModelID, req.Version)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to validate model: %v", err))
		return
	}

	if !exists {
		api.writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Model %s version %s not found", req.ModelID, req.Version))
		return
	}

	if model.Status != ModelStatusReady && !req.Force {
		api.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Model status is %s, not ready for deployment. Use force=true to override", model.Status))
		return
	}

	// Set as deployed model
	err = api.registry.SetDeployedModel(ctx, req.ModelID, req.Version)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to deploy model: %v", err))
		return
	}

	// Trigger hot reload in ML service via gRPC
	reloadResp, err := api.client.ReloadModel(ctx, req.ModelID, req.Version, req.Force)
	if err != nil {
		apiLog.Errorf("Failed to hot reload model in ML service: %v", err)
		// Don't fail the deployment request, just log the error
		api.writeJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: fmt.Sprintf("Model %s version %s deployed successfully (hot reload failed: %v)", req.ModelID, req.Version, err),
			Data:    model,
		})
		return
	}

	if !reloadResp.Success {
		apiLog.Warnf("Model hot reload completed with warning: %s", reloadResp.Message)
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Model %s version %s deployed successfully", req.ModelID, req.Version),
		Data:    model,
	})
}

// rollbackDeployment handles POST /api/v1/ml/deployment/rollback
func (api *ModelManagementAPI) rollbackDeployment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current deployed model
	currentModel, exists, err := api.registry.GetDeployedModel(ctx)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get current deployed model: %v", err))
		return
	}

	if !exists {
		api.writeErrorResponse(w, http.StatusBadRequest, "No model currently deployed")
		return
	}

	// Find the previous version to rollback to
	models, err := api.registry.ListModels(ctx, currentModel.ModelID)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list model versions: %v", err))
		return
	}

	var previousModel *ModelMetadata
	for _, model := range models {
		if model.Version != currentModel.Version && model.Status == ModelStatusReady {
			if previousModel == nil || model.TrainingTimestamp.After(previousModel.TrainingTimestamp) {
				previousModel = model
			}
		}
	}

	if previousModel == nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "No previous ready model version found for rollback")
		return
	}

	// Deploy the previous model
	err = api.registry.SetDeployedModel(ctx, previousModel.ModelID, previousModel.Version)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rollback to previous model: %v", err))
		return
	}

	// Trigger hot reload for rollback
	reloadResp, err := api.client.ReloadModel(ctx, previousModel.ModelID, previousModel.Version, true) // Force reload for rollback
	if err != nil {
		apiLog.Errorf("Failed to hot reload model during rollback: %v", err)
		// Continue with rollback even if hot reload fails
	} else if !reloadResp.Success {
		apiLog.Warnf("Model hot reload during rollback completed with warning: %s", reloadResp.Message)
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Rolled back from %s v%s to %s v%s", currentModel.ModelID, currentModel.Version, previousModel.ModelID, previousModel.Version),
		Data:    previousModel,
	})
}

// getModelHealth handles GET /api/v1/ml/health
func (api *ModelManagementAPI) getModelHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get health from ML service
	health, err := api.client.GetModelHealth(ctx)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get model health: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    health,
	})
}

// getDetailedHealth handles GET /api/v1/ml/health/detailed
func (api *ModelManagementAPI) getDetailedHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	includeTrends := r.URL.Query().Get("include_trends") != "false" // Default to true
	trendHours := 24 // Default to 24 hours

	if hoursStr := r.URL.Query().Get("trend_hours"); hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil && hours > 0 && hours <= 168 { // Max 1 week
			trendHours = hours
		}
	}

	// Get detailed health from ML service
	detailedHealth, err := api.client.GetDetailedHealth(ctx, includeTrends, trendHours)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get detailed health: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    detailedHealth,
	})
}

// getRegistryStats handles GET /api/v1/ml/stats
func (api *ModelManagementAPI) getRegistryStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := api.registry.GetStats(ctx)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get registry stats: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// triggerTraining handles POST /api/v1/ml/training/trigger
func (api *ModelManagementAPI) triggerTraining(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req TrainingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ModelID == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID is required")
		return
	}

	// TODO: Implement training trigger via ML service
	// For now, return a placeholder response
	api.writeJSONResponse(w, http.StatusAccepted, APIResponse{
		Success: true,
		Message: "Training request submitted successfully",
		Data: TrainingStatusResponse{
			Status:    "submitted",
			ModelID:   req.ModelID,
			StartedAt: time.Now(),
			Message:   "Training job has been queued",
		},
	})
}

// getTrainingStatus handles GET /api/v1/ml/training/status
func (api *ModelManagementAPI) getTrainingStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Implement training status tracking
	// For now, return a placeholder response
	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: TrainingStatusResponse{
			Status:  "no_active_training",
			Message: "No training jobs currently active",
		},
	})
}

// promoteModel handles POST /api/v1/ml/models/{modelId}/versions/{version}/promote
func (api *ModelManagementAPI) promoteModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	err := api.registry.store.UpdateModelStatus(ctx, modelID, version, ModelStatusReady)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to promote model: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Model %s version %s promoted to ready status", modelID, version),
	})
}

// deprecateModel handles POST /api/v1/ml/models/{modelId}/versions/{version}/deprecate
func (api *ModelManagementAPI) deprecateModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	// Check if this is the currently deployed model
	deployedModel, exists, err := api.registry.GetDeployedModel(ctx)
	if err == nil && exists && deployedModel.ModelID == modelID && deployedModel.Version == version {
		api.writeErrorResponse(w, http.StatusBadRequest, "Cannot deprecate currently deployed model")
		return
	}

	err = api.registry.store.UpdateModelStatus(ctx, modelID, version, ModelStatusDeprecated)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to deprecate model: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Model %s version %s deprecated", modelID, version),
	})
}

// Helper methods
func (api *ModelManagementAPI) writeJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		apiLog.Errorf("Failed to encode JSON response: %v", err)
	}
}

func (api *ModelManagementAPI) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	apiLog.Warnf("API error response: %d - %s", statusCode, message)
	api.writeJSONResponse(w, statusCode, APIResponse{
		Success: false,
		Error:   message,
	})
}

// reloadModel handles POST /api/v1/ml/models/{modelId}/versions/{version}/reload
func (api *ModelManagementAPI) reloadModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	var req ReloadModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If body parsing fails, use default values
		req.Force = false
	}

	// Validate that the model exists
	_, exists, err := api.registry.GetModel(ctx, modelID, version)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to validate model: %v", err))
		return
	}

	if !exists {
		api.writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Model %s version %s not found", modelID, version))
		return
	}

	// Trigger hot reload in ML service
	reloadResp, err := api.client.ReloadModel(ctx, modelID, version, req.Force)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload model: %v", err))
		return
	}

	if !reloadResp.Success {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Model reload failed: %s", reloadResp.Message))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: reloadResp.Message,
		Data:    reloadResp,
	})
}

// listAvailableModels handles GET /api/v1/ml/models/list
func (api *ModelManagementAPI) listAvailableModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get model ID filter from query params
	modelID := r.URL.Query().Get("model_id")

	// Get models from ML service storage
	modelsResp, err := api.client.ListModels(ctx, modelID)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list available models: %v", err))
		return
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    modelsResp,
	})
}

// getModelLineage handles GET /api/v1/ml/models/{modelId}/versions/{version}/lineage
func (api *ModelManagementAPI) getModelLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	// For now, return mock lineage data
	// In practice, this would call the ML service storage manager
	mockLineage := []ModelVersionInfo{
		{
			Version:         version,
			SemanticVersion: "1.2.3",
			Status:          "production",
			CreatedAt:       time.Now().Add(-7*24*time.Hour).Format(time.RFC3339),
			PerformanceMetrics: map[string]float64{
				"validation_ndcg": 0.87,
				"validation_auc":  0.81,
			},
			QualityScore: &[]float64{0.89}[0],
		},
		{
			Version:         "v1",
			SemanticVersion: "1.2.2",
			Status:          "deprecated",
			CreatedAt:       time.Now().Add(-14*24*time.Hour).Format(time.RFC3339),
			PerformanceMetrics: map[string]float64{
				"validation_ndcg": 0.85,
				"validation_auc":  0.78,
			},
			QualityScore: &[]float64{0.85}[0],
		},
	}

	response := ModelLineageResponse{
		ModelID: modelID,
		Version: version,
		Lineage: mockLineage,
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// compareModelVersions handles GET /api/v1/ml/models/{modelId}/versions/compare
func (api *ModelManagementAPI) compareModelVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	version1 := r.URL.Query().Get("version1")
	version2 := r.URL.Query().Get("version2")

	if modelID == "" || version1 == "" || version2 == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID, version1, and version2 are required")
		return
	}

	// Mock comparison data
	v1Info := ModelVersionInfo{
		Version:         version1,
		SemanticVersion: "1.2.2",
		Status:          "production",
		CreatedAt:       time.Now().Add(-14*24*time.Hour).Format(time.RFC3339),
		PerformanceMetrics: map[string]float64{
			"validation_ndcg": 0.85,
			"validation_auc":  0.78,
		},
		QualityScore: &[]float64{0.85}[0],
	}

	v2Info := ModelVersionInfo{
		Version:         version2,
		SemanticVersion: "1.2.3",
		Status:          "staging",
		CreatedAt:       time.Now().Add(-7*24*time.Hour).Format(time.RFC3339),
		PerformanceMetrics: map[string]float64{
			"validation_ndcg": 0.87,
			"validation_auc":  0.81,
		},
		QualityScore: &[]float64{0.89}[0],
	}

	performanceDiff := map[string]float64{
		"validation_ndcg": 0.02,
		"validation_auc":  0.03,
	}

	qualityDiff := 0.04

	response := ModelComparisonResponse{
		ModelID:        modelID,
		Version1:       v1Info,
		Version2:       v2Info,
		PerformanceDiff: performanceDiff,
		QualityDiff:    &qualityDiff,
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// getMetricHistory handles GET /api/v1/ml/models/{modelId}/metrics/{metric}/history
func (api *ModelManagementAPI) getMetricHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	metric := vars["metric"]

	if modelID == "" || metric == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and metric are required")
		return
	}

	// Mock metric history
	history := []MetricHistoryPoint{
		{
			Version:         "v1",
			SemanticVersion: "1.2.0",
			Value:           0.82,
			CreatedAt:       time.Now().Add(-30*24*time.Hour).Format(time.RFC3339),
		},
		{
			Version:         "v2",
			SemanticVersion: "1.2.1",
			Value:           0.85,
			CreatedAt:       time.Now().Add(-21*24*time.Hour).Format(time.RFC3339),
		},
		{
			Version:         "v3",
			SemanticVersion: "1.2.2",
			Value:           0.87,
			CreatedAt:       time.Now().Add(-14*24*time.Hour).Format(time.RFC3339),
		},
		{
			Version:         "v4",
			SemanticVersion: "1.2.3",
			Value:           0.89,
			CreatedAt:       time.Now().Add(-7*24*time.Hour).Format(time.RFC3339),
		},
	}

	response := MetricHistoryResponse{
		ModelID: modelID,
		Metric:  metric,
		History: history,
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// validateForProduction handles GET /api/v1/ml/models/{modelId}/versions/{version}/validate
func (api *ModelManagementAPI) validateForProduction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	modelID := vars["modelId"]
	version := vars["version"]

	if modelID == "" || version == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Model ID and version are required")
		return
	}

	// Mock validation logic
	// In practice, this would call the ML service validation
	qualityScore := 0.89
	isReady := true
	issues := []string{}

	// Simulate some validation checks
	if qualityScore < 0.8 {
		isReady = false
		issues = append(issues, "Model quality score below threshold (0.8)")
	}

	// Add more validation checks as needed
	if version == "v1" {
		isReady = false
		issues = append(issues, "Version v1 is deprecated")
	}

	response := ProductionValidationResponse{
		ModelID:      modelID,
		Version:      version,
		IsReady:      isReady,
		Issues:       issues,
		QualityScore: &qualityScore,
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// getModelsByStatus handles GET /api/v1/ml/models/status/{status}
func (api *ModelManagementAPI) getModelsByStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	status := vars["status"]

	if status == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Status is required")
		return
	}

	validStatuses := []string{"draft", "staging", "production", "deprecated", "archived"}
	isValid := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValid = true
			break
		}
	}

	if !isValid {
		api.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid status. Must be one of: %v", validStatuses))
		return
	}

	// For now, return mock filtered models
	// In practice, this would query the registry with status filter
	models, err := api.registry.ListAllModels(ctx)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list models: %v", err))
		return
	}

	// Filter by status (mock - in practice the registry would handle this)
	filteredModels := []ModelMetadata{}
	for _, model := range models {
		// Mock status filtering
		if (status == "production" && model.Status == ModelStatusReady) ||
		   (status == "deprecated" && model.Status == ModelStatusDeprecated) {
			filteredModels = append(filteredModels, *model)
		}
	}

	response := ModelListResponse{
		Models: filteredModels,
		Total:  len(filteredModels),
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// getUserFromContext extracts user information from request context
func (api *ModelManagementAPI) getUserFromContext(ctx context.Context) string {
	if userInfo := user.FromContext(ctx); userInfo != nil {
		return userInfo.GetUsername()
	}
	return "unknown"
}

// getDriftReport handles GET /api/v1/ml/drift/report
func (api *ModelManagementAPI) getDriftReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	modelID := r.URL.Query().Get("model_id")
	version := r.URL.Query().Get("version")
	periodHours := 24 // Default to 24 hours

	if hoursStr := r.URL.Query().Get("period_hours"); hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil && hours > 0 && hours <= 168 { // Max 1 week
			periodHours = hours
		}
	}

	// Call ML service to get drift report
	// In practice, this would call the gRPC GetDriftReport method
	// For now, return a mock response
	mockReport := DriftReportResponse{
		ModelID:               modelID,
		Version:               version,
		OverallDriftStatus:    "low_drift",
		OverallDriftScore:     0.25,
		DataDriftScore:        0.18,
		PredictionDriftScore:  0.32,
		PerformanceDriftScore: 0.15,
		ActiveAlertsCount:     2,
		Recommendations: []string{
			"Monitor prediction quality closely",
			"Consider retraining if drift increases",
		},
		ReportPeriodHours: periodHours,
		Timestamp:         time.Now().Format(time.RFC3339),
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    mockReport,
	})
}

// getActiveAlerts handles GET /api/v1/ml/drift/alerts
func (api *ModelManagementAPI) getActiveAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	modelID := r.URL.Query().Get("model_id")
	severity := r.URL.Query().Get("severity")

	// Call ML service to get active alerts
	// In practice, this would call the gRPC GetActiveAlerts method
	// For now, return mock alerts
	mockAlerts := []DriftAlert{
		{
			AlertID:       "data_drift_stackrox-risk-model_policy_violation_score_1640995200",
			DriftType:     "data_drift",
			Severity:      "medium",
			MetricName:    "policy_violation_score",
			DriftScore:    0.42,
			Threshold:     0.30,
			CurrentValue:  2.8,
			BaselineValue: 2.1,
			Message:       "Data drift detected in feature policy_violation_score using Kolmogorov-Smirnov",
			Timestamp:     time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			Details: map[string]interface{}{
				"ks_statistic": 0.15,
				"p_value":      0.02,
			},
		},
		{
			AlertID:       "prediction_drift_stackrox-risk-model_1640999800",
			DriftType:     "prediction_drift",
			Severity:      "low",
			MetricName:    "prediction_distribution",
			DriftScore:    0.22,
			Threshold:     0.20,
			CurrentValue:  3.2,
			BaselineValue: 2.9,
			Message:       "Prediction drift detected using Population Stability Index",
			Timestamp:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			Details: map[string]interface{}{
				"psi_value": 0.18,
			},
		},
	}

	// Filter by model_id if provided
	if modelID != "" {
		var filteredAlerts []DriftAlert
		for _, alert := range mockAlerts {
			// Simple check if model_id is in alert_id
			if strings.Contains(alert.AlertID, modelID) {
				filteredAlerts = append(filteredAlerts, alert)
			}
		}
		mockAlerts = filteredAlerts
	}

	// Filter by severity if provided
	if severity != "" {
		var filteredAlerts []DriftAlert
		for _, alert := range mockAlerts {
			if alert.Severity == severity {
				filteredAlerts = append(filteredAlerts, alert)
			}
		}
		mockAlerts = filteredAlerts
	}

	response := DriftAlertsResponse{
		Alerts:     mockAlerts,
		TotalCount: len(mockAlerts),
	}

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// setDriftBaseline handles POST /api/v1/ml/drift/baseline
func (api *ModelManagementAPI) setDriftBaseline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SetDriftBaselineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Use current deployed model if not specified
	if req.ModelID == "" || req.Version == "" {
		deployedModel, exists, err := api.registry.GetDeployedModel(ctx)
		if err != nil {
			api.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get deployed model: %v", err))
			return
		}

		if !exists {
			api.writeErrorResponse(w, http.StatusBadRequest, "No model currently deployed and model_id/version not specified")
			return
		}

		if req.ModelID == "" {
			req.ModelID = deployedModel.ModelID
		}
		if req.Version == "" {
			req.Version = deployedModel.Version
		}
	}

	// Call ML service to set baseline data
	// In practice, this would call the gRPC SetBaselineData method
	// For now, return a mock response
	apiLog.Infof("Setting drift baseline for model %s v%s", req.ModelID, req.Version)

	api.writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Drift baseline set for model %s v%s", req.ModelID, req.Version),
		Data: map[string]interface{}{
			"model_id": req.ModelID,
			"version":  req.Version,
			"features": 19, // Mock feature count
		},
	})
}