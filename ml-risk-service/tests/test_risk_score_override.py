"""
Unit test to verify that providing risk_score skips baseline computation.
"""

import pytest
from src.feature_extraction.baseline_features import BaselineFeatureExtractor


def test_risk_score_override_skips_baseline():
    """
    Test that when risk_score is provided, baseline computation is skipped.
    """
    extractor = BaselineFeatureExtractor()

    # Sample deployment data
    deployment_data = {
        'id': 'test-deployment',
        'name': 'test-app',
        'namespace': 'default',
        'policy_violations': [
            {'policy': {'severity': 'HIGH_SEVERITY'}}
        ]
    }

    # Sample image data
    image_data_list = [{
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

    # Test 1: Provide risk_score - should skip baseline computation
    result_with_score = extractor.create_training_sample(
        deployment_data=deployment_data,
        image_data_list=image_data_list,
        alert_data=[],
        baseline_violations=[],
        risk_score=42.5  # Provide explicit risk score
    )

    # Verify risk_score was used directly
    assert result_with_score['risk_score'] == 42.5, "Should use provided risk_score"

    # Verify baseline_factors was NOT computed (should not be in result)
    assert 'baseline_factors' not in result_with_score, \
        "Should not include baseline_factors when risk_score is provided"

    # Verify features are still extracted
    assert 'features' in result_with_score, "Should still extract features"
    assert len(result_with_score['features']) > 0, "Should have extracted features"

    # Test 2: Don't provide risk_score - should compute baseline
    result_without_score = extractor.create_training_sample(
        deployment_data=deployment_data,
        image_data_list=image_data_list,
        alert_data=[],
        baseline_violations=[]
        # No risk_score parameter - should compute from baseline
    )

    # Verify risk_score was computed
    assert 'risk_score' in result_without_score, "Should have computed risk_score"
    assert result_without_score['risk_score'] > 0, "Computed risk_score should be > 0"

    # Verify baseline_factors WAS computed (should be in result)
    assert 'baseline_factors' in result_without_score, \
        "Should include baseline_factors when computing risk_score"

    # Verify features are still extracted
    assert 'features' in result_without_score, "Should still extract features"
    assert len(result_without_score['features']) > 0, "Should have extracted features"

    # Test 3: Verify feature extraction is same in both cases
    # (features should be independent of risk_score computation method)
    # Note: We can't directly compare dicts because baseline computation might affect features
    # through _ensure_risk_score_variance, but we can verify keys are similar
    assert set(result_with_score['features'].keys()) == set(result_without_score['features'].keys()), \
        "Feature keys should be the same regardless of risk_score source"


def test_risk_score_zero_is_valid():
    """
    Test that risk_score=0.0 is treated as a valid score (not None).
    """
    extractor = BaselineFeatureExtractor()

    deployment_data = {
        'id': 'safe-deployment',
        'name': 'safe-app',
        'namespace': 'default'
    }

    result = extractor.create_training_sample(
        deployment_data=deployment_data,
        image_data_list=[],
        alert_data=[],
        baseline_violations=[],
        risk_score=0.0  # Explicit zero score
    )

    # Verify zero score was used (not computed from baseline)
    assert result['risk_score'] == 0.0, "Should accept 0.0 as valid risk_score"
    assert 'baseline_factors' not in result, "Should not compute baseline for 0.0 score"
