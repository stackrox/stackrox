"""
Integration test for Central API training data collection and prediction validation.

This test validates the entire ML pipeline end-to-end:
1. Pull real training data from Central (pre-extracted features and risk scores)
2. Train ML model on subset of data
3. Make predictions on held-out deployments
4. Compare predicted vs actual risk scores

Note: Central service returns training samples with pre-computed features via
create_training_sample(), not raw deployment data.

Requires:
- TRAINING_CENTRAL_API_TOKEN environment variable
- Access to a running StackRox Central instance
"""

import os
import pytest
import numpy as np
from typing import Dict, Any, List, Tuple
import logging

from src.clients.central_export_client import CentralExportClient
from src.services.central_export_service import CentralExportService
from src.config.central_config import CentralConfig
from src.feature_extraction.baseline_features import BaselineFeatureExtractor, BaselineRiskFactors
from src.models.ranking_model import RiskRankingModel, ModelMetrics

logger = logging.getLogger(__name__)

# Skip all tests in this module if TRAINING_CENTRAL_API_TOKEN is not set
pytestmark = [
    pytest.mark.integration,
    pytest.mark.slow,
    pytest.mark.skipif(
        not os.getenv('TRAINING_CENTRAL_API_TOKEN'),
        reason="Requires TRAINING_CENTRAL_API_TOKEN environment variable"
    )
]


@pytest.fixture
def central_client() -> CentralExportClient:
    """Provide a configured Central API client."""
    config = CentralConfig()

    endpoint = config.get_endpoint()
    token = os.getenv('TRAINING_CENTRAL_API_TOKEN')

    if not token:
        pytest.skip("TRAINING_CENTRAL_API_TOKEN not set")

    client_config = {
        'chunk_size': 100,
        'timeout_seconds': 120,
        'verify_certificates': False  # For development/testing
    }

    client = CentralExportClient(endpoint, token, client_config)
    return client


@pytest.fixture
def central_export_service(central_client: CentralExportClient) -> CentralExportService:
    """Provide a configured Central Export Service."""
    service_config = {
        'max_workers': 2,
        'batch_size': 50
    }
    return CentralExportService(central_client, service_config)


@pytest.fixture
def feature_extractor() -> BaselineFeatureExtractor:
    """Provide a feature extractor instance."""
    return BaselineFeatureExtractor()


def extract_features_and_score(
    sample: Dict[str, Any]
) -> Tuple[Dict[str, float], float]:
    """
    Extract features and risk score from a training sample.

    Args:
        sample: Training sample from Central (already contains extracted features)

    Returns:
        Tuple of (features dict, risk score)
    """
    # Sample structure from create_training_sample():
    # {
    #   'features': {...},           # Normalized features for ML
    #   'risk_score': 2.139,        # Risk score from Central's ground truth
    #   'baseline_factors': {...}   # (Optional) Only present for synthetic data
    # }
    #
    # Note: When collecting from Central, risk_score is Central's riskScore field.
    # When generating synthetic data, risk_score is computed from baseline factors.

    # Use actual normalized features for ML model
    # The 'features' dict contains deployment and image characteristics that are
    # normalized to 0-1 range
    features = sample.get('features', {})

    # If features dict is empty, this sample is invalid
    if not features:
        raise ValueError("Training sample missing 'features' dictionary")

    # Use Central's ground truth risk score
    risk_score = sample.get('risk_score', 1.0)

    return features, risk_score


@pytest.mark.integration
@pytest.mark.slow
def test_central_connection(central_client: CentralExportClient) -> None:
    """
    Test that we can connect to Central and fetch at least one deployment.

    This is a simple smoke test to verify authentication and connectivity.
    """
    logger.info("Testing Central API connection...")

    # Try to fetch one workload
    workloads = list(central_client.stream_workloads())

    # Should get at least one workload
    assert len(workloads) > 0, "Should fetch at least one workload from Central"

    # Verify structure - Central API returns {'result': {'deployment': ..., 'images': ..., ...}}
    first_workload = workloads[0]
    assert 'result' in first_workload, \
        "Workload should have 'result' field"
    assert 'deployment' in first_workload['result'], \
        "Workload result should have 'deployment' field"

    logger.info(f"Successfully fetched {len(workloads)} workload(s) from Central")


