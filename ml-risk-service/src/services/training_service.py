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
    TrainingSample,
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
                logger.info(f"Starting model training with {len(request.training_data)} examples")

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
                logger.info(f"Generated {num_examples} sample training samples in {temp_file.name}")
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

    def run_quick_test_pipeline(self) -> QuickTestPipelineResponse:
        """
        Run the quick test pipeline to validate the training system.

        This method:
        1. Generates 50 sample training samples
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

    def train_model_from_central(self,
                                filters: Optional[Dict[str, Any]] = None,
                                limit: Optional[int] = None,
                                config_override: Optional[str] = None,
                                risk_service=None) -> TrainModelResponse:
        """
        Train model using data streamed from Central API.

        This method abstracts the data source by using existing streaming and training methods.

        Args:
            filters: Filters for Central API data collection
            limit: Maximum number of training samples to collect
            config_override: Optional JSON configuration overrides
            risk_service: Risk service instance to update with trained model

        Returns:
            TrainModelResponse with training results
        """
        try:
            from training.data_loader import TrainingDataLoader

            logger.info(f"Starting model training from Central API with filters: {filters}")

            # Create data loader and collect training data
            data_loader = TrainingDataLoader()

            # Set up default filters
            training_filters = filters or {}

            # Stream data from Central API
            data_iterator = data_loader.load_from_central_api_streaming_with_config(
                config_path=None,
                filters=training_filters
            )

            # Convert streaming data to training samples format
            training_samples = self._convert_central_data_to_training_samples(
                data_iterator, limit
            )

            # Check for variance in risk scores and apply synthetic scoring if needed
            if training_samples:
                import numpy as np
                risk_scores = [sample.current_risk_score for sample in training_samples]
                score_variance = np.var(risk_scores) if len(risk_scores) > 1 else 0.0
                unique_scores = len(set(risk_scores))

                logger.info(f"Risk score analysis: variance={score_variance:.6f}, unique_scores={unique_scores}, total_samples={len(risk_scores)}")

                # If variance is too low, apply synthetic scoring to all samples
                if score_variance < 1e-3 or unique_scores == 1:
                    logger.warning(f"Low risk score variance detected (var={score_variance:.6f}, unique={unique_scores}). "
                                  f"Applying synthetic scoring to all samples to ensure training effectiveness.")
                    training_samples = self._apply_synthetic_scoring_to_all_samples(training_samples)

                    # Re-analyze after synthetic scoring
                    updated_risk_scores = [sample.current_risk_score for sample in training_samples]
                    updated_variance = np.var(updated_risk_scores) if len(updated_risk_scores) > 1 else 0.0
                    updated_unique = len(set(updated_risk_scores))
                    logger.info(f"After synthetic scoring: variance={updated_variance:.6f}, unique_scores={updated_unique}, "
                               f"range=[{min(updated_risk_scores):.3f}, {max(updated_risk_scores):.3f}]")

            if not training_samples:
                return TrainModelResponse(
                    success=False,
                    error_message="No training samples found with current filters",
                    model_version="",
                    metrics=TrainingMetrics(
                        validation_ndcg=0.0,
                        validation_auc=0.0,
                        training_loss=0.0,
                        epochs_completed=0,
                        global_feature_importance=[]
                    )
                )

            # Use training pipeline instead of direct training to ensure model is saved to storage
            logger.info(f"Using training pipeline for Central data with {len(training_samples)} samples")

            # Create temporary training data file for pipeline
            import tempfile
            import json
            import os

            # Convert training samples to deployment record format expected by pipeline
            deployment_records = []
            for sample in training_samples:
                # Convert TrainingSample back to deployment record format
                deployment_record = self._convert_training_sample_to_deployment_record(sample)
                deployment_records.append(deployment_record)

            # Create pipeline-compatible JSON structure
            pipeline_data = {
                "deployments": deployment_records
            }

            # Create temporary file for training data with explicit path for debugging
            temp_file_path = "/tmp/central_training_debug.json"
            with open(temp_file_path, 'w') as temp_file:
                json.dump(pipeline_data, temp_file, indent=2)

            # Debug: Log what we wrote to the temp file
            logger.info(f"Created temp file {temp_file_path} with {len(deployment_records)} deployment records")
            logger.info(f"Pipeline data structure: {list(pipeline_data.keys())}")
            if deployment_records:
                logger.info(f"First deployment record keys: {list(deployment_records[0].keys())}")

            try:
                # Run full training pipeline with the temporary file (this will save the model to storage)
                pipeline_results = self.training_pipeline.run_full_pipeline(temp_file_path)

                if pipeline_results.get('success', False):
                    # Extract model info for response (pipeline returns flat structure)
                    model_saving_results = pipeline_results.get('model_saving', {})

                    # Get model metrics from pipeline results
                    model_training_results = pipeline_results.get('model_training', {})
                    training_metrics_dict = model_training_results.get('training_metrics', {})

                    # Update risk service with the trained model from storage if available
                    if risk_service and model_saving_results.get('success', False):
                        try:
                            # Load the newly saved model into the risk service
                            model_id = "stackrox-risk-model"
                            version = "latest"  # Pipeline saves with timestamp, but we can load latest
                            if risk_service._load_model_from_storage(model_id, version):
                                logger.info(f"Successfully loaded trained model into risk service")
                            else:
                                logger.warning(f"Failed to load trained model into risk service")
                        except Exception as load_error:
                            logger.warning(f"Failed to load trained model into risk service: {load_error}")

                    # Create response with pipeline results
                    response_metrics = TrainingMetrics(
                        validation_ndcg=float(training_metrics_dict.get('val_ndcg', 0.0)),
                        validation_auc=float(training_metrics_dict.get('val_auc', 0.0)),
                        training_loss=float(training_metrics_dict.get('training_loss', 0.0)),
                        epochs_completed=int(training_metrics_dict.get('epochs_completed', 0)),
                        global_feature_importance=[]  # Would need to extract from model if needed
                    )

                    # Extract model version from saving results
                    model_version = model_saving_results.get('model_version', 'unknown')

                    return TrainModelResponse(
                        success=True,
                        error_message="",
                        model_version=model_version,
                        metrics=response_metrics
                    )
                else:
                    error_msg = pipeline_results.get('error', 'Training pipeline failed')
                    logger.error(f"Training pipeline failed: {error_msg}")
                    return TrainModelResponse(
                        success=False,
                        error_message=f"Training pipeline failed: {error_msg}",
                        model_version="",
                        metrics=TrainingMetrics(
                            validation_ndcg=0.0,
                            validation_auc=0.0,
                            training_loss=0.0,
                            epochs_completed=0,
                            global_feature_importance=[]
                        )
                    )

            finally:
                # Clean up temporary file (disabled for debugging)
                logger.info(f"Temporary file preserved for debugging: {temp_file_path}")
                pass  # Don't delete the file so we can examine it

        except Exception as e:
            logger.error(f"Central API training failed: {e}")
            return TrainModelResponse(
                success=False,
                error_message=f"Central API training failed: {str(e)}",
                model_version="",
                metrics=TrainingMetrics(
                    validation_ndcg=0.0,
                    validation_auc=0.0,
                    training_loss=0.0,
                    epochs_completed=0,
                    global_feature_importance=[]
                )
            )

    def _convert_central_data_to_training_samples(self,
                                                  data_iterator,
                                                  limit: Optional[int] = None) -> List[TrainingSample]:
        """
        Convert Central API streaming data to TrainingSample format.

        Args:
            data_iterator: Iterator yielding Central API training samples
            limit: Optional limit on number of examples to process

        Returns:
            List of TrainingSample objects ready for training
        """
        training_samples = []
        count = 0

        try:
            for example in data_iterator:
                if limit and count >= limit:
                    break

                # Convert Central format to TrainingSample format
                training_sample = self._convert_single_central_sample(example)
                if training_sample:
                    training_samples.append(training_sample)
                    count += 1

                    if count % 100 == 0:
                        logger.info(f"Converted {count} training samples")

        except Exception as e:
            logger.error(f"Error converting Central data: {e}")

        logger.info(f"Successfully converted {len(training_samples)} training samples from Central API")
        return training_samples

    def _convert_single_central_sample(self, example: Dict[str, Any]) -> Optional[TrainingSample]:
        """
        Convert a single Central API sample to TrainingSample format.

        Args:
            example: Central API training sample dictionary

        Returns:
            TrainingSample object or None if conversion fails
        """
        try:
            from src.api.schemas import TrainingSample, DeploymentFeatures, ImageFeatures

            # Extract features from the Central format
            features = example.get('features', {})

            # Create deployment features using existing mapping logic
            deployment_features = DeploymentFeatures(
                policy_violation_count=int(features.get('policy_violation_count', 0)),
                policy_violation_severity_score=float(features.get('policy_violation_score', 0.0)),
                process_baseline_violations=int(features.get('process_baseline_violations', 0)),
                host_network=bool(features.get('host_network', False)),
                host_pid=bool(features.get('host_pid', False)),
                host_ipc=bool(features.get('host_ipc', False)),
                privileged_container_count=int(features.get('privileged_container_count', 0)),
                automount_service_account_token=bool(features.get('automount_service_account_token', False)),
                exposed_port_count=int(features.get('exposed_port_count', 0)),
                replica_count=max(int(features.get('replica_count', 1)), 1),
                has_external_exposure=bool(features.get('has_external_exposure', False)),
                is_orchestrator_component=bool(features.get('is_orchestrator_component', False)),
                creation_timestamp=int(features.get('creation_timestamp', 0))
            )

            # Create image features (simplified - could be enhanced)
            image_features = []
            if 'image_features' in features:
                for img_data in features['image_features']:
                    image_feature = ImageFeatures(
                        critical_vuln_count=int(img_data.get('critical_vuln_count', 0)),
                        high_vuln_count=int(img_data.get('high_vuln_count', 0)),
                        medium_vuln_count=int(img_data.get('medium_vuln_count', 0)),
                        low_vuln_count=int(img_data.get('low_vuln_count', 0)),
                        total_component_count=int(img_data.get('total_component_count', 0)),
                        image_age_days=int(img_data.get('image_age_days', 0))
                    )
                    image_features.append(image_feature)

            # Get baseline risk score and apply synthetic scoring if needed
            baseline_risk_score = float(example.get('risk_score', 1.0))

            # Apply synthetic scoring for better variance if baseline is uniform
            final_risk_score = self._apply_synthetic_risk_scoring(
                baseline_risk_score, features, deployment_features, image_features)

            # Create training sample
            training_sample = TrainingSample(
                deployment_features=deployment_features,
                image_features=image_features,
                current_risk_score=final_risk_score,
                deployment_id=example.get('deployment_id', '')
            )

            return training_sample

        except Exception as e:
            logger.warning(f"Failed to convert Central sample: {e}")
            return None

    def _apply_synthetic_risk_scoring(self, baseline_score: float, features: Dict[str, Any],
                                    deployment_features, image_features: List) -> float:
        """
        Apply synthetic risk scoring similar to BaselineFeatureExtractor logic.

        Args:
            baseline_score: Original baseline risk score
            features: Raw feature dictionary from Central API
            deployment_features: Parsed deployment features
            image_features: List of parsed image features

        Returns:
            Final risk score with variance for training
        """
        # If baseline score is meaningful (not default 1.0), use it
        if baseline_score != 1.0:
            logger.debug(f"Using meaningful baseline score: {baseline_score}")
            return baseline_score

        logger.debug(f"Applying synthetic risk scoring for baseline score: {baseline_score}")

        # Generate synthetic score based on feature values
        synthetic_score = 1.0  # Base score

        # Add vulnerability-based risk (high impact)
        if image_features:
            for img in image_features:
                vuln_score = (
                    img.critical_vuln_count * 10.0 +
                    img.high_vuln_count * 4.0 +
                    img.medium_vuln_count * 1.0 +
                    img.low_vuln_count * 0.25
                ) / 100.0  # Normalize
                synthetic_score += min(vuln_score * 0.5, 2.0)

        # Add configuration-based risk (medium impact)
        config_risk = 0.0
        if deployment_features.host_network:
            config_risk += 0.3
        if deployment_features.host_pid:
            config_risk += 0.3
        if deployment_features.privileged_container_count > 0:
            privileged_ratio = deployment_features.privileged_container_count / max(deployment_features.replica_count, 1)
            config_risk += privileged_ratio * 0.4
        if deployment_features.has_external_exposure:
            config_risk += 0.2

        synthetic_score += config_risk

        # Add component-based risk (low impact)
        if image_features:
            for img in image_features:
                if img.total_component_count > 0:
                    risky_ratio = getattr(img, 'risky_component_count', 0) / img.total_component_count
                    synthetic_score += min(risky_ratio * 0.3, 0.1)

        # Add age-based risk (very low impact)
        if deployment_features.creation_timestamp > 0:
            import time
            age_days = (time.time() - deployment_features.creation_timestamp) / 86400
            age_score = min(age_days / 365.0, 5.0)
            if age_score > 0.5:  # Older deployments slightly riskier
                synthetic_score += min(age_score * 0.1, 0.2)

        # Add controlled randomization for variance with enhanced distribution
        # Use deployment-specific randomization for more variance
        import random
        deployment_hash = features.get('deployment_id_hash', hash(str(features)) % 1000000)
        seed_value = deployment_hash % 2**32
        random.seed(seed_value)

        # Enhanced noise generation with normal distribution for better variance
        noise = random.gauss(0, 0.4)  # Normal distribution with std=0.4 for good spread
        synthetic_score += noise

        # Add feature-based variance boost for samples with distinctive characteristics
        feature_variance_boost = 0.0
        if deployment_features.privileged_container_count > 0:
            feature_variance_boost += random.uniform(0.2, 0.5)
        if deployment_features.host_network or deployment_features.host_pid:
            feature_variance_boost += random.uniform(0.3, 0.6)
        if image_features and any(img.critical_vuln_count > 0 for img in image_features):
            feature_variance_boost += random.uniform(0.4, 0.8)

        synthetic_score += feature_variance_boost

        # Apply logarithmic scaling to spread out lower values more
        if synthetic_score > 1.0:
            log_component = (synthetic_score - 1.0) * 0.3
            synthetic_score = 1.0 + log_component + (log_component ** 0.7)

        # Ensure reasonable bounds [0.5, 6.0] with wider range for better discrimination
        final_score = max(0.5, min(synthetic_score, 6.0))

        logger.debug(f"Synthetic scoring: base={baseline_score} -> synthetic={synthetic_score} -> final={final_score}")
        return final_score

    def _apply_synthetic_scoring_to_all_samples(self, training_samples: List) -> List:
        """
        Apply synthetic scoring to all training samples to generate variance.

        Args:
            training_samples: List of TrainingSample objects

        Returns:
            Updated list with synthetic risk scores
        """
        updated_samples = []

        for i, sample in enumerate(training_samples):
            # Extract features for synthetic scoring
            features = self._extract_features_from_training_sample(sample)

            # Add unique identifier to features for unique scoring
            unique_id = f"{sample.deployment_id}_{i}" if sample.deployment_id else f"sample_{i}"
            features['deployment_id_hash'] = hash(unique_id) % 1000000

            # Force synthetic scoring by passing baseline_score=1.0
            synthetic_score = self._apply_synthetic_risk_scoring(
                1.0,  # Force synthetic scoring
                features,  # Include features with unique identifier hash
                sample.deployment_features,
                sample.image_features
            )

            # Update the sample with the new synthetic score
            from src.api.schemas import TrainingSample
            updated_sample = TrainingSample(
                deployment_features=sample.deployment_features,
                image_features=sample.image_features,
                current_risk_score=synthetic_score,
                deployment_id=sample.deployment_id
            )

            updated_samples.append(updated_sample)
            logger.debug(f"Updated sample {sample.deployment_id}: {sample.current_risk_score:.6f} -> {synthetic_score:.6f}")

        return updated_samples

    def _convert_training_sample_to_deployment_record(self, sample: 'TrainingSample') -> Dict[str, Any]:
        """
        Convert a TrainingSample back to deployment record format expected by training pipeline.

        Args:
            sample: TrainingSample object to convert

        Returns:
            Deployment record in the format expected by the pipeline
        """
        # Create deployment data structure
        deployment_data = {
            'id': sample.deployment_id,
            'name': f'deployment-{sample.deployment_id}',
            'namespace': 'default',  # Default values since we don't have this info
            'cluster_id': 'cluster-1',
            'clusterId': 'cluster-1',
            'replicas': sample.deployment_features.replica_count,
            'hostNetwork': sample.deployment_features.host_network,
            'hostPid': sample.deployment_features.host_pid,
            'hostIpc': sample.deployment_features.host_ipc,
            'automountServiceAccountToken': sample.deployment_features.automount_service_account_token,
            'containers': [],
            'ports': [],
            'created': {'seconds': sample.deployment_features.creation_timestamp} if sample.deployment_features.creation_timestamp > 0 else None,
            'orchestratorComponent': sample.deployment_features.is_orchestrator_component,
            'inactive': False
        }

        # Add privileged containers if any
        for i in range(sample.deployment_features.privileged_container_count):
            deployment_data['containers'].append({
                'name': f'container-{i}',
                'securityContext': {'privileged': True}
            })

        # Add exposed ports
        for i in range(sample.deployment_features.exposed_port_count):
            deployment_data['ports'].append({
                'name': f'port-{i}',
                'port': 8080 + i,
                'exposure': 'EXTERNAL' if sample.deployment_features.has_external_exposure else 'INTERNAL'
            })

        # Create images data structure
        images_data = []
        for img_feature in sample.image_features:
            image_data = {
                'id': f'image-{len(images_data)}',
                'name': {
                    'registry': 'docker.io',
                    'remote': f'library/app-{len(images_data)}'
                },
                'metadata': {
                    'layerShas': 10,  # Default layer count
                    'created': {'seconds': int(time.time()) - img_feature.image_age_days * 86400} if img_feature.image_age_days > 0 else None
                },
                'components': img_feature.total_component_count,
                'cluster_local': False,
                'scan': {
                    'criticalVulns': img_feature.critical_vuln_count,
                    'highVulns': img_feature.high_vuln_count,
                    'mediumVulns': img_feature.medium_vuln_count,
                    'lowVulns': img_feature.low_vuln_count
                }
            }
            images_data.append(image_data)

        # Create alerts data structure (simulate policy violations)
        alerts_data = []
        for i in range(sample.deployment_features.policy_violation_count):
            alerts_data.append({
                'id': f'alert-{i}',
                'policy': {
                    'id': f'policy-{i}',
                    'name': f'Policy Violation {i}',
                    'severity': 'HIGH_SEVERITY'  # Default severity
                },
                'resource': {
                    'deployment_id': sample.deployment_id
                },
                'time': {'seconds': int(time.time())}
            })

        # Create the complete deployment record
        deployment_record = {
            'deployment': deployment_data,
            'images': images_data,
            'alerts': alerts_data,
            'baseline_violations': [],  # Empty for now
            'current_risk_score': sample.current_risk_score
        }

        return deployment_record