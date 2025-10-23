"""
Deployment feature extraction that mirrors StackRox risk multipliers.
"""

import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from datetime import datetime, timezone
import yaml

logger = logging.getLogger(__name__)


@dataclass
class DeploymentFeatures:
    """Feature vector for deployment risk assessment."""

    # Policy violations (highest priority)
    policy_violation_count: int = 0
    policy_violation_severity_score: float = 0.0

    # Process baseline violations
    process_baseline_violations: int = 0

    # Host access features
    host_network: bool = False
    host_pid: bool = False
    host_ipc: bool = False

    # Container security
    privileged_container_count: int = 0
    automount_service_account_token: bool = False

    # Port exposure
    exposed_port_count: int = 0
    has_external_exposure: bool = False

    # Service account permissions
    service_account_permission_level: float = 0.0

    # Deployment configuration
    replica_count: int = 1
    is_orchestrator_component: bool = False
    is_platform_component: bool = False

    # Cluster context
    cluster_id: str = ""
    namespace: str = ""

    # Age and activity
    creation_timestamp: int = 0
    is_inactive: bool = False


class DeploymentFeatureExtractor:
    """Extracts features from deployment data matching StackRox risk factors."""

    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_config(config_path)
        self.severity_weights = {
            "CRITICAL": 10.0,
            "HIGH": 4.0,
            "MEDIUM": 1.0,
            "LOW": 0.25
        }

    def _load_config(self, config_path: Optional[str]) -> Dict[str, Any]:
        """Load feature extraction configuration."""
        if config_path:
            with open(config_path, 'r') as f:
                return yaml.safe_load(f)
        else:
            # Default configuration
            return {
                'features': {
                    'deployment': {
                        'policy_violations': {'enabled': True, 'weight': 1.0},
                        'process_baseline_violations': {'enabled': True, 'weight': 0.8},
                        'host_network': {'enabled': True, 'weight': 0.7},
                        'privileged_containers': {'enabled': True, 'weight': 0.9}
                    }
                }
            }

    def extract_features(self, deployment_data: Dict[str, Any],
                        alert_data: List[Dict[str, Any]] = None) -> DeploymentFeatures:
        """
        Extract features from deployment protobuf data.

        Args:
            deployment_data: Deployment protobuf as dict
            alert_data: List of alerts for this deployment

        Returns:
            DeploymentFeatures object
        """
        features = DeploymentFeatures()

        # Basic deployment info
        features.cluster_id = deployment_data.get('cluster_id', '')
        features.namespace = deployment_data.get('namespace', '')
        features.replica_count = deployment_data.get('replicas', 1)
        features.is_orchestrator_component = deployment_data.get('orchestrator_component', False)
        features.is_platform_component = deployment_data.get('platform_component', False)
        features.is_inactive = deployment_data.get('inactive', False)

        # Creation timestamp
        created = deployment_data.get('created')
        if created:
            # Convert protobuf timestamp to unix timestamp
            features.creation_timestamp = int(created.get('seconds', 0))

        # Host access features - mirrors violations multiplier
        features.host_network = deployment_data.get('host_network', False)
        features.host_pid = deployment_data.get('host_pid', False)
        features.host_ipc = deployment_data.get('host_ipc', False)

        # Service account configuration
        features.automount_service_account_token = deployment_data.get(
            'automount_service_account_token', False)

        # Service account permission level
        perm_level = deployment_data.get('service_account_permission_level')
        if perm_level is not None:
            features.service_account_permission_level = float(perm_level)

        # Container-level features
        containers = deployment_data.get('containers', [])
        for container in containers:
            security_context = container.get('security_context', {})
            if security_context.get('privileged', False):
                features.privileged_container_count += 1

        # Port exposure - mirrors port exposure multiplier
        ports = deployment_data.get('ports', [])
        features.exposed_port_count = len(ports)
        for port in ports:
            # Check for external exposure (LoadBalancer, NodePort)
            if port.get('exposure') in ['EXTERNAL', 'NODE']:
                features.has_external_exposure = True
                break

        # Policy violations - mirrors violations multiplier
        if alert_data:
            features.policy_violation_count = len(alert_data)
            features.policy_violation_severity_score = self._calculate_severity_score(alert_data)

        return features

    def _calculate_severity_score(self, alerts: List[Dict[str, Any]]) -> float:
        """
        Calculate severity score matching StackRox violations multiplier.
        Mirrors the logic in central/risk/multipliers/deployment/violations.go
        """
        severity_sum = 0.0

        for alert in alerts:
            policy = alert.get('policy', {})
            severity = policy.get('severity', 'LOW_SEVERITY')

            # Clean severity string (remove _SEVERITY suffix)
            clean_severity = severity.replace('_SEVERITY', '')

            # Apply severity impact: severity * severity (like in violations.go:80)
            severity_value = self._get_severity_value(clean_severity)
            severity_sum += severity_value * severity_value

        return severity_sum

    def _get_severity_value(self, severity: str) -> float:
        """Get numeric value for severity level."""
        severity_values = {
            'CRITICAL': 4.0,
            'HIGH': 3.0,
            'MEDIUM': 2.0,
            'LOW': 1.0
        }
        return severity_values.get(severity.upper(), 1.0)

    def extract_process_baseline_features(self, deployment_id: str,
                                        baseline_data: Dict[str, Any] = None) -> int:
        """
        Extract process baseline violation count.
        This would connect to process baseline evaluator.
        """
        if not baseline_data:
            return 0

        violations = baseline_data.get('violations', [])
        return len(violations)

    def normalize_features(self, features: DeploymentFeatures) -> Dict[str, float]:
        """
        Normalize features to 0-1 range for ML model input.
        """
        config = self.config.get('features', {}).get('deployment', {})

        normalized = {}

        # Policy violations - use saturation normalization like StackRox
        if config.get('policy_violations', {}).get('enabled', True):
            saturation = config.get('policy_violations', {}).get('normalize_saturation', 50)
            max_val = config.get('policy_violations', {}).get('max_value', 4.0)
            normalized['policy_violation_score'] = self._normalize_score(
                features.policy_violation_severity_score, saturation, max_val)

        # Binary features
        normalized['host_network'] = float(features.host_network)
        normalized['host_pid'] = float(features.host_pid)
        normalized['host_ipc'] = float(features.host_ipc)
        normalized['has_external_exposure'] = float(features.has_external_exposure)
        normalized['is_orchestrator_component'] = float(features.is_orchestrator_component)
        normalized['automount_service_account_token'] = float(features.automount_service_account_token)

        # Count features (log normalize)
        normalized['log_replica_count'] = self._log_normalize(features.replica_count)
        normalized['log_exposed_port_count'] = self._log_normalize(features.exposed_port_count)
        normalized['privileged_container_ratio'] = min(features.privileged_container_count / max(features.replica_count, 1), 1.0)

        # Age feature (days since creation)
        if features.creation_timestamp > 0:
            age_days = (datetime.now(timezone.utc).timestamp() - features.creation_timestamp) / 86400
            normalized['age_days'] = min(age_days / 365.0, 5.0)  # Cap at 5 years
        else:
            normalized['age_days'] = 0.0

        return normalized

    def _normalize_score(self, score: float, saturation: float, max_value: float) -> float:
        """
        Normalize score using StackRox's normalization function.
        Mirrors central/risk/multipliers/utils.go:NormalizeScore
        """
        if score > saturation:
            return max_value
        return 1 + (score / saturation) * (max_value - 1)

    def _log_normalize(self, value: int) -> float:
        """Log normalize count values."""
        import math
        return math.log1p(value) / math.log1p(100)  # Normalize to reasonable range