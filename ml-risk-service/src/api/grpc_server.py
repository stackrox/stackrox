"""
gRPC server for ML risk ranking service with hot model reloading support.
"""

import logging
import asyncio
import grpc
from concurrent import futures
import json
import os
import threading
import time
from typing import Dict, Any, List, Optional
import numpy as np
from contextlib import asynccontextmanager

# Import generated protobuf classes (would be generated from .proto files)
# For now, we'll create mock classes that match the protobuf definitions
from dataclasses import dataclass
from typing import List as TypingList

from ..models.ranking_model import RiskRankingModel
from ..models.feature_importance import FeatureImportanceAnalyzer
from ..storage.model_storage import ModelStorageManager, StorageConfig
from ..monitoring.health_checker import ModelHealthChecker
from ..monitoring.drift_detector import ModelDriftMonitor, DriftReport
from ..monitoring.alerting import AlertManager
from ...training.train_pipeline import TrainingPipeline

logger = logging.getLogger(__name__)


# Mock protobuf message classes (in practice, these would be auto-generated)
@dataclass
class DeploymentFeatures:
    policy_violation_count: int = 0
    policy_violation_severity_score: float = 0.0
    process_baseline_violations: int = 0
    host_network: bool = False
    host_pid: bool = False
    host_ipc: bool = False
    privileged_container_count: int = 0
    automount_service_account_token: bool = False
    exposed_port_count: int = 0
    has_external_exposure: bool = False
    service_account_permission_level: float = 0.0
    replica_count: int = 1
    is_orchestrator_component: bool = False
    is_platform_component: bool = False
    cluster_id: str = ""
    namespace: str = ""
    creation_timestamp: int = 0
    is_inactive: bool = False


@dataclass
class ImageFeatures:
    image_id: str = ""
    image_name: str = ""
    critical_vuln_count: int = 0
    high_vuln_count: int = 0
    medium_vuln_count: int = 0
    low_vuln_count: int = 0
    avg_cvss_score: float = 0.0
    max_cvss_score: float = 0.0
    total_component_count: int = 0
    risky_component_count: int = 0
    image_creation_timestamp: int = 0
    image_age_days: int = 0
    is_cluster_local: bool = False
    base_image: str = ""
    layer_count: int = 0


@dataclass
class FeatureImportance:
    feature_name: str
    importance_score: float
    feature_category: str
    description: str


@dataclass
class DeploymentRiskRequest:
    deployment_id: str
    deployment_features: DeploymentFeatures
    image_features: TypingList[ImageFeatures]


@dataclass
class DeploymentRiskResponse:
    deployment_id: str
    risk_score: float
    feature_importances: TypingList[FeatureImportance]
    model_version: str
    timestamp: int


@dataclass
class BatchDeploymentRiskRequest:
    requests: TypingList[DeploymentRiskRequest]


@dataclass
class BatchDeploymentRiskResponse:
    responses: TypingList[DeploymentRiskResponse]


@dataclass
class TrainingExample:
    deployment_features: DeploymentFeatures
    image_features: TypingList[ImageFeatures]
    current_risk_score: float
    deployment_id: str


@dataclass
class TrainModelRequest:
    training_data: TypingList[TrainingExample]
    config_override: str = ""


@dataclass
class TrainingMetrics:
    validation_ndcg: float
    validation_auc: float
    training_loss: float
    epochs_completed: int
    global_feature_importance: TypingList[FeatureImportance]


@dataclass
class TrainModelResponse:
    success: bool
    model_version: str
    metrics: TrainingMetrics
    error_message: str = ""


@dataclass
class ModelHealthRequest:
    pass


@dataclass
class ModelMetrics:
    current_ndcg: float
    current_auc: float
    predictions_served: int
    avg_prediction_time_ms: float


@dataclass
class ModelHealthResponse:
    healthy: bool
    model_version: str
    last_training_time: int
    training_examples_count: int
    current_metrics: ModelMetrics


@dataclass
class ReloadModelRequest:
    model_id: str
    version: str = ""  # Empty for latest version
    force_reload: bool = False


@dataclass
class ReloadModelResponse:
    success: bool
    message: str
    previous_model_version: str
    new_model_version: str
    reload_time_ms: float


@dataclass
class ListModelsRequest:
    model_id: str = ""  # Empty for all models


@dataclass
class ModelInfo:
    model_id: str
    version: str
    algorithm: str
    training_timestamp: int
    model_size_bytes: int
    performance_metrics: Dict[str, float]
    status: str


