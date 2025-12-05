"""
Training data loader for ML risk ranking system.
Uses the unified streaming architecture to load data from any source.
"""

import logging
import json
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional, Tuple, Iterator
import pandas as pd
import numpy as np

from src.streaming import SampleStreamSource, SampleStream
from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logger = logging.getLogger(__name__)


class TrainingDataLoader:
    """
    Loads and processes training data using unified streaming architecture.

    This class now uses SampleStream + SampleStreamSource for all data loading,
    eliminating code duplication across different sources.
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}
        self.baseline_extractor = BaselineFeatureExtractor()

    def stream_from_source(self,
                          source: SampleStreamSource,
                          filters: Optional[Dict[str, Any]] = None,
                          limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream training samples from any data source.

        This is the new unified method that works with any SampleStreamSource
        (Central API, JSON files, etc.).

        Args:
            source: Data source to stream from (CentralStreamSource, JSONFileStreamSource, etc.)
            filters: Optional filtering criteria (source-specific)
            limit: Maximum number of samples to yield

        Yields:
            Training samples ready for model training

        Example:
            # Stream from Central API
            from src.streaming import CentralStreamSource
            from src.config.central_config import create_central_client_from_config

            client = create_central_client_from_config()
            source = CentralStreamSource(client, config)
            loader = TrainingDataLoader()

            for sample in loader.stream_from_source(source, filters={'clusters': ['prod']}, limit=1000):
                # Process sample
                pass

            # Stream from JSON file
            from src.streaming import JSONFileStreamSource

            source = JSONFileStreamSource('training_data.json')
            for sample in loader.stream_from_source(source, limit=500):
                # Process sample
                pass
        """
        logger.info(f"Streaming samples from {source.__class__.__name__}")

        # Create sample stream with our feature extractor
        sample_stream = SampleStream(source, self.baseline_extractor, self.config)

        # Stream and yield samples
        samples_yielded = 0
        for sample in sample_stream.stream(filters, limit):
            yield sample
            samples_yielded += 1

        logger.info(f"Completed streaming: {samples_yielded} samples from {source.__class__.__name__}")

    def create_ranking_dataset(self, training_samples: List[Dict[str, Any]]) -> Tuple[np.ndarray, np.ndarray, np.ndarray]:
        """
        Create ranking dataset for learning-to-rank algorithms.

        Args:
            training_samples: List of training samples

        Returns:
            Tuple of (X, y, groups) where:
            - X: Feature matrix
            - y: Risk scores
            - groups: Group assignments for ranking
        """
        if not training_samples:
            raise ValueError("No training samples provided")

        # Extract features and risk scores
        feature_names = None
        feature_vectors = []
        risk_scores = []
        groups = []

        # Group deployments by cluster for ranking
        cluster_groups = {}
        for i, sample in enumerate(training_samples):
            cluster_id = sample.get('cluster_id', 'unknown')
            if cluster_id not in cluster_groups:
                cluster_groups[cluster_id] = []
            cluster_groups[cluster_id].append(i)

        # Build feature matrix
        user_adjusted_count = 0
        for sample in training_samples:
            features = sample['features']

            # Track user adjustments
            if sample.get('has_user_adjustment', False):
                user_adjusted_count += 1

            if feature_names is None:
                feature_names = sorted(features.keys())

            # Create feature vector in consistent order
            feature_vector = [features.get(name, 0.0) for name in feature_names]
            feature_vectors.append(feature_vector)
            risk_scores.append(sample['risk_score'])

        # Create group assignments
        group_sizes = []
        for cluster_id in sorted(cluster_groups.keys()):
            group_sizes.append(len(cluster_groups[cluster_id]))

        X = np.array(feature_vectors, dtype=np.float32)
        y_float = np.array(risk_scores, dtype=np.float32)
        groups = np.array(group_sizes, dtype=np.int32)

        logger.info(f"Created ranking dataset: {X.shape[0]} samples, {X.shape[1]} features, {len(groups)} groups")
        logger.info(f"Risk score range: {y_float.min():.6f} - {y_float.max():.6f}, unique values: {len(np.unique(y_float))}")
        logger.info(f"Training data sources: {user_adjusted_count} user-adjusted ({100.0*user_adjusted_count/X.shape[0]:.1f}%), "
                   f"{X.shape[0] - user_adjusted_count} ML scores ({100.0*(X.shape[0] - user_adjusted_count)/X.shape[0]:.1f}%)")

        # Return float scores - ranking transformation will be done in the model after data splitting
        return X, y_float, groups

    def save_processed_data(self, training_samples: List[Dict[str, Any]],
                          output_path: str) -> None:
        """
        Save processed training data to file.

        Args:
            training_samples: Processed training samples
            output_path: Path to save processed data
        """
        try:
            output_data = {
                'training_samples': training_samples,
                'metadata': {
                    'count': len(training_samples),
                    'feature_extractor': 'BaselineFeatureExtractor',
                    'timestamp': pd.Timestamp.now().isoformat()
                }
            }

            with open(output_path, 'w') as f:
                json.dump(output_data, f, indent=2, default=str)

            logger.info(f"Saved {len(training_samples)} training samples to {output_path}")

        except Exception as e:
            logger.error(f"Failed to save processed data to {output_path}: {e}")
            raise

    def load_processed_data(self, input_path: str) -> List[Dict[str, Any]]:
        """
        Load previously processed training data.

        Args:
            input_path: Path to processed data file

        Returns:
            List of training samples
        """
        try:
            with open(input_path, 'r') as f:
                data = json.load(f)

            training_samples = data.get('training_samples', [])
            metadata = data.get('metadata', {})

            logger.info(f"Loaded {len(training_samples)} training samples from {input_path}")
            logger.info(f"Data metadata: {metadata}")

            return training_samples

        except Exception as e:
            logger.error(f"Failed to load processed data from {input_path}: {e}")
            raise

    def validate_training_data(self, training_samples: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Validate training data quality and consistency.

        Args:
            training_samples: List of training samples

        Returns:
            Validation report
        """
        if not training_samples:
            return {'valid': False, 'error': 'No training samples provided'}

        validation_report = {
            'valid': True,
            'total_samples': len(training_samples),
            'feature_consistency': True,
            'risk_score_stats': {},
            'issues': []
        }

        # Check feature consistency
        feature_names = None
        risk_scores = []

        for i, example in enumerate(training_samples):
            # Check required fields
            if 'features' not in example:
                validation_report['issues'].append(f"Example {i}: Missing 'features' field")
                continue

            if 'risk_score' not in example:
                validation_report['issues'].append(f"Example {i}: Missing 'risk_score' field")
                continue

            # Check feature names consistency
            current_features = set(example['features'].keys())
            if feature_names is None:
                feature_names = current_features
            elif current_features != feature_names:
                validation_report['feature_consistency'] = False
                validation_report['issues'].append(f"Example {i}: Inconsistent feature names")

            # Collect risk scores
            risk_score = example['risk_score']
            if isinstance(risk_score, (int, float)) and risk_score > 0:
                risk_scores.append(risk_score)
            else:
                validation_report['issues'].append(f"Example {i}: Invalid risk score: {risk_score}")

        # Risk score statistics
        if risk_scores:
            validation_report['risk_score_stats'] = {
                'min': min(risk_scores),
                'max': max(risk_scores),
                'mean': np.mean(risk_scores),
                'std': np.std(risk_scores),
                'count': len(risk_scores)
            }

        # Overall validation
        if validation_report['issues']:
            validation_report['valid'] = len(validation_report['issues']) < len(training_samples) * 0.1  # Allow 10% errors

        logger.info(f"Validation complete: {validation_report}")
        return validation_report


class JSONTrainingDataGenerator:
    """
    Generates JSON training data files by reproducing current StackRox risk calculations.
    This is used to create initial training datasets.
    """

    def __init__(self):
        self.baseline_extractor = BaselineFeatureExtractor()

    def generate_sample_data(self, n_samples: int = 100) -> List[Dict[str, Any]]:
        """
        Generate sample training data and return it as a list.

        Args:
            n_samples: Number of samples to generate

        Returns:
            List of deployment dictionaries with deployment, images, and alerts data
        """
        import random

        deployments = []

        for i in range(n_samples):
            deployment_data = self._generate_sample_deployment(i)
            images_data = self._generate_sample_images(random.randint(1, 3))
            alerts_data = self._generate_sample_alerts(random.randint(0, 5))

            deployments.append({
                'deployment': deployment_data,
                'images': images_data,
                'alerts': alerts_data,
                'baseline_violations': []
            })

        return deployments

    def generate_sample_training_data(self, output_file: str, num_samples: int = 100) -> None:
        """
        Generate sample training data for testing.
        Creates synthetic deployment data with realistic risk patterns.

        Args:
            output_file: Path to output JSON file
            num_samples: Number of samples to generate
        """
        import random
        import uuid

        deployments = []

        for i in range(num_samples):
            deployment_data = self._generate_sample_deployment(i)
            images_data = self._generate_sample_images(random.randint(1, 3))
            alerts_data = self._generate_sample_alerts(random.randint(0, 5))

            deployments.append({
                'deployment': deployment_data,
                'images': images_data,
                'alerts': alerts_data,
                'baseline_violations': []
            })

        training_data = {
            'deployments': deployments,
            'metadata': {
                'generated_at': datetime.now().isoformat(),
                'num_samples': num_samples,
                'generator': 'JSONTrainingDataGenerator'
            }
        }

        with open(output_file, 'w') as f:
            json.dump(training_data, f, indent=2, default=str)

        logger.info(f"Generated {num_samples} sample training samples in {output_file}")

    def _generate_sample_deployment(self, index: int) -> Dict[str, Any]:
        """Generate sample deployment data."""
        import random
        import uuid

        return {
            'id': str(uuid.uuid4()),
            'name': f'sample-deployment-{index}',
            'namespace': random.choice(['default', 'kube-system', 'monitoring', 'app-namespace']),
            'cluster_id': str(uuid.uuid4()),
            'replicas': random.randint(1, 10),
            'host_network': random.choice([True, False]) if random.random() < 0.1 else False,
            'host_pid': random.choice([True, False]) if random.random() < 0.05 else False,
            'host_ipc': random.choice([True, False]) if random.random() < 0.05 else False,
            'automount_service_account_token': random.choice([True, False]),
            'orchestrator_component': random.choice([True, False]) if random.random() < 0.05 else False,
            'platform_component': random.choice([True, False]) if random.random() < 0.03 else False,
            'created': {
                'seconds': int((datetime.now() - timedelta(days=random.randint(0, 1000))).timestamp())
            },
            'containers': self._generate_sample_containers(random.randint(1, 4)),
            'ports': self._generate_sample_ports(random.randint(0, 3))
        }

    def _generate_sample_containers(self, count: int) -> List[Dict[str, Any]]:
        """Generate sample container data."""
        import random

        containers = []
        for i in range(count):
            containers.append({
                'id': f'container-{i}',
                'name': f'container-{i}',
                'security_context': {
                    'privileged': random.choice([True, False]) if random.random() < 0.1 else False,
                    'read_only_root_filesystem': random.choice([True, False]),
                    'run_as_non_root': random.choice([True, False]),
                }
            })
        return containers

    def _generate_sample_ports(self, count: int) -> List[Dict[str, Any]]:
        """Generate sample port configurations."""
        import random

        ports = []
        for i in range(count):
            ports.append({
                'container_port': random.randint(8000, 9000),
                'protocol': 'TCP',
                'exposure': random.choice(['INTERNAL', 'EXTERNAL', 'NODE']) if random.random() < 0.3 else 'INTERNAL'
            })
        return ports

    def _generate_sample_images(self, count: int) -> List[Dict[str, Any]]:
        """Generate sample image data."""
        import random

        images = []
        for i in range(count):
            vuln_count = random.randint(0, 50)
            images.append({
                'id': f'image-id-{i}',
                'name': {
                    'registry': 'docker.io',
                    'remote': f'sample/image-{i}',
                    'tag': 'latest'
                },
                'metadata': {
                    'layerShas': [f'sha256:layer{j}' for j in range(random.randint(5, 20))],
                    'created': {
                        'seconds': int((datetime.now() - timedelta(days=random.randint(0, 500))).timestamp())
                    }
                },
                'scan': {
                    'components': self._generate_sample_components(random.randint(10, 200), vuln_count)
                },
                'cluster_local': random.choice([True, False]) if random.random() < 0.1 else False
            })
        return images

    def _generate_sample_components(self, comp_count: int, vuln_count: int) -> List[Dict[str, Any]]:
        """Generate sample component data with vulnerabilities."""
        import random

        components = []
        remaining_vulns = vuln_count

        for i in range(comp_count):
            component_vulns = random.randint(0, min(remaining_vulns, 5)) if remaining_vulns > 0 else 0
            remaining_vulns -= component_vulns

            vulns = []
            for j in range(component_vulns):
                severity = random.choices(
                    ['CRITICAL_SEVERITY', 'HIGH_SEVERITY', 'MEDIUM_SEVERITY', 'LOW_SEVERITY'],
                    weights=[0.1, 0.2, 0.4, 0.3]
                )[0]

                vulns.append({
                    'cve': f'CVE-2023-{random.randint(1000, 9999)}',
                    'severity': severity,
                    'cvss': random.uniform(1.0, 10.0)
                })

            components.append({
                'name': f'component-{i}',
                'version': f'1.{random.randint(0, 10)}.{random.randint(0, 10)}',
                'vulns': vulns
            })

        return components

    def _generate_sample_alerts(self, count: int) -> List[Dict[str, Any]]:
        """Generate sample policy violation alerts."""
        import random

        alerts = []
        for i in range(count):
            severity = random.choices(
                ['CRITICAL_SEVERITY', 'HIGH_SEVERITY', 'MEDIUM_SEVERITY', 'LOW_SEVERITY'],
                weights=[0.1, 0.2, 0.4, 0.3]
            )[0]

            alerts.append({
                'id': f'alert-{i}',
                'policy': {
                    'id': f'policy-{random.randint(1, 20)}',
                    'name': f'Sample Policy {i}',
                    'severity': severity
                },
                'violation_state': 'ACTIVE'
            })

        return alerts
