"""
Test suite for synthetic deployment data generation and risk score predictions.

This test validates the end-to-end workflow:
1. Generate synthetic deployment data
2. Extract features from deployments
3. Train ML model
4. Predict risk scores
"""

import pytest
import numpy as np
import tempfile
import json
from pathlib import Path
from typing import Dict, Any, List

from src.models.ranking_model import RiskRankingModel, PredictionResult, ModelMetrics
from src.feature_extraction.baseline_features import BaselineFeatureExtractor, BaselineRiskFactors
from training.data_loader import JSONTrainingDataGenerator, TrainingDataLoader


@pytest.fixture
def synthetic_data_generator() -> JSONTrainingDataGenerator:
    """Provide a synthetic data generator instance."""
    return JSONTrainingDataGenerator()


@pytest.fixture
def feature_extractor() -> BaselineFeatureExtractor:
    """Provide a feature extractor instance."""
    return BaselineFeatureExtractor()


@pytest.fixture
def sample_deployments(synthetic_data_generator: JSONTrainingDataGenerator) -> List[Dict[str, Any]]:
    """Generate sample deployment data in-memory."""
    import random

    deployments = []
    for i in range(20):  # Generate 20 samples for training
        deployment_data = synthetic_data_generator._generate_sample_deployment(i)
        images_data = synthetic_data_generator._generate_sample_images(random.randint(1, 3))
        alerts_data = synthetic_data_generator._generate_sample_alerts(random.randint(0, 5))

        deployments.append({
            'deployment': deployment_data,
            'images': images_data,
            'alerts': alerts_data,
            'baseline_violations': []
        })

    return deployments


@pytest.fixture
def trained_model(sample_deployments: List[Dict[str, Any]],
                  feature_extractor: BaselineFeatureExtractor) -> RiskRankingModel:
    """Provide a pre-trained model on synthetic data."""
    # Extract features from sample deployments
    features_list = []
    scores_list = []

    for sample in sample_deployments:
        baseline_features = feature_extractor.extract_baseline_features(
            sample['deployment'],
            sample.get('images', []),
            sample.get('alerts', []),
            sample.get('baseline_violations', [])
        )

        # Extract score from features object
        baseline_score = baseline_features.overall_score

        # Convert BaselineRiskFactors to dict for model training
        features_dict = {
            'policy_violations': baseline_features.policy_violations_multiplier,
            'process_baseline': baseline_features.process_baseline_multiplier,
            'vulnerabilities': baseline_features.vulnerabilities_multiplier,
            'risky_components': baseline_features.risky_component_multiplier,
            'component_count': baseline_features.component_count_multiplier,
            'image_age': baseline_features.image_age_multiplier,
            'service_config': baseline_features.service_config_multiplier,
            'reachability': baseline_features.reachability_multiplier,
        }

        features_list.append(features_dict)
        scores_list.append(baseline_score)

    # Convert to numpy arrays
    feature_names = list(features_list[0].keys())
    X = np.array([[f[name] for name in feature_names] for f in features_list])
    y = np.array(scores_list)

    # Train model
    model = RiskRankingModel()
    model.train(X, y, feature_names=feature_names)

    return model


@pytest.mark.unit
def test_synthetic_data_generation_and_prediction(
    sample_deployments: List[Dict[str, Any]],
    feature_extractor: BaselineFeatureExtractor,
    trained_model: RiskRankingModel
) -> None:
    """
    Test end-to-end workflow: generate synthetic data, train model, and predict.
    """
    # Verify we have sample deployments
    assert len(sample_deployments) > 0, "Should generate at least one sample deployment"
    assert len(sample_deployments) == 20, "Should generate exactly 20 samples"

    # Extract features from a new test sample
    test_sample = sample_deployments[0]
    baseline_features = feature_extractor.extract_baseline_features(
        test_sample['deployment'],
        test_sample.get('images', []),
        test_sample.get('alerts', []),
        test_sample.get('baseline_violations', [])
    )

    # Convert to dict
    features = {
        'policy_violations': baseline_features.policy_violations_multiplier,
        'process_baseline': baseline_features.process_baseline_multiplier,
        'vulnerabilities': baseline_features.vulnerabilities_multiplier,
        'risky_components': baseline_features.risky_component_multiplier,
        'component_count': baseline_features.component_count_multiplier,
        'image_age': baseline_features.image_age_multiplier,
        'service_config': baseline_features.service_config_multiplier,
        'reachability': baseline_features.reachability_multiplier,
    }

    # Verify features were extracted
    assert features is not None, "Features should be extracted"
    assert len(features) > 0, "Should have extracted features"

    # Convert features to array for prediction
    feature_names = list(features.keys())
    X_test = np.array([[features[name] for name in feature_names]])

    # Make prediction
    predictions = trained_model.predict(X_test, explain=True)

    # Assertions
    assert len(predictions) == 1, "Should return one prediction"

    prediction = predictions[0]
    assert isinstance(prediction, PredictionResult), "Should return PredictionResult object"
    assert isinstance(prediction.risk_score, float), "Risk score should be a float"
    assert prediction.risk_score >= 0, "Risk score should be non-negative"
    assert isinstance(prediction.feature_importance, dict), "Feature importance should be a dictionary"
    assert len(prediction.feature_importance) > 0, "Should have feature importance values"
    assert prediction.model_version is not None, "Model version should be set"
    assert isinstance(prediction.confidence, float), "Confidence should be a float"


