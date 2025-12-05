"""
Test service configuration features (volumes, secrets, capabilities).

Verifies that the ML service correctly extracts and scores service configuration
features to match Central's risk multipliers in central/risk/multipliers/deployment/config.go
"""

import pytest
from src.feature_extraction.deployment_features import DeploymentFeatureExtractor
from src.feature_extraction.baseline_features import BaselineFeatureExtractor


class TestVolumeExtraction:
    """Test volume mount extraction (RW vs RO)."""

    def test_read_write_volume_counted(self):
        """Test that read-write volumes are counted."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [
                        {'name': 'vol1', 'readOnly': False},  # RW - should count
                        {'name': 'vol2'}  # Default is RW - should count
                    ]
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.rw_volume_mount_count == 2

    def test_read_only_volume_not_counted(self):
        """Test that read-only volumes are not counted."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [
                        {'name': 'vol1', 'readOnly': True},  # RO - should NOT count
                        {'name': 'vol2', 'readOnly': False}  # RW - should count
                    ]
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.rw_volume_mount_count == 1

    def test_volumes_across_multiple_containers(self):
        """Test volume counting across multiple containers."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [
                        {'name': 'vol1', 'readOnly': False},
                        {'name': 'vol2', 'readOnly': True}
                    ]
                },
                {
                    'name': 'container-2',
                    'volumes': [
                        {'name': 'vol3'},  # Default RW
                        {'name': 'vol4', 'readOnly': False}
                    ]
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.rw_volume_mount_count == 3  # vol1, vol3, vol4

    def test_no_volumes(self):
        """Test deployment with no volumes."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {'name': 'container-1', 'volumes': []}
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.rw_volume_mount_count == 0


class TestSecretExtraction:
    """Test secret usage extraction."""

    def test_secrets_counted(self):
        """Test that secrets are counted."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'secrets': [
                        {'name': 'secret1'},
                        {'name': 'secret2'},
                        {'name': 'secret3'}
                    ]
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.secret_count == 3

    def test_secrets_across_multiple_containers(self):
        """Test secret counting across multiple containers."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'secrets': [{'name': 'secret1'}, {'name': 'secret2'}]
                },
                {
                    'name': 'container-2',
                    'secrets': [{'name': 'secret3'}]
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.secret_count == 3

    def test_no_secrets(self):
        """Test deployment with no secrets."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {'name': 'container-1', 'secrets': []}
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.secret_count == 0


class TestCapabilityExtraction:
    """Test capability extraction (risky adds, missing drops)."""

    def test_risky_capabilities_added(self):
        """Test that risky capabilities are counted."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'addCapabilities': ['SYS_ADMIN', 'NET_ADMIN']
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.risky_capabilities_added_count == 2

    def test_all_capability_is_risky(self):
        """Test that ALL capability is considered risky."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'addCapabilities': ['ALL']
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.risky_capabilities_added_count == 1

    def test_non_risky_capabilities_not_counted(self):
        """Test that non-risky capabilities are not counted."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'addCapabilities': ['NET_BIND_SERVICE', 'CHOWN']  # Not in risky list
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.risky_capabilities_added_count == 0

    def test_mixed_risky_and_safe_capabilities(self):
        """Test mixture of risky and safe capabilities."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'addCapabilities': [
                            'SYS_ADMIN',  # Risky
                            'NET_BIND_SERVICE',  # Safe
                            'SYS_MODULE',  # Risky
                            'CHOWN'  # Safe
                        ]
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.risky_capabilities_added_count == 2

    def test_no_capabilities_dropped_flag(self):
        """Test that no_capabilities_dropped flag is set when no caps are dropped."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'dropCapabilities': []  # Empty - risky
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.no_capabilities_dropped is True

    def test_capabilities_dropped_flag_false_when_any_dropped(self):
        """Test that flag is False when any container drops capabilities."""
        extractor = DeploymentFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.no_capabilities_dropped is False

    def test_no_capabilities_dropped_across_containers(self):
        """Test that flag checks all containers."""
        extractor = DeploymentFeatureExtractor()

        # If ANY container drops caps, flag should be False
        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'securityContext': {
                        'dropCapabilities': []  # No drops
                    }
                },
                {
                    'name': 'container-2',
                    'securityContext': {
                        'dropCapabilities': ['NET_RAW']  # Drops one
                    }
                }
            ]
        }

        features = extractor.extract_features(deployment_data)
        assert features.no_capabilities_dropped is False


