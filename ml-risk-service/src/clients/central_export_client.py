"""
StackRox Central Export API Client for streaming bulk data.
Uses Central's /v1/export/* endpoints for efficient data collection.
"""

import json
import logging
import time
from typing import Dict, Any, List, Optional, Iterator, Union
from datetime import datetime, timezone
from urllib.parse import urljoin, urlencode
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
import urllib3

logger = logging.getLogger(__name__)


class CentralExportClient:
    """Client for StackRox Central's streaming export APIs."""

    def __init__(self, endpoint: str, auth_token: str, config: Optional[Dict[str, Any]] = None):
        """
        Initialize Central Export Client.

        Args:
            endpoint: Central API endpoint (e.g., "https://central.stackrox.io")
            auth_token: API authentication token
            config: Optional configuration dictionary
        """
        self.endpoint = endpoint.rstrip('/')
        self.auth_token = auth_token
        self.config = config or {}

        # Client configuration (must be set before session creation)
        self.chunk_size = self.config.get('chunk_size', 1000)
        self.timeout = self.config.get('timeout_seconds', 300)
        self.max_retries = self.config.get('max_retries', 3)

        # Initialize session with retry strategy
        self.session = self._create_session()

        logger.info(f"Initialized Central Export Client for {self.endpoint}")

    def _create_session(self) -> requests.Session:
        """Create HTTP session with retry strategy and authentication."""
        session = requests.Session()

        # Configure retry strategy
        retry_strategy = Retry(
            total=self.max_retries,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["HEAD", "GET", "OPTIONS"],
            backoff_factor=2,
            raise_on_status=False
        )

        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("http://", adapter)
        session.mount("https://", adapter)

        # Set authentication headers
        session.headers.update({
            'Authorization': f'Bearer {self.auth_token}',
            'Accept': 'application/json',
            'User-Agent': 'StackRox-ML-Risk-Service/1.0'
        })

        # Configure SSL verification
        verify_certs = self.config.get('verify_certificates', True)
        ca_bundle_path = self.config.get('ca_bundle_path', '')

        if not verify_certs:
            session.verify = False
            # Disable SSL warnings when verification is disabled
            urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
            logger.warning("SSL certificate verification is DISABLED - this should only be used in development/testing")
        elif ca_bundle_path:
            session.verify = ca_bundle_path
            logger.info(f"Using custom CA bundle: {ca_bundle_path}")
        # else: use default verification (session.verify = True by default)

        return session

    def stream_workloads(self, filters: Optional[Dict[str, Any]] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream workload data (deployments + images + vulnerabilities) from Central's export API.

        This endpoint provides deployments with their associated images and vulnerability data
        in a single stream, eliminating the need for separate deployment/image correlation.

        Args:
            filters: Optional filters for the export query

        Yields:
            Individual workload records containing deployment, images, and vulnerability data
        """
        endpoint = f"{self.endpoint}/v1/export/vuln-mgmt/workloads"

        # Default filters for ML training data
        default_filters = {
            'format': 'json'
        }

        if filters:
            default_filters.update(filters)

        logger.info(f"Starting workloads export stream with filters: {default_filters}")

        try:
            yield from self._stream_export_data(endpoint, default_filters, "workloads")
        except Exception as e:
            logger.error(f"Failed to stream workloads: {e}")
            raise

    def stream_alerts(self, filters: Optional[Dict[str, Any]] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream alert/violation data from Central's export API.

        Args:
            filters: Optional filters for the export query

        Yields:
            Individual alert records as dictionaries
        """
        endpoint = f"{self.endpoint}/v1/export/alerts"

        default_filters = {
            'format': 'json'
        }

        if filters:
            default_filters.update(filters)

        logger.info(f"Starting alert export stream with filters: {default_filters}")

        try:
            yield from self._stream_export_data(endpoint, default_filters, "alerts")
        except Exception as e:
            logger.error(f"Failed to stream alerts: {e}")
            raise

    def stream_policies(self, filters: Optional[Dict[str, Any]] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream policy data from Central's export API.

        Args:
            filters: Optional filters for the export query

        Yields:
            Individual policy records as dictionaries
        """
        endpoint = f"{self.endpoint}/v1/export/policies"

        default_filters = {
            'format': 'json'
        }

        if filters:
            default_filters.update(filters)

        logger.info(f"Starting policy export stream with filters: {default_filters}")

        try:
            yield from self._stream_export_data(endpoint, default_filters, "policies")
        except Exception as e:
            logger.error(f"Failed to stream policies: {e}")
            raise

    def _stream_export_data(self, url: str, filters: Dict[str, Any], data_type: str) -> Iterator[Dict[str, Any]]:
        """
        Generic method to stream data from export endpoints.

        Args:
            url: Export endpoint URL
            filters: Query parameters for filtering
            data_type: Type of data being exported (for logging)

        Yields:
            Individual records from the export stream
        """
        # Build query string
        if filters:
            query_string = urlencode(filters, doseq=True)
            full_url = f"{url}?{query_string}"
        else:
            full_url = url

        logger.debug(f"Requesting export from: {full_url}")

        try:
            # Make streaming request
            response = self.session.get(
                full_url,
                stream=True,
                timeout=self.timeout
            )

            response.raise_for_status()

            # Track progress
            records_processed = 0
            start_time = time.time()

            # Process streaming response line by line
            for line in response.iter_lines(decode_unicode=True):
                if not line.strip():
                    continue

                try:
                    # Parse JSON line
                    record = json.loads(line)

                    # Validate record has minimum required fields
                    if self._validate_record(record, data_type):
                        yield record
                        records_processed += 1

                        # Log progress periodically
                        if records_processed % self.chunk_size == 0:
                            elapsed = time.time() - start_time
                            rate = records_processed / elapsed if elapsed > 0 else 0
                            logger.info(f"Processed {records_processed} {data_type} records "
                                      f"({rate:.1f} records/sec)")
                    else:
                        # Extract ID for logging based on data type
                        record_id = 'unknown'
                        if data_type == 'workloads' and 'result' in record and isinstance(record['result'], dict):
                            deployment = record['result'].get('deployment', {})
                            if isinstance(deployment, dict):
                                record_id = deployment.get('id', 'unknown')
                        else:
                            record_id = record.get('id', 'unknown')
                        logger.warning(f"Skipping invalid {data_type} record: {record_id}")

                except json.JSONDecodeError as e:
                    logger.warning(f"Failed to parse JSON line in {data_type} stream: {e}")
                    continue
                except Exception as e:
                    logger.error(f"Error processing {data_type} record: {e}")
                    continue

            # Final progress log
            elapsed = time.time() - start_time
            rate = records_processed / elapsed if elapsed > 0 else 0
            logger.info(f"Completed {data_type} export: {records_processed} records "
                       f"in {elapsed:.1f}s ({rate:.1f} records/sec)")

        except requests.exceptions.RequestException as e:
            logger.error(f"HTTP error during {data_type} export: {e}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error during {data_type} export: {e}")
            raise

    def _validate_record(self, record: Dict[str, Any], data_type: str) -> bool:
        """
        Validate that a record has minimum required fields.

        Args:
            record: Record to validate
            data_type: Type of record (workloads, alerts, policies)

        Returns:
            True if record is valid, False otherwise
        """
        if not isinstance(record, dict):
            return False

        # Type-specific validation
        if data_type == 'workloads':
            # Workloads from Central API have structure: {"result": {"deployment": {...}, "images": [...], "livePods": [...]}}
            if 'result' not in record:
                return False

            result = record['result']
            if not isinstance(result, dict):
                return False

            # Check required fields in result
            return ('deployment' in result and
                    isinstance(result['deployment'], dict) and
                    'id' in result['deployment'] and
                    'images' in result)

        elif data_type == 'alerts':
            # For alerts, check for either direct fields or nested in result
            if 'result' in record:
                result = record['result']
                return isinstance(result, dict) and ('policy' in result or 'violation' in result)
            return 'policy' in record or 'violation' in record

        elif data_type == 'policies':
            # For policies, check for either direct fields or nested in result
            if 'result' in record:
                result = record['result']
                return isinstance(result, dict) and 'name' in result
            return 'name' in record

        return True

    def test_connection(self) -> Dict[str, Any]:
        """
        Test connection to Central API.

        Returns:
            Connection test results
        """
        try:
            # Try a simple API call to verify connectivity
            response = self.session.get(
                f"{self.endpoint}/v1/metadata",
                timeout=10
            )

            if response.status_code == 200:
                return {
                    'success': True,
                    'message': 'Successfully connected to Central API',
                    'central_version': response.json().get('version', 'unknown')
                }
            else:
                return {
                    'success': False,
                    'message': f'HTTP {response.status_code}: {response.text}',
                    'status_code': response.status_code
                }

        except requests.exceptions.RequestException as e:
            return {
                'success': False,
                'message': f'Connection failed: {str(e)}',
                'error_type': type(e).__name__
            }
        except Exception as e:
            return {
                'success': False,
                'message': f'Unexpected error: {str(e)}',
                'error_type': type(e).__name__
            }

    def get_export_capabilities(self) -> Dict[str, Any]:
        """
        Query Central for available export endpoints and capabilities.

        Returns:
            Dictionary describing available export capabilities
        """
        capabilities = {
            'endpoints': {
                'workloads': f"{self.endpoint}/v1/export/vuln-mgmt/workloads",
                'alerts': f"{self.endpoint}/v1/export/alerts",
                'policies': f"{self.endpoint}/v1/export/policies"
            },
            'primary_endpoint': 'workloads',
            'supported_formats': ['json'],
            'streaming_supported': True,
            'max_timeout': self.timeout,
            'chunk_size': self.chunk_size
        }

        # Test workloads endpoint availability
        try:
            response = self.session.head(capabilities['endpoints']['workloads'], timeout=5)
            capabilities['workloads_available'] = response.status_code in [200, 404]  # 404 is OK for HEAD
        except:
            capabilities['workloads_available'] = False

        # Test other endpoints
        for name in ['alerts', 'policies']:
            try:
                response = self.session.head(capabilities['endpoints'][name], timeout=5)
                capabilities[f'{name}_available'] = response.status_code in [200, 404]
            except:
                capabilities[f'{name}_available'] = False

        return capabilities

    def close(self):
        """Close the HTTP session."""
        if hasattr(self, 'session'):
            self.session.close()
            logger.debug("Closed Central Export Client session")

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()


class ExportFilters:
    """Helper class for building export API filters."""

    @staticmethod
    def by_date_range(start_date: datetime, end_date: Optional[datetime] = None) -> Dict[str, str]:
        """
        Create date range filter.

        Args:
            start_date: Start date for filtering
            end_date: End date for filtering (default: now)

        Returns:
            Filter dictionary for date range
        """
        if end_date is None:
            end_date = datetime.now(timezone.utc)

        return {
            'created_after': start_date.isoformat(),
            'created_before': end_date.isoformat()
        }

    @staticmethod
    def by_clusters(cluster_ids: List[str]) -> Dict[str, str]:
        """
        Create cluster filter.

        Args:
            cluster_ids: List of cluster IDs to filter by

        Returns:
            Filter dictionary for clusters
        """
        return {
            'cluster': ','.join(cluster_ids)
        }

    @staticmethod
    def by_namespaces(namespaces: List[str]) -> Dict[str, str]:
        """
        Create namespace filter.

        Args:
            namespaces: List of namespaces to filter by

        Returns:
            Filter dictionary for namespaces
        """
        return {
            'namespace': ','.join(namespaces)
        }

    @staticmethod
    def by_severity(min_severity: str = "MEDIUM_SEVERITY") -> Dict[str, str]:
        """
        Create severity filter.

        Args:
            min_severity: Minimum severity level

        Returns:
            Filter dictionary for severity
        """
        return {
            'severity': min_severity
        }

    @staticmethod
    def combine_filters(*filter_dicts: Dict[str, str]) -> Dict[str, str]:
        """
        Combine multiple filter dictionaries.

        Args:
            filter_dicts: Multiple filter dictionaries to combine

        Returns:
            Combined filter dictionary
        """
        combined = {}
        for filter_dict in filter_dicts:
            combined.update(filter_dict)
        return combined