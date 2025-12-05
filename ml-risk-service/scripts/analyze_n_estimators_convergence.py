#!/usr/bin/env python3
"""
Analyze how n_estimators affects model quality (NDCG) and training time.

This script trains RandomForest models with varying n_estimators values to:
1. Identify the optimal number of trees for convergence
2. Understand the tradeoff between model quality and training time
3. Generate visualizations showing NDCG and time vs n_estimators

Usage:
    python scripts/analyze_n_estimators_convergence.py [--data-source central|synthetic] [--limit 1000]
"""

import argparse
import json
import logging
import sys
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Tuple, Any
import yaml

import numpy as np
import matplotlib
matplotlib.use('Agg')  # Non-interactive backend
import matplotlib.pyplot as plt

# Add project root to path
project_root = Path(__file__).parent.parent
sys.path.insert(0, str(project_root))

from src.models.ranking_model import RiskRankingModel
from src.config.central_config import create_central_client_from_config
from src.streaming import CentralStreamSource, SampleStream
from src.training.data_loader import JSONTrainingDataGenerator

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class NEstimatorsAnalyzer:
    """Analyzes RandomForest convergence as n_estimators increases."""

    def __init__(self, config_path: str = None):
        """Initialize analyzer with configuration."""
        self.config_path = config_path or str(project_root / "src" / "config" / "feature_config.yaml")
        self.config = self._load_config()
        self.results = []

    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from YAML."""
        with open(self.config_path, 'r') as f:
            return yaml.safe_load(f)

    def load_training_data_from_central(self, limit: int = 1000) -> Tuple[np.ndarray, np.ndarray, List[str]]:
        """
        Load training data from Central API.

        Args:
            limit: Maximum number of samples to load

        Returns:
            X: Feature matrix (n_samples, n_features)
            y: Risk scores (n_samples,)
            feature_names: List of feature names
        """
        logger.info(f"Loading training data from Central API (limit={limit})")

        # Initialize Central client and stream source using new architecture
        client = create_central_client_from_config(self.config_path)
        source = CentralStreamSource(client, self.config)
        sample_stream = SampleStream(source, config=self.config)

        # Collect training data from Central using streaming
        logger.info("Streaming training data from Central...")
        deployments = list(sample_stream.stream(filters=None, limit=limit))

        if not deployments:
            raise ValueError("No training data collected from Central API")

        logger.info(f"Collected {len(deployments)} training samples from Central")

        # Extract features from pre-processed training samples
        X_list = []
        y_list = []
        feature_names = None

        for sample in deployments:
            try:
                # Samples are already processed with features and risk_score
                if feature_names is None:
                    feature_names = list(sample['features'].keys())

                # Convert features to array
                feature_vector = [sample['features'][name] for name in feature_names]
                X_list.append(feature_vector)
                y_list.append(sample['risk_score'])

            except Exception as e:
                logger.warning(f"Failed to extract features from sample: {e}")
                continue

        if not X_list:
            raise ValueError("No valid training samples extracted")

        X = np.array(X_list)
        y = np.array(y_list)

        logger.info(f"Extracted {X.shape[0]} training samples with {X.shape[1]} features")
        logger.info(f"Risk score range: [{y.min():.2f}, {y.max():.2f}]")

        return X, y, feature_names

    def load_training_data_synthetic(self, n_samples: int = 1000) -> Tuple[np.ndarray, np.ndarray, List[str]]:
        """
        Generate synthetic training data.

        Args:
            n_samples: Number of synthetic samples to generate

        Returns:
            X: Feature matrix (n_samples, n_features)
            y: Risk scores (n_samples,)
            feature_names: List of feature names
        """
        logger.info(f"Generating {n_samples} synthetic training samples")

        generator = JSONTrainingDataGenerator()
        data = generator.generate_sample_data(n_samples)

        # Extract features
        extractor = BaselineFeatureExtractor()
        X_list = []
        y_list = []
        feature_names = None

        for item in data:
            try:
                sample = extractor.create_training_sample(
                    deployment_data=item['deployment'],
                    image_data_list=item.get('images', []),
                    alert_data=item.get('alerts', []),
                    baseline_violations=[]
                )

                if feature_names is None:
                    feature_names = list(sample['features'].keys())

                feature_vector = [sample['features'][name] for name in feature_names]
                X_list.append(feature_vector)
                y_list.append(sample['risk_score'])

            except Exception as e:
                logger.warning(f"Failed to extract features: {e}")
                continue

        X = np.array(X_list)
        y = np.array(y_list)

        logger.info(f"Generated {X.shape[0]} samples with {X.shape[1]} features")

        return X, y, feature_names

    def analyze_convergence(
        self,
        X: np.ndarray,
        y: np.ndarray,
        feature_names: List[str],
        n_estimators_values: List[int] = None
    ) -> List[Dict[str, Any]]:
        """
        Analyze model convergence for different n_estimators values.

        Args:
            X: Feature matrix
            y: Risk scores
            feature_names: Feature names
            n_estimators_values: List of n_estimators to test

        Returns:
            List of result dictionaries
        """
        if n_estimators_values is None:
            n_estimators_values = [10, 25, 50, 100, 200, 500, 1000, 1500, 2000]

        logger.info(f"Analyzing convergence for n_estimators: {n_estimators_values}")

        results = []

        for n_est in n_estimators_values:
            logger.info(f"\n{'='*60}")
            logger.info(f"Training with n_estimators={n_est}")
            logger.info(f"{'='*60}")

            # Create config with specific n_estimators
            config = self.config.copy()
            config['model']['sklearn_params']['n_estimators'] = n_est

            # Initialize model
            model = RiskRankingModel(config)

            # Train and measure time
            start_time = time.time()
            metrics = model.train(X, y, feature_names=feature_names)
            training_time = time.time() - start_time

            # Record results
            result = {
                'n_estimators': n_est,
                'train_ndcg': metrics.train_ndcg,
                'val_ndcg': metrics.val_ndcg,
                'training_time_seconds': training_time,
                'n_samples': X.shape[0],
                'n_features': X.shape[1]
            }

            results.append(result)

            logger.info(f"Results: Train NDCG={metrics.train_ndcg:.4f}, "
                       f"Val NDCG={metrics.val_ndcg:.4f}, "
                       f"Time={training_time:.2f}s")

        self.results = results
        return results

    def plot_convergence(self, output_path: str):
        """
        Generate convergence plots.

        Args:
            output_path: Path to save the plot
        """
        if not self.results:
            raise ValueError("No results to plot. Run analyze_convergence first.")

        # Extract data
        n_estimators = [r['n_estimators'] for r in self.results]
        train_ndcg = [r['train_ndcg'] for r in self.results]
        val_ndcg = [r['val_ndcg'] for r in self.results]
        training_time = [r['training_time_seconds'] for r in self.results]

        # Create figure with 3 subplots
        fig, (ax1, ax2, ax3) = plt.subplots(3, 1, figsize=(12, 10))

        # Plot 1: NDCG vs n_estimators
        ax1.plot(n_estimators, train_ndcg, 'o-', label='Train NDCG', linewidth=2, markersize=8)
        ax1.plot(n_estimators, val_ndcg, 's-', label='Validation NDCG', linewidth=2, markersize=8)
        ax1.set_xlabel('n_estimators (number of trees)', fontsize=12)
        ax1.set_ylabel('NDCG Score', fontsize=12)
        ax1.set_title('Model Quality vs n_estimators', fontsize=14, fontweight='bold')
        ax1.legend(fontsize=10)
        ax1.grid(True, alpha=0.3)
        ax1.set_xscale('log')

        # Add reference line for best validation NDCG
        best_val_idx = np.argmax(val_ndcg)
        best_n_est = n_estimators[best_val_idx]
        best_val = val_ndcg[best_val_idx]
        ax1.axhline(y=best_val, color='green', linestyle='--', alpha=0.5,
                   label=f'Best Val NDCG: {best_val:.4f} (n={best_n_est})')
        ax1.legend(fontsize=10)

        # Plot 2: Training time vs n_estimators
        ax2.plot(n_estimators, training_time, 'o-', color='orange', linewidth=2, markersize=8)
        ax2.set_xlabel('n_estimators (number of trees)', fontsize=12)
        ax2.set_ylabel('Training Time (seconds)', fontsize=12)
        ax2.set_title('Training Time vs n_estimators', fontsize=14, fontweight='bold')
        ax2.grid(True, alpha=0.3)
        ax2.set_xscale('log')

        # Plot 3: Combined view with dual y-axes
        ax3_left = ax3
        ax3_right = ax3.twinx()

        line1 = ax3_left.plot(n_estimators, val_ndcg, 's-', color='blue',
                              label='Validation NDCG', linewidth=2, markersize=8)
        line2 = ax3_right.plot(n_estimators, training_time, 'o-', color='orange',
                               label='Training Time', linewidth=2, markersize=8)

        ax3_left.set_xlabel('n_estimators (number of trees)', fontsize=12)
        ax3_left.set_ylabel('Validation NDCG', color='blue', fontsize=12)
        ax3_right.set_ylabel('Training Time (seconds)', color='orange', fontsize=12)
        ax3_left.set_title('Quality vs Time Tradeoff', fontsize=14, fontweight='bold')
        ax3_left.tick_params(axis='y', labelcolor='blue')
        ax3_right.tick_params(axis='y', labelcolor='orange')
        ax3_left.grid(True, alpha=0.3)
        ax3_left.set_xscale('log')

        # Combined legend
        lines = line1 + line2
        labels = [l.get_label() for l in lines]
        ax3_left.legend(lines, labels, loc='center right', fontsize=10)

        plt.tight_layout()
        plt.savefig(output_path, dpi=150, bbox_inches='tight')
        logger.info(f"Convergence plot saved to: {output_path}")

    def save_results(self, output_path: str):
        """
        Save results to JSON file.

        Args:
            output_path: Path to save the JSON results
        """
        if not self.results:
            raise ValueError("No results to save. Run analyze_convergence first.")

        output = {
            'timestamp': datetime.now().isoformat(),
            'config_path': self.config_path,
            'results': self.results,
            'summary': {
                'best_n_estimators': self.results[np.argmax([r['val_ndcg'] for r in self.results])]['n_estimators'],
                'best_val_ndcg': max(r['val_ndcg'] for r in self.results),
                'fastest_training_time': min(r['training_time_seconds'] for r in self.results),
                'slowest_training_time': max(r['training_time_seconds'] for r in self.results)
            }
        }

        with open(output_path, 'w') as f:
            json.dump(output, f, indent=2)

        logger.info(f"Results saved to: {output_path}")

    def print_summary(self):
        """Print summary of results."""
        if not self.results:
            logger.warning("No results to summarize")
            return

        print("\n" + "="*70)
        print("N_ESTIMATORS CONVERGENCE ANALYSIS SUMMARY")
        print("="*70)
        print(f"{'n_estimators':<15} {'Train NDCG':<12} {'Val NDCG':<12} {'Time (s)':<12}")
        print("-"*70)

        for r in self.results:
            print(f"{r['n_estimators']:<15} {r['train_ndcg']:<12.4f} {r['val_ndcg']:<12.4f} {r['training_time_seconds']:<12.2f}")

        print("="*70)

        # Best validation NDCG
        best_idx = np.argmax([r['val_ndcg'] for r in self.results])
        best = self.results[best_idx]
        print(f"\nBest Validation NDCG: {best['val_ndcg']:.4f} (n_estimators={best['n_estimators']})")

        # Diminishing returns analysis
        val_ndcgs = [r['val_ndcg'] for r in self.results]
        improvements = [(val_ndcgs[i] - val_ndcgs[i-1]) for i in range(1, len(val_ndcgs))]

        print(f"\nDiminishing Returns Analysis:")
        if improvements:
            print(f"  Largest improvement: {max(improvements):.4f}")
            print(f"  Last improvement: {improvements[-1]:.4f}")

            if improvements[-1] < 0.001:
                print(f"  => Model has converged (improvement < 0.001)")
        else:
            print(f"  Not enough data points to analyze diminishing returns (need at least 2)")

        print("="*70 + "\n")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description='Analyze n_estimators convergence')
    parser.add_argument(
        '--data-source',
        choices=['central', 'synthetic'],
        default='central',
        help='Data source: central API or synthetic data'
    )
    parser.add_argument(
        '--limit',
        type=int,
        default=1000,
        help='Maximum number of training samples'
    )
    parser.add_argument(
        '--output-dir',
        default=None,
        help='Output directory for results (optional - if not provided, only prints summary)'
    )
    parser.add_argument(
        '--n-estimators',
        type=int,
        nargs='+',
        default=[10, 50, 100, 500, 1000],
        help='List of n_estimators values to test'
    )

    args = parser.parse_args()

    # Initialize analyzer
    analyzer = NEstimatorsAnalyzer()

    try:
        # Load training data
        if args.data_source == 'central':
            X, y, feature_names = analyzer.load_training_data_from_central(limit=args.limit)
        else:
            X, y, feature_names = analyzer.load_training_data_synthetic(n_samples=args.limit)

        # Run convergence analysis
        results = analyzer.analyze_convergence(X, y, feature_names, n_estimators_values=args.n_estimators)

        # Generate outputs if output directory is specified
        if args.output_dir:
            output_dir = Path(args.output_dir)
            output_dir.mkdir(parents=True, exist_ok=True)

            plot_path = output_dir / 'n_estimators_convergence.png'
            json_path = output_dir / 'n_estimators_results.json'

            analyzer.plot_convergence(str(plot_path))
            analyzer.save_results(str(json_path))

            print(f"\nResults saved:")
            print(f"  Plot: {plot_path}")
            print(f"  Data: {json_path}")

        # Always print summary to stdout
        analyzer.print_summary()

    except Exception as e:
        logger.error(f"Analysis failed: {e}", exc_info=True)
        sys.exit(1)


if __name__ == '__main__':
    main()
