"""
Feature importance analysis and explanation generation for risk ranking.
"""

import logging
import numpy as np
import pandas as pd
from typing import Dict, Any, List, Tuple, Optional
from dataclasses import dataclass
import json

try:
    import shap
    SHAP_AVAILABLE = True
except ImportError:
    SHAP_AVAILABLE = False

logger = logging.getLogger(__name__)


@dataclass
class FeatureExplanation:
    """Explanation for a single feature's contribution to risk score."""
    feature_name: str
    importance_score: float
    feature_category: str
    description: str
    value: float = 0.0
    baseline_value: float = 0.0


@dataclass
class RiskExplanation:
    """Complete explanation for a deployment's risk score."""
    deployment_id: str
    risk_score: float
    baseline_score: float
    top_features: List[FeatureExplanation]
    category_contributions: Dict[str, float]
    explanation_summary: str


class FeatureImportanceAnalyzer:
    """
    Analyzes and explains feature importance for risk predictions.
    Provides both global and local explanations.
    """

    def __init__(self, feature_categories: Optional[Dict[str, str]] = None):
        self.feature_categories = feature_categories or self._default_feature_categories()
        self.feature_descriptions = self._default_feature_descriptions()

    def _default_feature_categories(self) -> Dict[str, str]:
        """Default mapping of features to categories."""
        return {
            # Policy and violations
            'policy_violation_score': 'policy',
            'process_baseline_violations': 'policy',

            # Host access
            'host_network': 'host_access',
            'host_pid': 'host_access',
            'host_ipc': 'host_access',

            # Container security
            'privileged_container_ratio': 'container_security',
            'automount_service_account_token': 'container_security',

            # Network exposure
            'has_external_exposure': 'network',
            'log_exposed_port_count': 'network',

            # Image vulnerabilities
            'avg_vulnerability_score': 'vulnerabilities',
            'max_vulnerability_score': 'vulnerabilities',
            'avg_avg_cvss_score': 'vulnerabilities',
            'max_avg_cvss_score': 'vulnerabilities',
            'sum_vulnerability_score': 'vulnerabilities',

            # Image components
            'avg_risky_component_ratio': 'components',
            'max_risky_component_ratio': 'components',
            'avg_component_count_score': 'components',

            # Image age
            'avg_age_score': 'image_age',
            'max_age_score': 'image_age',

            # Deployment configuration
            'log_replica_count': 'configuration',
            'is_orchestrator_component': 'configuration',
            'age_days': 'configuration'
        }

    def _default_feature_descriptions(self) -> Dict[str, str]:
        """Default descriptions for features."""
        return {
            'policy_violation_score': 'Policy violations with severity weighting',
            'process_baseline_violations': 'Number of process baseline violations',
            'host_network': 'Container uses host network namespace',
            'host_pid': 'Container uses host PID namespace',
            'host_ipc': 'Container uses host IPC namespace',
            'privileged_container_ratio': 'Ratio of privileged containers',
            'automount_service_account_token': 'Service account token auto-mounted',
            'has_external_exposure': 'Deployment exposed to external traffic',
            'log_exposed_port_count': 'Number of exposed ports (log normalized)',
            'avg_vulnerability_score': 'Average vulnerability score across images',
            'max_vulnerability_score': 'Maximum vulnerability score across images',
            'avg_avg_cvss_score': 'Average CVSS score across images',
            'max_avg_cvss_score': 'Maximum CVSS score across images',
            'sum_vulnerability_score': 'Total vulnerability score across images',
            'avg_risky_component_ratio': 'Average ratio of risky components',
            'max_risky_component_ratio': 'Maximum ratio of risky components',
            'avg_component_count_score': 'Average component count score',
            'avg_age_score': 'Average image age score',
            'max_age_score': 'Maximum image age score',
            'log_replica_count': 'Number of replicas (log normalized)',
            'is_orchestrator_component': 'Component is part of orchestrator',
            'age_days': 'Age of deployment in days'
        }

    def analyze_global_importance(self, model, feature_names: List[str],
                                 X_sample: Optional[np.ndarray] = None) -> Dict[str, Any]:
        """
        Analyze global feature importance across the model.

        Args:
            model: Trained ML model
            feature_names: List of feature names
            X_sample: Sample data for SHAP analysis (optional)

        Returns:
            Global importance analysis
        """
        logger.info("Analyzing global feature importance")

        analysis = {
            'feature_importance': {},
            'category_importance': {},
            'top_features': [],
            'importance_method': 'model_default'
        }

        # Get model-specific feature importance
        if hasattr(model, 'feature_importance'):
            # LightGBM
            importance_scores = model.feature_importance()
            analysis['importance_method'] = 'lightgbm_gain'
        elif hasattr(model, 'feature_importances_'):
            # Sklearn models
            importance_scores = model.feature_importances_
            analysis['importance_method'] = 'sklearn_impurity'
        else:
            logger.warning("Model does not support feature importance")
            return analysis

        # Map to feature names
        feature_importance = dict(zip(feature_names, importance_scores))

        # Normalize importance scores
        total_importance = sum(importance_scores)
        if total_importance > 0:
            feature_importance = {k: v / total_importance for k, v in feature_importance.items()}

        analysis['feature_importance'] = feature_importance

        # Calculate category-level importance
        category_importance = {}
        for feature, importance in feature_importance.items():
            category = self.feature_categories.get(feature, 'other')
            category_importance[category] = category_importance.get(category, 0) + importance

        analysis['category_importance'] = category_importance

        # Top features
        top_features = sorted(feature_importance.items(), key=lambda x: x[1], reverse=True)[:10]
        analysis['top_features'] = [
            {
                'feature': feature,
                'importance': importance,
                'category': self.feature_categories.get(feature, 'other'),
                'description': self.feature_descriptions.get(feature, 'No description')
            }
            for feature, importance in top_features
        ]

        logger.info(f"Global importance analysis complete. Top feature: {top_features[0][0]}")
        return analysis

    def explain_prediction(self, model, X: np.ndarray, prediction: float,
                          feature_names: List[str], deployment_id: str = "",
                          shap_explainer=None) -> RiskExplanation:
        """
        Generate explanation for a single prediction.

        Args:
            model: Trained ML model
            X: Feature vector for single prediction (1D array)
            prediction: Model prediction
            feature_names: List of feature names
            deployment_id: ID of deployment being explained
            shap_explainer: SHAP explainer (optional)

        Returns:
            Detailed risk explanation
        """
        logger.debug(f"Explaining prediction for deployment {deployment_id}")

        # Ensure X is 2D
        if X.ndim == 1:
            X = X.reshape(1, -1)

        feature_explanations = []
        baseline_score = 1.0  # Default baseline risk score

        # Get feature contributions
        if shap_explainer is not None and SHAP_AVAILABLE:
            # SHAP explanations
            try:
                shap_values = shap_explainer.shap_values(X)
                if isinstance(shap_values, list):
                    shap_values = shap_values[0]  # Multi-class case

                base_value = shap_explainer.expected_value
                if isinstance(base_value, np.ndarray):
                    base_value = base_value[0]

                baseline_score = float(base_value)

                for i, (feature_name, shap_value) in enumerate(zip(feature_names, shap_values[0])):
                    feature_explanations.append(FeatureExplanation(
                        feature_name=feature_name,
                        importance_score=float(shap_value),
                        feature_category=self.feature_categories.get(feature_name, 'other'),
                        description=self.feature_descriptions.get(feature_name, 'No description'),
                        value=float(X[0, i]),
                        baseline_value=0.0  # SHAP baseline is already in base_value
                    ))

            except Exception as e:
                logger.warning(f"SHAP explanation failed: {e}")
                feature_explanations = self._fallback_explanation(X[0], feature_names, model)

        else:
            # Fallback explanation using feature importance
            feature_explanations = self._fallback_explanation(X[0], feature_names, model)

        # Sort by absolute importance
        feature_explanations.sort(key=lambda x: abs(x.importance_score), reverse=True)

        # Take top features
        top_features = feature_explanations[:10]

        # Calculate category contributions
        category_contributions = {}
        for explanation in feature_explanations:
            category = explanation.feature_category
            category_contributions[category] = category_contributions.get(category, 0) + explanation.importance_score

        # Generate summary
        explanation_summary = self._generate_explanation_summary(
            prediction, baseline_score, top_features, category_contributions)

        return RiskExplanation(
            deployment_id=deployment_id,
            risk_score=float(prediction),
            baseline_score=baseline_score,
            top_features=top_features,
            category_contributions=category_contributions,
            explanation_summary=explanation_summary
        )

    def _fallback_explanation(self, x: np.ndarray, feature_names: List[str], model) -> List[FeatureExplanation]:
        """Generate fallback explanation when SHAP is not available."""

        explanations = []

        # Get global feature importance if available
        global_importance = {}
        if hasattr(model, 'feature_importance'):
            global_importance = dict(zip(feature_names, model.feature_importance()))
        elif hasattr(model, 'feature_importances_'):
            global_importance = dict(zip(feature_names, model.feature_importances_))

        # Create explanations based on feature values and global importance
        for i, feature_name in enumerate(feature_names):
            feature_value = float(x[i])
            global_imp = global_importance.get(feature_name, 0.0)

            # Simple heuristic: importance proportional to value deviation from 0.5 (normalized baseline)
            # and global importance
            local_importance = (feature_value - 0.5) * global_imp

            explanations.append(FeatureExplanation(
                feature_name=feature_name,
                importance_score=local_importance,
                feature_category=self.feature_categories.get(feature_name, 'other'),
                description=self.feature_descriptions.get(feature_name, 'No description'),
                value=feature_value,
                baseline_value=0.5
            ))

        return explanations

    def _generate_explanation_summary(self, risk_score: float, baseline_score: float,
                                    top_features: List[FeatureExplanation],
                                    category_contributions: Dict[str, float]) -> str:
        """Generate human-readable explanation summary."""

        # Risk level assessment
        if risk_score < 1.5:
            risk_level = "LOW"
        elif risk_score < 2.5:
            risk_level = "MEDIUM"
        elif risk_score < 4.0:
            risk_level = "HIGH"
        else:
            risk_level = "CRITICAL"

        summary = f"Risk Level: {risk_level} (Score: {risk_score:.2f})\n\n"

        # Top contributing factors
        if top_features:
            summary += "Top Risk Factors:\n"
            for i, feature in enumerate(top_features[:5], 1):
                direction = "increases" if feature.importance_score > 0 else "decreases"
                summary += f"{i}. {feature.description} - {direction} risk by {abs(feature.importance_score):.3f}\n"

        # Category analysis
        if category_contributions:
            summary += "\nRisk by Category:\n"
            sorted_categories = sorted(category_contributions.items(), key=lambda x: abs(x[1]), reverse=True)
            for category, contribution in sorted_categories[:5]:
                if abs(contribution) > 0.01:  # Only show significant contributions
                    direction = "increases" if contribution > 0 else "decreases"
                    summary += f"- {category.title()}: {direction} risk by {abs(contribution):.3f}\n"

        return summary

    def generate_comparison_explanation(self, deployment1: RiskExplanation,
                                      deployment2: RiskExplanation) -> str:
        """
        Generate explanation comparing two deployments.

        Args:
            deployment1: First deployment explanation
            deployment2: Second deployment explanation

        Returns:
            Comparison explanation
        """
        score_diff = deployment1.risk_score - deployment2.risk_score

        if abs(score_diff) < 0.1:
            comparison = f"Both deployments have similar risk scores ({deployment1.risk_score:.2f} vs {deployment2.risk_score:.2f})"
        else:
            higher_risk = deployment1 if score_diff > 0 else deployment2
            lower_risk = deployment2 if score_diff > 0 else deployment1
            comparison = f"{higher_risk.deployment_id} has higher risk ({higher_risk.risk_score:.2f}) than {lower_risk.deployment_id} ({lower_risk.risk_score:.2f})"

        # Find key differences
        comparison += "\n\nKey Differences:\n"

        # Compare top features
        dep1_features = {f.feature_name: f.importance_score for f in deployment1.top_features}
        dep2_features = {f.feature_name: f.importance_score for f in deployment2.top_features}

        all_features = set(dep1_features.keys()) | set(dep2_features.keys())
        feature_diffs = []

        for feature in all_features:
            diff = dep1_features.get(feature, 0) - dep2_features.get(feature, 0)
            if abs(diff) > 0.05:  # Significant difference
                feature_diffs.append((feature, diff))

        feature_diffs.sort(key=lambda x: abs(x[1]), reverse=True)

        for feature, diff in feature_diffs[:5]:
            direction = "higher" if diff > 0 else "lower"
            description = self.feature_descriptions.get(feature, feature)
            comparison += f"- {description}: {direction} in {deployment1.deployment_id} by {abs(diff):.3f}\n"

        return comparison

    def export_explanations(self, explanations: List[RiskExplanation], output_file: str) -> None:
        """
        Export explanations to JSON file.

        Args:
            explanations: List of risk explanations
            output_file: Output file path
        """
        export_data = []

        for explanation in explanations:
            export_data.append({
                'deployment_id': explanation.deployment_id,
                'risk_score': explanation.risk_score,
                'baseline_score': explanation.baseline_score,
                'top_features': [
                    {
                        'feature_name': f.feature_name,
                        'importance_score': f.importance_score,
                        'feature_category': f.feature_category,
                        'description': f.description,
                        'value': f.value,
                        'baseline_value': f.baseline_value
                    }
                    for f in explanation.top_features
                ],
                'category_contributions': explanation.category_contributions,
                'explanation_summary': explanation.explanation_summary
            })

        with open(output_file, 'w') as f:
            json.dump(export_data, f, indent=2, default=str)

        logger.info(f"Exported {len(explanations)} explanations to {output_file}")

    def create_feature_importance_report(self, global_analysis: Dict[str, Any],
                                       sample_explanations: List[RiskExplanation],
                                       output_file: str) -> None:
        """
        Create comprehensive feature importance report.

        Args:
            global_analysis: Global feature importance analysis
            sample_explanations: Sample individual explanations
            output_file: Output file path
        """
        report = {
            'global_analysis': global_analysis,
            'sample_explanations': [
                {
                    'deployment_id': exp.deployment_id,
                    'risk_score': exp.risk_score,
                    'top_3_factors': [
                        f"{f.feature_name}: {f.importance_score:.3f}"
                        for f in exp.top_features[:3]
                    ],
                    'category_contributions': exp.category_contributions
                }
                for exp in sample_explanations[:10]  # Include first 10 as examples
            ],
            'summary_statistics': self._calculate_explanation_statistics(sample_explanations),
            'metadata': {
                'total_explanations': len(sample_explanations),
                'feature_count': len(global_analysis.get('feature_importance', {})),
                'category_count': len(global_analysis.get('category_importance', {}))
            }
        }

        with open(output_file, 'w') as f:
            json.dump(report, f, indent=2, default=str)

        logger.info(f"Feature importance report saved to {output_file}")

    def _calculate_explanation_statistics(self, explanations: List[RiskExplanation]) -> Dict[str, Any]:
        """Calculate statistics across multiple explanations."""
        if not explanations:
            return {}

        risk_scores = [exp.risk_score for exp in explanations]

        # Feature frequency in top features
        feature_frequency = {}
        for exp in explanations:
            for feature in exp.top_features[:5]:  # Top 5 features
                feature_frequency[feature.feature_name] = feature_frequency.get(feature.feature_name, 0) + 1

        # Category frequency
        category_frequency = {}
        for exp in explanations:
            for category, contribution in exp.category_contributions.items():
                if abs(contribution) > 0.01:  # Significant contribution
                    category_frequency[category] = category_frequency.get(category, 0) + 1

        return {
            'risk_score_stats': {
                'mean': np.mean(risk_scores),
                'std': np.std(risk_scores),
                'min': np.min(risk_scores),
                'max': np.max(risk_scores)
            },
            'most_frequent_features': sorted(feature_frequency.items(), key=lambda x: x[1], reverse=True)[:10],
            'most_frequent_categories': sorted(category_frequency.items(), key=lambda x: x[1], reverse=True)
        }
