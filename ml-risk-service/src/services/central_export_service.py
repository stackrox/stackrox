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

        logger.info("Initialized Central Export Service")

    def collect_training_data(self,
                            filters: Optional[Dict[str, Any]] = None,
                            limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Collect training data from Central using export APIs.

        Args:
            filters: Export filters to apply
            limit: Maximum number of training examples to collect

        Yields:
            Training examples as dictionaries
        """
        logger.info(f"Starting training data collection with filters: {filters}")

        try:
            # Process workloads directly - much simpler than correlation approach
            training_examples = self._stream_workload_data(filters, limit)

            examples_yielded = 0
            for example in training_examples:
                yield example
                examples_yielded += 1

                if limit and examples_yielded >= limit:
                    logger.info(f"Reached limit of {limit} training examples")
                    break

                # Log progress periodically
                if examples_yielded % self.batch_size == 0:
                    logger.info(f"Yielded {examples_yielded} training examples")

            logger.info(f"Training data collection completed: {examples_yielded} examples")

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
            limit: Maximum number of training examples

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
                training_example = self._create_training_example_from_workload(workload)

                if training_example:
                    yield training_example
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

    def _create_training_example_from_workload(self, workload: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create training example directly from workload data.
        No correlation needed - all data is already linked.

        Args:
            workload: Workload record containing deployment, images, and vulnerabilities

        Returns:
            Training example dictionary or None if invalid
        """
        try:
            deployment_data = workload.get('deployment', {})
            images_data = workload.get('images', [])
            vulnerabilities = workload.get('vulnerabilities', [])

            # Get any cached alerts for this deployment
            deployment_id = deployment_data.get('id')
            alerts_data = []
            if deployment_id:
                with self._cache_lock:
                    alerts_data = self._alert_cache.get(deployment_id, [])

            # Direct feature extraction using baseline extractor
            training_example = self.feature_extractor.create_training_example(
                deployment_data=deployment_data,
                image_data_list=images_data,
                alert_data=alerts_data,
                baseline_violations=[]  # Could extract from workload if available
            )

            # Add workload-specific metadata
            training_example['workload_metadata'] = {
                'deployment_id': deployment_id,
                'deployment_name': deployment_data.get('name', ''),
                'namespace': deployment_data.get('namespace', ''),
                'cluster_id': deployment_data.get('cluster_id', ''),
                'total_vulnerabilities': len(vulnerabilities),
                'workload_cvss': workload.get('workload_cvss', 0.0),
                'image_count': len(images_data),
                'alert_count': len(alerts_data),
                'collected_at': datetime.now(timezone.utc).isoformat()
            }

            return training_example

        except Exception as e:
            logger.error(f"Failed to create training example from workload: {e}")
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
            # Common filters
            if 'clusters' in base_filters:
                filters.update(ExportFilters.by_clusters(base_filters['clusters']))
            if 'namespaces' in base_filters:
                filters.update(ExportFilters.by_namespaces(base_filters['namespaces']))
            if 'start_date' in base_filters:
                end_date = base_filters.get('end_date')
                filters.update(ExportFilters.by_date_range(base_filters['start_date'], end_date))

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

    def close(self):
        """Clean up resources."""
        self.clear_cache()
        if hasattr(self.client, 'close'):
            self.client.close()
        logger.info("Closed Central Export Service")