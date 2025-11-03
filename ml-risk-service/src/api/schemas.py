"""
Pydantic schemas for ML Risk Service REST API.
These models provide automatic validation and OpenAPI documentation.
"""

from typing import List, Dict, Optional, Any
from pydantic import BaseModel, Field
from datetime import datetime


class DeploymentFeatures(BaseModel):
    """Features extracted from a deployment for risk assessment."""

    policy_violation_count: int = Field(
        default=0,
        description="Number of policy violations",
        ge=0
    )
    policy_violation_severity_score: float = Field(
        default=0.0,
        description="Aggregated severity score of policy violations",
        ge=0.0
    )
    process_baseline_violations: int = Field(
        default=0,
        description="Number of process baseline violations",
        ge=0
    )
    host_network: bool = Field(
        default=False,
        description="Whether deployment uses host network"
    )
    host_pid: bool = Field(
        default=False,
        description="Whether deployment uses host PID namespace"
    )
    host_ipc: bool = Field(
        default=False,
        description="Whether deployment uses host IPC namespace"
    )
    privileged_container_count: int = Field(
        default=0,
        description="Number of privileged containers",
        ge=0
    )
    automount_service_account_token: bool = Field(
        default=False,
        description="Whether service account token is automounted"
    )
    exposed_port_count: int = Field(
        default=0,
        description="Number of exposed ports",
        ge=0
    )
    has_external_exposure: bool = Field(
        default=False,
        description="Whether deployment has external network exposure"
    )
    service_account_permission_level: float = Field(
        default=0.0,
        description="Service account permission level score",
        ge=0.0
    )
    replica_count: int = Field(
        default=1,
        description="Number of replicas",
        ge=1
    )
    is_orchestrator_component: bool = Field(
        default=False,
        description="Whether this is an orchestrator component"
    )
    is_platform_component: bool = Field(
        default=False,
        description="Whether this is a platform component"
    )
    cluster_id: str = Field(
        default="",
        description="Cluster identifier"
    )
    namespace: str = Field(
        default="",
        description="Kubernetes namespace"
    )
    creation_timestamp: int = Field(
        default=0,
        description="Unix timestamp of deployment creation",
        ge=0
    )
    is_inactive: bool = Field(
        default=False,
        description="Whether deployment is inactive"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "policy_violation_count": 2,
                "policy_violation_severity_score": 7.5,
                "process_baseline_violations": 1,
                "host_network": False,
                "host_pid": False,
                "host_ipc": False,
                "privileged_container_count": 0,
                "automount_service_account_token": True,
                "exposed_port_count": 2,
                "has_external_exposure": True,
                "service_account_permission_level": 3.0,
                "replica_count": 3,
                "is_orchestrator_component": False,
                "is_platform_component": False,
                "cluster_id": "prod-cluster-1",
                "namespace": "app-namespace",
                "creation_timestamp": 1640995200,
                "is_inactive": False
            }
        }


class ImageFeatures(BaseModel):
    """Features extracted from container images for risk assessment."""

    image_id: str = Field(
        default="",
        description="Unique image identifier"
    )
    image_name: str = Field(
        default="",
        description="Image name and tag"
    )
    critical_vuln_count: int = Field(
        default=0,
        description="Number of critical vulnerabilities",
        ge=0
    )
    high_vuln_count: int = Field(
        default=0,
        description="Number of high severity vulnerabilities",
        ge=0
    )
    medium_vuln_count: int = Field(
        default=0,
        description="Number of medium severity vulnerabilities",
        ge=0
    )
    low_vuln_count: int = Field(
        default=0,
        description="Number of low severity vulnerabilities",
        ge=0
    )
    avg_cvss_score: float = Field(
        default=0.0,
        description="Average CVSS score of vulnerabilities",
        ge=0.0,
        le=10.0
    )
    max_cvss_score: float = Field(
        default=0.0,
        description="Maximum CVSS score among vulnerabilities",
        ge=0.0,
        le=10.0
    )
    total_component_count: int = Field(
        default=0,
        description="Total number of software components",
        ge=0
    )
    risky_component_count: int = Field(
        default=0,
        description="Number of components with known risks",
        ge=0
    )
    image_creation_timestamp: int = Field(
        default=0,
        description="Unix timestamp of image creation",
        ge=0
    )
    image_age_days: int = Field(
        default=0,
        description="Age of image in days",
        ge=0
    )
    is_cluster_local: bool = Field(
        default=False,
        description="Whether image is stored locally in cluster"
    )
    base_image: str = Field(
        default="",
        description="Base image name"
    )
    layer_count: int = Field(
        default=0,
        description="Number of layers in image",
        ge=0
    )

    class Config:
        json_schema_extra = {
            "example": {
                "image_id": "sha256:abc123",
                "image_name": "nginx:1.21.0",
                "critical_vuln_count": 1,
                "high_vuln_count": 3,
                "medium_vuln_count": 5,
                "low_vuln_count": 2,
                "avg_cvss_score": 6.2,
                "max_cvss_score": 9.1,
                "total_component_count": 150,
                "risky_component_count": 8,
                "image_creation_timestamp": 1620000000,
                "image_age_days": 45,
                "is_cluster_local": False,
                "base_image": "alpine:3.14",
                "layer_count": 8
            }
        }


