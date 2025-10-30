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
        Much simpler than the previous correlation approach.

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

        logger.info("Starting workload data streaming")

        # Optional: collect alerts and policies in parallel if needed
        alert_future = None
        policy_future = None

        if self._need_additional_data(filters):
            with ThreadPoolExecutor(max_workers=2) as executor:
                alert_future = executor.submit(self._collect_alerts, alert_filters)
                policy_future = executor.submit(self._collect_policies, policy_filters)

        # Stream workloads directly - no correlation needed
        examples_yielded = 0
        try:
            for workload in self.client.stream_workloads(workload_filters):
                # Direct processing - no correlation needed
                training_sample = self._create_training_sample_from_workload(workload)

                if training_sample:
                    yield training_sample
                    examples_yielded += 1

                    if limit and examples_yielded >= limit:
                        logger.info(f"Reached workload processing limit: {limit}")
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

        except Exception as e:
            logger.error(f"Error during workload streaming: {e}")
            if alert_future:
                alert_future.cancel()
                policy_future.cancel()
            raise

        logger.info(f"Workload streaming completed: {examples_yielded} examples")

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
            # Add diagnostic logging for workload structure
            logger.debug(f"Raw workload structure keys: {list(workload.keys())}")

            # Handle nested 'result' structure from Central API
            if 'result' in workload:
                result_data = workload['result']
                logger.debug(f"Result structure keys: {list(result_data.keys()) if isinstance(result_data, dict) else 'not-dict'}")
                deployment_data = result_data.get('deployment', {})
                images_data = result_data.get('images', [])
                # Vulnerabilities might be nested in images or separate
                vulnerabilities = result_data.get('vulnerabilities', [])
            else:
                # Fallback to flat structure (legacy or different API version)
                logger.debug("Using flat workload structure (no 'result' key)")
                deployment_data = workload.get('deployment', {})
                images_data = workload.get('images', [])
                vulnerabilities = workload.get('vulnerabilities', [])

            # Log deployment data structure for debugging
            logger.debug(f"Deployment data keys: {list(deployment_data.keys()) if isinstance(deployment_data, dict) else 'not-dict'}")
            logger.debug(f"Images data type/count: {type(images_data)} / {len(images_data) if isinstance(images_data, list) else 'not-list'}")

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

            # Log extracted metadata values
            logger.debug(f"Extracted metadata: id='{deployment_id}', name='{deployment_name}', "
                        f"namespace='{namespace}', cluster='{cluster_id}'")

            # Get any cached alerts for this deployment
            alerts_data = []
            if deployment_id:
                with self._cache_lock:
                    alerts_data = self._alert_cache.get(deployment_id, [])

            # Check for vulnerabilities in images if not found at workload level
            if not vulnerabilities and images_data:
                # Collect vulnerabilities from images
                for image in images_data:
                    if isinstance(image, dict) and 'vulnerabilities' in image:
                        vulnerabilities.extend(image.get('vulnerabilities', []))

            logger.debug(f"Total vulnerabilities found: {len(vulnerabilities)}")

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
        """Build filters specific to workload export endpoint."""
        filters = {'format': 'json'}

        if base_filters:
            # Convert days_back to start_date if provided
            if 'days_back' in base_filters:
                from datetime import datetime, timezone, timedelta
                start_date = datetime.now(timezone.utc) - timedelta(days=base_filters['days_back'])
                filters.update(ExportFilters.by_date_range(start_date, None))
                logger.debug(f"Converted days_back={base_filters['days_back']} to start_date={start_date}")
            # Use explicit start_date if provided (takes precedence over days_back)
            elif 'start_date' in base_filters:
                end_date = base_filters.get('end_date')
                filters.update(ExportFilters.by_date_range(base_filters['start_date'], end_date))

            # Common filters
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
        """
        Log risk score information for a deployment.

        Args:
            training_sample: Training sample containing risk score and metadata
        """
        try:
            risk_score = training_sample.get('risk_score', 0.0)
            baseline_factors = training_sample.get('baseline_factors', {})
            baseline_score = baseline_factors.get('baseline_score', 0.0)
            workload_metadata = training_sample.get('workload_metadata', {})

            # Extract deployment info
            deployment_name = workload_metadata.get('deployment_name', 'unknown')
            namespace = workload_metadata.get('namespace', 'unknown')
            cluster_id = workload_metadata.get('cluster_id', 'unknown')

            # Additional diagnostic data for uniform score investigation
            vuln_count = workload_metadata.get('total_vulnerabilities', 0)
            image_count = workload_metadata.get('image_count', 0)
            alert_count = workload_metadata.get('alert_count', 0)

            # Track score statistics
            self._risk_score_stats['scores'].append(risk_score)
            self._risk_score_stats['total_count'] += 1

            # Enhanced logging with diagnostic information
            logger.debug(f"Training sample: deployment={deployment_name} namespace={namespace} "
                        f"cluster={cluster_id} risk_score={risk_score:.3f} baseline_score={baseline_score:.3f} "
                        f"vulns={vuln_count} images={image_count} alerts={alert_count}")

            # Log baseline factors breakdown for uniform score investigation
            if baseline_factors:
                logger.debug(f"Baseline factors for {deployment_name}: "
                            f"policy_violations={baseline_factors.get('policy_violations', 1.0):.3f} "
                            f"vulnerabilities={baseline_factors.get('vulnerabilities', 1.0):.3f} "
                            f"service_config={baseline_factors.get('service_config', 1.0):.3f} "
                            f"reachability={baseline_factors.get('reachability', 1.0):.3f}")

            # Log batch summary periodically at INFO level
            if self._risk_score_stats['total_count'] % self.batch_size == 0:
                self._log_batch_risk_score_summary()

        except Exception as e:
            logger.warning(f"Failed to log risk score for deployment: {e}")

    def _log_batch_risk_score_summary(self):
        """Log summary statistics for the current batch of risk scores."""
        try:
            scores = self._risk_score_stats['scores']
            if not scores:
                return

            recent_scores = scores[-self.batch_size:] if len(scores) >= self.batch_size else scores

            avg_risk = sum(recent_scores) / len(recent_scores)
            min_risk = min(recent_scores)
            max_risk = max(recent_scores)
            count = len(recent_scores)

            logger.info(f"Training batch processed: count={count} avg_risk={avg_risk:.3f} "
                       f"min_risk={min_risk:.3f} max_risk={max_risk:.3f}")

        except Exception as e:
            logger.warning(f"Failed to log batch risk score summary: {e}")

    def _log_final_risk_score_summary(self):
        """Log final summary statistics for all collected risk scores."""
        try:
            scores = self._risk_score_stats['scores']
            total_count = self._risk_score_stats['total_count']
            failed_count = self._risk_score_stats['failed_count']

            if not scores:
                logger.info(f"Training collection complete: total_processed={total_count} "
                           f"successful=0 failed={failed_count} - no valid risk scores generated")
                return

            # Calculate comprehensive statistics
            avg_risk = sum(scores) / len(scores)
            min_risk = min(scores)
            max_risk = max(scores)

            # Calculate variance
            variance = sum((score - avg_risk) ** 2 for score in scores) / len(scores)

            # Count unique scores for diversity assessment
            unique_scores = len(set(f"{score:.3f}" for score in scores))

            logger.info(f"Training collection complete: total_processed={total_count} successful={len(scores)} "
                       f"failed={failed_count} avg_risk={avg_risk:.3f} min_risk={min_risk:.3f} "
                       f"max_risk={max_risk:.3f} risk_variance={variance:.6f} unique_scores={unique_scores}")

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
                'within_30_percent': float,  # Percentage within ±30%
                'predictions': List[Dict]  # Individual prediction results
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
                    # Extract features and actual risk score
                    baseline_factors = sample.get('baseline_factors', {})
                    actual_score = sample.get('risk_score', 0.0)

                    # Get feature names from model
                    feature_names = model.feature_names if hasattr(model, 'feature_names') else []

                    # Build feature vector matching model's expected features
                    features = {
                        'policy_violations': baseline_factors.get('policy_violations', 1.0),
                        'process_baseline': baseline_factors.get('process_baseline', 1.0),
                        'vulnerabilities': baseline_factors.get('vulnerabilities', 1.0),
                        'risky_components': baseline_factors.get('risky_components', 1.0),
                        'component_count': baseline_factors.get('component_count', 1.0),
                        'image_age': baseline_factors.get('image_age', 1.0),
                        'service_config': baseline_factors.get('service_config', 1.0),
                        'reachability': baseline_factors.get('reachability', 1.0),
                    }

                    # Convert to numpy array in correct order
                    if feature_names:
                        X = np.array([[features.get(name, 1.0) for name in feature_names]])
                    else:
                        X = np.array([[features[k] for k in sorted(features.keys())]])

                    # Make prediction
                    predictions = model.predict(X, explain=False)
                    predicted_score = predictions[0].risk_score

                    # Track scores
                    actual_scores.append(actual_score)
                    predicted_scores.append(predicted_score)

                    # Store prediction result
                    workload_metadata = sample.get('workload_metadata', {})
                    validation_results['predictions'].append({
                        'deployment_name': workload_metadata.get('deployment_name', 'unknown'),
                        'namespace': workload_metadata.get('namespace', 'unknown'),
                        'cluster_id': workload_metadata.get('cluster_id', 'unknown'),
                        'actual_score': float(actual_score),
                        'predicted_score': float(predicted_score),
                        'absolute_error': float(abs(predicted_score - actual_score)),
                        'percent_error': float(abs(predicted_score - actual_score) / (actual_score + 1e-10) * 100)
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

                # Percentage within ±30%
                within_range = float(np.mean(
                    np.abs(predicted_array - actual_array) / (actual_array + 1e-10) <= 0.3
                ) * 100)

                validation_results.update({
                    'mae': mae,
                    'rmse': rmse,
                    'correlation': correlation,
                    'within_30_percent': within_range,
                    'mean_actual_score': float(np.mean(actual_array)),
                    'mean_predicted_score': float(np.mean(predicted_array)),
                    'score_variance_actual': float(np.var(actual_array)),
                    'score_variance_predicted': float(np.var(predicted_array))
                })

                logger.info(f"Validation complete: {validation_results['successful_predictions']} samples")
                logger.info(f"  MAE: {mae:.4f}, RMSE: {rmse:.4f}")
                logger.info(f"  Correlation: {correlation:.4f}")
                logger.info(f"  Within ±30%: {within_range:.1f}%")
            else:
                logger.warning("No valid predictions to calculate metrics")
                validation_results.update({
                    'mae': 0.0,
                    'rmse': 0.0,
                    'correlation': 0.0,
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