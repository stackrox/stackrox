"""
Unit tests for model reload functionality.
"""

import pytest
from unittest.mock import Mock, MagicMock, patch
from src.services.prediction_service import RiskPredictionService
from src.api.schemas import ReloadModelRequest


def test_reload_tracks_actual_version_not_latest():
    """Test that reload stores the actual model version, not the string 'latest'."""

    # Create mock storage manager
    mock_storage_manager = Mock()

    # Create service with mocked storage
    with patch('src.services.prediction_service.ModelStorageManager', return_value=mock_storage_manager):
        service = RiskPredictionService()

        # Mock the model's load_model_from_storage to return success
        # and set the model_version to an actual version string
        actual_version = "sklearn_ranksvm_20250103_143022"
        service.model.model_version = actual_version

        with patch.object(service.model, 'load_model_from_storage', return_value=True):
            # Call reload without specifying version (should load "latest")
            request = ReloadModelRequest(
                model_id="stackrox-risk-model",
                version=None,  # Load latest
                force_reload=True
            )

            response = service.reload_model(request)

            # Verify reload succeeded
            assert response.success is True

            # Verify the service stored the ACTUAL version, not "latest"
            assert service.current_model_version == actual_version
            assert service.current_model_version != "latest"

            # Verify response contains the actual version
            assert response.new_model_version == actual_version


def test_reload_with_force_loads_new_version():
    """Test that reload with force_reload=True loads a new version when available."""

    # Create mock storage manager
    mock_storage_manager = Mock()

    # Create service with mocked storage
    with patch('src.services.prediction_service.ModelStorageManager', return_value=mock_storage_manager):
        service = RiskPredictionService()

        # Simulate first load with version 1
        version_1 = "sklearn_ranksvm_20250103_100000"
        service.model.model_version = version_1

        with patch.object(service.model, 'load_model_from_storage', return_value=True):
            request_1 = ReloadModelRequest(
                model_id="stackrox-risk-model",
                version=None,
                force_reload=True
            )

            response_1 = service.reload_model(request_1)
            assert response_1.success is True
            assert service.current_model_version == version_1

            # Now simulate a new version is available
            version_2 = "sklearn_ranksvm_20250103_143022"
            service.model.model_version = version_2

            # Call reload again WITH force_reload to get the latest
            request_2 = ReloadModelRequest(
                model_id="stackrox-risk-model",
                version=None,
                force_reload=True  # Force reload to get latest
            )

            # Mock to return True (simulating new version available)
            with patch.object(service.model, 'load_model_from_storage', return_value=True):
                response_2 = service.reload_model(request_2)

                # Should reload and get the new version
                assert response_2.success is True
                assert service.current_model_version == version_2
                assert response_2.new_model_version == version_2
                assert response_2.previous_model_version == version_1


def test_reload_with_explicit_version():
    """Test that reload works correctly when specifying an explicit version."""

    # Create mock storage manager
    mock_storage_manager = Mock()

    # Create service with mocked storage
    with patch('src.services.prediction_service.ModelStorageManager', return_value=mock_storage_manager):
        service = RiskPredictionService()

        # Load a specific version
        specific_version = "sklearn_ranksvm_20250103_120000"
        service.model.model_version = specific_version

        with patch.object(service.model, 'load_model_from_storage', return_value=True):
            request = ReloadModelRequest(
                model_id="stackrox-risk-model",
                version=specific_version,
                force_reload=True
            )

            response = service.reload_model(request)

            # Verify reload succeeded
            assert response.success is True

            # Verify the service stored the actual version from the model
            # (should prefer model_version over the requested version)
            assert service.current_model_version == specific_version
            assert response.new_model_version == specific_version


def test_reload_without_version_loads_newer_version():
    """
    Test that reload WITHOUT version parameter loads a newer version when available.

    This is the key test case for the bug:
    - Version 1 is currently loaded
    - Version 2 (newer) exists in storage
    - Call reload() without specifying version (to get "latest")
    - Should load version 2, NOT return "already loaded"
    """

    # Create mock storage manager
    mock_storage_manager = Mock()

    # Create service with mocked storage
    with patch('src.services.prediction_service.ModelStorageManager', return_value=mock_storage_manager):
        service = RiskPredictionService()

        # Initial state: version 1 is loaded
        version_1 = "sklearn_ranksvm_20250103_090416"
        service.model.model_version = version_1
        service.model_loaded = True
        service.current_model_id = "stackrox-risk-model"
        service.current_model_version = version_1

        # Simulate that a newer version (version 2) is now available in storage
        version_2 = "sklearn_ranksvm_20250103_143022"

        # Mock load_model_from_storage to simulate loading version 2 from storage
        def mock_load_from_storage(storage_mgr, model_id, version):
            # Simulate loading the LATEST version from storage (version 2)
            service.model.model_version = version_2
            return True

        with patch.object(service.model, 'load_model_from_storage', side_effect=mock_load_from_storage):
            # Call reload WITHOUT version (requesting "latest") and WITHOUT force_reload
            request = ReloadModelRequest(
                model_id="stackrox-risk-model",
                version=None,  # Request latest
                force_reload=False  # Don't force - but should still reload if newer version exists
            )

            response = service.reload_model(request)

            # CRITICAL TEST: Should successfully load the newer version
            # NOT return "already loaded"
            assert response.success is True

            # Verify the service loaded the NEW version (version 2), not kept the old one
            assert service.current_model_version == version_2, \
                f"Expected version {version_2}, but got {service.current_model_version}"

            assert response.new_model_version == version_2, \
                f"Response should show new version {version_2}, but got {response.new_model_version}"

            assert response.previous_model_version == version_1, \
                f"Response should show previous version {version_1}, but got {response.previous_model_version}"

            # Verify load_model_from_storage was actually called (not skipped)
            service.model.load_model_from_storage.assert_called_once_with(
                mock_storage_manager, "stackrox-risk-model", None
            )
