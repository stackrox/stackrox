"""
Unit tests for validation with top features output.
"""

import pytest
import numpy as np
from unittest.mock import Mock, MagicMock, patch
from src.services.central_export_service import CentralExportService


def test_validate_predictions_includes_top_features():
    """Test that validate_predictions includes top 5 features per deployment."""

    # Create mock client
    mock_client = Mock()

    # Create service
    service = CentralExportService(client=mock_client, config={})

    # Create mock model with predict method that returns feature importance
    mock_model = Mock()
    mock_prediction = Mock()
    mock_prediction.risk_score = 5.5
    mock_prediction.feature_importance = {
        'feature_1': 0.8,
        'feature_2': -0.6,
        'feature_3': 0.5,
        'feature_4': 0.3,
        'feature_5': -0.2,
        'feature_6': 0.1,
        'feature_7': 0.05
    }
    mock_model.predict.return_value = [mock_prediction]
    mock_model.feature_names = ['feature_1', 'feature_2', 'feature_3', 'feature_4', 'feature_5', 'feature_6', 'feature_7']

    # Create mock prediction client that streams one sample
    mock_prediction_client = Mock()

    # Create mock training sample
    mock_sample = {
        'features': {
            'feature_1': 0.5,
            'feature_2': 0.3,
            'feature_3': 0.2,
            'feature_4': 0.1,
            'feature_5': 0.05,
            'feature_6': 0.02,
            'feature_7': 0.01
        },
        'risk_score': 5.0,
        'workload_metadata': {
            'deployment_name': 'test-deployment',
            'namespace': 'test-namespace',
            'cluster_id': 'test-cluster'
        }
    }

    # Mock the streaming architecture to return one sample
    from src.streaming import SampleStream
    with patch.object(SampleStream, 'stream', return_value=[mock_sample]):
        with patch.object(service, 'close'):
            # Run validation
            results = service.validate_predictions(
                model=mock_model,
                prediction_client=mock_prediction_client,
                filters={},
                limit=1
            )

    # Verify results structure
    assert results['total_samples'] == 1
    assert results['successful_predictions'] == 1
    assert results['failed_predictions'] == 0
    assert len(results['predictions']) == 1

    # Verify prediction has top_features
    prediction = results['predictions'][0]
    assert 'top_features' in prediction
    assert len(prediction['top_features']) == 5

    # Verify top features are sorted by absolute importance
    top_features = prediction['top_features']
    assert top_features[0]['name'] == 'feature_1'  # |0.8| = 0.8
    assert top_features[0]['importance'] == 0.8
    assert top_features[1]['name'] == 'feature_2'  # |-0.6| = 0.6
    assert top_features[1]['importance'] == -0.6
    assert top_features[2]['name'] == 'feature_3'  # |0.5| = 0.5
    assert top_features[2]['importance'] == 0.5
    assert top_features[3]['name'] == 'feature_4'  # |0.3| = 0.3
    assert top_features[3]['importance'] == 0.3
    assert top_features[4]['name'] == 'feature_5'  # |-0.2| = 0.2
    assert top_features[4]['importance'] == -0.2

    # Verify other prediction fields
    assert prediction['deployment_name'] == 'test-deployment'
    assert prediction['namespace'] == 'test-namespace'
    assert prediction['cluster_id'] == 'test-cluster'
    assert prediction['actual_score'] == 5.0
    assert prediction['predicted_score'] == 5.5


def test_validate_predictions_handles_fewer_than_5_features():
    """Test that validate_predictions handles models with fewer than 5 features."""

    # Create mock client
    mock_client = Mock()

    # Create service
    service = CentralExportService(client=mock_client, config={})

    # Create mock model with only 3 features
    mock_model = Mock()
    mock_prediction = Mock()
    mock_prediction.risk_score = 3.0
    mock_prediction.feature_importance = {
        'feature_1': 0.5,
        'feature_2': 0.3,
        'feature_3': 0.2
    }
    mock_model.predict.return_value = [mock_prediction]
    mock_model.feature_names = ['feature_1', 'feature_2', 'feature_3']

    # Create mock prediction client
    mock_prediction_client = Mock()

    # Create mock training sample
    mock_sample = {
        'features': {
            'feature_1': 0.5,
            'feature_2': 0.3,
            'feature_3': 0.2
        },
        'risk_score': 3.0,
        'workload_metadata': {
            'deployment_name': 'test-deployment',
            'namespace': 'test-namespace',
            'cluster_id': 'test-cluster'
        }
    }

    # Mock the streaming architecture to return one sample
    from src.streaming import SampleStream
    with patch.object(SampleStream, 'stream', return_value=[mock_sample]):
        with patch.object(service, 'close'):
            # Run validation
            results = service.validate_predictions(
                model=mock_model,
                prediction_client=mock_prediction_client,
                filters={},
                limit=1
            )

    # Verify results
    assert results['successful_predictions'] == 1
    prediction = results['predictions'][0]

    # Should only have 3 features (not 5)
    assert 'top_features' in prediction
    assert len(prediction['top_features']) == 3
    assert prediction['top_features'][0]['name'] == 'feature_1'
    assert prediction['top_features'][1]['name'] == 'feature_2'
    assert prediction['top_features'][2]['name'] == 'feature_3'