@dataclass
class ListModelsResponse:
    models: TypingList[ModelInfo]
    total_count: int

@dataclass
class DetailedHealthRequest:
    include_trends: bool = True
    trend_hours: int = 24

@dataclass
class HealthCheckDetail:
    check_name: str
    status: str
    score: float
    message: str
    details: dict

@dataclass
class DetailedHealthResponse:
    model_id: str
    version: str
    overall_status: str
    overall_score: float
    health_checks: TypingList[HealthCheckDetail]
    recommendations: TypingList[str]
    trends: dict
    timestamp: int


class MLRiskServiceImpl:
    """Implementation of ML Risk Service gRPC methods with hot reloading support."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}
        self.model = RiskRankingModel(config)
        self.feature_analyzer = FeatureImportanceAnalyzer()
        self.training_pipeline = TrainingPipeline()

        # Initialize storage manager
        storage_config = self._create_storage_config()
        self.storage_manager = ModelStorageManager(storage_config)

        # Initialize health checker
        self.health_checker = ModelHealthChecker()
        self._setup_health_checks()

        # Initialize drift monitoring
        drift_config = self.config.get('drift_monitoring', {})
        self.drift_monitor = ModelDriftMonitor(drift_config)

        # Initialize alerting system
        alert_config = self.config.get('alerting', {})
        self.alert_manager = AlertManager(alert_config)

        # Service metrics
        self.predictions_served = 0
        self.total_prediction_time = 0.0
        self.last_training_time = 0
        self.training_examples_count = 0
        self.model_loaded = False
        self.current_model_id = None
        self.current_model_version = None

        # Thread safety
        self._model_lock = threading.RLock()

        # Try to load existing model if configured
        model_file = self.config.get('model_file')
        if model_file and os.path.exists(model_file):
            self._load_model_from_file(model_file)
        else:
            # Try to load the default or latest model from storage
            self._load_default_model()

    def _create_storage_config(self) -> StorageConfig:
        """Create storage configuration from service config."""
        # Use environment variables to create storage config
        return StorageConfig.from_env()

    def _setup_health_checks(self):
        """Setup default health checks."""
        try:
            # Setup baseline metrics if available
            baseline_metrics = self.config.get('health_baseline_metrics', {
                'validation_ndcg': 0.75,
                'validation_auc': 0.70,
                'training_loss': 0.4
            })

            self.health_checker.setup_default_checks(baseline_metrics)
            logger.info("Health checks configured successfully")

        except Exception as e:
            logger.error(f"Failed to setup health checks: {e}")

    def _load_default_model(self):
        """Load the default model from storage."""
        try:
            # Try to find the default model ID from config or environment
            default_model_id = (
                self.config.get('default_model_id') or
                os.environ.get('ROX_ML_DEFAULT_MODEL_ID', 'stackrox-risk-model')
            )

            # Try to load the latest version of the default model
            models = self.storage_manager.list_models(default_model_id)
            if models:
                latest_model = models[0]  # list_models returns sorted by timestamp desc
                if self._load_model_from_storage(latest_model.model_id, latest_model.version):
                    logger.info(f"Loaded default model {latest_model.model_id} v{latest_model.version}")
                    return

            logger.warning(f"No default model found for {default_model_id}")

        except Exception as e:
            logger.error(f"Failed to load default model: {e}")

    def _load_model_from_file(self, model_file: str):
        """Load model from a file (legacy method)."""
        try:
            with self._model_lock:
                self.model.load_model(model_file)
                self.model_loaded = True
                self.current_model_version = getattr(self.model, 'model_version', 'file-loaded')
                logger.info(f"Model loaded from file: {model_file}")
        except Exception as e:
            logger.error(f"Failed to load model from file {model_file}: {e}")

    def _load_model_from_storage(self, model_id: str, version: Optional[str] = None) -> bool:
        """Load model from storage manager."""
        try:
            with self._model_lock:
                success = self.model.load_model_from_storage(self.storage_manager, model_id, version)
                if success:
                    self.model_loaded = True
                    self.current_model_id = model_id
                    self.current_model_version = version or "latest"
                    logger.info(f"Model loaded from storage: {model_id} v{self.current_model_version}")
                return success
        except Exception as e:
            logger.error(f"Failed to load model from storage {model_id} v{version}: {e}")
            return False

    def GetDeploymentRisk(self, request: DeploymentRiskRequest, context) -> DeploymentRiskResponse:
        """Get risk score for a single deployment."""
        start_time = time.time()

        try:
            with self._model_lock:
                if not self.model_loaded:
                    context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                    context.set_details("No trained model available")
                    return DeploymentRiskResponse(
                        deployment_id=request.deployment_id,
                        risk_score=0.0,
                        feature_importances=[],
                        model_version="none",
                        timestamp=int(time.time())
                    )

                # Convert request to feature vector
                features = self._extract_features_from_request(request)

                # Get prediction
                predictions = self.model.predict(features.reshape(1, -1), explain=True)
                prediction = predictions[0]

                # Convert feature importance to response format
                feature_importances = [
                    FeatureImportance(
                        feature_name=name,
                        importance_score=score,
                        feature_category=self.feature_analyzer.feature_categories.get(name, 'other'),
                        description=self.feature_analyzer.feature_descriptions.get(name, 'No description')
                    )
                    for name, score in prediction.feature_importance.items()
                ]

                # Update metrics
                prediction_time = (time.time() - start_time) * 1000  # ms
                self.predictions_served += 1
                self.total_prediction_time += prediction_time

                # Record prediction for health monitoring
                prediction_data = {
                    'deployment_id': request.deployment_id,
                    'risk_score': prediction.risk_score,
                    'model_version': prediction.model_version,
                    'latency_ms': prediction_time,
                    'feature_count': len(features),
                    'prediction_timestamp': time.time()
                }
                self.health_checker.record_prediction(prediction_data)

                # Record prediction for drift monitoring
                feature_dict = self._convert_features_to_dict(features, request)
                self.drift_monitor.record_prediction(
                    self.current_model_id or "unknown",
                    self.current_model_version or "unknown",
                    feature_dict,
                    prediction.risk_score,
                    time.time()
                )

                return DeploymentRiskResponse(
                    deployment_id=request.deployment_id,
                    risk_score=prediction.risk_score,
                    feature_importances=feature_importances,
                    model_version=prediction.model_version,
                    timestamp=int(time.time())
                )

        except Exception as e:
            logger.error(f"Prediction failed for deployment {request.deployment_id}: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Prediction failed: {str(e)}")
            return DeploymentRiskResponse(
                deployment_id=request.deployment_id,
                risk_score=0.0,
                feature_importances=[],
                model_version="error",
                timestamp=int(time.time())
            )

    def GetBatchDeploymentRisk(self, request: BatchDeploymentRiskRequest, context) -> BatchDeploymentRiskResponse:
        """Get risk scores for multiple deployments."""
        responses = []

        for single_request in request.requests:
            response = self.GetDeploymentRisk(single_request, context)
            responses.append(response)

        return BatchDeploymentRiskResponse(responses=responses)

    def TrainModel(self, request: TrainModelRequest, context) -> TrainModelResponse:
        """Train the model with new data."""
        try:
            with self._model_lock:
                logger.info(f"Starting model training with {len(request.training_data)} examples")

                # Convert training data to internal format
                training_examples = []
                for example in request.training_data:
                    # Convert protobuf training example to internal format
                    features = self._extract_features_from_training_example(example)
                    training_examples.append({
                        'features': features,
                        'risk_score': example.current_risk_score,
                        'deployment_id': example.deployment_id
                    })

                # Create ranking dataset
                X, y, groups = self.training_pipeline.data_loader.create_ranking_dataset(training_examples)
                feature_names = sorted(training_examples[0]['features'].keys())

                # Train model
                training_metrics = self.model.train(X, y, groups, feature_names)

                # Update service state
                self.model_loaded = True
                self.last_training_time = int(time.time())
                self.training_examples_count = len(training_examples)

                # Convert metrics to response format
                global_importance = self.model.get_global_feature_importance()
                feature_importances = [
                    FeatureImportance(
                        feature_name=name,
                        importance_score=score,
                        feature_category=self.feature_analyzer.feature_categories.get(name, 'other'),
                        description=self.feature_analyzer.feature_descriptions.get(name, 'No description')
                    )
                    for name, score in global_importance.items()
                ]

                response_metrics = TrainingMetrics(
                    validation_ndcg=training_metrics.val_ndcg,
                    validation_auc=training_metrics.val_auc,
                    training_loss=training_metrics.training_loss,
                    epochs_completed=training_metrics.epochs_completed,
                    global_feature_importance=feature_importances
                )

                logger.info(f"Model training completed. Validation NDCG: {training_metrics.val_ndcg:.4f}")

                return TrainModelResponse(
                    success=True,
                    model_version=self.model.model_version or "unknown",
                    metrics=response_metrics,
                    error_message=""
                )

        except Exception as e:
            logger.error(f"Model training failed: {e}")
            return TrainModelResponse(
                success=False,
                model_version="",
                metrics=TrainingMetrics(0, 0, 0, 0, []),
                error_message=str(e)
            )

    def GetModelHealth(self, request: ModelHealthRequest, context) -> ModelHealthResponse:
        """Get model health and metrics."""
        try:
            with self._model_lock:
                # Calculate current metrics
                avg_prediction_time = (
                    self.total_prediction_time / max(self.predictions_served, 1)
                    if self.predictions_served > 0 else 0.0
                )

                # Run comprehensive health checks
                current_performance_metrics = {
                    'avg_prediction_time_ms': avg_prediction_time,
                    'predictions_served': self.predictions_served,
                    'model_loaded': 1.0 if self.model_loaded else 0.0
                }

                # Add model-specific metrics if available
                if hasattr(self.model, 'get_current_metrics'):
                    try:
                        model_metrics = self.model.get_current_metrics()
                        current_performance_metrics.update(model_metrics)
                    except Exception as e:
                        logger.warning(f"Failed to get model metrics: {e}")

                # Run health checks
                health_report = self.health_checker.run_health_checks(
                    model=self.model,
                    model_id=self.current_model_id or "unknown",
                    version=self.current_model_version or "unknown",
                    current_metrics=current_performance_metrics
                )

                # Determine overall health based on the report
                is_healthy = (self.model_loaded and
                            health_report.overall_status in ["healthy", "warning"])

                current_metrics = ModelMetrics(
                    current_ndcg=current_performance_metrics.get('validation_ndcg', 0.0),
                    current_auc=current_performance_metrics.get('validation_auc', 0.0),
                    predictions_served=self.predictions_served,
                    avg_prediction_time_ms=avg_prediction_time
                )

                return ModelHealthResponse(
                    healthy=is_healthy,
                    model_version=self.model.model_version or "none",
                    last_training_time=self.last_training_time,
                    training_examples_count=self.training_examples_count,
                    current_metrics=current_metrics
                )

        except Exception as e:
            logger.error(f"Health check failed: {e}")
            return ModelHealthResponse(
                healthy=False,
                model_version="error",
                last_training_time=0,
                training_examples_count=0,
                current_metrics=ModelMetrics(0, 0, 0, 0)
            )

    def _extract_features_from_request(self, request: DeploymentRiskRequest) -> np.ndarray:
        """Extract feature vector from gRPC request."""
        # Extract deployment features
        deployment_features = {
            'policy_violation_score': self._normalize_score(
                request.deployment_features.policy_violation_severity_score, 50, 4.0),
            'host_network': float(request.deployment_features.host_network),
            'host_pid': float(request.deployment_features.host_pid),
            'host_ipc': float(request.deployment_features.host_ipc),
            'has_external_exposure': float(request.deployment_features.has_external_exposure),
            'is_orchestrator_component': float(request.deployment_features.is_orchestrator_component),
            'automount_service_account_token': float(request.deployment_features.automount_service_account_token),
            'log_replica_count': self._log_normalize(request.deployment_features.replica_count),
            'log_exposed_port_count': self._log_normalize(request.deployment_features.exposed_port_count),
            'privileged_container_ratio': min(
                request.deployment_features.privileged_container_count /
                max(request.deployment_features.replica_count, 1), 1.0),
        }

        # Calculate deployment age
        if request.deployment_features.creation_timestamp > 0:
            age_days = (time.time() - request.deployment_features.creation_timestamp) / 86400
            deployment_features['age_days'] = min(age_days / 365.0, 5.0)
        else:
            deployment_features['age_days'] = 0.0

        # Extract image features (aggregate across images)
        if request.image_features:
            image_vulnerability_scores = []
            image_component_scores = []
            image_age_scores = []

            for img in request.image_features:
                # Vulnerability score
                vuln_score = (
                    img.critical_vuln_count * 10.0 +
                    img.high_vuln_count * 4.0 +
                    img.medium_vuln_count * 1.0 +
                    img.low_vuln_count * 0.25
                )
                image_vulnerability_scores.append(min(vuln_score / 100.0, 10.0))

                # Component score
                component_score = self._normalize_score(img.total_component_count, 500, 1.5)
                image_component_scores.append(component_score)

                # Age score
                if img.image_age_days > 0:
                    age_score = min(img.image_age_days / 365.0, 2.0)
                    image_age_scores.append(self._normalize_score(age_score, 1.0, 1.3))
                else:
                    image_age_scores.append(1.0)

            # Aggregate image features
            deployment_features.update({
                'avg_vulnerability_score': np.mean(image_vulnerability_scores),
                'max_vulnerability_score': np.max(image_vulnerability_scores),
                'sum_vulnerability_score': np.sum(image_vulnerability_scores),
                'avg_component_count_score': np.mean(image_component_scores),
                'avg_age_score': np.mean(image_age_scores),
                'max_age_score': np.max(image_age_scores),
            })

            # Risky component ratios
            risky_ratios = [
                img.risky_component_count / max(img.total_component_count, 1)
                for img in request.image_features
            ]
            deployment_features.update({
                'avg_risky_component_ratio': np.mean(risky_ratios),
                'max_risky_component_ratio': np.max(risky_ratios),
            })

        else:
            # No images - use default values
            deployment_features.update({
                'avg_vulnerability_score': 0.0,
                'max_vulnerability_score': 0.0,
                'sum_vulnerability_score': 0.0,
                'avg_component_count_score': 1.0,
                'avg_age_score': 1.0,
                'max_age_score': 1.0,
                'avg_risky_component_ratio': 0.0,
                'max_risky_component_ratio': 0.0,
            })

        # Convert to numpy array in consistent order
        if self.model.feature_names:
            feature_vector = np.array([
                deployment_features.get(name, 0.0)
                for name in self.model.feature_names
            ])
        else:
            # Fallback order
            feature_names = sorted(deployment_features.keys())
            feature_vector = np.array([deployment_features[name] for name in feature_names])

        return feature_vector

    def _extract_features_from_training_example(self, example: TrainingExample) -> Dict[str, float]:
        """Extract features from training example."""
        # Create a mock request and extract features
        mock_request = DeploymentRiskRequest(
            deployment_id=example.deployment_id,
            deployment_features=example.deployment_features,
            image_features=example.image_features
        )

        feature_vector = self._extract_features_from_request(mock_request)

        # Convert back to dictionary (this is inefficient but works for the prototype)
        feature_names = [
            'policy_violation_score', 'host_network', 'host_pid', 'host_ipc',
            'has_external_exposure', 'is_orchestrator_component',
            'automount_service_account_token', 'log_replica_count',
            'log_exposed_port_count', 'privileged_container_ratio', 'age_days',
            'avg_vulnerability_score', 'max_vulnerability_score',
            'sum_vulnerability_score', 'avg_component_count_score',
            'avg_age_score', 'max_age_score', 'avg_risky_component_ratio',
            'max_risky_component_ratio'
        ]

        return dict(zip(feature_names[:len(feature_vector)], feature_vector))

    def _convert_features_to_dict(self, feature_vector: np.ndarray, request: DeploymentRiskRequest) -> Dict[str, float]:
        """Convert feature vector to named dictionary for drift monitoring."""
        feature_names = [
            'policy_violation_score', 'host_network', 'host_pid', 'host_ipc',
            'has_external_exposure', 'is_orchestrator_component',
            'automount_service_account_token', 'log_replica_count',
            'log_exposed_port_count', 'privileged_container_ratio', 'age_days',
            'avg_vulnerability_score', 'max_vulnerability_score',
            'sum_vulnerability_score', 'avg_component_count_score',
            'avg_age_score', 'max_age_score', 'avg_risky_component_ratio',
            'max_risky_component_ratio'
        ]

        # Use model's feature names if available, otherwise use default
        if hasattr(self.model, 'feature_names') and self.model.feature_names:
            feature_names = self.model.feature_names

        return dict(zip(feature_names[:len(feature_vector)], feature_vector.tolist()))

    def _normalize_score(self, score: float, saturation: float, max_value: float) -> float:
        """Normalize score using StackRox normalization."""
        if score > saturation:
            return max_value
        return 1 + (score / saturation) * (max_value - 1)

    def _log_normalize(self, value: int) -> float:
        """Log normalize count values."""
        import math
        return math.log1p(value) / math.log1p(100)

    def _load_model(self, model_file: str) -> bool:
        """Load a trained model from file."""
        try:
            with self._model_lock:
                self.model.load_model(model_file)
                self.model_loaded = True
                logger.info(f"Model loaded from {model_file}")
                return True
        except Exception as e:
            logger.error(f"Failed to load model from {model_file}: {e}")
            return False

    def ReloadModel(self, request: ReloadModelRequest, context) -> ReloadModelResponse:
        """Hot reload a model from storage."""
        start_time = time.time()
        previous_version = self.current_model_version or "none"

        try:
            if not request.model_id:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Model ID is required")
                return ReloadModelResponse(
                    success=False,
                    message="Model ID is required",
                    previous_model_version=previous_version,
                    new_model_version="",
                    reload_time_ms=0.0
                )

            # Check if the model is already loaded (unless force reload)
            if (not request.force_reload and
                self.current_model_id == request.model_id and
                (not request.version or self.current_model_version == request.version)):
                reload_time = (time.time() - start_time) * 1000
                return ReloadModelResponse(
                    success=True,
                    message=f"Model {request.model_id} v{self.current_model_version} already loaded",
                    previous_model_version=previous_version,
                    new_model_version=self.current_model_version,
                    reload_time_ms=reload_time
                )

            # Attempt to load the model from storage
            success = self._load_model_from_storage(request.model_id, request.version)
            reload_time = (time.time() - start_time) * 1000

            if success:
                new_version = self.current_model_version
                message = f"Successfully reloaded model {request.model_id} v{new_version}"
                logger.info(f"Hot reload successful: {request.model_id} v{new_version} (took {reload_time:.1f}ms)")

                return ReloadModelResponse(
                    success=True,
                    message=message,
                    previous_model_version=previous_version,
                    new_model_version=new_version,
                    reload_time_ms=reload_time
                )
            else:
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(f"Failed to load model {request.model_id} v{request.version}")
                return ReloadModelResponse(
                    success=False,
                    message=f"Failed to load model {request.model_id} v{request.version}",
                    previous_model_version=previous_version,
                    new_model_version="",
                    reload_time_ms=reload_time
                )

        except Exception as e:
            reload_time = (time.time() - start_time) * 1000
            logger.error(f"Model reload failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Model reload failed: {str(e)}")
            return ReloadModelResponse(
                success=False,
                message=f"Model reload failed: {str(e)}",
                previous_model_version=previous_version,
                new_model_version="",
                reload_time_ms=reload_time
            )

    def ListModels(self, request: ListModelsRequest, context) -> ListModelsResponse:
        """List available models in storage."""
        try:
            if request.model_id:
                # List versions for specific model
                models = self.storage_manager.list_models(request.model_id)
            else:
                # List all models
                models = self.storage_manager.list_models()

            model_infos = []
            for model in models:
                # Convert performance metrics to simple dict
                metrics = {}
                if hasattr(model, 'performance_metrics') and model.performance_metrics:
                    metrics = {k: float(v) for k, v in model.performance_metrics.items()
                              if isinstance(v, (int, float))}

                model_info = ModelInfo(
                    model_id=model.model_id,
                    version=model.version,
                    algorithm=model.algorithm,
                    training_timestamp=int(model.training_timestamp.timestamp()),
                    model_size_bytes=model.model_size_bytes,
                    performance_metrics=metrics,
                    status="ready"  # Default status, could be enhanced
                )
                model_infos.append(model_info)

            return ListModelsResponse(
                models=model_infos,
                total_count=len(model_infos)
            )

        except Exception as e:
            logger.error(f"Failed to list models: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to list models: {str(e)}")
            return ListModelsResponse(models=[], total_count=0)

    def GetDetailedHealth(self, request: DetailedHealthRequest, context) -> DetailedHealthResponse:
        """Get detailed health report with trends and recommendations."""
        try:
            with self._model_lock:
                if not self.model_loaded:
                    return DetailedHealthResponse(
                        model_id="none",
                        version="none",
                        overall_status="error",
                        overall_score=0.0,
                        health_checks=[],
                        recommendations=["No model loaded"],
                        trends={},
                        timestamp=int(time.time())
                    )

                # Calculate current metrics
                avg_prediction_time = (
                    self.total_prediction_time / max(self.predictions_served, 1)
                    if self.predictions_served > 0 else 0.0
                )

                current_performance_metrics = {
                    'avg_prediction_time_ms': avg_prediction_time,
                    'predictions_served': self.predictions_served,
                    'model_loaded': 1.0 if self.model_loaded else 0.0
                }

                # Add model-specific metrics if available
                if hasattr(self.model, 'get_current_metrics'):
                    try:
                        model_metrics = self.model.get_current_metrics()
                        current_performance_metrics.update(model_metrics)
                    except Exception as e:
                        logger.warning(f"Failed to get model metrics: {e}")

                # Run health checks
                health_report = self.health_checker.run_health_checks(
                    model=self.model,
                    model_id=self.current_model_id or "unknown",
                    version=self.current_model_version or "unknown",
                    current_metrics=current_performance_metrics
                )

                # Convert health checks to response format
                health_check_details = []
                for check in health_report.health_checks:
                    health_check_details.append(HealthCheckDetail(
                        check_name=check.check_name,
                        status=check.status,
                        score=check.score,
                        message=check.message,
                        details=check.details or {}
                    ))

                # Get trends if requested
                trends = {}
                if request.include_trends:
                    trends = self.health_checker.get_health_trends(request.trend_hours)

                return DetailedHealthResponse(
                    model_id=self.current_model_id or "unknown",
                    version=self.current_model_version or "unknown",
                    overall_status=health_report.overall_status,
                    overall_score=health_report.overall_score,
                    health_checks=health_check_details,
                    recommendations=health_report.recommendations,
                    trends=trends,
                    timestamp=int(time.time())
                )

        except Exception as e:
            logger.error(f"Detailed health check failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Detailed health check failed: {str(e)}")
            return DetailedHealthResponse(
                model_id="error",
                version="error",
                overall_status="error",
                overall_score=0.0,
                health_checks=[],
                recommendations=[f"Health check error: {str(e)}"],
                trends={},
                timestamp=int(time.time())
            )

    def SetBaselineData(self, request, context):
        """Set baseline data for drift monitoring."""
        try:
            # This would be called after training or when setting up monitoring
            # For now, we'll use current prediction history as baseline
            model_id = self.current_model_id or "unknown"
            version = self.current_model_version or "unknown"

            # Extract baseline feature data from recent predictions
            recent_predictions = list(self.drift_monitor.prediction_history)[-1000:]  # Last 1000 predictions
            if len(recent_predictions) < 100:
                context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                context.set_details("Insufficient prediction history for baseline (need at least 100)")
                return

            # Group features by name
            baseline_features = {}
            for pred in recent_predictions:
                if pred['model_id'] == model_id:
                    for feature_name, value in pred['features'].items():
                        if feature_name not in baseline_features:
                            baseline_features[feature_name] = []
                        baseline_features[feature_name].append(value)

            # Convert to numpy arrays
            baseline_arrays = {
                name: np.array(values) for name, values in baseline_features.items()
                if len(values) >= 50  # Minimum samples per feature
            }

            if not baseline_arrays:
                context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                context.set_details("No sufficient feature data for baseline")
                return

            # Set baseline data
            self.drift_monitor.set_baseline_data(model_id, version, baseline_arrays)

            return {"success": True, "message": f"Baseline set for {model_id} v{version} with {len(baseline_arrays)} features"}

        except Exception as e:
            logger.error(f"Failed to set baseline data: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to set baseline data: {str(e)}")
            return {"success": False, "message": str(e)}

    def GetDriftReport(self, request, context):
        """Get comprehensive drift report for a model."""
        try:
            model_id = getattr(request, 'model_id', self.current_model_id or "unknown")
            version = getattr(request, 'version', self.current_model_version or "unknown")
            period_hours = getattr(request, 'period_hours', 24)

            # Generate drift report
            drift_report = self.drift_monitor.generate_drift_report(model_id, version, period_hours)

            # Send alerts if necessary
            if drift_report.active_alerts:
                asyncio.create_task(self._send_drift_alerts(drift_report.active_alerts))

            # Convert to response format
            return {
                "model_id": drift_report.model_id,
                "version": drift_report.version,
                "overall_drift_status": drift_report.overall_drift_status,
                "overall_drift_score": drift_report.overall_drift_score,
                "data_drift_score": drift_report.data_drift_score,
                "prediction_drift_score": drift_report.prediction_drift_score,
                "performance_drift_score": drift_report.performance_drift_score,
                "active_alerts_count": len(drift_report.active_alerts),
                "recommendations": drift_report.recommendations,
                "timestamp": drift_report.timestamp
            }

        except Exception as e:
            logger.error(f"Failed to generate drift report: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to generate drift report: {str(e)}")
            return {
                "model_id": "error",
                "version": "error",
                "overall_drift_status": "error",
                "overall_drift_score": 0.0,
                "data_drift_score": 0.0,
                "prediction_drift_score": 0.0,
                "performance_drift_score": 0.0,
                "active_alerts_count": 0,
                "recommendations": [f"Error generating report: {str(e)}"],
                "timestamp": int(time.time())
            }

    def GetActiveAlerts(self, request, context):
        """Get active drift alerts."""
        try:
            model_id = getattr(request, 'model_id', None)
            severity = getattr(request, 'severity', None)

            alerts = self.drift_monitor.get_active_alerts(model_id, severity)

            return {
                "alerts": [
                    {
                        "alert_id": alert.alert_id,
                        "drift_type": alert.drift_type,
                        "severity": alert.severity,
                        "metric_name": alert.metric_name,
                        "drift_score": alert.drift_score,
                        "threshold": alert.threshold,
                        "message": alert.message,
                        "timestamp": alert.timestamp
                    }
                    for alert in alerts
                ],
                "total_count": len(alerts)
            }

        except Exception as e:
            logger.error(f"Failed to get active alerts: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to get active alerts: {str(e)}")
            return {"alerts": [], "total_count": 0}

    async def _send_drift_alerts(self, alerts):
        """Send drift alerts through alert manager."""
        try:
            for alert in alerts:
                await self.alert_manager.send_alert(alert)
        except Exception as e:
            logger.error(f"Failed to send drift alerts: {e}")


class MLRiskServer:
    """gRPC server for ML Risk Service."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}
        self.server = None
        self.service_impl = MLRiskServiceImpl(config)

    def start_server(self, port: int = 8080, max_workers: int = 10) -> None:
        """Start the gRPC server."""
        self.server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))

        # Add service implementation
        # In practice, this would use generated gRPC code:
        # ml_risk_service_pb2_grpc.add_MLRiskServiceServicer_to_server(self.service_impl, self.server)

        # For now, we'll create a mock servicer registration
        logger.info("Adding ML Risk Service to gRPC server")

        listen_addr = f'[::]:{port}'
        self.server.add_insecure_port(listen_addr)

        self.server.start()
        logger.info(f"ML Risk Service gRPC server started on {listen_addr}")

    def stop_server(self, grace_period: float = 30.0) -> None:
        """Stop the gRPC server."""
        if self.server:
            logger.info("Stopping ML Risk Service gRPC server")
            self.server.stop(grace_period)
            self.server = None

    def serve_forever(self) -> None:
        """Start server and wait for termination."""
        port = self.config.get('api', {}).get('grpc_port', 8080)
        max_workers = self.config.get('api', {}).get('max_workers', 10)

        self.start_server(port, max_workers)

        try:
            # Keep server running
            while True:
                time.sleep(86400)  # Sleep for a day
        except KeyboardInterrupt:
            logger.info("Server interrupted")
        finally:
            self.stop_server()


