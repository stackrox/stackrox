"""
Risk prediction endpoints for ML Risk Service REST API.
"""

import logging
from typing import Dict, Any
from fastapi import APIRouter, HTTPException, Depends, status
from fastapi.responses import JSONResponse

from src.api.schemas import (
    DeploymentRiskRequest,
    DeploymentRiskResponse,
    BatchDeploymentRiskRequest,
    BatchDeploymentRiskResponse,
    ErrorResponse
)
from src.api.dependencies import get_prediction_service

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/prediction", tags=["prediction"])


@router.post(
    "/deployment",
    response_model=DeploymentRiskResponse,
    responses={
        500: {"model": ErrorResponse, "description": "Prediction failed"},
        503: {"model": ErrorResponse, "description": "Model not available"}
    },
    summary="Predict deployment risk",
    description="Get risk score and feature importance for a single deployment"
)
async def predict_deployment_risk(
    request: DeploymentRiskRequest,
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> DeploymentRiskResponse:
    """
    Predict risk score for a single deployment.

    This endpoint analyzes deployment and image features to calculate a risk score
    and provides feature importance explanations for interpretability.

    - **deployment_id**: Unique identifier for the deployment
    - **deployment_features**: Security and configuration features of the deployment
    - **image_features**: Vulnerability and component information for container images

    Returns risk score (higher = more risk) and feature importance rankings.
    """
    try:
        if not prediction_service.is_model_loaded():
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail="No trained model available"
            )

        response = prediction_service.predict_deployment_risk(request)
        return response

    except ValueError as e:
        logger.error(f"Prediction failed for deployment {request.deployment_id}: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=str(e)
        )
    except Exception as e:
        logger.error(f"Internal error during prediction for {request.deployment_id}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error during prediction"
        )


@router.post(
    "/batch",
    response_model=BatchDeploymentRiskResponse,
    responses={
        500: {"model": ErrorResponse, "description": "Batch prediction failed"},
        503: {"model": ErrorResponse, "description": "Model not available"}
    },
    summary="Predict batch deployment risk",
    description="Get risk scores for multiple deployments in a single request"
)
async def predict_batch_deployment_risk(
    request: BatchDeploymentRiskRequest,
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> BatchDeploymentRiskResponse:
    """
    Predict risk scores for multiple deployments in batch.

    This endpoint processes multiple deployment risk requests efficiently.
    Individual failures are handled gracefully - failed predictions return
    risk_score=0.0 and model_version="error".

    - **requests**: List of deployment risk requests

    Returns list of risk predictions with same ordering as input.
    """
    try:
        if not prediction_service.is_model_loaded():
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail="No trained model available"
            )

        if not request.requests:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="No deployment requests provided"
            )

        if len(request.requests) > 100:  # Reasonable batch size limit
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Batch size too large (max 100 deployments)"
            )

        response = prediction_service.predict_batch_deployment_risk(request)
        return response

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Internal error during batch prediction: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error during batch prediction"
        )


@router.get(
    "/explain/{deployment_id}",
    response_model=Dict[str, Any],
    responses={
        404: {"model": ErrorResponse, "description": "Explanation not found"},
        501: {"model": ErrorResponse, "description": "Feature not implemented"}
    },
    summary="Get prediction explanation",
    description="Get detailed explanation for a previous risk prediction"
)
async def get_prediction_explanation(
    deployment_id: str,
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> Dict[str, Any]:
    """
    Get detailed explanation for a risk prediction.

    This endpoint provides SHAP values and detailed feature contributions
    for a previously computed risk score.

    Note: This endpoint is not yet fully implemented and returns a placeholder.
    """
    # This would typically require storing prediction explanations
    # and retrieving them by deployment_id
    return JSONResponse(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        content={
            "error": "Prediction explanation retrieval not yet implemented",
            "detail": f"Explanation for deployment {deployment_id} not available"
        }
    )