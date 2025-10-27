"""
Model management service for loading, reloading, and listing models.
This service is shared between gRPC and REST APIs.
"""

import logging
import time
import threading
from typing import Dict, Any, List, Optional

from src.storage.model_storage import ModelStorageManager, StorageConfig
from src.api.schemas import (
    ReloadModelRequest,
    ReloadModelResponse,
    ListModelsResponse,
    ModelInfo
)

logger = logging.getLogger(__name__)


class ModelManagementService:
    """Service for managing ML models with hot reloading support."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or {}

        # Initialize storage manager
        storage_config = self._create_storage_config()
        self.storage_manager = ModelStorageManager(storage_config)

        # Current model state
        self.current_model_id = None
        self.current_model_version = None

        # Thread safety
        self._model_lock = threading.RLock()

    def _create_storage_config(self) -> StorageConfig:
        """Create storage configuration from service config."""
        # Use environment variables to create storage config
        return StorageConfig.from_env()

    def reload_model(self, request: ReloadModelRequest, risk_service) -> ReloadModelResponse:
        """
        Hot reload a model from storage.

        Args:
            request: Model reload request
            risk_service: Risk service instance to update

        Returns:
            Model reload response
        """
        start_time = time.time()
        previous_version = self.current_model_version or "none"

        try:
            if not request.model_id:
                return ReloadModelResponse(
                    success=False,
                    message="Model ID is required",
                    previous_model_version=previous_version,
                    new_model_version="",
                    reload_time_ms=0.0
                )

            # Check if the model is already loaded (unless force reload)
            if (not request.force_reload and
                self.current_model_id == request.model_id and
                (not request.version or self.current_model_version == request.version)):
                reload_time = (time.time() - start_time) * 1000
                return ReloadModelResponse(
                    success=True,
                    message=f"Model {request.model_id} v{self.current_model_version} already loaded",
                    previous_model_version=previous_version,
                    new_model_version=self.current_model_version,
                    reload_time_ms=reload_time
                )

            # Attempt to load the model from storage
            success = self._load_model_from_storage(request.model_id, request.version, risk_service)
            reload_time = (time.time() - start_time) * 1000

            if success:
                new_version = self.current_model_version
                message = f"Successfully reloaded model {request.model_id} v{new_version}"
                logger.info(f"Hot reload successful: {request.model_id} v{new_version} (took {reload_time:.1f}ms)")

                return ReloadModelResponse(
                    success=True,
                    message=message,
                    previous_model_version=previous_version,
                    new_model_version=new_version,
                    reload_time_ms=reload_time
                )
            else:
                version_display = request.version or "latest"
                return ReloadModelResponse(
                    success=False,
                    message=f"Failed to load model {request.model_id} v{version_display}",
                    previous_model_version=previous_version,
                    new_model_version="",
                    reload_time_ms=reload_time
                )

        except Exception as e:
            reload_time = (time.time() - start_time) * 1000
            logger.error(f"Model reload failed: {e}")
            return ReloadModelResponse(
                success=False,
                message=f"Model reload failed: {str(e)}",
                previous_model_version=previous_version,
                new_model_version="",
                reload_time_ms=reload_time
            )

    def list_models(self, model_id: Optional[str] = None) -> ListModelsResponse:
        """
        List available models in storage.

        Args:
            model_id: Optional specific model ID to filter by

        Returns:
            List of available models
        """
        try:
            if model_id:
                # List versions for specific model
                models = self.storage_manager.list_models(model_id)
            else:
                # List all models
                models = self.storage_manager.list_models()

            model_infos = []
            for model in models:
                # Convert performance metrics to simple dict
                metrics = {}
                if hasattr(model, 'performance_metrics') and model.performance_metrics:
                    metrics = {k: float(v) for k, v in model.performance_metrics.items()
                              if isinstance(v, (int, float))}

                # Parse training timestamp string to datetime, then to unix timestamp
                from datetime import datetime
                try:
                    if isinstance(model.training_timestamp, str):
                        dt = datetime.fromisoformat(model.training_timestamp.replace('Z', '+00:00'))
                        training_timestamp = int(dt.timestamp())
                    else:
                        training_timestamp = int(model.training_timestamp.timestamp())
                except (ValueError, AttributeError):
                    # Fallback to current time if parsing fails
                    training_timestamp = int(datetime.now().timestamp())

                model_info = ModelInfo(
                    model_id=model.model_id,
                    version=model.version,
                    algorithm=model.algorithm,
                    training_timestamp=training_timestamp,
                    model_size_bytes=model.model_size_bytes,
                    performance_metrics=metrics,
                    status="ready"  # Default status, could be enhanced
                )
                model_infos.append(model_info)

            return ListModelsResponse(
                models=model_infos,
                total_count=len(model_infos)
            )

        except Exception as e:
            logger.error(f"Failed to list models: {e}")
            return ListModelsResponse(models=[], total_count=0)

    def _load_model_from_storage(self, model_id: str, version: Optional[str], risk_service) -> bool:
        """
        Load model from storage manager.

        Args:
            model_id: Model identifier
            version: Model version (optional)
            risk_service: Risk service to update with loaded model

        Returns:
            True if successful
        """
        try:
            with self._model_lock:
                success = risk_service.model.load_model_from_storage(
                    self.storage_manager, model_id, version
                )
                if success:
                    risk_service.model_loaded = True
                    risk_service.current_model_id = model_id
                    risk_service.current_model_version = version or "latest"

                    # Update our tracking
                    self.current_model_id = model_id
                    self.current_model_version = version or "latest"

                    logger.info(f"Model loaded from storage: {model_id} v{self.current_model_version}")
                return success
        except Exception as e:
            version_display = version or "latest"
            logger.error(f"Failed to load model from storage {model_id} v{version_display}: {e}")
            return False

    def get_current_model_info(self) -> Dict[str, Any]:
        """Get information about the currently loaded model."""
        return {
            'model_id': self.current_model_id,
            'version': self.current_model_version,
            'loaded': self.current_model_id is not None
        }