"""
Training data loader for ML risk ranking system.
Loads deployment and image data from JSON files or Central API.
"""

import json
import logging
from typing import Dict, Any, List, Optional, Tuple, Iterator
from pathlib import Path
import pandas as pd
import numpy as np
from dataclasses import asdict
from datetime import datetime, timedelta

from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logger = logging.getLogger(__name__)


class TrainingDataLoader:
    """Loads and processes training data for ML risk ranking."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}
        self.baseline_extractor = BaselineFeatureExtractor()

    def load_from_json(self, json_file_path: str) -> List[Dict[str, Any]]:
        """
        Load training data from JSON file.

        Expected JSON format:
        {
            "deployments": [
                {
                    "deployment": {...},  // Deployment protobuf as dict
                    "images": [...],      // List of image protobuf as dict
                    "alerts": [...],      // List of policy violation alerts
                    "baseline_violations": [...],  // Process baseline violations
                    "current_risk_score": 2.5     // Optional: existing risk score
                }
            ]
        }

        Args:
            json_file_path: Path to JSON training data file

        Returns:
            List of training examples
        """
        try:
            with open(json_file_path, 'r') as f:
                data = json.load(f)

            training_examples = []
            deployments = data.get('deployments', [])

            logger.info(f"Loading {len(deployments)} deployment examples from {json_file_path}")

            for i, deployment_record in enumerate(deployments):
                try:
                    example = self._process_deployment_record(deployment_record)
                    training_examples.append(example)

                    if (i + 1) % 100 == 0:
                        logger.info(f"Processed {i + 1} examples")

                except Exception as e:
                    logger.warning(f"Failed to process deployment record {i}: {e}")
                    continue

            logger.info(f"Successfully loaded {len(training_examples)} training examples")
            return training_examples

        except Exception as e:
            logger.error(f"Failed to load training data from {json_file_path}: {e}")
            raise

    def load_from_central_api(self, central_endpoint: str,
                            auth_token: str,
                            limit: int = 1000) -> List[Dict[str, Any]]:
        """
        Load training data from Central API.
        This would connect to Central's gRPC or REST API to fetch real deployment data.

        Args:
            central_endpoint: Central API endpoint
            auth_token: Authentication token
            limit: Maximum number of deployments to fetch

        Returns:
            List of training examples
        """
        # This is a placeholder - would implement actual Central API client
        logger.info(f"Loading training data from Central API: {central_endpoint}")

        # TODO: Implement Central API client
        # - Connect to Central's deployment service
        # - Fetch deployments with images and risk data
        # - Process through baseline feature extractor

        raise NotImplementedError("Central API integration not yet implemented")

    def _process_deployment_record(self, record: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process a single deployment record into a training example.

        Args:
            record: Raw deployment record from JSON

        Returns:
            Processed training example
        """
        deployment_data = record.get('deployment', {})
        images_data = record.get('images', [])
        alerts_data = record.get('alerts', [])
        baseline_violations = record.get('baseline_violations', [])
        existing_risk_score = record.get('current_risk_score')

        # Create training example using baseline feature extractor
        example = self.baseline_extractor.create_training_example(
            deployment_data=deployment_data,
            image_data_list=images_data,
            alert_data=alerts_data,
            baseline_violations=baseline_violations
        )

        # Add metadata
        example['deployment_id'] = deployment_data.get('id', '')
        example['deployment_name'] = deployment_data.get('name', '')
        example['namespace'] = deployment_data.get('namespace', '')
        example['cluster_id'] = deployment_data.get('cluster_id', '')

        # Use existing risk score if provided, otherwise use calculated baseline
        if existing_risk_score is not None:
            example['risk_score'] = existing_risk_score

        return example

    def create_ranking_dataset(self, training_examples: List[Dict[str, Any]]) -> Tuple[np.ndarray, np.ndarray, np.ndarray]:
        """
        Create ranking dataset for learning-to-rank algorithms.

        Args:
            training_examples: List of training examples

        Returns:
            Tuple of (X, y, groups) where:
            - X: Feature matrix
            - y: Risk scores
            - groups: Group assignments for ranking
        """
        if not training_examples:
            raise ValueError("No training examples provided")

        # Extract features and risk scores
        feature_names = None
        feature_vectors = []
        risk_scores = []
        groups = []

        # Group deployments by cluster for ranking
        cluster_groups = {}
        for i, example in enumerate(training_examples):
            cluster_id = example.get('cluster_id', 'unknown')
            if cluster_id not in cluster_groups:
                cluster_groups[cluster_id] = []
            cluster_groups[cluster_id].append(i)

        # Build feature matrix
        for example in training_examples:
            features = example['features']

            if feature_names is None:
                feature_names = sorted(features.keys())

            # Create feature vector in consistent order
            feature_vector = [features.get(name, 0.0) for name in feature_names]
            feature_vectors.append(feature_vector)
            risk_scores.append(example['risk_score'])

        # Create group assignments
        group_sizes = []
        for cluster_id in sorted(cluster_groups.keys()):
            group_sizes.append(len(cluster_groups[cluster_id]))

        X = np.array(feature_vectors, dtype=np.float32)
        y = np.array(risk_scores, dtype=np.float32)
        groups = np.array(group_sizes, dtype=np.int32)

        logger.info(f"Created ranking dataset: {X.shape[0]} examples, {X.shape[1]} features, {len(groups)} groups")

        return X, y, groups

    def save_processed_data(self, training_examples: List[Dict[str, Any]],
                          output_path: str) -> None:
        """
        Save processed training data to file.

        Args:
            training_examples: Processed training examples
            output_path: Path to save processed data
        """
        try:
            output_data = {
                'training_examples': training_examples,
                'metadata': {
                    'count': len(training_examples),
                    'feature_extractor': 'BaselineFeatureExtractor',
                    'timestamp': pd.Timestamp.now().isoformat()
                }
            }

            with open(output_path, 'w') as f:
                json.dump(output_data, f, indent=2, default=str)

            logger.info(f"Saved {len(training_examples)} training examples to {output_path}")

        except Exception as e:
            logger.error(f"Failed to save processed data to {output_path}: {e}")
            raise

    def load_processed_data(self, input_path: str) -> List[Dict[str, Any]]:
        """
        Load previously processed training data.

        Args:
            input_path: Path to processed data file

        Returns:
            List of training examples
        """
        try:
            with open(input_path, 'r') as f:
                data = json.load(f)

            training_examples = data.get('training_examples', [])
            metadata = data.get('metadata', {})

            logger.info(f"Loaded {len(training_examples)} training examples from {input_path}")
            logger.info(f"Data metadata: {metadata}")

            return training_examples

        except Exception as e:
            logger.error(f"Failed to load processed data from {input_path}: {e}")
            raise

    def validate_training_data(self, training_examples: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Validate training data quality and consistency.

        Args:
            training_examples: List of training examples

        Returns:
            Validation report
        """
        if not training_examples:
            return {'valid': False, 'error': 'No training examples provided'}

        validation_report = {
            'valid': True,
            'total_examples': len(training_examples),
            'feature_consistency': True,
            'risk_score_stats': {},
            'issues': []
        }

        # Check feature consistency
        feature_names = None
        risk_scores = []

        for i, example in enumerate(training_examples):
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
            validation_report['valid'] = len(validation_report['issues']) < len(training_examples) * 0.1  # Allow 10% errors

        logger.info(f"Validation complete: {validation_report}")
        return validation_report


class JSONTrainingDataGenerator:
    """
    Generates JSON training data files by reproducing current StackRox risk calculations.
    This is used to create initial training datasets.
    """

    def __init__(self):
        self.baseline_extractor = BaselineFeatureExtractor()

    def generate_sample_training_data(self, output_file: str, num_examples: int = 100) -> None:
        """
        Generate sample training data for testing.
        Creates synthetic deployment data with realistic risk patterns.

        Args:
            output_file: Path to output JSON file
            num_examples: Number of examples to generate
        """
        import random
        import uuid

        deployments = []

        for i in range(num_examples):
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
                'num_examples': num_examples,
                'generator': 'JSONTrainingDataGenerator'
            }
        }

        with open(output_file, 'w') as f:
            json.dump(training_data, f, indent=2, default=str)

        logger.info(f"Generated {num_examples} sample training examples in {output_file}")

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