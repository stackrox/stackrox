"""
Test to verify feature consistency across deployments with and without images.
"""

import pytest
from src.feature_extraction.baseline_features import BaselineFeatureExtractor


def test_features_consistent_with_and_without_images():
    """
    Test that deployments with and without images have the same feature keys.
    This prevents KeyError when creating numpy arrays for ML training.
    """
    extractor = BaselineFeatureExtractor()

    # Deployment 1: Has images
    deployment_with_images = {
        'id': 'deployment-with-images',
        'name': 'app-with-images',
        'namespace': 'default'
    }

    images_data = [{
        'scan': {
            'components': [
                {'vulns': [{'cvss': 7.5, 'severity': 'IMPORTANT_VULNERABILITY_SEVERITY'}]}
            ]
        },
        'metadata': {
            'v1': {
                'created': '2020-01-01T00:00:00Z'
            }
        }
    }]

    result_with_images = extractor.create_training_sample(
        deployment_data=deployment_with_images,
        image_data_list=images_data,
        alert_data=[],
        baseline_violations=[],
        risk_score=50.0
    )

    # Deployment 2: No images
    deployment_without_images = {
        'id': 'deployment-without-images',
        'name': 'app-without-images',
        'namespace': 'default'
    }

    result_without_images = extractor.create_training_sample(
        deployment_data=deployment_without_images,
        image_data_list=[],  # Empty list - no images
        alert_data=[],
        baseline_violations=[],
        risk_score=10.0
    )

    # Verify both have features
    assert 'features' in result_with_images
    assert 'features' in result_without_images

    features_with_images = result_with_images['features']
    features_without_images = result_without_images['features']

    # Verify both have same keys (critical for ML training)
    assert set(features_with_images.keys()) == set(features_without_images.keys()), \
        f"Feature keys don't match!\n" \
        f"With images: {sorted(features_with_images.keys())}\n" \
        f"Without images: {sorted(features_without_images.keys())}"

    # Verify image features exist in both
    expected_image_features = [
        'avg_vulnerability_score', 'max_vulnerability_score', 'sum_vulnerability_score',
        'avg_avg_cvss_score', 'max_avg_cvss_score', 'sum_avg_cvss_score',
        'avg_max_cvss_score', 'max_max_cvss_score', 'sum_max_cvss_score',
        'avg_component_count_score', 'max_component_count_score', 'sum_component_count_score',
        'avg_risky_component_ratio', 'max_risky_component_ratio', 'sum_risky_component_ratio',
        'avg_age_score', 'max_age_score', 'sum_age_score',
        'avg_is_cluster_local', 'max_is_cluster_local', 'sum_is_cluster_local',
        'avg_log_layer_count', 'max_log_layer_count', 'sum_log_layer_count'
    ]

    for feature in expected_image_features:
        assert feature in features_with_images, f"Missing {feature} in deployment with images"
        assert feature in features_without_images, f"Missing {feature} in deployment without images"

    # Verify that features without images have zero values for image features
    for feature in expected_image_features:
        assert features_without_images[feature] == 0.0, \
            f"{feature} should be 0.0 for deployment without images, got {features_without_images[feature]}"

    # Note: We don't test that features_with_images has non-zero values because
    # that depends on the mock data structure being complete, which is not the
    # focus of this test. The key test is that both have the same feature keys.


def test_multiple_deployments_same_feature_count():
    """
    Test that multiple deployments produce consistent feature counts.
    This simulates what happens during ML training.
    """
    extractor = BaselineFeatureExtractor()

    # Create 3 deployments: with images, without images, with images again
    deployments = [
        {
            'deployment': {'id': 'deploy-1', 'name': 'app-1', 'namespace': 'default'},
            'images': [{'scan': {'components': []}, 'metadata': {'v1': {'created': '2020-01-01T00:00:00Z'}}}],
            'risk_score': 30.0
        },
        {
            'deployment': {'id': 'deploy-2', 'name': 'app-2', 'namespace': 'default'},
            'images': [],  # No images
            'risk_score': 20.0
        },
        {
            'deployment': {'id': 'deploy-3', 'name': 'app-3', 'namespace': 'default'},
            'images': [{'scan': {'components': []}, 'metadata': {'v1': {'created': '2021-01-01T00:00:00Z'}}}],
            'risk_score': 40.0
        }
    ]

    all_features = []
    for d in deployments:
        result = extractor.create_training_sample(
            deployment_data=d['deployment'],
            image_data_list=d['images'],
            alert_data=[],
            baseline_violations=[],
            risk_score=d['risk_score']
        )
        all_features.append(result['features'])

    # Verify all have the same number of features
    feature_counts = [len(f) for f in all_features]
    assert len(set(feature_counts)) == 1, \
        f"Feature counts should be consistent, got: {feature_counts}"

    # Verify all have the same keys
    keys_list = [set(f.keys()) for f in all_features]
    assert all(k == keys_list[0] for k in keys_list), \
        "All deployments should have identical feature keys"