@pytest.mark.unit
def test_prediction_output_format(trained_model: RiskRankingModel) -> None:
    """
    Test that prediction output has the correct structure and data types.
    """
    # Create random test data matching the model's feature dimensions
    num_features = len(trained_model.feature_names)
    X_test = np.random.rand(3, num_features)

    # Make predictions
    predictions = trained_model.predict(X_test, explain=True)

    # Verify we get predictions for all samples
    assert len(predictions) == 3, "Should return predictions for all 3 samples"

    for i, prediction in enumerate(predictions):
        # Check type
        assert isinstance(prediction, PredictionResult), f"Prediction {i} should be PredictionResult"

        # Check risk_score
        assert isinstance(prediction.risk_score, float), f"Prediction {i} risk_score should be float"
        assert not np.isnan(prediction.risk_score), f"Prediction {i} risk_score should not be NaN"
        assert not np.isinf(prediction.risk_score), f"Prediction {i} risk_score should not be infinite"

        # Check feature_importance
        assert isinstance(prediction.feature_importance, dict), f"Prediction {i} feature_importance should be dict"
        assert len(prediction.feature_importance) > 0, f"Prediction {i} should have feature importance"

        # Verify all feature importance values are numeric
        for feature, importance in prediction.feature_importance.items():
            assert isinstance(feature, str), f"Feature name should be string"
            assert isinstance(importance, (int, float, np.number)), f"Importance value should be numeric"

        # Check model_version
        assert prediction.model_version is not None, f"Prediction {i} should have model_version"
        assert isinstance(prediction.model_version, str), f"Prediction {i} model_version should be string"

        # Check confidence
        assert isinstance(prediction.confidence, float), f"Prediction {i} confidence should be float"
        assert prediction.confidence >= 0, f"Prediction {i} confidence should be non-negative"


@pytest.mark.unit
def test_model_training_with_synthetic_data(
    sample_deployments: List[Dict[str, Any]],
    feature_extractor: BaselineFeatureExtractor
) -> None:
    """
    Test that model can be trained with synthetic data and metrics are computed.
    """
    # Extract features from sample deployments
    features_list = []
    scores_list = []

    for sample in sample_deployments:
        features = feature_extractor.extract_baseline_features(
            sample['deployment'],
            sample.get('images', []),
            sample.get('alerts', []),
            sample.get('baseline_violations', [])
        )

        # Extract score from features object
        baseline_score = features.overall_score

        # Convert BaselineRiskFactors to dict for model training
        features_dict = {
            'policy_violations': features.policy_violations_multiplier,
            'process_baseline': features.process_baseline_multiplier,
            'vulnerabilities': features.vulnerabilities_multiplier,
            'risky_components': features.risky_component_multiplier,
            'component_count': features.component_count_multiplier,
            'image_age': features.image_age_multiplier,
            'service_config': features.service_config_multiplier,
            'reachability': features.reachability_multiplier,
        }

        features_list.append(features_dict)
        scores_list.append(baseline_score)

    # Convert to numpy arrays
    feature_names = list(features_list[0].keys())
    X = np.array([[f[name] for name in feature_names] for f in features_list])
    y = np.array(scores_list)

    # Train model
    model = RiskRankingModel()
    metrics = model.train(X, y, feature_names=feature_names)

    # Verify training completed
    assert metrics is not None, "Training should return metrics"
    assert isinstance(metrics, ModelMetrics), "Should return ModelMetrics object"

    # Check that metrics have reasonable values
    assert isinstance(metrics.train_ndcg, float), "train_ndcg should be float"
    assert isinstance(metrics.val_ndcg, float), "val_ndcg should be float"
    assert metrics.train_ndcg >= 0, "train_ndcg should be non-negative"
    assert metrics.val_ndcg >= 0, "val_ndcg should be non-negative"

    # Verify model is trained
    assert model.model is not None, "Model should be trained"
    assert model.scaler is not None, "Scaler should be fitted"
    assert model.feature_names is not None, "Feature names should be set"
    assert model.model_version is not None, "Model version should be set"
    assert len(model.feature_names) == X.shape[1], "Feature names count should match features"


