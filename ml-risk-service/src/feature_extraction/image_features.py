"""
Image feature extraction that mirrors StackRox image risk multipliers.
"""

import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from datetime import datetime, timezone
import math

logger = logging.getLogger(__name__)


@dataclass
class ImageFeatures:
    """Feature vector for image risk assessment."""

    # Image identification
    image_id: str = ""
    image_name: str = ""

    # Vulnerability metrics - mirrors vulnerabilities multiplier
    critical_vuln_count: int = 0
    high_vuln_count: int = 0
    medium_vuln_count: int = 0
    low_vuln_count: int = 0
    avg_cvss_score: float = 0.0
    max_cvss_score: float = 0.0

    # Component metrics - mirrors component multipliers
    total_component_count: int = 0
    risky_component_count: int = 0

    # Image metadata - mirrors image age multiplier
    image_creation_timestamp: int = 0
    image_age_days: int = 0
    is_cluster_local: bool = False

    # Base image info
    base_image: str = ""
    layer_count: int = 0


class ImageFeatureExtractor:
    """Extracts features from image data matching StackRox image risk factors."""

    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_default_config()

        # Severity weights matching StackRox vulnerability scoring
        self.severity_weights = {
            "CRITICAL": 10.0,
            "HIGH": 4.0,
            "MEDIUM": 1.0,
            "LOW": 0.25
        }

    def _load_default_config(self) -> Dict[str, Any]:
        """Load default configuration for image features."""
        return {
            'features': {
                'image': {
                    'vulnerabilities': {
                        'enabled': True,
                        'weight': 1.0,
                        'cvss_threshold': 7.0
                    },
                    'risky_components': {
                        'enabled': True,
                        'weight': 0.7,
                        'risk_threshold': 5
                    },
                    'component_count': {
                        'enabled': True,
                        'weight': 0.4,
                        'saturation': 500,
                        'max_value': 1.5
                    },
                    'image_age': {
                        'enabled': True,
                        'weight': 0.5,
                        'age_threshold_days': 365,
                        'max_multiplier': 1.3
                    }
                }
            }
        }

    def extract_features(self, image_data: Dict[str, Any]) -> ImageFeatures:
        """
        Extract features from image protobuf data.

        Args:
            image_data: Image protobuf as dict

        Returns:
            ImageFeatures object
        """
        features = ImageFeatures()

        # Basic image info
        features.image_id = image_data.get('id', '')

        # Image name
        name = image_data.get('name', {})
        if isinstance(name, dict):
            features.image_name = f"{name.get('registry', '')}/{name.get('remote', '')}"
        else:
            features.image_name = str(name)

        # Metadata - handle both layer SHA lists and layer counts from Central API
        metadata = image_data.get('metadata', {})
        layer_shas = metadata.get('layerShas', [])
        if isinstance(layer_shas, (int, float)):
            # Central API returns layer count as integer
            features.layer_count = int(layer_shas)
            logger.debug(f"Using layer count from Central API: {layer_shas}")
        elif isinstance(layer_shas, (list, tuple)):
            # Central API returns actual layer SHA list
            features.layer_count = len(layer_shas)
        else:
            logger.warning(f"Unexpected layerShas type {type(layer_shas)}: {layer_shas}, using default 0")
            features.layer_count = 0

        # Creation timestamp - handle both protobuf and ISO string formats
        if 'created' in metadata:
            created = metadata['created']
            if isinstance(created, dict):
                # Protobuf style: {"seconds": 1698509065}
                features.image_creation_timestamp = int(created.get('seconds', 0))
            elif isinstance(created, str):
                # ISO string: "2023-10-28T18:54:25.638Z"
                try:
                    from datetime import datetime
                    created_clean = created.replace('Z', '+00:00') if created.endswith('Z') else created
                    dt = datetime.fromisoformat(created_clean)
                    features.image_creation_timestamp = int(dt.timestamp())
                except Exception as e:
                    logger.warning(f"Failed to parse image timestamp '{created}': {e}")
                    features.image_creation_timestamp = 0
            else:
                logger.warning(f"Unknown image timestamp format: {type(created)} - {created}")
                features.image_creation_timestamp = 0

            features.image_age_days = self._calculate_age_days(features.image_creation_timestamp)

        # Cluster local flag
        features.is_cluster_local = image_data.get('cluster_local', False)

        # Component analysis - handle both component counts and component lists from Central API
        components = image_data.get('components', [])
        if isinstance(components, (int, float)):
            # Central API returns component count as integer
            features.total_component_count = int(components)
            # Estimate risky components as 10% of total (reasonable default)
            features.risky_component_count = max(1, int(components * 0.1)) if components > 0 else 0
            logger.debug(f"Using component count from Central API: {components} total, {features.risky_component_count} estimated risky")
        elif isinstance(components, (list, tuple)):
            # Central API returns actual component list
            features.total_component_count = len(components)
            features.risky_component_count = self._count_risky_components(components)
        else:
            logger.warning(f"Unexpected components type {type(components)}: {components}, using defaults")
            features.total_component_count = 0
            features.risky_component_count = 0

        # Vulnerability analysis - mirrors vulnerability multiplier
        self._extract_vulnerability_features(image_data, features)

        return features

    def _extract_vulnerability_features(self, image_data: Dict[str, Any],
                                      features: ImageFeatures) -> None:
        """
        Extract vulnerability features matching StackRox vulnerability multiplier.
        Mirrors central/risk/multipliers/image/vulnerabilities.go
        """
        scan = image_data.get('scan', {})
        components = scan.get('components', [])

        # Check if we have count-based data instead of detailed scan results
        if not isinstance(components, (list, tuple)):
            # Handle count-based or summary scan data
            logger.debug(f"Scan components is not a list: {type(components)}, looking for vulnerability counts")
            self._extract_vulnerability_counts_from_scan(scan, features)
            return

        cvss_scores = []

        for component in components:
            vulns = component.get('vulns', [])

            for vuln in vulns:
                severity = vuln.get('severity', 'LOW_SEVERITY')
                clean_severity = severity.replace('_SEVERITY', '')

                # Count by severity
                if clean_severity == 'CRITICAL':
                    features.critical_vuln_count += 1
                elif clean_severity == 'HIGH':
                    features.high_vuln_count += 1
                elif clean_severity == 'MEDIUM':
                    features.medium_vuln_count += 1
                elif clean_severity == 'LOW':
                    features.low_vuln_count += 1

                # CVSS score analysis
                cvss = vuln.get('cvss', 0.0)
                if cvss > 0:
                    cvss_scores.append(cvss)

        # Calculate CVSS statistics
        if cvss_scores:
            features.avg_cvss_score = sum(cvss_scores) / len(cvss_scores)
            features.max_cvss_score = max(cvss_scores)

    def _count_risky_components(self, components: List[Dict[str, Any]]) -> int:
        """
        Count risky components matching StackRox risky component multiplier.
        Mirrors central/risk/multipliers/image/risky_component.go
        """
        risky_count = 0

        for component in components:
            vulns = component.get('vulns', [])

            # Component is risky if it has high/critical vulnerabilities
            for vuln in vulns:
                severity = vuln.get('severity', 'LOW_SEVERITY')
                if severity in ['CRITICAL_SEVERITY', 'HIGH_SEVERITY']:
                    risky_count += 1
                    break  # Count component only once

        return risky_count

    def _extract_vulnerability_counts_from_scan(self, scan: Dict[str, Any], features: ImageFeatures) -> None:
        """
        Extract vulnerability features from count-based scan data.
        Used when Central API provides summary counts instead of detailed vulnerability lists.
        """
        # Look for direct vulnerability count fields
        features.critical_vuln_count = int(scan.get('criticalVulns', scan.get('critical_vulns', 0)))
        features.high_vuln_count = int(scan.get('highVulns', scan.get('high_vulns', 0)))
        features.medium_vuln_count = int(scan.get('mediumVulns', scan.get('medium_vulns', 0)))
        features.low_vuln_count = int(scan.get('lowVulns', scan.get('low_vulns', 0)))

        # Estimate CVSS scores from severity distribution
        total_vulns = (features.critical_vuln_count + features.high_vuln_count +
                      features.medium_vuln_count + features.low_vuln_count)

        if total_vulns > 0:
            # Estimate average CVSS based on severity distribution
            weighted_score = (features.critical_vuln_count * 9.5 +  # Critical: ~9.5
                            features.high_vuln_count * 7.5 +        # High: ~7.5
                            features.medium_vuln_count * 5.0 +      # Medium: ~5.0
                            features.low_vuln_count * 2.0)          # Low: ~2.0
            features.avg_cvss_score = weighted_score / total_vulns

            # Estimate max CVSS (critical vulns likely have high CVSS)
            if features.critical_vuln_count > 0:
                features.max_cvss_score = 9.8  # Typical critical
            elif features.high_vuln_count > 0:
                features.max_cvss_score = 8.5  # Typical high
            elif features.medium_vuln_count > 0:
                features.max_cvss_score = 6.0  # Typical medium
            else:
                features.max_cvss_score = 3.0  # Typical low

        logger.debug(f"Extracted vulnerability counts: Critical={features.critical_vuln_count}, "
                    f"High={features.high_vuln_count}, Medium={features.medium_vuln_count}, "
                    f"Low={features.low_vuln_count}, AvgCVSS={features.avg_cvss_score:.1f}")

    def _calculate_age_days(self, timestamp: int) -> int:
        """Calculate image age in days."""
        if timestamp <= 0:
            return 0

        current_time = datetime.now(timezone.utc).timestamp()
        age_seconds = current_time - timestamp
        return int(age_seconds / 86400)

    def normalize_features(self, features: ImageFeatures) -> Dict[str, float]:
        """
        Normalize image features for ML model input.
        Applies normalization similar to StackRox multipliers.
        """
        normalized = {}
        config = self.config.get('features', {}).get('image', {})

        # Vulnerability score - weighted by severity (mirrors vulnerabilities.go)
        vuln_score = (
            features.critical_vuln_count * self.severity_weights['CRITICAL'] +
            features.high_vuln_count * self.severity_weights['HIGH'] +
            features.medium_vuln_count * self.severity_weights['MEDIUM'] +
            features.low_vuln_count * self.severity_weights['LOW']
        )
        normalized['vulnerability_score'] = min(vuln_score / 100.0, 10.0)  # Cap and normalize

        # CVSS scores
        normalized['avg_cvss_score'] = features.avg_cvss_score / 10.0  # Normalize to 0-1
        normalized['max_cvss_score'] = features.max_cvss_score / 10.0

        # Component metrics - mirrors component count multiplier
        component_config = config.get('component_count', {})
        saturation = component_config.get('saturation', 500)
        max_value = component_config.get('max_value', 1.5)

        normalized['component_count_score'] = self._normalize_score(
            features.total_component_count, saturation, max_value)

        # Risky component ratio
        if features.total_component_count > 0:
            normalized['risky_component_ratio'] = features.risky_component_count / features.total_component_count
        else:
            normalized['risky_component_ratio'] = 0.0

        # Image age - mirrors image age multiplier
        age_config = config.get('image_age', {})
        age_threshold = age_config.get('age_threshold_days', 365)
        max_multiplier = age_config.get('max_multiplier', 1.3)

        if features.image_age_days > 0:
            age_score = min(features.image_age_days / age_threshold, 2.0)
            normalized['age_score'] = self._normalize_score(age_score, 1.0, max_multiplier)
        else:
            normalized['age_score'] = 1.0

        # Binary features
        normalized['is_cluster_local'] = float(features.is_cluster_local)

        # Layer count (log normalize)
        normalized['log_layer_count'] = self._log_normalize(features.layer_count)

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
        return math.log1p(value) / math.log1p(50)  # Normalize layer count