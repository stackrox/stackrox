"""
Model management endpoints for ML Risk Service REST API.
"""

import logging
from typing import Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, status, Query

from src.api.schemas import (
    ReloadModelRequest,
    ReloadModelResponse,
    ListModelsResponse,
    ModelHealthResponse,
    ErrorResponse
)
from src.services.model_service import ModelManagementService
from src.services.risk_service import RiskPredictionService

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/models", tags=["models"])

# Global service instances (in production, use dependency injection)
_model_service = None
_risk_service = None


def get_model_service() -> ModelManagementService:
    """Get model management service instance."""
    global _model_service
    if _model_service is None:
        _model_service = ModelManagementService()
    return _model_service


def get_risk_service() -> RiskPredictionService:
    """Get risk prediction service instance."""
    global _risk_service
    if _risk_service is None:
        _risk_service = RiskPredictionService()
    return _risk_service


@router.get(
    "",
    response_model=ListModelsResponse,
    summary="List available models",
    description="Get list of all available models or models for a specific model ID"
)
async def list_models(
    model_id: Optional[str] = Query(None, description="Filter by specific model ID"),
    model_service: ModelManagementService = Depends(get_model_service)
) -> ListModelsResponse:
    """
    List available models in storage.

    - **model_id**: Optional filter to show only versions of a specific model

    Returns list of models with metadata including version, algorithm,
    training timestamp, and performance metrics.
    """
    try:
        response = model_service.list_models(model_id)
        return response

    except Exception as e:
        logger.error(f"Failed to list models: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to retrieve model list"
        )


@router.get(
    "/{model_id}",
    response_model=Dict[str, Any],
    responses={
        404: {"model": ErrorResponse, "description": "Model not found"}
    },
    summary="Get specific model info",
    description="Get detailed information about a specific model"
)
async def get_model_info(
    model_id: str,
    model_service: ModelManagementService = Depends(get_model_service)
) -> Dict[str, Any]:
    """
    Get detailed information about a specific model.

    - **model_id**: Unique identifier for the model

    Returns model metadata including available versions, performance metrics,
    and training information.
    """
    try:
        response = model_service.list_models(model_id)

        if not response.models:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Model {model_id} not found"
            )

        # Return the first (latest) version's info plus summary
        latest_model = response.models[0]
        return {
            "model_id": model_id,
            "latest_version": latest_model.version,
            "algorithm": latest_model.algorithm,
            "total_versions": response.total_count,
            "latest_training_timestamp": latest_model.training_timestamp,
            "latest_performance_metrics": latest_model.performance_metrics,
            "status": latest_model.status,
            "all_versions": [model.version for model in response.models]
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to get model info for {model_id}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to retrieve model information"
        )


@router.post(
    "/{model_id}/reload",
    response_model=ReloadModelResponse,
    responses={
        400: {"model": ErrorResponse, "description": "Invalid request"},
        404: {"model": ErrorResponse, "description": "Model not found"},
        500: {"model": ErrorResponse, "description": "Reload failed"}
    },
    summary="Hot reload model",
    description="Reload a model from storage with zero downtime"
)
async def reload_model(
    model_id: str,
    version: Optional[str] = Query("", description="Model version (empty for latest)"),
    force_reload: bool = Query(False, description="Force reload even if already loaded"),
    model_service: ModelManagementService = Depends(get_model_service),
    risk_service: RiskPredictionService = Depends(get_risk_service)
) -> ReloadModelResponse:
    """
    Hot reload a model from storage.

    This endpoint provides zero-downtime model updates by loading a new model
    version while keeping the service running.

    - **model_id**: Unique identifier for the model
    - **version**: Specific version to load (empty for latest)
    - **force_reload**: Force reload even if already loaded

    Returns reload status, timing information, and version details.
    """
    try:
        # Convert empty string to None for latest version
        version_param = version if version else None

        request = ReloadModelRequest(
            model_id=model_id,
            version=version_param,
            force_reload=force_reload
        )

        response = model_service.reload_model(request, risk_service)
        return response

    except Exception as e:
        logger.error(f"Model reload failed for {model_id}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Model reload failed: {str(e)}"
        )


@router.get(
    "/{model_id}/health",
    response_model=ModelHealthResponse,
    responses={
        404: {"model": ErrorResponse, "description": "Model not found"}
    },
    summary="Get model health",
    description="Get health status and performance metrics for a specific model"
)
async def get_model_health(
    model_id: str,
    risk_service: RiskPredictionService = Depends(get_risk_service),
    model_service: ModelManagementService = Depends(get_model_service)
) -> ModelHealthResponse:
    """
    Get health status and metrics for a model.

    - **model_id**: Unique identifier for the model

    Returns health status, performance metrics, and operational statistics.
    """
    try:
        # Check if this is the currently loaded model
        current_info = model_service.get_current_model_info()

        if current_info.get('model_id') != model_id:
            # Model is not currently loaded
            return ModelHealthResponse(
                healthy=False,
                model_version="not_loaded",
                last_training_time=0,
                training_examples_count=0,
                current_metrics={
                    "current_ndcg": 0.0,
                    "current_auc": 0.0,
                    "predictions_served": 0,
                    "avg_prediction_time_ms": 0.0
                }
            )

        # Get health for currently loaded model
        health_response = risk_service.get_model_health()
        return health_response

    except Exception as e:
        logger.error(f"Failed to get health for model {model_id}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to retrieve model health"
        )


@router.get(
    "/current/info",
    response_model=Dict[str, Any],
    summary="Get current model info",
    description="Get information about the currently loaded model"
)
async def get_current_model_info(
    model_service: ModelManagementService = Depends(get_model_service),
    risk_service: RiskPredictionService = Depends(get_risk_service)
) -> Dict[str, Any]:
    """
    Get information about the currently loaded model.

    Returns details about which model is currently active and serving predictions.
    """
    try:
        current_info = model_service.get_current_model_info()
        model_info = risk_service.get_model_info()

        return {
            **current_info,
            "model_details": model_info,
            "is_healthy": risk_service.is_model_loaded()
        }

    except Exception as e:
        logger.error(f"Failed to get current model info: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to retrieve current model information"
        )