def create_server_from_config(config_file: str) -> MLRiskServer:
    """Create server from configuration file."""
    import yaml

    with open(config_file, 'r') as f:
        config = yaml.safe_load(f)

    return MLRiskServer(config)


def main():
    """Main entry point for the gRPC server."""
    import argparse
    import yaml

    parser = argparse.ArgumentParser(description='ML Risk Service gRPC Server')
    parser.add_argument('--config', help='Configuration file path')
    parser.add_argument('--port', type=int, default=8080, help='Server port')
    parser.add_argument('--workers', type=int, default=10, help='Max worker threads')
    parser.add_argument('--model', help='Pre-trained model file to load')

    args = parser.parse_args()

    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Load configuration
    config = {}
    if args.config:
        with open(args.config, 'r') as f:
            config = yaml.safe_load(f)

    # Override with command line args
    if not config.get('api'):
        config['api'] = {}
    config['api']['grpc_port'] = args.port
    config['api']['max_workers'] = args.workers

    if args.model:
        config['model_file'] = args.model

    # Create and start server
    server = MLRiskServer(config)
    logger.info("Starting ML Risk Service...")

    try:
        server.serve_forever()
    except Exception as e:
        logger.error(f"Server failed: {e}")
        return 1

    return 0


if __name__ == '__main__':
    exit(main())