@pytest.mark.unit
def test_synthetic_data_structure(sample_deployments: List[Dict[str, Any]]) -> None:
    """
    Test that generated synthetic data has the expected structure.
    """
    assert len(sample_deployments) > 0, "Should have sample deployments"

    for i, sample in enumerate(sample_deployments):
        # Check required keys
        assert 'deployment' in sample, f"Sample {i} should have 'deployment' key"
        assert 'images' in sample, f"Sample {i} should have 'images' key"
        assert 'alerts' in sample, f"Sample {i} should have 'alerts' key"

        # Check deployment structure
        deployment = sample['deployment']
        assert isinstance(deployment, dict), f"Sample {i} deployment should be a dict"
        assert 'id' in deployment, f"Sample {i} deployment should have 'id'"
        assert 'name' in deployment, f"Sample {i} deployment should have 'name'"
        assert 'namespace' in deployment, f"Sample {i} deployment should have 'namespace'"

        # Check images structure
        images = sample['images']
        assert isinstance(images, list), f"Sample {i} images should be a list"

        # Check alerts structure
        alerts = sample['alerts']
        assert isinstance(alerts, list), f"Sample {i} alerts should be a list"


@pytest.mark.unit
def test_feature_extraction_from_synthetic_data(
    sample_deployments: List[Dict[str, Any]],
    feature_extractor: BaselineFeatureExtractor
) -> None:
    """
    Test that features can be extracted from synthetic deployment data.
    """
    for i, sample in enumerate(sample_deployments[:5]):  # Test first 5 samples
        baseline_features = feature_extractor.extract_baseline_features(
            sample['deployment'],
            sample.get('images', []),
            sample.get('alerts', []),
            sample.get('baseline_violations', [])
        )

        # Convert to dict for validation
        features = {
            'policy_violations': baseline_features.policy_violations_multiplier,
            'process_baseline': baseline_features.process_baseline_multiplier,
            'vulnerabilities': baseline_features.vulnerabilities_multiplier,
            'risky_components': baseline_features.risky_component_multiplier,
            'component_count': baseline_features.component_count_multiplier,
            'image_age': baseline_features.image_age_multiplier,
            'service_config': baseline_features.service_config_multiplier,
            'reachability': baseline_features.reachability_multiplier,
        }

        assert features is not None, f"Sample {i} should extract features"
        assert isinstance(features, dict), f"Sample {i} features should be a dict"
        assert len(features) > 0, f"Sample {i} should have at least one feature"

        # Verify all feature values are numeric
        for feature_name, feature_value in features.items():
            assert isinstance(feature_name, str), f"Sample {i} feature name should be string"
            assert isinstance(feature_value, (int, float, np.number)), \
                f"Sample {i} feature '{feature_name}' value should be numeric, got {type(feature_value)}"
            assert not np.isnan(feature_value), \
                f"Sample {i} feature '{feature_name}' should not be NaN"
            assert not np.isinf(feature_value), \
                f"Sample {i} feature '{feature_name}' should not be infinite"


@pytest.mark.unit
def test_predictions_consistency(trained_model: RiskRankingModel) -> None:
    """
    Test that predictions are consistent for the same input.
    """
    # Create test data
    num_features = len(trained_model.feature_names)
    X_test = np.random.rand(1, num_features)

    # Make predictions twice
    predictions1 = trained_model.predict(X_test, explain=False)
    predictions2 = trained_model.predict(X_test, explain=False)

    # Verify predictions are the same
    assert len(predictions1) == len(predictions2), "Should return same number of predictions"
    assert predictions1[0].risk_score == predictions2[0].risk_score, \
        "Predictions should be deterministic for same input"


@pytest.mark.unit
def test_batch_predictions(trained_model: RiskRankingModel) -> None:
    """
    Test that model can handle batch predictions.
    """
    # Create batch of test data
    num_features = len(trained_model.feature_names)
    batch_sizes = [1, 5, 10, 50]

    for batch_size in batch_sizes:
        X_test = np.random.rand(batch_size, num_features)
        predictions = trained_model.predict(X_test, explain=True)

        assert len(predictions) == batch_size, \
            f"Should return {batch_size} predictions for batch size {batch_size}"

        for i, prediction in enumerate(predictions):
            assert isinstance(prediction, PredictionResult), \
                f"Batch {batch_size}, prediction {i} should be PredictionResult"
            assert isinstance(prediction.risk_score, float), \
                f"Batch {batch_size}, prediction {i} risk_score should be float"
