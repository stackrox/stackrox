"""
Configuration management for data sources (Central API, files).
Handles authentication, environment variables, and multi-source configuration.
"""

import os
import logging
import ssl
from typing import Dict, Any, List, Optional, Tuple
from pathlib import Path
from dataclasses import dataclass
import yaml

logger = logging.getLogger(__name__)


@dataclass
class SourceConfig:
    """Configuration for a single data source (Central or file)."""

    type: str  # "central" or "file"
    name: str
    enabled: bool = True

    # Central-specific fields
    endpoint: Optional[str] = None
    authentication: Optional[Dict[str, Any]] = None
    filters: Optional[Dict[str, Any]] = None

    # File-specific fields
    path: Optional[str] = None
    format: Optional[str] = None  # "json" or "jsonl"

    def is_central(self) -> bool:
        """Check if this is a Central API source."""
        return self.type == "central"

    def is_file(self) -> bool:
        """Check if this is a file source."""
        return self.type == "file"


class DataSourceConfig:
    """
    Configuration manager for data sources.
    Supports multiple Central instances and file sources for training/prediction.
    """

    def __init__(self, config_path: Optional[str] = None, source_type: str = "training"):
        """
        Initialize data source configuration.

        Args:
            config_path: Optional path to configuration file
            source_type: Type of sources to load: "training" or "prediction"
        """
        self.source_type = source_type
        self.config_path = config_path or self._find_config_file()
        self.full_config = self._load_full_config()
        self.common_config = self.full_config.get('common', {})

        # Load source-specific configuration
        self.source_config = self._load_source_config()
        self.sources = self._parse_sources()

    @classmethod
    def for_training(cls, config_path: Optional[str] = None) -> 'DataSourceConfig':
        """Create configuration for training sources."""
        return cls(config_path=config_path, source_type="training")

    @classmethod
    def for_prediction(cls, config_path: Optional[str] = None) -> 'DataSourceConfig':
        """Create configuration for prediction sources."""
        return cls(config_path=config_path, source_type="prediction")

    def _find_config_file(self) -> str:
        """Find the feature configuration file."""
        possible_paths = [
            '/app/config/feature_config.yaml',  # Production
            os.path.join(os.path.dirname(__file__), 'feature_config.yaml'),  # Dev
            'src/config/feature_config.yaml'  # Relative path fallback
        ]

        for path in possible_paths:
            if os.path.exists(path):
                return path

        raise FileNotFoundError("Could not find feature_config.yaml file")

    def _load_full_config(self) -> Dict[str, Any]:
        """Load full configuration from YAML file."""
        try:
            with open(self.config_path, 'r') as f:
                return yaml.safe_load(f) or {}
        except Exception as e:
            logger.warning(f"Failed to load config from {self.config_path}: {e}")
            return {}

    def _load_source_config(self) -> Dict[str, Any]:
        """Load source-specific configuration section."""
        # Try new format first
        section = self.full_config.get(self.source_type, {})

        if section:
            logger.debug(f"Loading '{self.source_type}' configuration section")
            return section

        # Fall back to legacy format
        legacy_key = f"{self.source_type}_central_api"
        legacy_section = self.full_config.get(legacy_key, {})

        if legacy_section:
            logger.warning(
                f"Using legacy '{legacy_key}' section. "
                f"Please migrate to new '{self.source_type}' section format."
            )
            return self._convert_legacy_config(legacy_section)

        return {}

    def _convert_legacy_config(self, legacy_config: Dict[str, Any]) -> Dict[str, Any]:
        """Convert legacy config format to new multi-source format."""
        if not legacy_config.get('enabled', False):
            return {'sources': []}

        # Create a single source from legacy config
        source = {
            'type': 'central',
            'name': f'legacy-{self.source_type}-central',
            'enabled': True,
            'endpoint': legacy_config.get('endpoint', ''),
            'authentication': legacy_config.get('authentication', {}),
            'filters': legacy_config.get('export_settings', {}).get('filters', {})
        }

        return {
            'sources': [source],
            f'{self.source_type}_settings': legacy_config.get(self.source_type, {})
        }

    def _parse_sources(self) -> List[SourceConfig]:
        """Parse sources list from configuration."""
        sources_list = self.source_config.get('sources', [])
        parsed_sources = []

        for idx, source_dict in enumerate(sources_list):
            try:
                source = SourceConfig(
                    type=source_dict.get('type', 'central'),
                    name=source_dict.get('name', f'source-{idx}'),
                    enabled=source_dict.get('enabled', True),
                    endpoint=source_dict.get('endpoint'),
                    authentication=source_dict.get('authentication'),
                    filters=source_dict.get('filters'),
                    path=source_dict.get('path'),
                    format=source_dict.get('format', 'json')
                )
                parsed_sources.append(source)
            except Exception as e:
                logger.error(f"Failed to parse source {idx}: {e}")
                continue

        return parsed_sources

    def get_sources(self, enabled_only: bool = True) -> List[SourceConfig]:
        """
        Get list of configured sources.

        Args:
            enabled_only: If True, return only enabled sources

        Returns:
            List of SourceConfig objects
        """
        if enabled_only:
            return [s for s in self.sources if s.enabled]
        return self.sources

    def get_central_sources(self, enabled_only: bool = True) -> List[SourceConfig]:
        """Get only Central API sources."""
        sources = self.get_sources(enabled_only=enabled_only)
        return [s for s in sources if s.is_central()]

    def get_file_sources(self, enabled_only: bool = True) -> List[SourceConfig]:
        """Get only file sources."""
        sources = self.get_sources(enabled_only=enabled_only)
        return [s for s in sources if s.is_file()]

    def get_authentication_config(self, source: SourceConfig) -> Dict[str, Any]:
        """
        Get authentication configuration for a Central source with env var resolution.

        Args:
            source: SourceConfig for a Central instance

        Returns:
            Dictionary with authentication settings
        """
        if not source.is_central():
            raise ValueError(f"Authentication only applies to Central sources, got {source.type}")

        auth_config = source.authentication or {}
        method = auth_config.get('method', 'api_token')

        if method == 'api_token':
            return self._get_token_auth_config(auth_config, source.name)
        elif method == 'mtls':
            return self._get_mtls_auth_config(auth_config, source.name)
        else:
            raise ValueError(f"Unsupported authentication method: {method}")

    def _get_token_auth_config(self, auth_config: Dict[str, Any], source_name: str) -> Dict[str, Any]:
        """Get API token authentication configuration."""
        token = None

        # New format: api_token_env specifies the environment variable NAME
        if 'api_token_env' in auth_config:
            env_var_name = auth_config['api_token_env']
            token = os.getenv(env_var_name)
            if not token:
                raise ValueError(
                    f"Authentication token not found in environment variable '{env_var_name}' "
                    f"for source '{source_name}'"
                )
        # Legacy format: api_token with ${ENV_VAR} interpolation
        elif 'api_token' in auth_config:
            token_value = auth_config['api_token']
            if token_value and token_value.startswith('${') and token_value.endswith('}'):
                env_var_name = token_value[2:-1]
                token = os.getenv(env_var_name)
                logger.warning(
                    f"Using legacy ${{ENV_VAR}} format for source '{source_name}'. "
                    f"Please migrate to 'api_token_env' format."
                )
            else:
                # Direct token value (not recommended)
                token = token_value

        if not token:
            raise ValueError(f"No authentication token configured for source '{source_name}'")

        return {
            'method': 'api_token',
            'token': token
        }

    def _get_mtls_auth_config(self, auth_config: Dict[str, Any], source_name: str) -> Dict[str, Any]:
        """Get mTLS authentication configuration."""
        # New format: *_env fields specify environment variable NAMES
        if 'client_cert_path_env' in auth_config:
            cert_path = self._resolve_env_var(auth_config.get('client_cert_path_env'), 'client certificate')
            key_path = self._resolve_env_var(auth_config.get('client_key_path_env'), 'client key')
            ca_path = self._resolve_env_var(auth_config.get('ca_cert_path_env'), 'CA certificate')
        # Legacy format: ${ENV_VAR} interpolation
        else:
            cert_path = self._resolve_path(auth_config.get('client_cert_path', ''))
            key_path = self._resolve_path(auth_config.get('client_key_path', ''))
            ca_path = self._resolve_path(auth_config.get('ca_cert_path', ''))
            logger.warning(
                f"Using legacy ${{ENV_VAR}} format for mTLS config in source '{source_name}'. "
                f"Please migrate to '*_env' format."
            )

        # Validate certificate files exist
        for path, name in [(cert_path, 'client certificate'),
                          (key_path, 'client key'),
                          (ca_path, 'CA certificate')]:
            if not path or not os.path.exists(path):
                raise FileNotFoundError(
                    f"{name} not found at: {path} for source '{source_name}'"
                )

        return {
            'method': 'mtls',
            'client_cert': cert_path,
            'client_key': key_path,
            'ca_cert': ca_path
        }

    def _resolve_env_var(self, env_var_name: Optional[str], description: str) -> str:
        """Resolve an environment variable by name."""
        if not env_var_name:
            raise ValueError(f"Environment variable name not specified for {description}")

        value = os.getenv(env_var_name)
        if not value:
            raise ValueError(f"{description} path not found in environment variable '{env_var_name}'")

        return value

    def _resolve_path(self, path: str) -> str:
        """Resolve path with legacy ${ENV_VAR} substitution."""
        if not path:
            return ''

        # Substitute environment variables (legacy format)
        if path.startswith('${') and path.endswith('}'):
            env_var = path[2:-1]
            resolved_path = os.getenv(env_var)
            if not resolved_path:
                logger.warning(f"Environment variable {env_var} not set")
                return ''
            return resolved_path

        return path

    def get_common_config(self) -> Dict[str, Any]:
        """Get common configuration shared by all sources."""
        return self.common_config

    def get_ssl_settings(self) -> Dict[str, Any]:
        """Get SSL/TLS configuration settings from common config."""
        ssl_config = self.common_config.get('ssl', {})
        return {
            'verify_certificates': ssl_config.get('verify_certificates', True),
            'ca_bundle_path': ssl_config.get('ca_bundle_path', '')
        }

    def get_retry_settings(self) -> Dict[str, Any]:
        """Get retry configuration from common config."""
        retry_config = self.common_config.get('retry', {})
        return {
            'max_attempts': retry_config.get('max_attempts', 3),
            'backoff_factor': retry_config.get('backoff_factor', 2),
            'resume_broken_streams': retry_config.get('resume_broken_streams', True)
        }

    def get_performance_settings(self) -> Dict[str, Any]:
        """Get performance tuning settings from common config."""
        perf_config = self.common_config.get('performance', {})
        return {
            'batch_size': perf_config.get('batch_size', 100),
            'max_concurrent_streams': perf_config.get('max_concurrent_streams', 3),
            'memory_limit_mb': perf_config.get('memory_limit_mb', 1024)
        }

    def get_export_settings(self) -> Dict[str, Any]:
        """Get export API settings from common config."""
        settings = self.common_config.get('export_settings', {})
        return {
            'chunk_size': settings.get('chunk_size', 1000),
            'timeout_seconds': settings.get('timeout_seconds', 300),
            'max_workers': settings.get('max_workers', 3),
            'workload_settings': settings.get('workload_settings', {}),
            'default_filters': settings.get('default_filters', {})
        }

    def get_source_filters(self, source: SourceConfig) -> Dict[str, Any]:
        """
        Get filters for a specific source, with fallback to common default_filters.

        Args:
            source: SourceConfig object

        Returns:
            Dictionary with filter settings
        """
        # Source-specific filters override common defaults
        if source.filters:
            return source.filters

        # Fall back to common default filters
        export_settings = self.get_export_settings()
        return export_settings.get('default_filters', {})

    def create_ssl_context(self, auth_config: Dict[str, Any]) -> Optional[ssl.SSLContext]:
        """
        Create SSL context for mTLS authentication.

        Args:
            auth_config: Authentication configuration dictionary

        Returns:
            SSL context or None if not using mTLS
        """
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

    def get_client_config_for_source(self, source: SourceConfig) -> Dict[str, Any]:
        """
        Get complete configuration for a Central export client.

        Args:
            source: SourceConfig for a Central instance

        Returns:
            Dictionary with all client configuration
        """
        if not source.is_central():
            raise ValueError(f"Client config only applies to Central sources, got {source.type}")

        auth_config = self.get_authentication_config(source)
        export_settings = self.get_export_settings()
        retry_settings = self.get_retry_settings()
        ssl_settings = self.get_ssl_settings()

        config = {
            'endpoint': source.endpoint.rstrip('/') if source.endpoint else '',
            'authentication': auth_config,
            **export_settings,
            **retry_settings,
            **ssl_settings
        }

        # Add SSL context for mTLS
        if auth_config['method'] == 'mtls':
            config['ssl_context'] = self.create_ssl_context(auth_config)

        return config

    def validate_source(self, source: SourceConfig) -> Tuple[bool, List[str]]:
        """
        Validate a single source configuration.

        Args:
            source: SourceConfig to validate

        Returns:
            Tuple of (is_valid, list_of_issues)
        """
        issues = []

        if not source.enabled:
            return True, []  # Disabled sources don't need validation

        if source.is_central():
            # Validate Central source
            if not source.endpoint:
                issues.append(f"Source '{source.name}': Missing endpoint")

            try:
                self.get_authentication_config(source)
            except (ValueError, FileNotFoundError, ssl.SSLError) as e:
                issues.append(f"Source '{source.name}': Authentication error - {e}")

        elif source.is_file():
            # Validate file source
            if not source.path:
                issues.append(f"Source '{source.name}': Missing file path")
            elif not os.path.exists(source.path):
                issues.append(f"Source '{source.name}': File not found - {source.path}")

            if source.format not in ['json', 'jsonl']:
                issues.append(f"Source '{source.name}': Invalid format '{source.format}', must be 'json' or 'jsonl'")

        else:
            issues.append(f"Source '{source.name}': Unknown type '{source.type}'")

        return len(issues) == 0, issues

    def validate_configuration(self) -> Tuple[bool, List[str]]:
        """
        Validate the complete configuration.

        Returns:
            Tuple of (is_valid, list_of_issues)
        """
        all_issues = []

        sources = self.get_sources(enabled_only=True)
        if not sources:
            all_issues.append(f"No enabled sources configured for {self.source_type}")

        for source in sources:
            is_valid, issues = self.validate_source(source)
            all_issues.extend(issues)

        # Validate common export settings
        export_settings = self.get_export_settings()
        if export_settings['chunk_size'] <= 0:
            all_issues.append("Invalid chunk_size in common config: must be > 0")
        if export_settings['timeout_seconds'] <= 0:
            all_issues.append("Invalid timeout_seconds in common config: must be > 0")

        return len(all_issues) == 0, all_issues

    def log_configuration_summary(self):
        """Log a summary of the current configuration."""
        logger.info(f"Data Source Configuration ({self.source_type}):")

        sources = self.get_sources(enabled_only=True)
        logger.info(f"  Enabled sources: {len(sources)}")

        for source in sources:
            logger.info(f"  - {source.name} ({source.type})")
            if source.is_central():
                logger.info(f"      Endpoint: {source.endpoint}")
                auth_method = source.authentication.get('method', 'unknown') if source.authentication else 'unknown'
                logger.info(f"      Auth: {auth_method}")
            elif source.is_file():
                logger.info(f"      Path: {source.path}")
                logger.info(f"      Format: {source.format}")


