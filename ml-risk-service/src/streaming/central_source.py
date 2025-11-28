"""
Central API sample stream source.
"""

import logging
import threading
from collections import defaultdict
from typing import Dict, Any, Iterator, Optional
from concurrent.futures import ThreadPoolExecutor

from src.streaming.sample_source import SampleStreamSource
from src.clients.central_export_client import CentralExportClient, ExportFilters

logger = logging.getLogger(__name__)


class CentralStreamSource(SampleStreamSource):
    """
    Stream deployment samples from Central API using export endpoints.

    This implementation extracts and consolidates the core streaming logic
    from CentralExportService._stream_workload_data().
    """

    def __init__(self,
                 client: CentralExportClient,
                 config: Optional[Dict[str, Any]] = None):
        """
        Initialize Central stream source.

        Args:
            client: Configured Central export client
            config: Optional configuration dictionary
        """
        self.client = client
        self.config = config or {}

        # Processing configuration
        self.max_workers = self.config.get('max_workers', 3)
        self.batch_size = self.config.get('batch_size', 100)

        # Optional caching for alerts and policies
        self._alert_cache = defaultdict(list)
        self._policy_cache = {}
        self._cache_lock = threading.RLock()

    def stream_samples(self,
                      filters: Optional[Dict[str, Any]] = None,
                      limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream deployment records from Central API.

        Args:
            filters: Export filters (clusters, namespaces, severity, etc.)
            limit: Maximum number of records to stream

        Yields:
            Raw deployment records in Central API format:
            {
                "result": {
                    "deployment": {...},
                    "images": [...],
                    "vulnerabilities": [...]
                },
                "workload_cvss": float
            }
        """
        # Build filters for workloads
        workload_filters = self._build_workload_filters(filters)
        alert_filters = self._build_alert_filters(filters)
        policy_filters = self._build_policy_filters(filters)

        logger.info(f"Starting Central API workload streaming (limit: {limit})")
        logger.debug(f"Workload filters: {workload_filters}")

        # Optional: collect alerts and policies in parallel if needed
        alert_future = None
        policy_future = None

        if self._need_additional_data(filters):
            with ThreadPoolExecutor(max_workers=2) as executor:
                alert_future = executor.submit(self._collect_alerts, alert_filters)
                policy_future = executor.submit(self._collect_policies, policy_filters)

        # Stream workloads directly
        workloads_received = 0
        try:
            for workload in self.client.stream_workloads(workload_filters):
                workloads_received += 1

                # Enhance workload with cached alerts if available
                if 'result' in workload and isinstance(workload['result'], dict):
                    deployment_data = workload['result'].get('deployment', {})
                    deployment_id = deployment_data.get('id')

                    if deployment_id:
                        with self._cache_lock:
                            cached_alerts = self._alert_cache.get(deployment_id, [])
                            if cached_alerts:
                                workload['result']['alerts'] = cached_alerts

                yield workload

                if limit and workloads_received >= limit:
                    logger.info(f"Reached limit: {limit} workloads")
                    break

                if workloads_received % self.batch_size == 0:
                    logger.info(f"Streamed {workloads_received} workloads from Central")

            # Wait for alerts/policies collection to complete
            if alert_future:
                try:
                    alert_result = alert_future.result(timeout=30)
                    policy_result = policy_future.result(timeout=30)
                    logger.info(f"Additional data collected: {alert_result}, {policy_result}")
                except Exception as e:
                    logger.warning(f"Failed to collect additional data: {e}")

            # Log summary
            if workloads_received == 0:
                logger.warning(f"No workloads received from Central API. "
                             f"Filters may be too restrictive or Central has no matching data. "
                             f"Workload filters used: {workload_filters}")
            else:
                logger.info(f"Central API streaming completed: {workloads_received} workloads")

        except Exception as e:
            logger.error(f"Error during Central API workload streaming: {e}")
            if alert_future:
                alert_future.cancel()
                policy_future.cancel()
            raise

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

    def close(self):
        """Clean up resources."""
        with self._cache_lock:
            self._alert_cache.clear()
            self._policy_cache.clear()

        if hasattr(self.client, 'close'):
            self.client.close()

        logger.debug("Closed Central stream source")