class TestServiceConfigMultiplier:
    """Test service configuration multiplier calculation."""

    def test_no_risk_factors_returns_one(self):
        """Test that no risk factors returns 1.0 multiplier."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [],
                    'secrets': [],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': [],
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)
        assert multiplier == 1.0

    def test_rw_volumes_increase_multiplier(self):
        """Test that RW volumes increase multiplier."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [
                        {'name': 'vol1', 'readOnly': False}
                    ],
                    'secrets': [],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': [],
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)
        assert multiplier > 1.0

    def test_secrets_increase_multiplier(self):
        """Test that secrets increase multiplier."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [],
                    'secrets': [{'name': 'secret1'}],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': [],
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)
        assert multiplier > 1.0

    def test_risky_capabilities_increase_multiplier(self):
        """Test that risky capabilities increase multiplier."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [],
                    'secrets': [],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': ['SYS_ADMIN'],
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)
        assert multiplier > 1.0

    def test_no_capabilities_dropped_increases_multiplier(self):
        """Test that missing capability drops increase multiplier."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [],
                    'secrets': [],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': [],
                        'dropCapabilities': []  # No drops - risky
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)
        assert multiplier > 1.0

    def test_privileged_container_multiplies_score(self):
        """Test that privileged containers multiply the score by 2."""
        extractor = BaselineFeatureExtractor()

        # First get baseline score with one RW volume
        deployment_no_priv = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [{'name': 'vol1', 'readOnly': False}],
                    'secrets': [],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': [],
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        multiplier_no_priv = extractor._calculate_service_config_multiplier(deployment_no_priv)

        # Now with privileged container
        deployment_priv = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [{'name': 'vol1', 'readOnly': False}],
                    'secrets': [],
                    'securityContext': {
                        'privileged': True,  # Should multiply score by 2
                        'addCapabilities': [],
                        'dropCapabilities': ['ALL']
                    }
                }
            ]
        }

        multiplier_priv = extractor._calculate_service_config_multiplier(deployment_priv)

        # Privileged should result in higher multiplier
        assert multiplier_priv > multiplier_no_priv

    def test_multiplier_caps_at_two(self):
        """Test that multiplier is capped at 2.0 (configMaxScore from Central)."""
        extractor = BaselineFeatureExtractor()

        # Create deployment with many risk factors to exceed saturation
        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [
                        {'name': f'vol{i}', 'readOnly': False}
                        for i in range(20)  # Many volumes
                    ],
                    'secrets': [{'name': f'secret{i}'} for i in range(20)],  # Many secrets
                    'securityContext': {
                        'privileged': True,
                        'addCapabilities': ['ALL', 'SYS_ADMIN', 'NET_ADMIN', 'SYS_MODULE'],
                        'dropCapabilities': []
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)
        assert multiplier <= 2.0

    def test_combined_risk_factors(self):
        """Test realistic scenario with multiple risk factors."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [
                        {'name': 'vol1', 'readOnly': False},
                        {'name': 'vol2', 'readOnly': False}
                    ],
                    'secrets': [
                        {'name': 'secret1'},
                        {'name': 'secret2'}
                    ],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': ['SYS_ADMIN', 'NET_ADMIN'],
                        'dropCapabilities': []  # No drops
                    }
                }
            ]
        }

        multiplier = extractor._calculate_service_config_multiplier(deployment_data)

        # Should be > 1.0 (has risk factors) and <= 2.0 (capped)
        assert 1.0 < multiplier <= 2.0


class TestIntegration:
    """Integration tests for service config features."""

    def test_baseline_features_include_service_config(self):
        """Test that baseline features include service config multiplier."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [{'name': 'vol1', 'readOnly': False}],
                    'secrets': [{'name': 'secret1'}],
                    'securityContext': {
                        'privileged': False,
                        'addCapabilities': ['SYS_ADMIN'],
                        'dropCapabilities': []
                    }
                }
            ]
        }

        baseline_factors = extractor.extract_baseline_features(
            deployment_data=deployment_data,
            image_data_list=[],
            alert_data=[],
            baseline_violations=[]
        )

        # Verify service_config_multiplier is included
        assert hasattr(baseline_factors, 'service_config_multiplier')
        assert baseline_factors.service_config_multiplier > 1.0

        # Verify it contributes to overall score
        assert baseline_factors.overall_score > 1.0

    def test_training_sample_uses_service_config(self):
        """Test that training samples properly use service config features."""
        extractor = BaselineFeatureExtractor()

        deployment_data = {
            'id': 'test-deployment',
            'namespace': 'default',
            'containers': [
                {
                    'name': 'container-1',
                    'volumes': [{'name': 'vol1', 'readOnly': False}],
                    'secrets': [{'name': 'secret1'}],
                    'securityContext': {
                        'privileged': True,
                        'addCapabilities': ['ALL'],
                        'dropCapabilities': []
                    }
                }
            ]
        }

        # Use synthetic scoring (no risk_score parameter)
        sample = extractor.create_training_sample(
            deployment_data=deployment_data,
            image_data_list=[],
            alert_data=[],
            baseline_violations=[]
        )

        # Verify sample includes baseline_factors
        assert 'baseline_factors' in sample
        assert 'service_config' in sample['baseline_factors']

        # Service config should have contributed to risk score
        service_config_multiplier = sample['baseline_factors']['service_config']
        assert service_config_multiplier > 1.0
