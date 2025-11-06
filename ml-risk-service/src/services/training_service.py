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
    TrainModelResponse,
    TrainingMetrics,
    FeatureImportance
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

