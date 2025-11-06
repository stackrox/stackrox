"""
Test model persistence and retrieval after training.

This test simulates the workflow that happens in test-train-predict-central:
1. Train a model (saving to storage)
2. List models (should find the newly trained model)
3. Reload the model (should load successfully)
"""

import pytest
import tempfile
import os
from pathlib import Path
from unittest.mock import Mock, patch
import numpy as np

from src.models.ranking_model import RiskRankingModel
from src.storage.model_storage import ModelStorageManager, StorageConfig, ModelMetadata
from src.services.prediction_service import RiskPredictionService
from src.api.schemas import ReloadModelRequest


def test_model_available_immediately_after_training():
    """
    Test that a model is immediately available after training.

    This reproduces the issue from 'make clean test-train-predict-central':
    1. Start with empty storage
    2. Train and save a model
    3. List models - should find the model immediately
    4. Reload model - should load successfully
    """

    # Create temporary directory for model storage
    with tempfile.TemporaryDirectory() as tmpdir:
        # Set up storage with temp directory
        storage_config = StorageConfig(
            backend='local',
            base_path=tmpdir
        )
        storage_manager = ModelStorageManager(storage_config)

        # Verify storage is initially empty
        models = storage_manager.list_models('stackrox-risk-model')
        assert len(models) == 0, "Storage should be empty initially"

        # Simulate training: create and save a model
        model = RiskRankingModel()

        # Create minimal training data
        X_train = np.random.rand(50, 10)
        y_train = np.random.rand(50)

        # Train the model
        model.train(X_train, y_train)

        # Save the model (this is what happens in training service)
        import joblib
        import io
        buffer = io.BytesIO()
        joblib.dump({
            'model': model.model,
            'scaler': model.scaler,
            'feature_names': model.feature_names,
            'model_version': model.model_version,
            'algorithm': model.algorithm,
            'config': model.config,
            'training_metrics': model.training_metrics
        }, buffer)
        model_data = buffer.getvalue()

        metadata = ModelMetadata(
            model_id='stackrox-risk-model',
            version=model.model_version,
            algorithm='sklearn_ranksvm',
            feature_count=10,
            training_timestamp='',  # Will be set by __post_init__
            model_size_bytes=0,  # Will be updated by save_model
            checksum='',  # Will be set by save_model
            performance_metrics={},
            config={}
        )

        # Save to storage
        success = storage_manager.save_model(model_data, metadata)
        assert success, "Model save should succeed"

        # CRITICAL TEST: List models immediately after save
        # This is where the bug would manifest - model not found
        models = storage_manager.list_models('stackrox-risk-model')
        assert len(models) == 1, f"Should find exactly 1 model, found {len(models)}"
        assert models[0].model_id == 'stackrox-risk-model'
        assert models[0].version == model.model_version

        # Create a NEW RiskPredictionService (simulating a fresh API call)
        # with the same storage configuration
        with patch('src.services.prediction_service.ModelStorageManager') as mock_storage_class:
            mock_storage_class.return_value = storage_manager

            service = RiskPredictionService()

            # Try to reload the model (this is what the Makefile test does)
            request = ReloadModelRequest(
                model_id='stackrox-risk-model',
                version=None,  # Load latest
                force_reload=True
            )

            response = service.reload_model(request)

            # Verify reload succeeded
            assert response.success, f"Reload should succeed: {response.message}"
            assert response.new_model_version == model.model_version
            assert service.model_loaded, "Model should be loaded after reload"


def test_model_persistence_across_service_instances():
    """
    Test that models persist across different service instances.

    This simulates what happens between docker container restarts:
    1. First instance trains and saves model
    2. Second instance should find and load the model
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        # Set up storage
        storage_config = StorageConfig(
            backend='local',
            base_path=tmpdir
        )

        # First service instance: train and save
        storage_manager_1 = ModelStorageManager(storage_config)
        model = RiskRankingModel()

        X_train = np.random.rand(50, 10)
        y_train = np.random.rand(50)
        model.train(X_train, y_train)

        # Save model
        import joblib
        import io
        buffer = io.BytesIO()
        joblib.dump({
            'model': model.model,
            'scaler': model.scaler,
            'feature_names': model.feature_names,
            'model_version': model.model_version,
            'algorithm': model.algorithm,
            'config': model.config,
            'training_metrics': model.training_metrics
        }, buffer)
        model_data = buffer.getvalue()

        metadata = ModelMetadata(
            model_id='stackrox-risk-model',
            version=model.model_version,
            algorithm='sklearn_ranksvm',
            feature_count=10,
            training_timestamp='',  # Will be set by __post_init__
            model_size_bytes=0,  # Will be updated by save_model
            checksum='',  # Will be set by save_model
            performance_metrics={},
            config={}
        )

        storage_manager_1.save_model(model_data, metadata)

        # Second service instance: should find the model
        storage_manager_2 = ModelStorageManager(storage_config)

        models = storage_manager_2.list_models('stackrox-risk-model')
        assert len(models) == 1, "Second instance should find the model"
        assert models[0].version == model.model_version

        # Should be able to load it
        with patch('src.services.prediction_service.ModelStorageManager') as mock_storage_class:
            mock_storage_class.return_value = storage_manager_2

            service = RiskPredictionService()

            request = ReloadModelRequest(
                model_id='stackrox-risk-model',
                version=model.model_version,
                force_reload=True
            )

            response = service.reload_model(request)
            assert response.success, f"Should load model from second instance: {response.message}"


def test_file_system_sync_after_save():
    """
    Test that saved models are immediately visible in the file system.

    This checks for potential file system buffering issues.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        storage_config = StorageConfig(
            backend='local',
            base_path=tmpdir
        )
        storage_manager = ModelStorageManager(storage_config)

        # Create and save a minimal model
        model = RiskRankingModel()
        X_train = np.random.rand(20, 5)
        y_train = np.random.rand(20)
        model.train(X_train, y_train)

        import joblib
        import io
        buffer = io.BytesIO()
        joblib.dump({
            'model': model.model,
            'scaler': model.scaler,
            'feature_names': model.feature_names,
            'model_version': model.model_version,
            'algorithm': model.algorithm,
            'config': model.config,
            'training_metrics': model.training_metrics
        }, buffer)
        model_data = buffer.getvalue()

        metadata = ModelMetadata(
            model_id='test-model',
            version='v1.0.0',
            algorithm='sklearn_ranksvm',
            feature_count=5,
            training_timestamp='',  # Will be set by __post_init__
            model_size_bytes=0,  # Will be updated by save_model
            checksum='',  # Will be set by save_model
            performance_metrics={},
            config={}
        )

        # Save the model
        storage_manager.save_model(model_data, metadata)

        # Immediately check if files exist on disk
        expected_dir = Path(tmpdir) / "models" / "test-model" / "vv1.0.0"
        expected_model_file = expected_dir / "model.joblib"
        expected_metadata_file = expected_dir / "metadata.json"

        assert expected_dir.exists(), f"Model directory should exist: {expected_dir}"
        assert expected_model_file.exists(), f"Model file should exist: {expected_model_file}"
        assert expected_metadata_file.exists(), f"Metadata file should exist: {expected_metadata_file}"

        # Verify we can read the files
        assert expected_model_file.stat().st_size > 0, "Model file should not be empty"
        assert expected_metadata_file.stat().st_size > 0, "Metadata file should not be empty"
