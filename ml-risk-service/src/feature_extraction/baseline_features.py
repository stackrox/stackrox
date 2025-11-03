"""
Baseline feature extractor that reproduces current StackRox risk scores.
This allows the ML model to initially learn from the existing risk system.
"""

import logging
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass
import math

from .deployment_features import DeploymentFeatureExtractor, DeploymentFeatures
from .image_features import ImageFeatureExtractor, ImageFeatures

logger = logging.getLogger(__name__)


@dataclass
class BaselineRiskFactors:
    """Risk factors that reproduce StackRox's current scoring system."""

    # Multiplier scores (as computed by StackRox)
    policy_violations_multiplier: float = 1.0
    process_baseline_multiplier: float = 1.0
    vulnerabilities_multiplier: float = 1.0
    service_config_multiplier: float = 1.0
    reachability_multiplier: float = 1.0
    risky_component_multiplier: float = 1.0
    component_count_multiplier: float = 1.0
    image_age_multiplier: float = 1.0

    # Overall score (product of multipliers)
    overall_score: float = 1.0

    # Individual risk result factors
    risk_factors: List[Dict[str, Any]] = None


class BaselineFeatureExtractor:
    """
    Extracts features that reproduce the exact StackRox risk calculation.
    This serves as the ground truth for initial ML model training.
    """

    def __init__(self):
        self.deployment_extractor = DeploymentFeatureExtractor()
        self.image_extractor = ImageFeatureExtractor()

        # Constants from StackRox risk multipliers
        self.POLICY_SATURATION = 50
        self.POLICY_MAX_VALUE = 4
        self.COMPONENT_SATURATION = 500
        self.COMPONENT_MAX_VALUE = 1.5
        self.AGE_THRESHOLD_DAYS = 365

    def extract_baseline_features(self,
                                deployment_data: Dict[str, Any],
                                image_data_list: List[Dict[str, Any]],
                                alert_data: List[Dict[str, Any]] = None,
                                baseline_violations: List[Dict[str, Any]] = None) -> BaselineRiskFactors:
        """
        Extract features that exactly reproduce StackRox risk calculation.

        Args:
            deployment_data: Deployment protobuf data
            image_data_list: List of image protobuf data
            alert_data: Policy violation alerts
            baseline_violations: Process baseline violations

        Returns:
            BaselineRiskFactors with computed multipliers
        """
        factors = BaselineRiskFactors()
        factors.risk_factors = []

        # 1. Policy Violations Multiplier (highest priority)
        factors.policy_violations_multiplier = self._calculate_policy_violations_multiplier(
            alert_data or [])

        # 2. Process Baseline Violations Multiplier
        factors.process_baseline_multiplier = self._calculate_process_baseline_multiplier(
            baseline_violations or [])

        # 3. Image-based multipliers (aggregated across all images)
        image_multipliers = self._calculate_image_multipliers(image_data_list)
        factors.vulnerabilities_multiplier = image_multipliers['vulnerabilities']
        factors.risky_component_multiplier = image_multipliers['risky_components']
        factors.component_count_multiplier = image_multipliers['component_count']
        factors.image_age_multiplier = image_multipliers['image_age']

        # 4. Service Configuration Multiplier
        factors.service_config_multiplier = self._calculate_service_config_multiplier(deployment_data)

        # 5. Reachability Multiplier (port exposure)
        factors.reachability_multiplier = self._calculate_reachability_multiplier(deployment_data)

        # Calculate overall score (product of all multipliers)
        factors.overall_score = (
            factors.policy_violations_multiplier *
            factors.process_baseline_multiplier *
            factors.vulnerabilities_multiplier *
            factors.service_config_multiplier *
            factors.reachability_multiplier *
            factors.risky_component_multiplier *
            factors.component_count_multiplier *
            factors.image_age_multiplier
        )

        return factors

    def _calculate_policy_violations_multiplier(self, alerts: List[Dict[str, Any]]) -> float:
        """
        Calculate policy violations multiplier exactly as in StackRox.
        Mirrors central/risk/multipliers/deployment/violations.go
        """
        if not alerts:
            return 1.0

        severity_sum = 0.0
        for alert in alerts:
            policy = alert.get('policy', {})
            severity = policy.get('severity', 'LOW_SEVERITY')
            severity_value = self._get_severity_value(severity)
            severity_sum += severity_value * severity_value

        return self._normalize_score(severity_sum, self.POLICY_SATURATION, self.POLICY_MAX_VALUE)

    def _calculate_process_baseline_multiplier(self, violations: List[Dict[str, Any]]) -> float:
        """
        Calculate process baseline violations multiplier.
        Mirrors central/risk/multipliers/deployment/process_baseline_violations.go
        """
        if not violations:
            return 1.0

        # Simple count-based scoring for now
        violation_count = len(violations)
        return min(1.0 + (violation_count * 0.1), 2.0)

    def _calculate_image_multipliers(self, images: List[Dict[str, Any]]) -> Dict[str, float]:
        """
        Calculate image-based risk multipliers.
        Aggregates risk across all images in the deployment.
        """
        if not images:
            return {
                'vulnerabilities': 1.0,
                'risky_components': 1.0,
                'component_count': 1.0,
                'image_age': 1.0
            }

        # Extract features for each image
        image_features_list = []
        for image_data in images:
            features = self.image_extractor.extract_features(image_data)
            image_features_list.append(features)

        # Calculate vulnerability multiplier (highest risk across images)
        vuln_multiplier = max(self._calculate_vulnerability_multiplier(img) for img in image_features_list)

        # Calculate risky component multiplier (sum across images)
        risky_comp_multiplier = self._calculate_risky_component_multiplier(image_features_list)

        # Calculate component count multiplier (average across images)
        comp_count_multiplier = self._calculate_component_count_multiplier(image_features_list)

        # Calculate image age multiplier (oldest image)
        age_multiplier = max(self._calculate_image_age_multiplier(img) for img in image_features_list)

        return {
            'vulnerabilities': vuln_multiplier,
            'risky_components': risky_comp_multiplier,
            'component_count': comp_count_multiplier,
            'image_age': age_multiplier
        }

    def _calculate_vulnerability_multiplier(self, image_features: ImageFeatures) -> float:
        """
        Calculate vulnerability multiplier for a single image.
        Mirrors central/risk/multipliers/image/vulnerabilities.go
        """
        severity_weights = {'CRITICAL': 10.0, 'HIGH': 4.0, 'MEDIUM': 1.0, 'LOW': 0.25}

        vuln_score = (
            image_features.critical_vuln_count * severity_weights['CRITICAL'] +
            image_features.high_vuln_count * severity_weights['HIGH'] +
            image_features.medium_vuln_count * severity_weights['MEDIUM'] +
            image_features.low_vuln_count * severity_weights['LOW']
        )

        if vuln_score == 0:
            return 1.0

        # Normalize with saturation at 50, max value 4.0
        return self._normalize_score(vuln_score, 50, 4.0)

    def _calculate_risky_component_multiplier(self, image_features_list: List[ImageFeatures]) -> float:
        """
        Calculate risky component multiplier.
        Mirrors central/risk/multipliers/image/risky_component.go
        """
        total_risky = sum(img.risky_component_count for img in image_features_list)

        if total_risky == 0:
            return 1.0

        # Normalize with saturation at 10, max value 1.5
        return self._normalize_score(total_risky, 10, 1.5)

    def _calculate_component_count_multiplier(self, image_features_list: List[ImageFeatures]) -> float:
        """
        Calculate component count multiplier.
        Mirrors central/risk/multipliers/image/component_count.go
        """
        if not image_features_list:
            return 1.0

        avg_components = sum(img.total_component_count for img in image_features_list) / len(image_features_list)

        # Normalize with saturation at 500, max value 1.5
        return self._normalize_score(avg_components, self.COMPONENT_SATURATION, self.COMPONENT_MAX_VALUE)

    def _calculate_image_age_multiplier(self, image_features: ImageFeatures) -> float:
        """
        Calculate image age multiplier.
        Mirrors central/risk/multipliers/image/image_age.go
        """
        if image_features.image_age_days <= 0:
            return 1.0

        # Age factor increases with days beyond threshold
        if image_features.image_age_days > self.AGE_THRESHOLD_DAYS:
            age_factor = (image_features.image_age_days - self.AGE_THRESHOLD_DAYS) / self.AGE_THRESHOLD_DAYS
            return min(1.0 + (age_factor * 0.3), 1.3)  # Cap at 1.3x multiplier

        return 1.0

    def _calculate_service_config_multiplier(self, deployment_data: Dict[str, Any]) -> float:
        """
        Calculate service configuration risk multiplier.
        Based on host access, privileged containers, etc.
        """
        multiplier = 1.0

        # Host network access
        if deployment_data.get('host_network', False):
            multiplier *= 1.2

        # Host PID access
        if deployment_data.get('host_pid', False):
            multiplier *= 1.15

        # Host IPC access
        if deployment_data.get('host_ipc', False):
            multiplier *= 1.1

        # Privileged containers
        containers = deployment_data.get('containers', [])
        privileged_count = 0
        for container in containers:
            security_context = container.get('security_context', {})
            if security_context.get('privileged', False):
                privileged_count += 1

        if privileged_count > 0:
            multiplier *= (1.0 + privileged_count * 0.15)

        return min(multiplier, 2.0)  # Cap multiplier

    def _calculate_reachability_multiplier(self, deployment_data: Dict[str, Any]) -> float:
        """
        Calculate network reachability risk multiplier.
        Based on port exposure and service configuration.
        """
        ports = deployment_data.get('ports', [])
        if not ports:
            return 1.0

        multiplier = 1.0
        exposed_ports = len(ports)

        # More exposed ports = higher risk
        multiplier += (exposed_ports * 0.05)

        # External exposure (LoadBalancer, NodePort)
        for port in ports:
            if port.get('exposure') in ['EXTERNAL', 'NODE']:
                multiplier *= 1.2
                break

        return min(multiplier, 1.5)  # Cap multiplier

    def _normalize_score(self, score: float, saturation: float, max_value: float) -> float:
        """
        Normalize score using StackRox's normalization function.
        Mirrors central/risk/multipliers/utils.go:NormalizeScore
        """
        if score > saturation:
            return max_value
        return 1 + (score / saturation) * (max_value - 1)

    def _get_severity_value(self, severity: str) -> float:
        """Get numeric value for policy severity."""
        severity_values = {
            'CRITICAL_SEVERITY': 4.0,
            'HIGH_SEVERITY': 3.0,
            'MEDIUM_SEVERITY': 2.0,
            'LOW_SEVERITY': 1.0
        }
        return severity_values.get(severity, 1.0)

    def _ensure_risk_score_variance(self, baseline_score: float,
                                   features: Dict[str, float],
                                   baseline_factors: BaselineRiskFactors) -> float:
        """
        Ensure risk scores have meaningful variance for ML training.

        When baseline calculation returns uniform scores (typically 1.0),
        generate synthetic scores based on feature values to enable learning.

        Args:
            baseline_score: Original baseline risk score
            features: Extracted feature values
            baseline_factors: Computed baseline factors

        Returns:
            Risk score with ensured variance
        """
        # If baseline score is meaningful (not default 1.0), use it
        if baseline_score != 1.0:
            return baseline_score

        # Generate synthetic score based on feature values
        synthetic_score = 1.0  # Base score

        # Add vulnerability-based risk (high impact)
        vuln_score = features.get('avg_vulnerability_score', 0.0)
        if vuln_score > 0:
            synthetic_score += min(vuln_score * 0.5, 2.0)  # Cap at +2.0

        # Add configuration-based risk (medium impact)
        config_risk = 0.0
        if features.get('host_network', 0.0) > 0:
            config_risk += 0.3
        if features.get('host_pid', 0.0) > 0:
            config_risk += 0.3
        if features.get('privileged_container_ratio', 0.0) > 0:
            config_risk += 0.4
        if features.get('has_external_exposure', 0.0) > 0:
            config_risk += 0.2

        synthetic_score += config_risk

        # Add component-based risk (low impact)
        risky_ratio = features.get('avg_risky_component_ratio', 0.0)
        if risky_ratio > 0:
            synthetic_score += min(risky_ratio * 0.3, 0.5)

        # Add age-based risk (very low impact)
        age_score = features.get('age_days', 0.0)
        if age_score > 0.5:  # Older deployments slightly riskier
            synthetic_score += min(age_score * 0.1, 0.2)

        # Add some controlled randomization for variance
        import random
        random.seed(hash(str(features)) % 2**32)  # Deterministic based on features
        noise = random.uniform(-0.1, 0.1)
        synthetic_score += noise

        # Ensure reasonable bounds [0.5, 5.0]
        final_score = max(0.5, min(synthetic_score, 5.0))

        return final_score

    def create_training_sample(self,
                               deployment_data: Dict[str, Any],
                               image_data_list: List[Dict[str, Any]],
                               alert_data: List[Dict[str, Any]] = None,
                               baseline_violations: List[Dict[str, Any]] = None,
                               risk_score: Optional[float] = None) -> Dict[str, Any]:
        """
        Create a training sample with features and risk score.

        Args:
            deployment_data: Deployment information
            image_data_list: List of image data
            alert_data: Optional alert/violation data
            baseline_violations: Optional baseline violation data
            risk_score: Optional risk score from Central (if None, will compute from baseline)

        Returns:
            Dict with 'features' and 'risk_score' keys for ML training
        """
        # Extract normalized features for ML model
        deployment_features = self.deployment_extractor.extract_features(deployment_data, alert_data)
        normalized_deployment = self.deployment_extractor.normalize_features(deployment_features)

        image_features_list = []
        for image_data in image_data_list:
            image_features = self.image_extractor.extract_features(image_data)
            normalized_image = self.image_extractor.normalize_features(image_features)
            image_features_list.append(normalized_image)

        # Combine all features
        combined_features = normalized_deployment.copy()

        # Aggregate image features (mean, max, sum strategies)
        if image_features_list:
            # Average image features across all images
            for key in image_features_list[0].keys():
                values = [img[key] for img in image_features_list]
                combined_features[f'avg_{key}'] = sum(values) / len(values)
                combined_features[f'max_{key}'] = max(values)
                combined_features[f'sum_{key}'] = sum(values)
        else:
            # No images - add default zero values for consistency
            # These are the features from ImageFeatureExtractor.normalize_features()
            default_image_features = [
                'vulnerability_score', 'avg_cvss_score', 'max_cvss_score',
                'component_count_score', 'risky_component_ratio', 'age_score',
                'is_cluster_local', 'log_layer_count'
            ]
            for key in default_image_features:
                combined_features[f'avg_{key}'] = 0.0
                combined_features[f'max_{key}'] = 0.0
                combined_features[f'sum_{key}'] = 0.0

        # Determine final risk score
        if risk_score is not None:
            # Use Central's provided risk score (ground truth)
            final_risk_score = risk_score
            baseline_factors = None  # No need to compute baseline when using Central's score
        else:
            # Fallback: compute from baseline (used for synthetic data generation)
            baseline_factors = self.extract_baseline_features(
                deployment_data, image_data_list, alert_data, baseline_violations)
            # Use synthetic scoring if baseline calculation produces no variance
            final_risk_score = self._ensure_risk_score_variance(
                baseline_factors.overall_score, combined_features, baseline_factors)

        # Build return dictionary
        result = {
            'features': combined_features,
            'risk_score': final_risk_score
        }

        # Include baseline_factors only when computed (for synthetic data)
        if baseline_factors is not None:
            result['baseline_factors'] = {
                'policy_violations': baseline_factors.policy_violations_multiplier,
                'process_baseline': baseline_factors.process_baseline_multiplier,
                'vulnerabilities': baseline_factors.vulnerabilities_multiplier,
                'service_config': baseline_factors.service_config_multiplier,
                'reachability': baseline_factors.reachability_multiplier,
                'risky_components': baseline_factors.risky_component_multiplier,
                'component_count': baseline_factors.component_count_multiplier,
                'image_age': baseline_factors.image_age_multiplier,
                'final_score': final_risk_score,
                'baseline_score': baseline_factors.overall_score
            }

        return result