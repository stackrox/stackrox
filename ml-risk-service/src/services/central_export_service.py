"""
Central Export Data Service for ML training data collection.
Provides helper methods and prediction validation using the new streaming architecture.
"""

import logging
from typing import Dict, Any, Optional

from src.clients.central_export_client import CentralExportClient
from src.streaming import CentralStreamSource, SampleStream
from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logger = logging.getLogger(__name__)


class CentralExportService:
    """
    Service for prediction validation using Central API.

    Note: Training data collection now uses SampleStream + CentralStreamSource directly.
    This class is kept mainly for validate_predictions() functionality.
    """

    def __init__(self, client: CentralExportClient, config: Optional[Dict[str, Any]] = None):
        """
        Initialize Central Export Service.

        Args:
            client: Configured Central export client
            config: Optional configuration dictionary
        """
        self.client = client
        self.config = config or {}
        self.feature_extractor = BaselineFeatureExtractor()

        logger.info("Initialized Central Export Service")


    def validate_predictions(self,
                           model,
                           prediction_client: CentralExportClient,
                           filters: Optional[Dict[str, Any]] = None,
                           limit: Optional[int] = None) -> Dict[str, Any]:
        """
        Validate model predictions against actual risk scores from a prediction Central instance.

        This method uses the new streaming architecture (SampleStream + CentralStreamSource)
        to pull deployments from a prediction Central instance.

        Args:
            model: Trained RiskRankingModel instance to validate
            prediction_client: CentralExportClient for prediction Central instance
            filters: Optional filters for data collection
            limit: Maximum number of deployments to validate

        Returns:
            Dictionary with validation results and metrics
        """
        import numpy as np

        logger.info("Starting prediction validation")
        logger.info(f"Validation filters: {filters}, limit: {limit}")

        # Create stream source and sample stream for prediction Central
        prediction_source = CentralStreamSource(prediction_client, self.config)
        sample_stream = SampleStream(prediction_source, self.feature_extractor, self.config)

        validation_results = {
            'total_samples': 0,
            'successful_predictions': 0,
            'failed_predictions': 0,
            'predictions': []
        }

        actual_scores = []
        predicted_scores = []

        try:
            # Stream samples from prediction Central using new architecture
            for i, sample in enumerate(sample_stream.stream(filters, limit)):
                validation_results['total_samples'] += 1

                try:
                    # Extract features and actual risk score
                    features = sample.get('features', {})
                    actual_score = sample.get('risk_score', 0.0)

                    if not features:
                        logger.warning(f"Sample {i} missing 'features' dictionary, skipping")
                        validation_results['failed_predictions'] += 1
                        continue

                    # Get feature names from model
                    feature_names = model.feature_names if hasattr(model, 'feature_names') else []

                    # Convert to numpy array in correct order
                    if feature_names:
                        X = np.array([[features.get(name, 0.0) for name in feature_names]])
                    else:
                        X = np.array([[features[k] for k in sorted(features.keys())]])

                    # Make prediction with explanations
                    predictions = model.predict(X, explain=True)
                    predicted_score = predictions[0].risk_score
                    feature_importance = predictions[0].feature_importance

                    # Extract top 5 features by importance
                    workload_metadata = sample.get('workload_metadata', {})
                    sorted_features = sorted(
                        feature_importance.items(),
                        key=lambda x: abs(x[1]),
                        reverse=True
                    )[:5]

                    top_features = [
                        {'name': name, 'importance': float(importance)}
                        for name, importance in sorted_features
                    ]

                    # Track scores
                    actual_scores.append(actual_score)
                    predicted_scores.append(predicted_score)

                    # Store prediction result
                    validation_results['predictions'].append({
                        'deployment_name': workload_metadata.get('deployment_name', 'unknown'),
                        'namespace': workload_metadata.get('namespace', 'unknown'),
                        'cluster_id': workload_metadata.get('cluster_id', 'unknown'),
                        'actual_score': float(actual_score),
                        'predicted_score': float(predicted_score),
                        'absolute_error': float(abs(predicted_score - actual_score)),
                        'percent_error': float(abs(predicted_score - actual_score) / (actual_score + 1e-10) * 100),
                        'top_features': top_features
                    })

                    validation_results['successful_predictions'] += 1

                    if validation_results['successful_predictions'] % 10 == 0:
                        logger.info(f"Validated {validation_results['successful_predictions']} predictions")

                except Exception as e:
                    logger.warning(f"Failed to validate prediction for sample {i}: {e}")
                    validation_results['failed_predictions'] += 1
                    continue

            # Calculate validation metrics
            if actual_scores and predicted_scores:
                from sklearn.metrics import ndcg_score

                actual_array = np.array(actual_scores)
                predicted_array = np.array(predicted_scores)

                mae = float(np.mean(np.abs(predicted_array - actual_array)))
                rmse = float(np.sqrt(np.mean((predicted_array - actual_array) ** 2)))

                if np.std(actual_array) > 0 and np.std(predicted_array) > 0:
                    correlation = float(np.corrcoef(actual_array, predicted_array)[0, 1])
                else:
                    correlation = 0.0

                if len(np.unique(actual_array)) > 1:
                    ndcg = float(ndcg_score([actual_array], [predicted_array]))
                else:
                    ndcg = 0.0

                within_range = float(np.mean(
                    np.abs(predicted_array - actual_array) / (actual_array + 1e-10) <= 0.3
                ) * 100)

                validation_results.update({
                    'mae': mae,
                    'rmse': rmse,
                    'correlation': correlation,
                    'ndcg': ndcg,
                    'within_30_percent': within_range,
                    'mean_actual_score': float(np.mean(actual_array)),
                    'mean_predicted_score': float(np.mean(predicted_array)),
                    'score_variance_actual': float(np.var(actual_array)),
                    'score_variance_predicted': float(np.var(predicted_array))
                })

                logger.info(f"Validation complete: {validation_results['successful_predictions']} samples")
                logger.info(f"  MAE: {mae:.4f}, RMSE: {rmse:.4f}, Correlation: {correlation:.4f}, NDCG: {ndcg:.4f}")
            else:
                logger.warning("No valid predictions to calculate metrics")
                validation_results.update({
                    'mae': 0.0,
                    'rmse': 0.0,
                    'correlation': 0.0,
                    'ndcg': 0.0,
                    'within_30_percent': 0.0
                })

        except Exception as e:
            logger.error(f"Validation failed: {e}")
            validation_results['error'] = str(e)
            raise

        return validation_results

    def close(self):
        """Clean up resources."""
        if hasattr(self.client, 'close'):
            self.client.close()
        logger.info("Closed Central Export Service")
