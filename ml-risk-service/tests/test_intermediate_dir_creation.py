"""
Test to verify intermediate directory creation is immediately visible.

This specifically tests whether creating /path/to/dir/subdir1/subdir2/file
makes /path/to/dir/subdir1/ immediately visible to exists() checks.
"""

import pytest
import tempfile
from pathlib import Path
import time


def test_intermediate_directory_visibility():
    """
    Test that intermediate directories created by mkdir(parents=True)
    are immediately visible to subsequent exists() checks.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base = Path(tmpdir)

        # Start with only base directory
        assert base.exists()

        # Create a deeply nested path all at once
        deep_path = base / "level1" / "level2" / "level3"
        deep_path.mkdir(parents=True, exist_ok=True)

        # Immediately check if intermediate directories are visible
        level1 = base / "level1"
        level2 = base / "level1" / "level2"

        # These should ALL be True immediately
        assert level1.exists(), "level1 should exist immediately after mkdir"
        assert level2.exists(), "level2 should exist immediately after mkdir"
        assert deep_path.exists(), "level3 should exist immediately after mkdir"

        # Verify with os.path.exists too
        import os
        assert os.path.exists(str(level1)), "level1 should exist via os.path.exists"
        assert os.path.exists(str(level2)), "level2 should exist via os.path.exists"


def test_file_creation_makes_parent_visible():
    """
    Test that creating a file with parents=True makes parent directory visible.

    This mirrors what happens in save_model:
    1. model_path.mkdir(parents=True) creates models/stackrox-risk-model/vXXX/
    2. We write model.joblib and metadata.json
    3. list_models checks if models/ exists
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base = Path(tmpdir)

        # Simulate what save_model does
        model_id = "test-model"
        version = "v1.0.0"
        model_path = base / "models" / model_id / version

        # This is what save_model does (line 326-327)
        model_path.mkdir(parents=True, exist_ok=True)

        # Write files (like save_model does)
        model_file = model_path / "model.joblib"
        metadata_file = model_path / "metadata.json"

        model_file.write_text("fake model data")
        metadata_file.write_text("{}")

        # Now check if the intermediate "models/" directory is visible
        # This is what list_models does (line 383-385)
        models_dir = base / "models"

        # THIS IS THE CRITICAL CHECK
        assert models_dir.exists(), f"models/ should exist after creating {model_path}"

        # Verify contents
        assert model_id in [d.name for d in models_dir.iterdir()]


def test_concurrent_directory_creation():
    """
    Test if there's any race condition in directory creation and existence checks.
    """

    import threading

    with tempfile.TemporaryDirectory() as tmpdir:
        base = Path(tmpdir)
        success_count = [0]
        failure_count = [0]

        def create_and_check():
            """Create a directory path and immediately check parent visibility."""
            try:
                # Each thread creates its own subdirectory
                thread_id = threading.current_thread().name
                path = base / "models" / f"thread-{thread_id}" / "v1.0.0"
                path.mkdir(parents=True, exist_ok=True)

                # Immediately check if parent exists
                models_dir = base / "models"
                if models_dir.exists():
                    success_count[0] += 1
                else:
                    failure_count[0] += 1
                    print(f"FAILURE: {thread_id} - models/ doesn't exist after creating {path}")
            except Exception as e:
                print(f"Exception in {thread_id}: {e}")
                failure_count[0] += 1

        # Run 10 threads concurrently
        threads = []
        for i in range(10):
            t = threading.Thread(target=create_and_check, name=f"T{i}")
            threads.append(t)
            t.start()

        # Wait for all threads
        for t in threads:
            t.join()

        # All should succeed
        assert failure_count[0] == 0, f"Had {failure_count[0]} failures out of 10 threads"
        assert success_count[0] == 10, f"Expected 10 successes, got {success_count[0]}"


def test_exact_save_model_scenario():
    """
    Replicate the EXACT scenario from save_model and list_models.

    This uses the exact same path construction and checks.
    """

    with tempfile.TemporaryDirectory() as tmpdir:
        base_path = Path(tmpdir)

        # Simulate save_model (lines 326-337 of model_storage.py)
        model_id = "stackrox-risk-model"
        version = "sklearn_ranksvm_20251103_120000"

        # This is _get_model_path(model_id, version)
        relative_path = f"models/{model_id}/v{version}"
        model_path = base_path / relative_path

        # This is what save_model does
        model_path.mkdir(parents=True, exist_ok=True)

        # Write files
        model_file = model_path / "model.joblib"
        metadata_file = model_path / "metadata.json"
        model_file.write_bytes(b"model data")
        metadata_file.write_text('{"model_id": "stackrox-risk-model"}')

        # NOW - simulate list_models (lines 383-386 of model_storage.py)
        models_dir = base_path / "models"

        # THE CRITICAL CHECK
        if not models_dir.exists():
            pytest.fail(f"REPRODUCED THE BUG: models/ doesn't exist after saving to {model_path}")

        # If we get here, the directory exists - verify we can list it
        assert models_dir.exists()
        assert model_id in [d.name for d in models_dir.iterdir()]

        # Find the version directory
        model_dir = models_dir / model_id
        assert model_dir.exists()

        version_dirs = list(model_dir.iterdir())
        assert len(version_dirs) == 1
        assert version_dirs[0].name == f"v{version}"