# Backward compatibility: Keep CentralConfig class as alias
class CentralConfig(DataSourceConfig):
    """
    DEPRECATED: Use DataSourceConfig instead.
    This class is kept for backward compatibility.
    """

    def __init__(self, config_path: Optional[str] = None, use_prediction_env: bool = False):
        """
        Initialize Central configuration (deprecated constructor).

        Args:
            config_path: Optional path to configuration file
            use_prediction_env: If True, use prediction sources instead of training
        """
        logger.warning(
            "CentralConfig is deprecated. Use DataSourceConfig.for_training() "
            "or DataSourceConfig.for_prediction() instead."
        )
        source_type = "prediction" if use_prediction_env else "training"
        super().__init__(config_path=config_path, source_type=source_type)
        self.use_prediction_env = use_prediction_env

    @classmethod
    def from_prediction_env(cls, config_path: Optional[str] = None) -> 'CentralConfig':
        """Create a CentralConfig instance for prediction Central (deprecated)."""
        logger.warning(
            "CentralConfig.from_prediction_env() is deprecated. "
            "Use DataSourceConfig.for_prediction() instead."
        )
        return cls(config_path=config_path, use_prediction_env=True)

    def is_enabled(self) -> bool:
        """Check if any sources are enabled (deprecated method)."""
        return len(self.get_sources(enabled_only=True)) > 0

    def get_endpoint(self) -> str:
        """Get endpoint from first enabled Central source (deprecated method)."""
        central_sources = self.get_central_sources(enabled_only=True)
        if not central_sources:
            raise ValueError("No enabled Central sources configured")
        return central_sources[0].endpoint.rstrip('/') if central_sources[0].endpoint else ''

    def get_client_config(self) -> Dict[str, Any]:
        """Get config for first Central source (deprecated method)."""
        central_sources = self.get_central_sources(enabled_only=True)
        if not central_sources:
            raise ValueError("No enabled Central sources configured")
        return self.get_client_config_for_source(central_sources[0])


def create_central_client_from_config(config_path: Optional[str] = None):
    """
    Create a Central export client from configuration.
    Uses the first enabled training Central source.

    Args:
        config_path: Optional path to configuration file

    Returns:
        Configured CentralExportClient instance
    """
    from src.clients.central_export_client import CentralExportClient

    config = DataSourceConfig.for_training(config_path)

    central_sources = config.get_central_sources(enabled_only=True)
    if not central_sources:
        raise RuntimeError("No enabled Central sources configured for training")

    # Use first Central source
    source = central_sources[0]

    # Validate source
    is_valid, issues = config.validate_source(source)
    if not is_valid:
        raise ValueError(f"Invalid Central source configuration: {'; '.join(issues)}")

    # Get client configuration
    client_config = config.get_client_config_for_source(source)
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
        client = CentralExportClient(
            endpoint=endpoint,
            auth_token='',  # Not used for mTLS
            config=client_config
        )

    logger.info(f"Created Central client for source: {source.name}")
    return client