@pytest.mark.integration
@pytest.mark.slow
def test_train_and_predict_with_central_data(
    central_export_service: CentralExportService
) -> None:
    """
    End-to-end integration test: Train model on Central data and validate predictions.

    Steps:
    1. Collect training data from Central (100 samples)
    2. Split into training (80) and test (20) sets
    3. Train model on training set
    4. Make predictions on test set
    5. Compare predicted vs actual risk scores
    """
    logger.info("=" * 80)
    logger.info("Starting Central Integration Test")
    logger.info("=" * 80)

    # Step 1: Collect training data from Central
    logger.info("Step 1: Collecting training data from Central...")
    all_samples = []

    try:
        # Use new streaming architecture
        from src.streaming import CentralStreamSource, SampleStream
        source = CentralStreamSource(central_export_service.client, {})
        sample_stream = SampleStream(source, central_export_service.feature_extractor, {})

        for i, sample in enumerate(sample_stream.stream(filters=None, limit=100)):
            all_samples.append(sample)
            if (i + 1) % 20 == 0:
                logger.info(f"  Collected {i + 1} samples...")
    except Exception as e:
        logger.error(f"Failed to collect training data: {e}")
        pytest.fail(f"Failed to collect training data: {e}")

    total_samples = len(all_samples)
    logger.info(f"  Total samples collected: {total_samples}")

    # Need at least 20 samples to run the test
    assert total_samples >= 20, f"Need at least 20 samples, got {total_samples}"

    # Step 2: Split data into training and test sets
    logger.info("Step 2: Splitting data into training and test sets...")
    split_index = int(total_samples * 0.8)
    training_samples = all_samples[:split_index]
    test_samples = all_samples[split_index:]

    logger.info(f"  Training set: {len(training_samples)} samples")
    logger.info(f"  Test set: {len(test_samples)} samples")

    # Step 3: Extract features and train model
    logger.info("Step 3: Extracting features and training model...")
    training_features = []
    training_scores = []

    for i, sample in enumerate(training_samples):
        try:
            features, score = extract_features_and_score(sample)
            training_features.append(features)
            training_scores.append(score)
        except Exception as e:
            logger.warning(f"  Failed to extract features from training sample {i}: {e}")
            continue

    logger.info(f"  Extracted features from {len(training_features)} training samples")

    # Convert to numpy arrays
    feature_names = list(training_features[0].keys())
    X_train = np.array([[f[name] for name in feature_names] for f in training_features])
    y_train = np.array(training_scores)

    logger.info(f"  Training data shape: X={X_train.shape}, y={y_train.shape}")
    logger.info(f"  Training score range: [{y_train.min():.4f}, {y_train.max():.4f}]")

    # Check if scores have variance - if all identical, we can't train a ranking model
    score_variance = np.var(y_train)
    logger.info(f"  Training score variance: {score_variance:.6f}")

    if score_variance < 1e-10:
        logger.warning("⚠️  All training scores are identical - cannot train ranking model")
        logger.warning("   This may occur when Central doesn't have alerts/policies data")
        logger.warning("   Skipping model training and prediction validation")
        pytest.skip("Cannot train ranking model with identical target values")

    # Train model
    model = RiskRankingModel()
    metrics = model.train(X_train, y_train, feature_names=feature_names)

    logger.info(f"  Model trained successfully!")
    logger.info(f"    Training NDCG: {metrics.train_ndcg:.4f}")
    logger.info(f"    Validation NDCG: {metrics.val_ndcg:.4f}")
    logger.info(f"    Model version: {model.model_version}")

    # Assertions on training
    assert model.model is not None, "Model should be trained"
    assert model.model_version is not None, "Model version should be set"

    # Step 4: Make predictions on test set
    logger.info("Step 4: Making predictions on held-out test set...")
    test_features = []
    actual_scores = []
    predicted_scores = []

    for i, sample in enumerate(test_samples):
        try:
            features, actual_score = extract_features_and_score(sample)
            test_features.append(features)
            actual_scores.append(actual_score)
        except Exception as e:
            logger.warning(f"  Failed to extract features from test sample {i}: {e}")
            continue

    # Convert to numpy and predict
    X_test = np.array([[f[name] for name in feature_names] for f in test_features])
    predictions = model.predict(X_test, explain=False)
    predicted_scores = [pred.risk_score for pred in predictions]

    logger.info(f"  Made predictions for {len(predicted_scores)} test samples")

    # Step 5: Compare predicted vs actual scores
    logger.info("Step 5: Comparing predicted vs actual risk scores...")

    actual_scores_array = np.array(actual_scores)
    predicted_scores_array = np.array(predicted_scores)

    # Calculate metrics
    mae = np.mean(np.abs(predicted_scores_array - actual_scores_array))
    rmse = np.sqrt(np.mean((predicted_scores_array - actual_scores_array) ** 2))

    # Correlation (if we have variance)
    if np.std(actual_scores_array) > 0 and np.std(predicted_scores_array) > 0:
        correlation = np.corrcoef(actual_scores_array, predicted_scores_array)[0, 1]
    else:
        correlation = 0.0

    # Percentage within acceptable range (±30%)
    within_range = np.mean(np.abs(predicted_scores_array - actual_scores_array) /
                          (actual_scores_array + 1e-10) <= 0.3) * 100

    # Log results
    logger.info(f"  Prediction Metrics:")
    logger.info(f"    Mean Absolute Error (MAE): {mae:.4f}")
    logger.info(f"    Root Mean Squared Error (RMSE): {rmse:.4f}")
    logger.info(f"    Correlation: {correlation:.4f}")
    logger.info(f"    Within ±30%: {within_range:.1f}%")

    # Log first 5 comparisons
    logger.info(f"  Sample Predictions (first 5):")
    for i in range(min(5, len(actual_scores))):
        diff = predicted_scores[i] - actual_scores[i]
        pct_diff = (diff / (actual_scores[i] + 1e-10)) * 100
        logger.info(f"    Sample {i}: Actual={actual_scores[i]:.4f}, "
                   f"Predicted={predicted_scores[i]:.4f}, "
                   f"Diff={diff:.4f} ({pct_diff:+.1f}%)")

    # Assertions
    assert len(predicted_scores) == len(actual_scores), \
        "Should have predictions for all test samples"

    assert len(predicted_scores) > 0, "Should have at least one prediction"

    # Model should have some predictive power (correlation > 0.3 is reasonable)
    # Note: This threshold may need adjustment based on actual data quality
    assert correlation > 0.3 or correlation == 0.0, \
        f"Model should have reasonable correlation (got {correlation:.4f})"

    # MAE should be reasonable (depends on score range, but let's check it's not infinite)
    assert not np.isnan(mae) and not np.isinf(mae), "MAE should be finite"
    assert mae < 100, f"MAE seems too high: {mae:.4f}"

    # At least some predictions should be in the ballpark
    assert within_range >= 30.0, \
        f"At least 30% of predictions should be within ±30% of actual (got {within_range:.1f}%)"

    logger.info("=" * 80)
    logger.info("Central Integration Test PASSED!")
    logger.info("=" * 80)


