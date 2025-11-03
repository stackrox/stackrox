"""
Test directory initialization to diagnose the make clean issue.

This test simulates what happens when:
1. make clean removes ./deploy/models/
2. docker-run recreates just ./deploy/models/ (not subdirectories)
3. Training tries to save to /app/models/models/{model_id}/v{version}/
"""

import pytest
import tempfile
import shutil
from pathlib import Path
from unittest.mock import patch
import numpy as np
import joblib
import io

from src.models.ranking_model import RiskRankingModel
from src.storage.model_storage import ModelStorageManager, StorageConfig, ModelMetadata


def test_save_to_empty_base_directory():
    """
    Test saving a model when only the base directory exists (simulating post-clean state).

    This reproduces the exact scenario:
    - Base directory exists: /app/models (mapped from ./deploy/models)
    - Subdirectory does NOT exist: /app/models/models/
    - Model storage must create full path including subdirectories
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir) / "models"

        # Simulate 'make clean' followed by 'mkdir -p ./deploy/models'
        # Only base directory exists, NO subdirectories
        base_path.mkdir(parents=True, exist_ok=True)
        assert base_path.exists(), "Base directory should exist"
        assert not (base_path / "models").exists(), "Subdirectory should NOT exist yet"

        # Set up storage with this base path
        storage_config = StorageConfig(
            backend='local',
            base_path=str(base_path)
        )
        storage_manager = ModelStorageManager(storage_config)

        # Create and train a minimal model
        model = RiskRankingModel()
        X_train = np.random.rand(50, 10)
        y_train = np.random.rand(50)
        model.train(X_train, y_train)

        # Serialize the model
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

        # THIS IS THE CRITICAL TEST:
        # Can storage_manager.save_model() create the full directory structure
        # when only base_path exists but base_path/models/ does not?
        success = storage_manager.save_model(model_data, metadata)
        assert success, "Model save should succeed even when subdirectories don't exist"

        # Verify the full directory structure was created
        expected_path = base_path / "models" / "stackrox-risk-model" / f"v{model.model_version}"
        assert expected_path.exists(), f"Full model directory should be created: {expected_path}"

        # Verify files exist
        model_file = expected_path / "model.joblib"
        metadata_file = expected_path / "metadata.json"
        assert model_file.exists(), f"Model file should exist: {model_file}"
        assert metadata_file.exists(), f"Metadata file should exist: {metadata_file}"

        # Verify we can list the model
        models = storage_manager.list_models('stackrox-risk-model')
        assert len(models) == 1, "Should find exactly 1 model"
        assert models[0].model_id == 'stackrox-risk-model'
        assert models[0].version == model.model_version


def test_list_models_with_missing_subdirectory():
    """
    Test list_models when base directory exists but 'models/' subdirectory doesn't.

    This simulates the state immediately after 'make clean && docker-run'
    before any training has occurred.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir) / "models"

        # Only create base directory
        base_path.mkdir(parents=True, exist_ok=True)
        assert base_path.exists()
        assert not (base_path / "models").exists(), "models/ subdirectory should not exist"

        # Set up storage
        storage_config = StorageConfig(
            backend='local',
            base_path=str(base_path)
        )
        storage_manager = ModelStorageManager(storage_config)

        # list_models should handle this gracefully and return empty list
        models = storage_manager.list_models('stackrox-risk-model')
        assert models == [], "Should return empty list when subdirectory doesn't exist"


def test_permission_simulation():
    """
    Test that chmod 777 permissions allow model saving.

    Simulates the exact permission setup from docker-run.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir) / "models"
        base_path.mkdir(parents=True, exist_ok=True)

        # Simulate 'chmod 777 ./deploy/models' from Makefile line 101
        base_path.chmod(0o777)

        # Verify permissions
        import stat
        mode = base_path.stat().st_mode
        assert stat.S_IMODE(mode) == 0o777, "Directory should have 777 permissions"

        # Now test if we can save a model
        storage_config = StorageConfig(
            backend='local',
            base_path=str(base_path)
        )
        storage_manager = ModelStorageManager(storage_config)

        # Create minimal model
        model = RiskRankingModel()
        X_train = np.random.rand(20, 5)
        y_train = np.random.rand(20)
        model.train(X_train, y_train)

        # Serialize
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

        # Save should work with 777 permissions
        success = storage_manager.save_model(model_data, metadata)
        assert success, "Model save should succeed with 777 permissions"
