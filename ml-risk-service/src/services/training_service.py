"""
Training service for ML model training and management.
This service is shared between gRPC and REST APIs.
"""

import logging
import threading
import time
from typing import Dict, Any, List, Optional

from src.training.train_pipeline import TrainingPipeline
from src.storage.model_storage import ModelStorageManager, StorageConfig
from src.api.schemas import (
    TrainModelRequest,
    TrainModelResponse,
    TrainingMetrics,
    FeatureImportance,
    TrainingSample
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
        self.training_samples_count = 0

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
                logger.info(f"Starting model training with {len(request.training_data)} samples")

                # Convert training data to internal format
                training_samples = []
                for example in request.training_data:
                    # Convert Pydantic models to dictionary format expected by training pipeline
                    features = self._extract_features_from_training_sample(example)
                    training_samples.append({
                        'features': features,
                        'risk_score': example.current_risk_score,
                        'deployment_id': example.deployment_id
                    })

                # Create ranking dataset
                X, y, groups = self.training_pipeline.data_loader.create_ranking_dataset(training_samples)
                feature_names = sorted(training_samples[0]['features'].keys())

                # Train model
                training_metrics = risk_service.model.train(X, y, groups, feature_names)

                # Update service state
                risk_service.model_loaded = True
                self.last_training_time = int(time.time())
                self.training_samples_count = len(training_samples)

                # Save model to storage
                self._save_trained_model(risk_service, training_samples, 'api', 'train')

                # Convert metrics to response format with feature importances
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

                logger.info(f"Model training completed. Validation NDCG: {training_metrics.val_ndcg:.4f}")

                return self._create_training_response(
                    success=True,
                    training_metrics=training_metrics,
                    risk_service=risk_service,
                    feature_importances=feature_importances
                )

        except Exception as e:
            logger.error(f"Model training failed: {e}")
            return self._create_training_response(
                success=False,
                training_metrics=None,
                risk_service=risk_service,
                error_message=str(e)
            )

    def _extract_features_from_training_sample(self, example: TrainingSample) -> Dict[str, float]:
        """
        Extract features from training sample.

        Args:
            example: Training sample with deployment and image features

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

    def _prepare_training_data(self, training_samples: List[Dict[str, Any]]):
        """
        Convert training samples to numpy arrays for model training.

        Args:
            training_samples: List of samples with 'features' and 'risk_score' fields

        Returns:
            Tuple of (X, y, feature_names) where:
                X: numpy array of feature vectors
                y: numpy array of risk scores
                feature_names: sorted list of feature names
        """
        import numpy as np

        feature_names = sorted(training_samples[0]['features'].keys())
        X = []
        y = []

        for sample in training_samples:
            feature_vector = [sample['features'][name] for name in feature_names]
            X.append(feature_vector)
            y.append(sample['risk_score'])

        return np.array(X), np.array(y), feature_names

    def _save_trained_model(self, risk_service, training_samples: List[Dict[str, Any]],
                           source_type: str, method_name: str, additional_tags: Optional[Dict[str, str]] = None):
        """
        Save trained model to storage with metadata.

        Args:
            risk_service: Risk service instance containing the trained model
            training_samples: List of training samples used
            source_type: Source type for tags (e.g., 'file', 'central-api')
            method_name: Training method name for tags (e.g., 'train', 'train-full')
            additional_tags: Optional additional tags to include in metadata
        """
        if not risk_service:
            return

        try:
            model_id = risk_service.get_default_model_id()
            description = f"Model trained from {source_type} with {len(training_samples)} samples"
            tags = {'source': source_type, 'method': method_name}

            if additional_tags:
                tags.update(additional_tags)

            if risk_service.model.save_model_to_storage(
                risk_service.storage_manager,
                model_id,
                description,
                tags
            ):
                logger.info(f"Model saved to storage with ID: {model_id}")
            else:
                logger.warning("Model trained successfully but failed to save to storage")
        except Exception as e:
            logger.warning(f"Failed to save model to storage: {e}. Model is trained but not persisted.")

    def _create_training_response(self, success: bool, training_metrics, risk_service,
                                  error_message: str = "",
                                  feature_importances: Optional[List[FeatureImportance]] = None) -> TrainModelResponse:
        """
        Create standardized training response.

        Args:
            success: Whether training succeeded
            training_metrics: Training metrics object (or None if failed)
            risk_service: Risk service instance
            error_message: Error message if training failed
            feature_importances: Optional list of feature importances (for detailed responses)

        Returns:
            TrainModelResponse with standardized format
        """
        if success and training_metrics:
            response_metrics = TrainingMetrics(
                validation_ndcg=training_metrics.val_ndcg,
                global_feature_importance=feature_importances or []
            )

            return TrainModelResponse(
                success=True,
                error_message="",
                model_version=risk_service.model.model_version if risk_service else "unknown",
                metrics=response_metrics
            )
        else:
            return TrainModelResponse(
                success=False,
                error_message=error_message,
                model_version="",
                metrics=TrainingMetrics(
                    validation_ndcg=0.0,
                    global_feature_importance=[]
                )
            )

    def generate_sample_training_data(self, num_samples: int = 100) -> str:
        """
        Generate sample training data for testing.

        Args:
            num_samples: Number of samples to generate

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
            result = self.training_pipeline.create_sample_training_data(temp_file.name, num_samples)

            if result['success']:
                logger.info(f"Generated {num_samples} sample training samples in {temp_file.name}")
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
            'training_samples_count': self.training_samples_count,
            'training_pipeline_ready': True
        }



    def train_model_from_central(self,
                                filters: Optional[Dict[str, Any]] = None,
                                limit: Optional[int] = None,
                                config_override: Optional[str] = None,
                                risk_service=None) -> TrainModelResponse:
        """
        Train model using data streamed from Central API.

        Uses the new streaming architecture (CentralStreamSource + SampleStream).

        Args:
            filters: Filters for Central API data collection
            limit: Maximum number of training samples to collect
            config_override: Optional JSON configuration overrides
            risk_service: Risk service instance to update with trained model

        Returns:
            TrainModelResponse with training results
        """
        try:
            from src.config.central_config import create_central_client_from_config
            from src.streaming import CentralStreamSource, SampleStream

            logger.info(f"Starting model training from Central API with filters: {filters}")

            # Create Central client and stream source
            client = create_central_client_from_config()
            source = CentralStreamSource(client, self.config)

            # Create sample stream
            sample_stream = SampleStream(source, config=self.config)

            # Stream and collect training samples
            training_samples = []
            for sample in sample_stream.stream(filters, limit):
                training_samples.append(sample)

            if not training_samples:
                return self._create_training_response(
                    success=False,
                    training_metrics=None,
                    risk_service=risk_service,
                    error_message="No training samples found with current filters"
                )

            logger.info(f"Collected {len(training_samples)} samples from Central API")

            # Prepare training data
            X, y, feature_names = self._prepare_training_data(training_samples)

            # Train the model
            logger.info(f"Training model with {len(training_samples)} samples from Central API")
            training_metrics = risk_service.model.train(X, y, feature_names=feature_names) if risk_service else None

            if not training_metrics:
                return self._create_training_response(
                    success=False,
                    training_metrics=None,
                    risk_service=risk_service,
                    error_message="Failed to train model"
                )

            # Update risk service state and save model
            if risk_service:
                risk_service.model_loaded = True
                self._save_trained_model(risk_service, training_samples, 'central-api', 'train-full')

            logger.info(f"Model training completed successfully. Validation NDCG: {training_metrics.val_ndcg:.4f}")

            return self._create_training_response(
                success=True,
                training_metrics=training_metrics,
                risk_service=risk_service
            )

        except Exception as e:
            logger.error(f"Central API training failed: {e}")
            return self._create_training_response(
                success=False,
                training_metrics=None,
                risk_service=risk_service,
                error_message=f"Central API training failed: {str(e)}"
            )

    def train_model_from_file(self,
                             file_path: str,
                             limit: Optional[int] = None,
                             risk_service=None) -> TrainModelResponse:
        """
        Train model using data from a JSON file.

        Uses the new streaming architecture (JSONFileStreamSource + SampleStream).

        Args:
            file_path: Path to JSON training data file
            limit: Maximum number of training samples to use
            risk_service: Risk service instance to update with trained model

        Returns:
            TrainModelResponse with training results
        """
        try:
            from src.streaming import JSONFileStreamSource, SampleStream

            logger.info(f"Starting model training from file: {file_path}")

            # Create file source and sample stream
            source = JSONFileStreamSource(file_path)
            sample_stream = SampleStream(source, config=self.config)

            # Stream and collect training samples
            training_samples = []
            for sample in sample_stream.stream(filters=None, limit=limit):
                training_samples.append(sample)

            if not training_samples:
                return self._create_training_response(
                    success=False,
                    training_metrics=None,
                    risk_service=risk_service,
                    error_message=f"No training samples found in file: {file_path}"
                )

            logger.info(f"Collected {len(training_samples)} samples from file")

            # Prepare training data
            X, y, feature_names = self._prepare_training_data(training_samples)

            # Train the model
            training_metrics = risk_service.model.train(X, y, feature_names=feature_names) if risk_service else None

            if not training_metrics:
                return self._create_training_response(
                    success=False,
                    training_metrics=None,
                    risk_service=risk_service,
                    error_message="Failed to train model"
                )

            # Update risk service state and save model
            if risk_service:
                risk_service.model_loaded = True
                self._save_trained_model(risk_service, training_samples, 'file', 'train-file', {'file': file_path})

            logger.info(f"Model training from file completed successfully. Validation NDCG: {training_metrics.val_ndcg:.4f}")

            return self._create_training_response(
                success=True,
                training_metrics=training_metrics,
                risk_service=risk_service
            )

        except Exception as e:
            logger.error(f"File-based training failed: {e}")
            return self._create_training_response(
                success=False,
                training_metrics=None,
                risk_service=risk_service,
                error_message=f"File-based training failed: {str(e)}"
            )