class FeatureImportance(BaseModel):
    """Feature importance information for model interpretability."""

    feature_name: str = Field(description="Name of the feature")
    importance_score: float = Field(description="Importance score of the feature")
    feature_category: str = Field(description="Category/group of the feature")
    description: str = Field(description="Human-readable description of the feature")

    class Config:
        json_schema_extra = {
            "example": {
                "feature_name": "critical_vuln_count",
                "importance_score": 0.85,
                "feature_category": "vulnerability",
                "description": "Number of critical vulnerabilities in container images"
            }
        }


class DeploymentRiskRequest(BaseModel):
    """Request to get risk score for a single deployment."""

    deployment_id: str = Field(description="Unique deployment identifier")
    deployment_features: DeploymentFeatures = Field(description="Deployment-level features")
    image_features: List[ImageFeatures] = Field(
        default=[],
        description="List of features for each container image"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "deployment_id": "nginx-deployment-123",
                "deployment_features": {
                    "policy_violation_count": 2,
                    "has_external_exposure": True,
                    "replica_count": 3,
                    "namespace": "production"
                },
                "image_features": [
                    {
                        "image_name": "nginx:1.21.0",
                        "critical_vuln_count": 1,
                        "high_vuln_count": 3
                    }
                ]
            }
        }


class DeploymentRiskResponse(BaseModel):
    """Response containing risk score and explanations for a deployment."""

    deployment_id: str = Field(description="Unique deployment identifier")
    risk_score: float = Field(
        description="Calculated risk score",
        ge=0.0
    )
    feature_importances: List[FeatureImportance] = Field(
        default=[],
        description="Feature importance scores for interpretability"
    )
    model_version: str = Field(description="Version of the model used for prediction")
    timestamp: int = Field(description="Unix timestamp of prediction")

    class Config:
        json_schema_extra = {
            "example": {
                "deployment_id": "nginx-deployment-123",
                "risk_score": 7.2,
                "feature_importances": [
                    {
                        "feature_name": "critical_vuln_count",
                        "importance_score": 0.85,
                        "feature_category": "vulnerability",
                        "description": "Number of critical vulnerabilities"
                    }
                ],
                "model_version": "v1.2.3",
                "timestamp": 1640995200
            }
        }


class BatchDeploymentRiskRequest(BaseModel):
    """Request to get risk scores for multiple deployments."""

    requests: List[DeploymentRiskRequest] = Field(
        description="List of deployment risk requests"
    )


class BatchDeploymentRiskResponse(BaseModel):
    """Response containing risk scores for multiple deployments."""

    responses: List[DeploymentRiskResponse] = Field(
        description="List of deployment risk responses"
    )


class TrainingSample(BaseModel):
    """Training sample for model training."""

    deployment_features: DeploymentFeatures = Field(description="Deployment features")
    image_features: List[ImageFeatures] = Field(
        default=[],
        description="Image features"
    )
    current_risk_score: float = Field(
        description="Target risk score for training",
        ge=0.0
    )
    deployment_id: str = Field(description="Deployment identifier")


class TrainModelRequest(BaseModel):
    """Request to train a new model."""

    training_data: List[TrainingSample] = Field(
        description="Training samples"
    )
    config_override: Optional[str] = Field(
        default="",
        description="JSON string with configuration overrides"
    )


class TrainingMetrics(BaseModel):
    """Training metrics and performance indicators for RandomForest model."""

    validation_ndcg: float = Field(
        description="Validation NDCG score",
        ge=0.0,
        le=1.0
    )
    global_feature_importance: List[FeatureImportance] = Field(
        default=[],
        description="Global feature importance rankings"
    )


class TrainModelResponse(BaseModel):
    """Response from model training."""

    success: bool = Field(description="Whether training succeeded")
    model_version: str = Field(description="Version of the trained model")
    metrics: TrainingMetrics = Field(description="Training metrics")
    error_message: Optional[str] = Field(
        default="",
        description="Error message if training failed"
    )


