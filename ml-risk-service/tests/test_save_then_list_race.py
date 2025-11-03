"""
Test to reproduce the exact save-then-list scenario from the Makefile test.

This simulates:
1. Training saves model (creates /app/models/models/{id}/v{version}/)
2. API endpoint lists models (checks if /app/models/models/ exists)

The hypothesis is that there's a timing or filesystem sync issue.
"""

import pytest
import tempfile
from pathlib import Path
import time
import numpy as np
import joblib
import io

from src.models.ranking_model import RiskRankingModel
from src.storage.model_storage import ModelStorageManager, StorageConfig, ModelMetadata
from src.services.risk_service import RiskPredictionService
from unittest.mock import patch


def test_save_model_then_list_immediately():
    """
    Test that list_models() finds a model immediately after save_model().

    This is the exact sequence that happens in the Makefile test.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir)

        # Simulate the initial state: only base directory exists
        base_path.mkdir(parents=True, exist_ok=True)

        # Set up storage
        storage_config = StorageConfig(
            backend='local',
            base_path=str(base_path)
        )
        storage_manager = ModelStorageManager(storage_config)

        # Verify initial state - no models
        models_before = storage_manager.list_models('stackrox-risk-model')
        assert len(models_before) == 0, "Should start with no models"

        # Check if models/ subdirectory exists
        models_dir = base_path / "models"
        print(f"Before save - models/ exists: {models_dir.exists()}")

        # Train and save a model (simulating training endpoint)
        model = RiskRankingModel()
        X_train = np.random.rand(50, 10)
        y_train = np.random.rand(50)
        model.train(X_train, y_train)

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
            training_timestamp='',
            model_size_bytes=0,
            checksum='',
            performance_metrics={},
            config={}
        )

        # Save the model
        save_success = storage_manager.save_model(model_data, metadata)
        assert save_success, "Model save must succeed"

        # Immediately check if models/ directory was created
        print(f"After save - models/ exists: {models_dir.exists()}")
        print(f"After save - models/ contents: {list(models_dir.iterdir()) if models_dir.exists() else 'N/A'}")

        # Verify the full path exists
        expected_path = base_path / "models" / "stackrox-risk-model" / f"v{model.model_version}"
        print(f"Expected path: {expected_path}")
        print(f"Expected path exists: {expected_path.exists()}")

        # NOW - list models immediately (simulating the API endpoint)
        models_after = storage_manager.list_models('stackrox-risk-model')

        # THIS IS WHERE THE BUG WOULD MANIFEST
        assert len(models_after) == 1, f"Should find 1 model immediately after save, found {len(models_after)}"
        assert models_after[0].model_id == 'stackrox-risk-model'
        assert models_after[0].version == model.model_version


def test_save_and_list_with_service_instances():
    """
    Test save and list using RiskPredictionService instances like the real API.

    This simulates:
    1. Training service saves model
    2. Models API lists models using a different service instance
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir)
        base_path.mkdir(parents=True, exist_ok=True)

        storage_config = StorageConfig(
            backend='local',
            base_path=str(base_path)
        )

        # Create storage manager and save a model
        storage_manager = ModelStorageManager(storage_config)

        model = RiskRankingModel()
        X_train = np.random.rand(50, 10)
        y_train = np.random.rand(50)
        model.train(X_train, y_train)

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
            training_timestamp='',
            model_size_bytes=0,
            checksum='',
            performance_metrics={},
            config={}
        )

        # Save via storage manager
        save_success = storage_manager.save_model(model_data, metadata)
        assert save_success

        # Now create a RiskPredictionService and list models
        # This simulates what the /api/v1/models endpoint does
        with patch('src.services.risk_service.ModelStorageManager') as mock_storage_class:
            mock_storage_class.return_value = storage_manager

            service = RiskPredictionService()

            # List models through the service (simulating API call)
            list_response = service.list_models('stackrox-risk-model')

            assert list_response.total_count == 1, f"Service should find 1 model, found {list_response.total_count}"
            assert len(list_response.models) == 1
            assert list_response.models[0].model_id == 'stackrox-risk-model'


def test_filesystem_visibility_after_save():
    """
    Test that saved files are immediately visible to os.listdir() and Path.iterdir().

    This checks for potential filesystem caching issues.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir)
        base_path.mkdir(parents=True, exist_ok=True)

        storage_config = StorageConfig(
            backend='local',
            base_path=str(base_path)
        )
        storage_manager = ModelStorageManager(storage_config)

        model = RiskRankingModel()
        X_train = np.random.rand(20, 5)
        y_train = np.random.rand(20)
        model.train(X_train, y_train)

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
            training_timestamp='',
            model_size_bytes=0,
            checksum='',
            performance_metrics={},
            config={}
        )

        # Save model
        storage_manager.save_model(model_data, metadata)

        # Immediately check filesystem visibility using multiple methods
        models_dir = base_path / "models"

        # Method 1: Path.exists()
        assert models_dir.exists(), f"models/ should exist via Path.exists(): {models_dir}"

        # Method 2: os.path.exists()
        import os
        assert os.path.exists(str(models_dir)), f"models/ should exist via os.path.exists(): {models_dir}"

        # Method 3: Path.iterdir()
        contents = list(models_dir.iterdir())
        assert len(contents) > 0, f"models/ should have contents via Path.iterdir(): {contents}"

        # Method 4: os.listdir()
        os_contents = os.listdir(str(models_dir))
        assert len(os_contents) > 0, f"models/ should have contents via os.listdir(): {os_contents}"

        print(f"✓ All filesystem visibility checks passed")
        print(f"  Path.iterdir(): {[p.name for p in contents]}")
        print(f"  os.listdir(): {os_contents}")
