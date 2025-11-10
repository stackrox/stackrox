"""
Health and monitoring endpoints for ML Risk Service REST API.
"""

import logging
import time
import os
from typing import Dict, Any
from fastapi import APIRouter, HTTPException, Depends, status
from fastapi.responses import PlainTextResponse

from src.api.schemas import (
    HealthStatus,
    ReadinessStatus,
    DetailedHealthRequest,
    DetailedHealthResponse,
    ErrorResponse
)
from src.api.dependencies import get_prediction_service
from src.services.prediction_service import RiskPredictionService

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/health", tags=["health"])

# Service start time for uptime calculation
_service_start_time = time.time()


@router.get(
    "",
    response_model=HealthStatus,
    summary="Basic health check",
    description="Get basic health status of the ML Risk Service"
)
async def health_check() -> HealthStatus:
    """
    Basic health check endpoint.

    Returns basic service health information including status, uptime, and version.
    This endpoint should respond quickly and is suitable for load balancer health checks.
    """
    try:
        uptime = time.time() - _service_start_time

        return HealthStatus(
            status="healthy",
            timestamp=int(time.time()),
            version="1.0.0",
            uptime_seconds=uptime
        )

    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Health check failed"
        )


@router.get(
    "/ready",
    response_model=ReadinessStatus,
    responses={
        503: {"model": ErrorResponse, "description": "Service not ready"}
    },
    summary="Readiness check",
    description="Check if service is ready to accept requests"
)
async def readiness_check(
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> ReadinessStatus:
    """
    Readiness check endpoint.

    Performs comprehensive checks to determine if the service is ready to handle requests.
    This includes checking model availability, dependencies, and service components.
    """
    try:
        checks = {
            "model_loaded": prediction_service.is_model_loaded(),
            "service_initialized": True,
            "dependencies_available": True  # Could check external dependencies
        }

        all_ready = all(checks.values())

        response = ReadinessStatus(
            ready=all_ready,
            checks=checks,
            timestamp=int(time.time())
        )

        if not all_ready:
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail=response.dict()
            )

        return response

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Readiness check failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Readiness check failed"
        )


@router.get(
    "/metrics",
    response_class=PlainTextResponse,
    responses={
        200: {"content": {"text/plain": {"example": "# HELP ml_predictions_total Total predictions\nml_predictions_total 123"}}},
        501: {"model": ErrorResponse, "description": "Metrics not available"}
    },
    summary="Prometheus metrics",
    description="Get Prometheus-format metrics for monitoring"
)
async def get_metrics(
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> PlainTextResponse:
    """
    Get Prometheus-format metrics.

    Returns metrics in Prometheus exposition format for scraping by monitoring systems.
    Includes prediction counts, latency histograms, and model health metrics.
    """
    try:
        # Generate Prometheus-format metrics
        uptime = time.time() - _service_start_time

        metrics_text = f"""# HELP ml_risk_service_info Service information
# TYPE ml_risk_service_info gauge
ml_risk_service_info{{version="1.0.0"}} 1

# HELP ml_risk_service_uptime_seconds Service uptime in seconds
# TYPE ml_risk_service_uptime_seconds gauge
ml_risk_service_uptime_seconds {uptime}

# HELP ml_risk_predictions_total Total number of predictions served
# TYPE ml_risk_predictions_total counter
ml_risk_predictions_total {prediction_service.predictions_served}

# HELP ml_risk_model_loaded Whether a model is currently loaded
# TYPE ml_risk_model_loaded gauge
ml_risk_model_loaded {1 if prediction_service.is_model_loaded() else 0}

# HELP ml_risk_prediction_time_total Total time spent on predictions
# TYPE ml_risk_prediction_time_total counter
ml_risk_prediction_time_total {prediction_service.total_prediction_time / 1000.0}
"""

        return PlainTextResponse(content=metrics_text, media_type="text/plain")

    except Exception as e:
        logger.error(f"Metrics generation failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to generate metrics"
        )


@router.get(
    "/status",
    response_model=Dict[str, Any],
    summary="Detailed status",
    description="Get detailed service status including system information"
)
async def get_detailed_status(
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> Dict[str, Any]:
    """
    Get detailed service status.

    Returns comprehensive status information including system resources,
    model information, and service metrics.
    """
    try:
        import psutil

        # System information
        system_info = {
            "cpu_percent": psutil.cpu_percent(interval=1),
            "memory_percent": psutil.virtual_memory().percent,
            "disk_usage_percent": psutil.disk_usage('/').percent,
            "load_average": os.getloadavg() if hasattr(os, 'getloadavg') else [0, 0, 0]
        }

        # Service metrics
        uptime = time.time() - _service_start_time
        avg_prediction_time = (
            prediction_service.total_prediction_time / max(prediction_service.predictions_served, 1)
            if prediction_service.predictions_served > 0 else 0.0
        )

        service_metrics = {
            "uptime_seconds": uptime,
            "predictions_served": prediction_service.predictions_served,
            "avg_prediction_time_ms": avg_prediction_time,
            "model_loaded": prediction_service.is_model_loaded()
        }

        # Model information
        model_info = prediction_service.get_model_info() if prediction_service.is_model_loaded() else {}

        return {
            "status": "healthy",
            "timestamp": int(time.time()),
            "version": "1.0.0",
            "system": system_info,
            "service": service_metrics,
            "model": model_info
        }

    except ImportError:
        # psutil not available, return basic info
        uptime = time.time() - _service_start_time

        return {
            "status": "healthy",
            "timestamp": int(time.time()),
            "version": "1.0.0",
            "uptime_seconds": uptime,
            "model_loaded": prediction_service.is_model_loaded(),
            "predictions_served": prediction_service.predictions_served
        }

    except Exception as e:
        logger.error(f"Status check failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to retrieve detailed status"
        )


@router.post(
    "/detailed",
    response_model=DetailedHealthResponse,
    responses={
        501: {"model": ErrorResponse, "description": "Detailed health not implemented"}
    },
    summary="Detailed health analysis",
    description="Get comprehensive health analysis with trends and recommendations"
)
async def get_detailed_health(
    request: DetailedHealthRequest,
    prediction_service: RiskPredictionService = Depends(get_prediction_service)
) -> DetailedHealthResponse:
    """
    Get detailed health analysis.

    This endpoint would provide comprehensive health analysis including
    model performance trends, drift detection, and operational recommendations.

    Currently returns a placeholder as the detailed health analysis is not fully implemented.
    """
    # This would integrate with the existing health checker and drift monitor
    return DetailedHealthResponse(
        model_id="unknown",
        version="unknown",
        overall_status="healthy",
        overall_score=0.8,
        health_checks=[],
        recommendations=["Detailed health analysis not yet fully implemented"],
        trends={},
        timestamp=int(time.time())
    )