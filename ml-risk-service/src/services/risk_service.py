"""
Risk prediction service containing business logic for risk assessment.
This service is shared between gRPC and REST APIs.
"""

import logging
import time
import threading
from typing import Dict, Any, List, Optional, Tuple
import numpy as np

from src.models.ranking_model import RiskRankingModel
from src.models.feature_importance import FeatureImportanceAnalyzer
from src.monitoring.health_checker import ModelHealthChecker
from src.monitoring.drift_detector import ModelDriftMonitor
from src.api.schemas import (
    DeploymentRiskRequest,
    DeploymentRiskResponse,
    BatchDeploymentRiskRequest,
    BatchDeploymentRiskResponse,
    FeatureImportance,
    ModelHealthResponse,
    ModelMetrics
)

logger = logging.getLogger(__name__)


class RiskPredictionService:
    """Service for handling risk predictions with shared business logic."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}
        self.model = RiskRankingModel(config)
        self.feature_analyzer = FeatureImportanceAnalyzer()

        # Service metrics
        self.predictions_served = 0
        self.total_prediction_time = 0.0
        self.model_loaded = False
        self.current_model_id = None
        self.current_model_version = None

        # Thread safety
        self._model_lock = threading.RLock()

    def predict_deployment_risk(self, request: DeploymentRiskRequest) -> DeploymentRiskResponse:
        """
        Predict risk score for a single deployment.

        Args:
            request: Deployment risk request with features

        Returns:
            Risk prediction response with score and explanations

        Raises:
            ValueError: If no model is loaded or prediction fails
        """
        start_time = time.time()

        try:
            with self._model_lock:
                if not self.model_loaded:
                    raise ValueError("No trained model available")

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

                return DeploymentRiskResponse(
                    deployment_id=request.deployment_id,
                    risk_score=prediction.risk_score,
                    feature_importances=feature_importances,
                    model_version=prediction.model_version,
                    timestamp=int(time.time())
                )

        except Exception as e:
            logger.error(f"Prediction failed for deployment {request.deployment_id}: {e}")
            raise

    def predict_batch_deployment_risk(self, request: BatchDeploymentRiskRequest) -> BatchDeploymentRiskResponse:
        """
        Predict risk scores for multiple deployments.

        Args:
            request: Batch deployment risk request

        Returns:
            Batch response with individual predictions
        """
        responses = []

        for single_request in request.requests:
            try:
                response = self.predict_deployment_risk(single_request)
                responses.append(response)
            except Exception as e:
                logger.warning(f"Failed to predict for deployment {single_request.deployment_id}: {e}")
                # Create error response
                responses.append(DeploymentRiskResponse(
                    deployment_id=single_request.deployment_id,
                    risk_score=0.0,
                    feature_importances=[],
                    model_version="error",
                    timestamp=int(time.time())
                ))

        return BatchDeploymentRiskResponse(responses=responses)

    def get_model_health(self) -> ModelHealthResponse:
        """
        Get current model health status and metrics.

        Returns:
            Model health response with status and metrics
        """
        try:
            with self._model_lock:
                # Calculate current metrics
                avg_prediction_time = (
                    self.total_prediction_time / max(self.predictions_served, 1)
                    if self.predictions_served > 0 else 0.0
                )

                current_metrics = ModelMetrics(
                    current_ndcg=0.0,  # Would be populated from actual metrics
                    current_auc=0.0,   # Would be populated from actual metrics
                    predictions_served=self.predictions_served,
                    avg_prediction_time_ms=avg_prediction_time
                )

                return ModelHealthResponse(
                    healthy=self.model_loaded,
                    model_version=self.model.model_version or "none",
                    last_training_time=0,  # Would track actual training time
                    training_examples_count=0,  # Would track training data size
                    current_metrics=current_metrics
                )

        except Exception as e:
            logger.error(f"Health check failed: {e}")
            return ModelHealthResponse(
                healthy=False,
                model_version="error",
                last_training_time=0,
                training_examples_count=0,
                current_metrics=ModelMetrics(
                    current_ndcg=0.0,
                    current_auc=0.0,
                    predictions_served=0,
                    avg_prediction_time_ms=0.0
                )
            )

    def _extract_features_from_request(self, request: DeploymentRiskRequest) -> np.ndarray:
        """
        Extract feature vector from API request.

        Args:
            request: Deployment risk request

        Returns:
            Feature vector as numpy array
        """
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

    def _normalize_score(self, score: float, saturation: float, max_value: float) -> float:
        """Normalize score using StackRox normalization."""
        if score > saturation:
            return max_value
        return 1 + (score / saturation) * (max_value - 1)

    def _log_normalize(self, value: int) -> float:
        """Log normalize count values."""
        import math
        return math.log1p(value) / math.log1p(100)

    def load_model(self, model_file: str) -> bool:
        """
        Load a trained model from file.

        Args:
            model_file: Path to model file

        Returns:
            True if model loaded successfully
        """
        try:
            with self._model_lock:
                self.model.load_model(model_file)
                self.model_loaded = True
                self.current_model_version = getattr(self.model, 'model_version', 'file-loaded')
                logger.info(f"Model loaded from {model_file}")
                return True
        except Exception as e:
            logger.error(f"Failed to load model from {model_file}: {e}")
            return False

    def is_model_loaded(self) -> bool:
        """Check if a model is currently loaded."""
        return self.model_loaded

    def get_model_info(self) -> Dict[str, Any]:
        """Get information about the currently loaded model."""
        if not self.model_loaded:
            return {}

        try:
            return self.model.get_model_info()
        except Exception as e:
            logger.error(f"Failed to get model info: {e}")
            return {}