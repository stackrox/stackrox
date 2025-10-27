"""
Training endpoints for ML Risk Service REST API.
"""

import logging
from typing import Dict, Any
from fastapi import APIRouter, HTTPException, Depends, status, Query

from src.api.schemas import (
    TrainModelRequest,
    TrainModelResponse,
    QuickTestPipelineResponse,
    ErrorResponse
)
from src.services.training_service import TrainingService
from src.services.risk_service import RiskPredictionService

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/training", tags=["training"])

# Global service instances (in production, use dependency injection)
_training_service = None
_risk_service = None


def get_training_service() -> TrainingService:
    """Get training service instance."""
    global _training_service
    if _training_service is None:
        _training_service = TrainingService()
    return _training_service


def get_risk_service() -> RiskPredictionService:
    """Get risk prediction service instance."""
    global _risk_service
    if _risk_service is None:
        _risk_service = RiskPredictionService()
    return _risk_service


@router.post(
    "/train",
    response_model=TrainModelResponse,
    responses={
        400: {"model": ErrorResponse, "description": "Invalid training data"},
        500: {"model": ErrorResponse, "description": "Training failed"}
    },
    summary="Train ML model",
    description="Train a new risk ranking model with provided training data"
)
async def train_model(
    request: TrainModelRequest,
    training_service: TrainingService = Depends(get_training_service),
    risk_service: RiskPredictionService = Depends(get_risk_service)
) -> TrainModelResponse:
    """
    Train a new risk ranking model.

    This endpoint trains a new ML model using the provided training examples.
    The training process includes:
    - Data validation and preprocessing
    - Feature extraction and ranking dataset creation
    - Model training with cross-validation
    - Performance evaluation and feature importance analysis

    - **training_data**: List of training examples with features and target scores
    - **config_override**: Optional JSON configuration overrides

    Returns training metrics, model version, and feature importance rankings.
    """
    try:
        if not request.training_data:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="No training data provided"
            )

        if len(request.training_data) < 10:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Insufficient training data (minimum 10 examples required)"
            )

        if len(request.training_data) > 10000:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Training data too large (maximum 10,000 examples)"
            )

        logger.info(f"Starting model training with {len(request.training_data)} examples")
        response = training_service.train_model(request, risk_service)

        if response.success:
            logger.info(f"Model training completed successfully. Version: {response.model_version}")
        else:
            logger.error(f"Model training failed: {response.error_message}")

        return response

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Training request failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Training failed: {str(e)}"
        )


@router.get(
    "/status/{job_id}",
    response_model=Dict[str, Any],
    responses={
        404: {"model": ErrorResponse, "description": "Training job not found"},
        501: {"model": ErrorResponse, "description": "Feature not implemented"}
    },
    summary="Get training job status",
    description="Get status and progress of a training job"
)
async def get_training_status(
    job_id: str,
    training_service: TrainingService = Depends(get_training_service)
) -> Dict[str, Any]:
    """
    Get training job status and progress.

    This endpoint would typically return the status of a long-running training job
    submitted asynchronously. Currently returns a placeholder response.

    - **job_id**: Unique identifier for the training job

    Note: Asynchronous training is not yet implemented.
    """
    # This would require implementing async training with job tracking
    return {
        "error": "Async training status tracking not yet implemented",
        "detail": f"Status for training job {job_id} not available",
        "recommendation": "Use synchronous training endpoint /training/train"
    }


@router.post(
    "/sample-data",
    response_model=Dict[str, Any],
    responses={
        400: {"model": ErrorResponse, "description": "Invalid parameters"},
        500: {"model": ErrorResponse, "description": "Sample data generation failed"}
    },
    summary="Generate sample training data",
    description="Generate synthetic training data for testing and development"
)
async def generate_sample_data(
    num_examples: int = Query(100, ge=10, le=1000, description="Number of examples to generate"),
    training_service: TrainingService = Depends(get_training_service)
) -> Dict[str, Any]:
    """
    Generate sample training data for testing.

    This endpoint creates synthetic deployment data with realistic risk patterns
    for testing the training pipeline and API endpoints.

    - **num_examples**: Number of training examples to generate (10-1000)

    Returns the path to generated data file and statistics.
    """
    try:
        logger.info(f"Generating {num_examples} sample training examples")
        sample_file = training_service.generate_sample_training_data(num_examples)

        return {
            "success": True,
            "sample_file": sample_file,
            "examples_generated": num_examples,
            "message": f"Generated {num_examples} sample training examples"
        }

    except Exception as e:
        logger.error(f"Sample data generation failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Sample data generation failed: {str(e)}"
        )


@router.get(
    "/info",
    response_model=Dict[str, Any],
    summary="Get training service info",
    description="Get information about training service status and capabilities"
)
async def get_training_info(
    training_service: TrainingService = Depends(get_training_service)
) -> Dict[str, Any]:
    """
    Get training service information and status.

    Returns information about the training pipeline, last training run,
    and service capabilities.
    """
    try:
        status_info = training_service.get_training_status()

        return {
            "service_name": "ML Risk Training Service",
            "version": "1.0.0",
            "status": "ready",
            "capabilities": [
                "Learning-to-rank model training",
                "Feature importance analysis",
                "Cross-validation evaluation",
                "Sample data generation"
            ],
            "supported_algorithms": [
                "LightGBM Ranker",
                "Random Forest",
                "Gradient Boosting"
            ],
            "training_status": status_info
        }

    except Exception as e:
        logger.error(f"Failed to get training info: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to retrieve training service information"
        )


@router.post(
    "/quick-test",
    response_model=QuickTestPipelineResponse,
    responses={
        500: {"model": ErrorResponse, "description": "Test pipeline execution failed"}
    },
    summary="Run quick test pipeline",
    description="Execute a complete test of the ML training pipeline with sample data"
)
async def run_quick_test_pipeline(
    training_service: TrainingService = Depends(get_training_service)
) -> QuickTestPipelineResponse:
    """
    Run a quick test of the complete ML training pipeline.

    This endpoint performs a comprehensive test of the training system by:

    1. **Sample Data Generation**: Creates 50 synthetic training examples with realistic patterns
    2. **Full Pipeline Execution**: Runs complete training workflow including:
       - Data loading and validation
       - Feature extraction from deployments and images
       - Model training with cross-validation
       - Performance evaluation and metrics calculation
       - Feature importance analysis
    3. **Automatic Cleanup**: Removes temporary files after completion
    4. **Comprehensive Results**: Returns detailed metrics and execution status

    **Use Cases:**
    - Validate training system functionality after deployment
    - Test new training configurations before production use
    - Verify pipeline performance and identify potential issues
    - Generate sample metrics for monitoring setup

    **Note:** This operation may take 30-60 seconds to complete as it runs
    the full training workflow. The endpoint will return detailed results
    including training metrics, execution time, and any errors encountered.

    Returns comprehensive test results including pipeline status, metrics,
    and execution time. If any stage fails, detailed error information is provided.
    """
    try:
        logger.info("Quick test pipeline requested via REST API")

        # Execute the quick test pipeline
        response = training_service.run_quick_test_pipeline()

        # Log the result
        if response.success:
            logger.info(f"Quick test pipeline completed successfully in {response.execution_time_seconds:.2f}s")
        else:
            logger.error(f"Quick test pipeline failed: {response.error_message}")

        return response

    except Exception as e:
        logger.error(f"Quick test pipeline request failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to execute quick test pipeline: {str(e)}"
        )