"""
Training service for ML model training and management.
This service is shared between gRPC and REST APIs.
"""

import logging
import threading
import time
from typing import Dict, Any, List, Optional

from training.train_pipeline import TrainingPipeline
from src.storage.model_storage import ModelStorageManager, StorageConfig
from src.api.schemas import (
    TrainModelRequest,
    TrainModelResponse,
    TrainingMetrics,
    FeatureImportance,
    TrainingExample,
    QuickTestPipelineResponse
)

logger = logging.getLogger(__name__)


class TrainingService:
    """Service for handling model training operations."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}

        # Initialize model storage manager
        storage_config = StorageConfig.from_env()
        self.storage_manager = ModelStorageManager(storage_config)

        # Initialize training pipeline with storage manager
        self.training_pipeline = TrainingPipeline(storage_manager=self.storage_manager)

        # Training state
        self.last_training_time = 0
        self.training_examples_count = 0

        # Thread safety
        self._training_lock = threading.RLock()

    def train_model(self, request: TrainModelRequest, risk_service) -> TrainModelResponse:
        """
        Train the model with new data.

        Args:
            request: Training request with data and configuration
            risk_service: Risk service instance to update with trained model

        Returns:
            Training response with metrics and status
        """
        try:
            with self._training_lock:
                logger.info(f"Starting model training with {len(request.training_data)} examples")

                # Convert training data to internal format
                training_examples = []
                for example in request.training_data:
                    # Convert Pydantic models to dictionary format expected by training pipeline
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
                training_metrics = risk_service.model.train(X, y, groups, feature_names)

                # Update service state
                risk_service.model_loaded = True
                self.last_training_time = int(time.time())
                self.training_examples_count = len(training_examples)

                # Convert metrics to response format
                global_importance = risk_service.model.get_global_feature_importance()
                feature_importances = [
                    FeatureImportance(
                        feature_name=name,
                        importance_score=score,
                        feature_category=risk_service.feature_analyzer.feature_categories.get(name, 'other'),
                        description=risk_service.feature_analyzer.feature_descriptions.get(name, 'No description')
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
                    model_version=risk_service.model.model_version or "unknown",
                    metrics=response_metrics,
                    error_message=""
                )

        except Exception as e:
            logger.error(f"Model training failed: {e}")
            return TrainModelResponse(
                success=False,
                model_version="",
                metrics=TrainingMetrics(
                    validation_ndcg=0.0,
                    validation_auc=0.0,
                    training_loss=0.0,
                    epochs_completed=0,
                    global_feature_importance=[]
                ),
                error_message=str(e)
            )

    def _extract_features_from_training_example(self, example: TrainingExample) -> Dict[str, float]:
        """
        Extract features from training example.

        Args:
            example: Training example with deployment and image features

        Returns:
            Dictionary of extracted features
        """
        # This is similar to the feature extraction in risk_service
        # but adapted for training data format

        deployment_features = {
            'policy_violation_score': self._normalize_score(
                example.deployment_features.policy_violation_severity_score, 50, 4.0),
            'host_network': float(example.deployment_features.host_network),
            'host_pid': float(example.deployment_features.host_pid),
            'host_ipc': float(example.deployment_features.host_ipc),
            'has_external_exposure': float(example.deployment_features.has_external_exposure),
            'is_orchestrator_component': float(example.deployment_features.is_orchestrator_component),
            'automount_service_account_token': float(example.deployment_features.automount_service_account_token),
            'log_replica_count': self._log_normalize(example.deployment_features.replica_count),
            'log_exposed_port_count': self._log_normalize(example.deployment_features.exposed_port_count),
            'privileged_container_ratio': min(
                example.deployment_features.privileged_container_count /
                max(example.deployment_features.replica_count, 1), 1.0),
        }

        # Calculate deployment age
        if example.deployment_features.creation_timestamp > 0:
            import time
            age_days = (time.time() - example.deployment_features.creation_timestamp) / 86400
            deployment_features['age_days'] = min(age_days / 365.0, 5.0)
        else:
            deployment_features['age_days'] = 0.0

        # Extract image features (aggregate across images)
        if example.image_features:
            import numpy as np

            image_vulnerability_scores = []
            image_component_scores = []
            image_age_scores = []

            for img in example.image_features:
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
                for img in example.image_features
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

        return deployment_features

    def _normalize_score(self, score: float, saturation: float, max_value: float) -> float:
        """Normalize score using StackRox normalization."""
        if score > saturation:
            return max_value
        return 1 + (score / saturation) * (max_value - 1)

    def _log_normalize(self, value: int) -> float:
        """Log normalize count values."""
        import math
        return math.log1p(value) / math.log1p(100)

    def generate_sample_training_data(self, num_examples: int = 100) -> str:
        """
        Generate sample training data for testing.

        Args:
            num_examples: Number of examples to generate

        Returns:
            Path to generated training data file
        """
        import tempfile
        import os

        try:
            # Create temporary file
            temp_file = tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False)
            temp_file.close()

            # Generate sample data
            result = self.training_pipeline.create_sample_training_data(temp_file.name, num_examples)

            if result['success']:
                logger.info(f"Generated {num_examples} sample training examples in {temp_file.name}")
                return temp_file.name
            else:
                # Clean up on failure
                os.unlink(temp_file.name)
                raise Exception(result.get('error', 'Unknown error generating sample data'))

        except Exception as e:
            logger.error(f"Failed to generate sample training data: {e}")
            raise

    def get_training_status(self) -> Dict[str, Any]:
        """Get current training status and metrics."""
        return {
            'last_training_time': self.last_training_time,
            'training_examples_count': self.training_examples_count,
            'training_pipeline_ready': True
        }

    def run_quick_test_pipeline(self) -> QuickTestPipelineResponse:
        """
        Run the quick test pipeline to validate the training system.

        This method:
        1. Generates 50 sample training examples
        2. Runs the complete training pipeline with this data
        3. Returns comprehensive results including metrics and status
        4. Cleans up temporary files automatically

        Returns:
            QuickTestPipelineResponse with test results and pipeline metrics
        """
        start_time = time.time()

        try:
            logger.info("Starting quick test pipeline execution")

            # Run the quick test pipeline
            results = self.training_pipeline.quick_test_pipeline()

            execution_time = time.time() - start_time

            if results['success']:
                logger.info(f"Quick test pipeline completed successfully in {execution_time:.2f} seconds")

                # Sanitize pipeline results to ensure JSON serialization
                sanitized_results = self._sanitize_float_values(results.get('pipeline_results', {}))

                return QuickTestPipelineResponse(
                    success=True,
                    test_completed=results.get('test_completed', True),
                    pipeline_results=sanitized_results,
                    error_message="",
                    execution_time_seconds=execution_time
                )
            else:
                error_msg = results.get('error', 'Unknown error during pipeline execution')
                logger.error(f"Quick test pipeline failed: {error_msg}")

                return QuickTestPipelineResponse(
                    success=False,
                    test_completed=False,
                    pipeline_results={},
                    error_message=error_msg,
                    execution_time_seconds=execution_time
                )

        except Exception as e:
            execution_time = time.time() - start_time
            error_msg = f"Quick test pipeline execution failed: {str(e)}"
            logger.error(error_msg)

            return QuickTestPipelineResponse(
                success=False,
                test_completed=False,
                pipeline_results={},
                error_message=error_msg,
                execution_time_seconds=execution_time
            )

    def _sanitize_float_values(self, data: Any) -> Any:
        """
        Recursively sanitize float values to ensure JSON serialization compatibility.
        Converts NaN, infinity, and -infinity to None or appropriate string representations.

        Args:
            data: Data structure to sanitize

        Returns:
            Sanitized data structure
        """
        import math

        if isinstance(data, dict):
            return {key: self._sanitize_float_values(value) for key, value in data.items()}
        elif isinstance(data, list):
            return [self._sanitize_float_values(item) for item in data]
        elif isinstance(data, float):
            if math.isnan(data):
                return None  # or could use "NaN" string
            elif math.isinf(data):
                return "Infinity" if data > 0 else "-Infinity"
            else:
                return data
        else:
            return data