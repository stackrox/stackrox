"""
Configuration management for Central API integration.
Handles authentication, environment variables, and configuration validation.
"""

import os
import logging
import ssl
from typing import Dict, Any, List, Optional, Tuple
from pathlib import Path
import yaml

logger = logging.getLogger(__name__)


class CentralConfig:
    """Configuration manager for Central API integration."""

    def __init__(self, config_path: Optional[str] = None, use_prediction_env: bool = False):
        """
        Initialize Central configuration.

        Args:
            config_path: Optional path to configuration file
            use_prediction_env: If True, use PREDICTION_CENTRAL_* environment variables
                              instead of TRAINING_CENTRAL_* variables
        """
        self.use_prediction_env = use_prediction_env
        self.config_path = config_path or self._find_config_file()
        self.config = self._load_config()

    @classmethod
    def from_prediction_env(cls, config_path: Optional[str] = None) -> 'CentralConfig':
        """
        Create a CentralConfig instance for prediction Central.

        This will use PREDICTION_CENTRAL_* environment variables instead of
        TRAINING_CENTRAL_* variables.

        Args:
            config_path: Optional path to configuration file

        Returns:
            CentralConfig instance configured for prediction Central
        """
        return cls(config_path=config_path, use_prediction_env=True)

    def _find_config_file(self) -> str:
        """Find the feature configuration file."""
        possible_paths = [
            '/app/config/feature_config.yaml',  # Template config (production)
            os.path.join(os.path.dirname(__file__), 'feature_config.yaml'),  # Template config (dev)
            'src/config/feature_config.yaml'  # Relative path fallback
        ]

        for path in possible_paths:
            if os.path.exists(path):
                return path

        raise FileNotFoundError("Could not find feature_config.yaml file")

    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from YAML file."""
        try:
            with open(self.config_path, 'r') as f:
                config = yaml.safe_load(f)

            # Load appropriate section based on prediction flag
            if self.use_prediction_env:
                section = config.get('prediction_central_api', {})
                logger.debug("Loading prediction_central_api configuration section")
            else:
                section = config.get('training_central_api', {})
                logger.debug("Loading training_central_api configuration section")

            # Fallback to legacy 'central_api' section if new sections don't exist
            if not section:
                section = config.get('central_api', {})
                logger.warning("Using legacy 'central_api' section - consider migrating to "
                             "'training_central_api' and 'prediction_central_api' sections")

            return section
        except Exception as e:
            logger.warning(f"Failed to load config from {self.config_path}: {e}")
            return {}

    def is_enabled(self) -> bool:
        """Check if Central API integration is enabled."""
        return self.config.get('enabled', False)

    def get_endpoint(self) -> str:
        """Get Central API endpoint."""
        endpoint = self.config.get('endpoint', '')

        # Allow environment variable override
        env_var = 'PREDICTION_CENTRAL_ENDPOINT' if self.use_prediction_env else 'TRAINING_CENTRAL_ENDPOINT'
        env_endpoint = os.getenv(env_var)
        if env_endpoint:
            endpoint = env_endpoint

        if not endpoint:
            central_type = "Prediction" if self.use_prediction_env else "Training"
            raise ValueError(f"{central_type} Central API endpoint not configured")

        return endpoint.rstrip('/')

    def get_authentication_config(self) -> Dict[str, Any]:
        """
        Get authentication configuration with environment variable substitution.

        Returns:
            Dictionary with authentication settings
        """
        auth_config = self.config.get('authentication', {})
        method = auth_config.get('method', 'api_token')

        if method == 'api_token':
            return self._get_token_auth_config(auth_config)
        elif method == 'mtls':
            return self._get_mtls_auth_config(auth_config)
        else:
            raise ValueError(f"Unsupported authentication method: {method}")

    def _get_token_auth_config(self, auth_config: Dict[str, Any]) -> Dict[str, Any]:
        """Get API token authentication configuration."""
        token = auth_config.get('api_token', '')

        # Check if token is a placeholder for environment variable substitution
        if token and token.startswith('${') and token.endswith('}'):
            env_var = token[2:-1]
            token = os.getenv(env_var)
            # If the placeholder env var doesn't exist, fall through to default behavior

        # If we have a non-empty token that's not a placeholder, use it directly
        if token and not token.strip().startswith('${'):
            # Token value is already provided in configuration
            pass
        else:
            # Fallback to direct environment variable lookup if no token found
            env_var = 'PREDICTION_CENTRAL_API_TOKEN' if self.use_prediction_env else 'TRAINING_CENTRAL_API_TOKEN'
            token = os.getenv(env_var)

        if not token:
            central_type = "Prediction" if self.use_prediction_env else "Training"
            raise ValueError(f"{central_type} Central API token not configured or found in environment")

        return {
            'method': 'api_token',
            'token': token
        }

    def _get_mtls_auth_config(self, auth_config: Dict[str, Any]) -> Dict[str, Any]:
        """Get mTLS authentication configuration."""
        cert_path = self._resolve_path(auth_config.get('client_cert_path', ''))
        key_path = self._resolve_path(auth_config.get('client_key_path', ''))
        ca_path = self._resolve_path(auth_config.get('ca_cert_path', ''))

        # Validate certificate files exist
        for path, name in [(cert_path, 'client certificate'),
                          (key_path, 'client key'),
                          (ca_path, 'CA certificate')]:
            if not path or not os.path.exists(path):
                raise FileNotFoundError(f"{name} not found at: {path}")

        return {
            'method': 'mtls',
            'client_cert': cert_path,
            'client_key': key_path,
            'ca_cert': ca_path
        }

    def _resolve_path(self, path: str) -> str:
        """Resolve path with environment variable substitution."""
        if not path:
            return ''

        # Substitute environment variables
        if path.startswith('${') and path.endswith('}'):
            env_var = path[2:-1]
            resolved_path = os.getenv(env_var)
            if not resolved_path:
                logger.warning(f"Environment variable {env_var} not set")
                return ''
            return resolved_path

        return path

    def get_export_settings(self) -> Dict[str, Any]:
        """Get export API settings."""
        settings = self.config.get('export_settings', {})

        return {
            'chunk_size': settings.get('chunk_size', 1000),
            'timeout_seconds': settings.get('timeout_seconds', 300),
            'max_workers': settings.get('max_workers', 3),
            'correlation_timeout_seconds': settings.get('correlation_timeout_seconds', 60)
        }

    def get_retry_settings(self) -> Dict[str, Any]:
        """Get retry configuration."""
        retry_config = self.config.get('retry', {})

        return {
            'max_attempts': retry_config.get('max_attempts', 3),
            'backoff_factor': retry_config.get('backoff_factor', 2),
            'resume_broken_streams': retry_config.get('resume_broken_streams', True)
        }

    def get_performance_settings(self) -> Dict[str, Any]:
        """Get performance tuning settings."""
        perf_config = self.config.get('performance', {})

        return {
            'batch_size': perf_config.get('batch_size', 100),
            'max_concurrent_streams': perf_config.get('max_concurrent_streams', 3),
            'memory_limit_mb': perf_config.get('memory_limit_mb', 1024)
        }

    def get_ssl_settings(self) -> Dict[str, Any]:
        """Get SSL/TLS configuration settings."""
        ssl_config = self.config.get('ssl', {})

        return {
            'verify_certificates': ssl_config.get('verify_certificates', True),
            'ca_bundle_path': ssl_config.get('ca_bundle_path', '')
        }

    def get_default_filters(self) -> Dict[str, Any]:
        """Get default filters for data collection."""
        filters = self.config.get('export_settings', {}).get('filters', {})

        # Convert days to datetime for processing
        from datetime import datetime, timezone, timedelta

        result = {}

        if 'deployment_age_days' in filters:
            days_back = filters['deployment_age_days']
            result['start_date'] = datetime.now(timezone.utc) - timedelta(days=days_back)

        if 'include_inactive' in filters:
            result['include_inactive'] = filters['include_inactive']

        if 'severity_threshold' in filters:
            result['severity_threshold'] = filters['severity_threshold']

        if 'clusters' in filters:
            result['clusters'] = filters['clusters']

        if 'namespaces' in filters:
            result['namespaces'] = filters['namespaces']

        if 'policy_categories' in filters:
            result['policy_categories'] = filters['policy_categories']

        return result

    def create_ssl_context(self) -> Optional[ssl.SSLContext]:
        """
        Create SSL context for mTLS authentication.

        Returns:
            SSL context or None if not using mTLS
        """
        auth_config = self.get_authentication_config()

        if auth_config['method'] != 'mtls':
            return None

        try:
            context = ssl.create_default_context(ssl.Purpose.SERVER_AUTH)
            context.load_cert_chain(
                auth_config['client_cert'],
                auth_config['client_key']
            )
            context.load_verify_locations(auth_config['ca_cert'])
            context.check_hostname = True
            context.verify_mode = ssl.CERT_REQUIRED

            return context

        except Exception as e:
            logger.error(f"Failed to create SSL context: {e}")
            raise

    def validate_configuration(self) -> Tuple[bool, List[str]]:
        """
        Validate the Central API configuration.

        Returns:
            Tuple of (is_valid, list_of_issues)
        """
        issues = []

        if not self.is_enabled():
            return True, []  # Not enabled, so no validation needed

        # Check endpoint
        try:
            self.get_endpoint()
        except ValueError as e:
            issues.append(f"Endpoint configuration error: {e}")

        # Check authentication
        try:
            auth_config = self.get_authentication_config()
            if auth_config['method'] == 'mtls':
                self.create_ssl_context()  # This will raise if certificates are invalid
        except (ValueError, FileNotFoundError, ssl.SSLError) as e:
            issues.append(f"Authentication configuration error: {e}")

        # Check export settings
        export_settings = self.get_export_settings()
        if export_settings['chunk_size'] <= 0:
            issues.append("Invalid chunk_size: must be > 0")
        if export_settings['timeout_seconds'] <= 0:
            issues.append("Invalid timeout_seconds: must be > 0")
        if export_settings['max_workers'] <= 0:
            issues.append("Invalid max_workers: must be > 0")

        return len(issues) == 0, issues

    def get_client_config(self) -> Dict[str, Any]:
        """
        Get complete configuration for Central export client.

        Returns:
            Dictionary with all client configuration
        """
        auth_config = self.get_authentication_config()
        export_settings = self.get_export_settings()
        retry_settings = self.get_retry_settings()
        ssl_settings = self.get_ssl_settings()

        config = {
            'endpoint': self.get_endpoint(),
            'authentication': auth_config,
            **export_settings,
            **retry_settings,
            **ssl_settings
        }

        # Add SSL context for mTLS
        if auth_config['method'] == 'mtls':
            config['ssl_context'] = self.create_ssl_context()

        return config

    def log_configuration_summary(self):
        """Log a summary of the current configuration."""
        if not self.is_enabled():
            logger.info("Central API integration is disabled")
            return

        try:
            endpoint = self.get_endpoint()
            auth_config = self.get_authentication_config()
            export_settings = self.get_export_settings()

            logger.info(f"Central API Configuration:")
            logger.info(f"  Endpoint: {endpoint}")
            logger.info(f"  Authentication: {auth_config['method']}")
            logger.info(f"  Chunk size: {export_settings['chunk_size']}")
            logger.info(f"  Timeout: {export_settings['timeout_seconds']}s")
            logger.info(f"  Max workers: {export_settings['max_workers']}")

        except Exception as e:
            logger.error(f"Error logging configuration summary: {e}")


def create_central_client_from_config(config_path: Optional[str] = None):
    """
    Create a Central export client from configuration.

    Args:
        config_path: Optional path to configuration file

    Returns:
        Configured CentralExportClient instance
    """
    from src.clients.central_export_client import CentralExportClient

    config = CentralConfig(config_path)

    if not config.is_enabled():
        raise RuntimeError("Central API integration is not enabled")

    # Validate configuration
    is_valid, issues = config.validate_configuration()
    if not is_valid:
        raise ValueError(f"Invalid Central API configuration: {'; '.join(issues)}")

    # Get client configuration
    client_config = config.get_client_config()
    auth_config = client_config.pop('authentication')
    endpoint = client_config.pop('endpoint')

    # Create client
    if auth_config['method'] == 'api_token':
        client = CentralExportClient(
            endpoint=endpoint,
            auth_token=auth_config['token'],
            config=client_config
        )
    else:  # mTLS
        # For mTLS, the auth token parameter is not used
        # SSL context is handled in the session creation
        client = CentralExportClient(
            endpoint=endpoint,
            auth_token='',  # Not used for mTLS
            config=client_config
        )

    config.log_configuration_summary()
    return client