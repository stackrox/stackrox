"""
Shared service instances for dependency injection across all API routers.

This module provides a single source of truth for service instances used
across the API. All routers import their dependencies from here to ensure
they share the same service instances.

This is critical for ensuring that when a model is trained, all endpoints
immediately have access to the newly trained model without requiring
manual reloads or restarts.
"""

from typing import Optional
from src.services.prediction_service import RiskPredictionService
from src.services.training_service import RiskTrainingService

# Global shared instances - initialized on first use
_prediction_service: Optional[RiskPredictionService] = None
_training_service: Optional[RiskTrainingService] = None


def get_prediction_service() -> RiskPredictionService:
    """
    Get the shared risk prediction service instance.

    All API endpoints use this same instance, ensuring that model
    updates (from training or hot-reloading) are immediately visible
    to all endpoints.

    Returns:
        Shared RiskPredictionService instance
    """
    global _prediction_service
    if _prediction_service is None:
        _prediction_service = RiskPredictionService()
    return _prediction_service


def get_training_service() -> RiskTrainingService:
    """
    Get the shared training service instance.

    Returns:
        Shared RiskTrainingService instance
    """
    global _training_service
    if _training_service is None:
        _training_service = RiskTrainingService()
    return _training_service
