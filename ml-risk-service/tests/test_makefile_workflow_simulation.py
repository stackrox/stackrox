"""
Simulate the exact workflow from the Makefile test-train-predict-central target.

This test reproduces:
1. make clean - removes ./deploy/models/
2. docker-run - recreates ./deploy/models/ (empty)
3. Training saves model
4. List models API call

The hypothesis: The issue occurs when the models.py service is initialized
BEFORE training completes, and somehow can't see the newly created model.
"""

import pytest
import tempfile
import shutil
from pathlib import Path
import numpy as np
import joblib
import io
from unittest.mock import patch

from src.models.ranking_model import RiskRankingModel
from src.storage.model_storage import ModelStorageManager, StorageConfig, ModelMetadata
from src.services.risk_service import RiskPredictionService
# Don't import TrainingService to avoid matplotlib dependency in tests


def test_makefile_workflow_exact_simulation():
    """
    Simulate the exact sequence of events from the Makefile test.

    1. Start with empty deploy/models/ directory (simulating post-clean state)
    2. Create models.py's global _risk_service (happens on first API call to /health or /models)
    3. Create training.py's global _training_service
    4. Training saves a model
    5. Use models.py's _risk_service to list models
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        deploy_models = Path(tmpdir) / "deploy" / "models"

        # Step 1: Simulate 'make clean' - directory doesn't exist
        assert not deploy_models.exists(), "Start with no directory (post-clean state)"

        # Step 2: Simulate 'docker-run' - mkdir -p ./deploy/models
        deploy_models.mkdir(parents=True, exist_ok=True)
        deploy_models.chmod(0o777)
        assert deploy_models.exists()
        assert list(deploy_models.iterdir()) == [], "Should be empty after clean+mkdir"

        # Configure storage to use our temp directory
        storage_config = StorageConfig(
            backend='local',
            base_path=str(deploy_models)
        )

        # Step 3: Simulate first API call (e.g., /health/ready or /models)
        # This creates models.py's global _risk_service
        with patch('src.services.risk_service.ModelStorageManager') as mock_storage_class1:
            # Create storage manager for models.py service
            models_api_storage = ModelStorageManager(storage_config)
            mock_storage_class1.return_value = models_api_storage

            models_api_service = RiskPredictionService()
            print(f"Step 3: models.py service created")
            print(f"  Storage base_path: {models_api_storage.primary_storage.base_path}")
            print(f"  models/ subdirectory exists: {(models_api_storage.primary_storage.base_path / 'models').exists()}")

            # Verify no models exist initially
            initial_list = models_api_service.list_models('stackrox-risk-model')
            print(f"  Initial model count: {initial_list.total_count}")
            assert initial_list.total_count == 0, "Should start with 0 models"

        # Step 4: Simulate training API call
        # Create storage manager for training (separate instance like in actual app)
        training_api_storage = ModelStorageManager(storage_config)

        print(f"\nStep 4: Training with Central data simulation")

        # Create and train a model (simulating train_full endpoint)
        model = RiskRankingModel()
        X_train = np.random.rand(100, 10)
        y_train = np.random.rand(100)
        model.train(X_train, y_train)

        # Serialize model
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

        # Save model using training service's storage manager
        save_success = training_api_storage.save_model(model_data, metadata)
        assert save_success, "Training should save model successfully"
        print(f"  Model saved: {model.model_version}")
        print(f"  models/ subdirectory now exists: {(deploy_models / 'models').exists()}")

        # Verify files were created
        expected_path = deploy_models / "models" / "stackrox-risk-model" / f"v{model.model_version}"
        print(f"  Expected path: {expected_path}")
        print(f"  Expected path exists: {expected_path.exists()}")
        assert expected_path.exists(), f"Model directory should exist: {expected_path}"

        # Step 5: Simulate /api/v1/models API call
        # This uses the SAME models_api_service created in step 3
        print(f"\nStep 5: List models using models.py service (created in step 3)")

        # THIS IS THE CRITICAL TEST - Can the models.py service (created before training)
        # find the model that was saved by the training.py service?
        final_list = models_api_service.list_models('stackrox-risk-model')
        print(f"  Final model count: {final_list.total_count}")

        # THIS IS WHERE THE BUG WOULD MANIFEST
        if final_list.total_count == 0:
            print(f"\n  ❌ BUG REPRODUCED!")
            print(f"  The models.py service (created in step 3) cannot see the model")
            print(f"  saved by training.py service (in step 4)")
            print(f"\n  Debugging info:")
            print(f"    deploy_models path: {deploy_models}")
            print(f"    models/ exists: {(deploy_models / 'models').exists()}")
            print(f"    models/ contents: {list((deploy_models / 'models').iterdir()) if (deploy_models / 'models').exists() else 'N/A'}")
            print(f"    Storage backend base_path: {models_api_storage.primary_storage.base_path}")

            # Try listing with the training service's storage to see if THAT can see it
            training_list = training_api_storage.list_models('stackrox-risk-model')
            print(f"    Training service can see: {len(training_list)} models")

            pytest.fail(f"REPRODUCED THE BUG: Found {final_list.total_count} models, expected 1")

        assert final_list.total_count == 1, f"Should find 1 model after training, found {final_list.total_count}"
        assert final_list.models[0].model_id == 'stackrox-risk-model'
        assert final_list.models[0].version == model.model_version
        print(f"  ✓ Test passed - model found successfully")


def test_same_storage_manager_instance():
    """
    Test if using the SAME storage manager instance (simulating shared singleton) works.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        deploy_models = Path(tmpdir) / "deploy" / "models"
        deploy_models.mkdir(parents=True, exist_ok=True)

        storage_config = StorageConfig(
            backend='local',
            base_path=str(deploy_models)
        )

        # Use the SAME storage manager for both services
        shared_storage = ModelStorageManager(storage_config)

        # Create models.py service
        with patch('src.services.risk_service.ModelStorageManager') as mock1:
            mock1.return_value = shared_storage
            models_service = RiskPredictionService()

        # Train and save model
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

        # Save using shared storage
        shared_storage.save_model(model_data, metadata)

        # List using models service
        result = models_service.list_models('stackrox-risk-model')
        assert result.total_count == 1, "Shared storage manager should work"