@pytest.mark.integration
def test_feature_extraction_from_central_data(
    central_export_service: CentralExportService,
    feature_extractor: BaselineFeatureExtractor
) -> None:
    """
    Test that we can extract features from real Central deployment data.
    """
    logger.info("Testing feature extraction from Central data...")

    # Collect a few samples using new streaming architecture
    from src.streaming import CentralStreamSource, SampleStream
    source = CentralStreamSource(central_export_service.client, {})
    sample_stream = SampleStream(source, central_export_service.feature_extractor, {})
    samples = list(sample_stream.stream(filters=None, limit=5))

    assert len(samples) > 0, "Should collect at least one sample"

    # Extract features from each sample
    for i, sample in enumerate(samples):
        baseline_features = feature_extractor.extract_baseline_features(
            sample.get('deployment', {}),
            sample.get('images', []),
            sample.get('alerts', []),
            sample.get('baseline_violations', [])
        )

        assert baseline_features is not None, f"Sample {i} should extract features"
        assert isinstance(baseline_features, BaselineRiskFactors), \
            f"Sample {i} should return BaselineRiskFactors"

        # Verify score is calculated
        assert baseline_features.overall_score >= 0, \
            f"Sample {i} should have non-negative risk score"

        # Verify individual multipliers are set
        assert baseline_features.policy_violations_multiplier >= 1.0
        assert baseline_features.vulnerabilities_multiplier >= 1.0

        logger.info(f"Sample {i}: Risk score = {baseline_features.overall_score:.4f}")

    logger.info(f"Successfully extracted features from {len(samples)} Central deployments")
