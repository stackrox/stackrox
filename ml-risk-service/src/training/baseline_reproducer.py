"""
Baseline reproducer that validates our feature extraction reproduces StackRox risk scores.
"""

import logging
import json
from typing import Dict, Any, List, Tuple
import numpy as np
from sklearn.metrics import mean_squared_error, mean_absolute_error
import matplotlib.pyplot as plt

from .data_loader import TrainingDataLoader
from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logger = logging.getLogger(__name__)


class BaselineReproducer:
    """
    Validates that our feature extraction can reproduce existing StackRox risk scores.
    This ensures the ML model starts with accurate baseline knowledge.
    """

    def __init__(self):
        self.baseline_extractor = BaselineFeatureExtractor()
        self.data_loader = TrainingDataLoader()

    def validate_baseline_reproduction(self, test_data: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Validate that we can reproduce StackRox risk scores from raw deployment data.

        Args:
            test_data: List of test deployment records with known risk scores

        Returns:
            Validation report with accuracy metrics
        """
        logger.info(f"Validating baseline reproduction on {len(test_data)} samples")

        actual_scores = []
        predicted_scores = []
        detailed_results = []

        for i, record in enumerate(test_data):
            try:
                # Extract known risk score
                known_score = record.get('current_risk_score')
                if known_score is None:
                    logger.warning(f"Example {i}: No known risk score, skipping")
                    continue

                # Calculate our baseline score
                deployment_data = record.get('deployment', {})
                images_data = record.get('images', [])
                alerts_data = record.get('alerts', [])
                baseline_violations = record.get('baseline_violations', [])

                baseline_factors = self.baseline_extractor.extract_baseline_features(
                    deployment_data, images_data, alerts_data, baseline_violations
                )

                predicted_score = baseline_factors.overall_score

                actual_scores.append(known_score)
                predicted_scores.append(predicted_score)

                detailed_results.append({
                    'deployment_id': deployment_data.get('id', f'example-{i}'),
                    'actual_score': known_score,
                    'predicted_score': predicted_score,
                    'error': abs(known_score - predicted_score),
                    'relative_error': abs(known_score - predicted_score) / max(known_score, 0.001),
                    'multipliers': {
                        'policy_violations': baseline_factors.policy_violations_multiplier,
                        'process_baseline': baseline_factors.process_baseline_multiplier,
                        'vulnerabilities': baseline_factors.vulnerabilities_multiplier,
                        'service_config': baseline_factors.service_config_multiplier,
                        'reachability': baseline_factors.reachability_multiplier,
                        'risky_components': baseline_factors.risky_component_multiplier,
                        'component_count': baseline_factors.component_count_multiplier,
                        'image_age': baseline_factors.image_age_multiplier
                    }
                })

                if (i + 1) % 50 == 0:
                    logger.info(f"Processed {i + 1} samples")

            except Exception as e:
                logger.warning(f"Failed to process example {i}: {e}")
                continue

        # Calculate validation metrics
        if not actual_scores:
            return {'valid': False, 'error': 'No valid samples to validate'}

        actual_scores = np.array(actual_scores)
        predicted_scores = np.array(predicted_scores)

        mse = mean_squared_error(actual_scores, predicted_scores)
        mae = mean_absolute_error(actual_scores, predicted_scores)
        rmse = np.sqrt(mse)

        # Calculate relative metrics
        relative_errors = np.abs(actual_scores - predicted_scores) / np.maximum(actual_scores, 0.001)
        mean_relative_error = np.mean(relative_errors)
        median_relative_error = np.median(relative_errors)

        # Correlation
        correlation = np.corrcoef(actual_scores, predicted_scores)[0, 1]

        # Accuracy within thresholds
        accuracy_5pct = np.mean(relative_errors < 0.05)
        accuracy_10pct = np.mean(relative_errors < 0.10)
        accuracy_20pct = np.mean(relative_errors < 0.20)

        validation_report = {
            'valid': True,
            'total_samples': len(actual_scores),
            'metrics': {
                'mse': float(mse),
                'mae': float(mae),
                'rmse': float(rmse),
                'mean_relative_error': float(mean_relative_error),
                'median_relative_error': float(median_relative_error),
                'correlation': float(correlation),
                'accuracy_5pct': float(accuracy_5pct),
                'accuracy_10pct': float(accuracy_10pct),
                'accuracy_20pct': float(accuracy_20pct)
            },
            'score_statistics': {
                'actual_min': float(np.min(actual_scores)),
                'actual_max': float(np.max(actual_scores)),
                'actual_mean': float(np.mean(actual_scores)),
                'predicted_min': float(np.min(predicted_scores)),
                'predicted_max': float(np.max(predicted_scores)),
                'predicted_mean': float(np.mean(predicted_scores))
            },
            'detailed_results': detailed_results
        }

        # Assessment
        if correlation > 0.95 and mean_relative_error < 0.1:
            validation_report['assessment'] = 'EXCELLENT'
        elif correlation > 0.9 and mean_relative_error < 0.2:
            validation_report['assessment'] = 'GOOD'
        elif correlation > 0.8 and mean_relative_error < 0.3:
            validation_report['assessment'] = 'ACCEPTABLE'
        else:
            validation_report['assessment'] = 'POOR'

        logger.info(f"Baseline reproduction validation complete: {validation_report['assessment']}")
        logger.info(f"Correlation: {correlation:.3f}, Mean Relative Error: {mean_relative_error:.3f}")

        return validation_report

    def analyze_multiplier_contributions(self, test_data: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Analyze the contribution of each risk multiplier to overall scores.

        Args:
            test_data: List of test deployment records

        Returns:
            Analysis report of multiplier importance
        """
        logger.info("Analyzing multiplier contributions")

        multiplier_data = {
            'policy_violations': [],
            'process_baseline': [],
            'vulnerabilities': [],
            'service_config': [],
            'reachability': [],
            'risky_components': [],
            'component_count': [],
            'image_age': []
        }

        overall_scores = []

        for record in test_data:
            try:
                deployment_data = record.get('deployment', {})
                images_data = record.get('images', [])
                alerts_data = record.get('alerts', [])
                baseline_violations = record.get('baseline_violations', [])

                baseline_factors = self.baseline_extractor.extract_baseline_features(
                    deployment_data, images_data, alerts_data, baseline_violations
                )

                # Collect multiplier values
                multiplier_data['policy_violations'].append(baseline_factors.policy_violations_multiplier)
                multiplier_data['process_baseline'].append(baseline_factors.process_baseline_multiplier)
                multiplier_data['vulnerabilities'].append(baseline_factors.vulnerabilities_multiplier)
                multiplier_data['service_config'].append(baseline_factors.service_config_multiplier)
                multiplier_data['reachability'].append(baseline_factors.reachability_multiplier)
                multiplier_data['risky_components'].append(baseline_factors.risky_component_multiplier)
                multiplier_data['component_count'].append(baseline_factors.component_count_multiplier)
                multiplier_data['image_age'].append(baseline_factors.image_age_multiplier)

                overall_scores.append(baseline_factors.overall_score)

            except Exception as e:
                logger.warning(f"Failed to analyze multipliers for record: {e}")
                continue

        # Calculate statistics for each multiplier
        analysis = {}
        for multiplier_name, values in multiplier_data.items():
            if values:
                values_array = np.array(values)
                analysis[multiplier_name] = {
                    'mean': float(np.mean(values_array)),
                    'std': float(np.std(values_array)),
                    'min': float(np.min(values_array)),
                    'max': float(np.max(values_array)),
                    'median': float(np.median(values_array)),
                    'above_baseline_pct': float(np.mean(values_array > 1.0)),
                    'high_impact_pct': float(np.mean(values_array > 1.5))
                }

                # Correlation with overall score
                if len(overall_scores) == len(values):
                    correlation = np.corrcoef(values_array, overall_scores)[0, 1]
                    analysis[multiplier_name]['correlation_with_overall'] = float(correlation)

        # Rank multipliers by impact
        impact_ranking = []
        for name, stats in analysis.items():
            impact_score = (
                stats.get('correlation_with_overall', 0) * 0.4 +
                stats['high_impact_pct'] * 0.3 +
                (stats['mean'] - 1.0) * 0.3
            )
            impact_ranking.append((name, impact_score))

        impact_ranking.sort(key=lambda x: x[1], reverse=True)

        analysis_report = {
            'multiplier_statistics': analysis,
            'impact_ranking': impact_ranking,
            'total_samples': len(overall_scores),
            'overall_score_stats': {
                'mean': float(np.mean(overall_scores)),
                'std': float(np.std(overall_scores)),
                'min': float(np.min(overall_scores)),
                'max': float(np.max(overall_scores))
            }
        }

        logger.info("Multiplier contribution analysis complete")
        logger.info(f"Top impact multipliers: {[name for name, _ in impact_ranking[:3]]}")

        return analysis_report

    def generate_baseline_validation_report(self, test_data_file: str, output_file: str) -> None:
        """
        Generate comprehensive baseline validation report.

        Args:
            test_data_file: Path to test data JSON file
            output_file: Path to output validation report
        """
        logger.info(f"Generating baseline validation report from {test_data_file}")

        # Load test data
        test_data = self.data_loader.load_from_json(test_data_file)

        # Run validation
        validation_report = self.validate_baseline_reproduction(test_data)
        multiplier_analysis = self.analyze_multiplier_contributions(test_data)

        # Combine reports
        full_report = {
            'validation_report': validation_report,
            'multiplier_analysis': multiplier_analysis,
            'metadata': {
                'test_data_file': test_data_file,
                'total_samples': len(test_data),
                'timestamp': pd.Timestamp.now().isoformat()
            }
        }

        # Save report
        with open(output_file, 'w') as f:
            json.dump(full_report, f, indent=2, default=str)

        logger.info(f"Baseline validation report saved to {output_file}")

    def create_baseline_comparison_plots(self, validation_report: Dict[str, Any], output_dir: str) -> None:
        """
        Create visualization plots comparing actual vs predicted risk scores.

        Args:
            validation_report: Validation report from validate_baseline_reproduction
            output_dir: Directory to save plots
        """
        try:
            import os
            os.makedirs(output_dir, exist_ok=True)

            detailed_results = validation_report.get('detailed_results', [])
            if not detailed_results:
                logger.warning("No detailed results available for plotting")
                return

            actual_scores = [r['actual_score'] for r in detailed_results]
            predicted_scores = [r['predicted_score'] for r in detailed_results]

            # Actual vs Predicted scatter plot
            plt.figure(figsize=(10, 8))
            plt.scatter(actual_scores, predicted_scores, alpha=0.6)
            plt.plot([min(actual_scores), max(actual_scores)],
                    [min(actual_scores), max(actual_scores)], 'r--', lw=2)
            plt.xlabel('Actual Risk Score')
            plt.ylabel('Predicted Risk Score')
            plt.title('Baseline Reproduction: Actual vs Predicted Risk Scores')
            plt.grid(True, alpha=0.3)

            # Add correlation info
            correlation = validation_report['metrics']['correlation']
            plt.text(0.05, 0.95, f'Correlation: {correlation:.3f}',
                    transform=plt.gca().transAxes, bbox=dict(boxstyle="round", facecolor='wheat'))

            plt.savefig(f"{output_dir}/actual_vs_predicted.png", dpi=300, bbox_inches='tight')
            plt.close()

            # Error distribution histogram
            errors = [r['error'] for r in detailed_results]
            plt.figure(figsize=(10, 6))
            plt.hist(errors, bins=50, alpha=0.7, edgecolor='black')
            plt.xlabel('Absolute Error')
            plt.ylabel('Frequency')
            plt.title('Distribution of Prediction Errors')
            plt.grid(True, alpha=0.3)
            plt.savefig(f"{output_dir}/error_distribution.png", dpi=300, bbox_inches='tight')
            plt.close()

            logger.info(f"Baseline comparison plots saved to {output_dir}")

        except ImportError:
            logger.warning("matplotlib not available, skipping plot generation")
        except Exception as e:
            logger.error(f"Failed to create plots: {e}")

    def test_feature_extraction_pipeline(self) -> Dict[str, Any]:
        """
        Test the complete feature extraction pipeline with sample data.

        Returns:
            Test results
        """
        logger.info("Testing feature extraction pipeline")

        try:
            # Generate small sample dataset
            from .data_loader import JSONTrainingDataGenerator
            generator = JSONTrainingDataGenerator()

            sample_file = "/tmp/sample_test_data.json"
            generator.generate_sample_training_data(sample_file, num_samples=10)

            # Load and process the sample data
            test_data = self.data_loader.load_from_json(sample_file)

            # Run validation
            validation_report = self.validate_baseline_reproduction(test_data)

            # Test ranking dataset creation
            X, y, groups = self.data_loader.create_ranking_dataset(test_data)

            test_results = {
                'success': True,
                'sample_samples': len(test_data),
                'feature_matrix_shape': X.shape,
                'risk_score_range': (float(np.min(y)), float(np.max(y))),
                'num_groups': len(groups),
                'validation_assessment': validation_report.get('assessment', 'UNKNOWN')
            }

            logger.info(f"Feature extraction pipeline test complete: {test_results}")

            # Cleanup
            import os
            if os.path.exists(sample_file):
                os.remove(sample_file)

            return test_results

        except Exception as e:
            logger.error(f"Feature extraction pipeline test failed: {e}")
            return {'success': False, 'error': str(e)}
