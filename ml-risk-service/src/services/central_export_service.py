"""
Central Export Data Service for ML training data collection.
Orchestrates streaming data from multiple Central export APIs.
"""

import logging
import threading
import time
from collections import defaultdict
from typing import Dict, Any, List, Optional, Iterator, Tuple
from datetime import datetime, timezone, timedelta
from concurrent.futures import ThreadPoolExecutor, as_completed

from src.clients.central_export_client import CentralExportClient, ExportFilters
from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logger = logging.getLogger(__name__)


class CentralExportService:
    """Service for collecting training data from Central's export APIs."""

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

        # Processing configuration
        self.max_workers = self.config.get('max_workers', 3)
        self.correlation_timeout = self.config.get('correlation_timeout_seconds', 60)
        self.batch_size = self.config.get('batch_size', 100)

        # Simplified data tracking (only for alerts and policies if needed)
        self._alert_cache = defaultdict(list)
        self._policy_cache = {}
        self._cache_lock = threading.RLock()

        # Risk score tracking for logging
        self._risk_score_stats = {
            'scores': [],
            'total_count': 0,
            'failed_count': 0
        }

        logger.info("Initialized Central Export Service")

    def collect_training_data(self,
                            filters: Optional[Dict[str, Any]] = None,
                            limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Collect training data from Central using export APIs.

        Args:
            filters: Export filters to apply
            limit: Maximum number of training samples to collect

        Yields:
            Training examples as dictionaries
        """
        logger.info(f"Starting training data collection with filters: {filters}")

        try:
            # Process workloads directly - much simpler than correlation approach
            training_samples = self._stream_workload_data(filters, limit)

            examples_yielded = 0
            for example in training_samples:
                yield example
                examples_yielded += 1

                if limit and examples_yielded >= limit:
                    logger.info(f"Reached limit of {limit} training samples")
                    break

                # Log progress periodically
                if examples_yielded % self.batch_size == 0:
                    logger.info(f"Yielded {examples_yielded} training samples")

            logger.info(f"Training data collection completed: {examples_yielded} examples")
            self._log_final_risk_score_summary()

        except Exception as e:
            logger.error(f"Failed to collect training data: {e}")
            raise

    def _stream_workload_data(self,
                            filters: Optional[Dict[str, Any]] = None,
                            limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream and process workload data directly from Central.

        Args:
            filters: Export filters to apply
            limit: Maximum number of training samples

        Yields:
            Training examples from workload data
        """
        # Build filters for workloads
        workload_filters = self._build_workload_filters(filters)
        alert_filters = self._build_alert_filters(filters)
        policy_filters = self._build_policy_filters(filters)

        logger.info(f"Starting workload data streaming (limit: {limit})")

        # Optional: collect alerts and policies in parallel if needed
        alert_future = None
        policy_future = None

        if self._need_additional_data(filters):
            with ThreadPoolExecutor(max_workers=2) as executor:
                alert_future = executor.submit(self._collect_alerts, alert_filters)
                policy_future = executor.submit(self._collect_policies, policy_filters)

        # Stream workloads directly - no correlation needed
        examples_yielded = 0
        workloads_received = 0
        try:
            for workload in self.client.stream_workloads(workload_filters):
                workloads_received += 1

                # Direct processing - no correlation needed
                training_sample = self._create_training_sample_from_workload(workload)

                if training_sample:
                    yield training_sample
                    examples_yielded += 1

                    if limit and examples_yielded >= limit:
                        logger.info(f"Reached limit: {limit} training samples")
                        break

                    if examples_yielded % self.batch_size == 0:
                        logger.info(f"Processed {examples_yielded} workloads")

            # Wait for alerts/policies if needed
            if alert_future:
                try:
                    alert_result = alert_future.result(timeout=30)
                    policy_result = policy_future.result(timeout=30)
                    logger.info(f"Additional data collected: {alert_result}, {policy_result}")
                except Exception as e:
                    logger.warning(f"Failed to collect additional data: {e}")

            # Log summary of what was received
            if workloads_received == 0:
                logger.warning(f"No workloads received from Central API. "
                             f"Filters may be too restrictive or Central has no matching data. "
                             f"Workload filters used: {workload_filters}")
            else:
                logger.info(f"Received {workloads_received} workloads from Central API, "
                          f"created {examples_yielded} valid training samples")

        except Exception as e:
            logger.error(f"Error during workload streaming: {e}")
            if alert_future:
                alert_future.cancel()
                policy_future.cancel()
            raise

        logger.info(f"Workload streaming completed: {examples_yielded} examples from {workloads_received} workloads")

    def _create_training_sample_from_workload(self, workload: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create training sample directly from workload data.
        No correlation needed - all data is already linked.

        Args:
            workload: Workload record containing deployment, images, and vulnerabilities

        Returns:
            Training example dictionary or None if invalid
        """
        try:
            # Handle nested 'result' structure from Central API
            if 'result' in workload:
                result_data = workload['result']
                deployment_data = result_data.get('deployment', {})
                images_data = result_data.get('images', [])
                vulnerabilities = result_data.get('vulnerabilities', [])
            else:
                deployment_data = workload.get('deployment', {})
                images_data = workload.get('images', [])
                vulnerabilities = workload.get('vulnerabilities', [])

            # Extract deployment metadata with field name fallbacks
            deployment_id = deployment_data.get('id') or deployment_data.get('deploymentId', '')
            deployment_name = (deployment_data.get('name') or
                             deployment_data.get('deploymentName') or
                             deployment_data.get('metadata', {}).get('name', ''))
            namespace = (deployment_data.get('namespace') or
                        deployment_data.get('namespaceName') or
                        deployment_data.get('metadata', {}).get('namespace', ''))
            cluster_id = (deployment_data.get('clusterId') or
                         deployment_data.get('cluster_id') or
                         deployment_data.get('clusterName', ''))

            # Get any cached alerts for this deployment
            alerts_data = []
            if deployment_id:
                with self._cache_lock:
                    alerts_data = self._alert_cache.get(deployment_id, [])

            # Check for vulnerabilities in images if not found at workload level
            if not vulnerabilities and images_data:
                for image in images_data:
                    if isinstance(image, dict) and 'vulnerabilities' in image:
                        vulnerabilities.extend(image.get('vulnerabilities', []))

            # Direct feature extraction using baseline extractor
            training_sample = self.feature_extractor.create_training_sample(
                deployment_data=deployment_data,
                image_data_list=images_data,
                alert_data=alerts_data,
                baseline_violations=[]  # Could extract from workload if available
            )

            # Add workload-specific metadata
            training_sample['workload_metadata'] = {
                'deployment_id': deployment_id,
                'deployment_name': deployment_name,
                'namespace': namespace,
                'cluster_id': cluster_id,
                'total_vulnerabilities': len(vulnerabilities),
                'workload_cvss': workload.get('workload_cvss', 0.0),
                'image_count': len(images_data),
                'alert_count': len(alerts_data),
                'collected_at': datetime.now(timezone.utc).isoformat()
            }

            # Log risk score for this deployment
            self._log_deployment_risk_score(training_sample)

            return training_sample

        except Exception as e:
            logger.error(f"Failed to create training example from workload: {e}")
            # Track failed sample creation
            self._risk_score_stats['failed_count'] += 1
            return None

    def _need_additional_data(self, filters: Optional[Dict[str, Any]]) -> bool:
        """Check if we need to collect alerts and policies."""
        if not filters:
            return True  # Default to collecting additional data
        return filters.get('include_alerts', True) or filters.get('include_policies', False)

    def _collect_alerts(self, filters: Dict[str, Any]) -> Dict[str, Any]:
        """Collect alert/violation data and cache for correlation."""
        logger.info("Starting alert collection")
        count = 0

        try:
            for alert in self.client.stream_alerts(filters):
                deployment_id = alert.get('deployment_id') or alert.get('resource', {}).get('deployment_id')
                if deployment_id:
                    with self._cache_lock:
                        self._alert_cache[deployment_id].append(alert)
                    count += 1

                    if count % 100 == 0:
                        logger.debug(f"Cached {count} alerts")

            return {'type': 'alerts', 'count': count}

        except Exception as e:
            logger.error(f"Error collecting alerts: {e}")
            return {'type': 'alerts', 'count': count, 'error': str(e)}

    def _collect_policies(self, filters: Dict[str, Any]) -> Dict[str, Any]:
        """Collect policy data for understanding violations."""
        logger.info("Starting policy collection")
        count = 0

        try:
            for policy in self.client.stream_policies(filters):
                policy_id = policy.get('id')
                if policy_id:
                    with self._cache_lock:
                        self._policy_cache[policy_id] = policy
                    count += 1

            return {'type': 'policies', 'count': count}

        except Exception as e:
            logger.error(f"Error collecting policies: {e}")
            return {'type': 'policies', 'count': count, 'error': str(e)}

    def _build_workload_filters(self, base_filters: Optional[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Build filters specific to workload export endpoint.

        Note: No date-based filtering - collects all deployments.
        Use cluster/namespace filters to focus on specific environments.
        """
        filters = {'format': 'json'}

        if base_filters:
            # Cluster/namespace filters (primary filtering mechanism)
            if 'clusters' in base_filters:
                filters.update(ExportFilters.by_clusters(base_filters['clusters']))
            if 'namespaces' in base_filters:
                filters.update(ExportFilters.by_namespaces(base_filters['namespaces']))

            # Workload-specific filters
            if 'severity_threshold' in base_filters:
                filters['min_cvss'] = self._severity_to_cvss(base_filters['severity_threshold'])
            if 'include_inactive' in base_filters and not base_filters['include_inactive']:
                filters['active'] = 'true'
            if 'vulnerability_states' in base_filters:
                filters['vuln_state'] = ','.join(base_filters['vulnerability_states'])
            if 'include_vulnerabilities' in base_filters:
                filters['include_vulns'] = str(base_filters['include_vulnerabilities']).lower()

        return filters

    def _severity_to_cvss(self, severity: str) -> float:
        """Convert severity string to CVSS score threshold."""
        severity_map = {
            'CRITICAL_SEVERITY': 9.0,
            'HIGH_SEVERITY': 7.0,
            'MEDIUM_SEVERITY': 4.0,
            'LOW_SEVERITY': 0.1
        }
        return severity_map.get(severity, 0.0)

    def _build_alert_filters(self, base_filters: Optional[Dict[str, Any]]) -> Dict[str, Any]:
        """Build filters specific to alert export."""
        filters = {'format': 'json'}

        if base_filters:
            if 'clusters' in base_filters:
                filters.update(ExportFilters.by_clusters(base_filters['clusters']))
            if 'severity_threshold' in base_filters:
                filters.update(ExportFilters.by_severity(base_filters['severity_threshold']))
            if 'start_date' in base_filters:
                end_date = base_filters.get('end_date')
                filters.update(ExportFilters.by_date_range(base_filters['start_date'], end_date))

        return filters

    def _build_policy_filters(self, base_filters: Optional[Dict[str, Any]]) -> Dict[str, Any]:
        """Build filters specific to policy export."""
        filters = {'format': 'json'}

        # Policies don't typically need filtering for training data
        # We want all active policies to understand violation context
        if base_filters and 'policy_categories' in base_filters:
            filters['category'] = ','.join(base_filters['policy_categories'])

        return filters

    def get_collection_stats(self) -> Dict[str, Any]:
        """Get statistics about the current data collection."""
        with self._cache_lock:
            return {
                'alerts_cached': sum(len(alerts) for alerts in self._alert_cache.values()),
                'policies_cached': len(self._policy_cache),
                'cache_timestamp': datetime.now(timezone.utc).isoformat(),
                'processing_mode': 'workloads_direct'  # No correlation needed
            }

    def clear_cache(self):
        """Clear all cached data."""
        with self._cache_lock:
            self._alert_cache.clear()
            self._policy_cache.clear()
        logger.info("Cleared cached data")

    def _log_deployment_risk_score(self, training_sample: Dict[str, Any]):
        """Track risk score statistics."""
        try:
            risk_score = training_sample.get('risk_score', 0.0)
            self._risk_score_stats['scores'].append(risk_score)
            self._risk_score_stats['total_count'] += 1

            # Log batch summary periodically
            if self._risk_score_stats['total_count'] % self.batch_size == 0:
                scores = self._risk_score_stats['scores']
                recent = scores[-self.batch_size:]
                avg = sum(recent) / len(recent)
                logger.info(f"Processed {self._risk_score_stats['total_count']} samples, avg_risk={avg:.2f}")
        except Exception as e:
            logger.warning(f"Failed to track risk score: {e}")

    def _log_final_risk_score_summary(self):
        """Log final summary statistics for all collected risk scores."""
        try:
            scores = self._risk_score_stats['scores']
            total_count = self._risk_score_stats['total_count']
            failed_count = self._risk_score_stats['failed_count']

            if scores:
                avg_risk = sum(scores) / len(scores)
                logger.info(f"Training collection complete: total={total_count} successful={len(scores)} "
                           f"failed={failed_count} avg_risk={avg_risk:.3f}")
            else:
                logger.info(f"Training collection complete: total={total_count} successful=0 failed={failed_count}")

            # Reset statistics for next collection
            self._risk_score_stats = {
                'scores': [],
                'total_count': 0,
                'failed_count': 0
            }

        except Exception as e:
            logger.warning(f"Failed to log final risk score summary: {e}")

    def validate_predictions(self,
                           model,
                           prediction_client: CentralExportClient,
                           filters: Optional[Dict[str, Any]] = None,
                           limit: Optional[int] = None) -> Dict[str, Any]:
        """
        Validate model predictions against actual risk scores from a prediction Central instance.

        This method:
        1. Pulls deployments from a prediction Central (different from training Central)
        2. Runs predictions on those deployments using the provided model
        3. Compares predicted risk scores with actual Central risk scores
        4. Returns validation metrics

        Args:
            model: Trained RiskRankingModel instance to validate
            prediction_client: CentralExportClient for prediction Central instance
            filters: Optional filters for data collection
            limit: Maximum number of deployments to validate

        Returns:
            Dictionary with validation results:
            {
                'total_samples': int,
                'successful_predictions': int,
                'failed_predictions': int,
                'mae': float,  # Mean Absolute Error
                'rmse': float,  # Root Mean Squared Error
                'correlation': float,  # Correlation coefficient
                'ndcg': float,  # Normalized Discounted Cumulative Gain (ranking quality)
                'within_30_percent': float,  # Percentage within ±30%
                'predictions': List[Dict]  # Individual prediction results with top_features
            }

            Each prediction includes:
            {
                'deployment_name': str,
                'namespace': str,
                'cluster_id': str,
                'actual_score': float,
                'predicted_score': float,
                'absolute_error': float,
                'percent_error': float,
                'top_features': [{'name': str, 'importance': float}, ...]  # Top 5 features
            }
        """
        import numpy as np

        logger.info("Starting prediction validation")
        logger.info(f"Validation filters: {filters}, limit: {limit}")

        # Create temporary service for prediction Central
        prediction_service = CentralExportService(
            client=prediction_client,
            config=self.config
        )

        validation_results = {
            'total_samples': 0,
            'successful_predictions': 0,
            'failed_predictions': 0,
            'predictions': []
        }

        actual_scores = []
        predicted_scores = []

        try:
            # Collect samples from prediction Central
            for i, sample in enumerate(prediction_service.collect_training_data(filters, limit)):
                validation_results['total_samples'] += 1

                try:
                    # Extract actual normalized features and risk score
                    # Use 'features' dict instead of 'baseline_factors' for better variance
                    features = sample.get('features', {})
                    actual_score = sample.get('risk_score', 0.0)

                    # Validate features exist
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

                    # Make prediction with explanations enabled for debugging
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

                    # Store prediction result with top features
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

                    # Log progress
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

                # Mean Absolute Error
                mae = float(np.mean(np.abs(predicted_array - actual_array)))

                # Root Mean Squared Error
                rmse = float(np.sqrt(np.mean((predicted_array - actual_array) ** 2)))

                # Correlation
                if np.std(actual_array) > 0 and np.std(predicted_array) > 0:
                    correlation = float(np.corrcoef(actual_array, predicted_array)[0, 1])
                else:
                    correlation = 0.0

                # NDCG (Normalized Discounted Cumulative Gain) - ranking quality metric
                if len(np.unique(actual_array)) > 1:
                    ndcg = float(ndcg_score([actual_array], [predicted_array]))
                else:
                    ndcg = 0.0

                # Percentage within ±30%
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
                logger.info(f"  MAE: {mae:.4f}, RMSE: {rmse:.4f}")
                logger.info(f"  Correlation: {correlation:.4f}, NDCG: {ndcg:.4f}")
                logger.info(f"  Within ±30%: {within_range:.1f}%")
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

        finally:
            # Clean up prediction service
            prediction_service.close()

        return validation_results

    def close(self):
        """Clean up resources."""
        self.clear_cache()
        if hasattr(self.client, 'close'):
            self.client.close()
        logger.info("Closed Central Export Service")