class ModelMetrics(BaseModel):
    """Current model performance metrics."""

    current_ndcg: float = Field(
        description="Current NDCG score",
        ge=0.0,
        le=1.0
    )
    current_auc: float = Field(
        description="Current AUC score",
        ge=0.0,
        le=1.0
    )
    predictions_served: int = Field(
        description="Number of predictions served",
        ge=0
    )
    avg_prediction_time_ms: float = Field(
        description="Average prediction time in milliseconds",
        ge=0.0
    )


class ModelHealthResponse(BaseModel):
    """Model health status and metrics."""

    healthy: bool = Field(description="Whether model is healthy")
    model_version: str = Field(description="Current model version")
    last_training_time: int = Field(
        description="Unix timestamp of last training",
        ge=0
    )
    training_samples_count: int = Field(
        description="Number of training samples used",
        ge=0
    )
    current_metrics: ModelMetrics = Field(description="Current performance metrics")


class ReloadModelRequest(BaseModel):
    """Request to reload a model."""

    model_id: str = Field(description="Model identifier")
    version: Optional[str] = Field(
        default="",
        description="Model version (empty for latest)"
    )
    force_reload: bool = Field(
        default=False,
        description="Force reload even if already loaded"
    )


class ReloadModelResponse(BaseModel):
    """Response from model reload operation."""

    success: bool = Field(description="Whether reload succeeded")
    message: str = Field(description="Status message")
    previous_model_version: str = Field(description="Previous model version")
    new_model_version: str = Field(description="New model version")
    reload_time_ms: float = Field(
        description="Time taken for reload in milliseconds",
        ge=0.0
    )


class ModelInfo(BaseModel):
    """Information about a stored model."""

    model_id: str = Field(description="Model identifier")
    version: str = Field(description="Model version")
    algorithm: str = Field(description="ML algorithm used")
    training_timestamp: int = Field(
        description="Unix timestamp of training",
        ge=0
    )
    model_size_bytes: int = Field(
        description="Model size in bytes",
        ge=0
    )
    performance_metrics: Dict[str, float] = Field(
        default={},
        description="Performance metrics"
    )
    status: str = Field(description="Model status")

    class Config:
        protected_namespaces = ()


class ListModelsResponse(BaseModel):
    """Response listing available models."""

    models: List[ModelInfo] = Field(description="List of available models")
    total_count: int = Field(
        description="Total number of models",
        ge=0
    )


class HealthCheckDetail(BaseModel):
    """Detailed health check information."""

    check_name: str = Field(description="Name of health check")
    status: str = Field(description="Health check status")
    score: float = Field(
        description="Health check score",
        ge=0.0,
        le=1.0
    )
    message: str = Field(description="Health check message")
    details: Dict = Field(
        default={},
        description="Additional details"
    )


class DetailedHealthRequest(BaseModel):
    """Request for detailed health information."""

    include_trends: bool = Field(
        default=True,
        description="Include trend analysis"
    )
    trend_hours: int = Field(
        default=24,
        description="Hours of trend data to include",
        ge=1,
        le=168  # Max 1 week
    )


class DetailedHealthResponse(BaseModel):
    """Detailed health status response."""

    model_id: str = Field(description="Model identifier")
    version: str = Field(description="Model version")
    overall_status: str = Field(description="Overall health status")
    overall_score: float = Field(
        description="Overall health score",
        ge=0.0,
        le=1.0
    )
    health_checks: List[HealthCheckDetail] = Field(
        default=[],
        description="Individual health check results"
    )
    recommendations: List[str] = Field(
        default=[],
        description="Health improvement recommendations"
    )
    trends: Dict = Field(
        default={},
        description="Trend analysis data"
    )
    timestamp: int = Field(
        description="Unix timestamp of health check",
        ge=0
    )


class HealthStatus(BaseModel):
    """Basic health status response."""

    status: str = Field(description="Health status")
    timestamp: int = Field(description="Unix timestamp")
    version: str = Field(description="Service version")
    uptime_seconds: float = Field(description="Service uptime in seconds")


class ReadinessStatus(BaseModel):
    """Readiness status response."""

    ready: bool = Field(description="Whether service is ready")
    checks: Dict[str, bool] = Field(description="Individual readiness checks")
    timestamp: int = Field(description="Unix timestamp")




class ErrorResponse(BaseModel):
    """Standard error response."""

    error: str = Field(description="Error message")
    detail: Optional[str] = Field(default=None, description="Additional error details")
    timestamp: int = Field(description="Unix timestamp")