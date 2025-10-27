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
from training.data_loader import TrainingDataLoader

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
    "/central/test-connection",
    response_model=Dict[str, Any],
    summary="Test Central API connection",
    description="Test connection to StackRox Central's workloads export API"
)
async def test_central_connection() -> Dict[str, Any]:
    """
    Test connection to Central API using configured settings.

    Returns connection status and Central version information.
    """
    try:
        from src.config.central_config import CentralConfig, create_central_client_from_config

        # Load configuration
        config = CentralConfig()

        if not config.is_enabled():
            return {
                "success": False,
                "message": "Central API integration is not enabled",
                "enabled": False
            }

        # Validate configuration
        is_valid, issues = config.validate_configuration()
        if not is_valid:
            return {
                "success": False,
                "message": f"Configuration validation failed: {'; '.join(issues)}",
                "enabled": True,
                "configuration_valid": False,
                "issues": issues
            }

        # Create and test client
        client = create_central_client_from_config()
        connection_test = client.test_connection()

        # Get capabilities
        capabilities = client.get_export_capabilities()

        client.close()

        return {
            "success": connection_test['success'],
            "message": connection_test['message'],
            "enabled": True,
            "configuration_valid": True,
            "central_version": connection_test.get('central_version', 'unknown'),
            "endpoint": config.get_endpoint(),
            "auth_method": config.get_authentication_config()['method'],
            "capabilities": capabilities
        }

    except ImportError as e:
        logger.error(f"Central API components not available: {e}")
        return {
            "success": False,
            "message": "Central API integration components not available",
            "enabled": False,
            "error": str(e)
        }
    except Exception as e:
        logger.error(f"Central connection test failed: {e}")
        return {
            "success": False,
            "message": f"Connection test failed: {str(e)}",
            "enabled": True,
            "error": str(e)
        }


@router.post(
    "/central/collect-sample",
    response_model=Dict[str, Any],
    summary="Collect sample training data from Central",
    description="Collect sample training data using Central's workloads export API"
)
async def collect_sample_from_central(
    limit: int = Query(10, description="Number of training examples to collect"),
    days_back: int = Query(7, description="Number of days back to look for data")
) -> Dict[str, Any]:
    """
    Collect a small sample of training data from Central's workloads API.

    This endpoint uses the /v1/export/vuln-mgmt/workloads endpoint to collect
    deployments with their associated images and vulnerability data efficiently.
    """
    try:
        from src.config.central_config import CentralConfig

        # Check if Central integration is enabled
        config = CentralConfig()
        if not config.is_enabled():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Central API integration is not enabled"
            )

        # Create data loader
        data_loader = TrainingDataLoader()

        # Set up filters for sample collection
        filters = {
            'days_back': days_back,
            'include_inactive': False,
            'severity_threshold': 'MEDIUM_SEVERITY'
        }

        # Collect sample data
        logger.info(f"Collecting {limit} training examples from Central (last {days_back} days)")

        # Collect sample data using streaming approach with limit
        training_examples = []
        streaming_data = data_loader.load_from_central_api_streaming_with_config(
            config_path=None,
            filters=filters
        )

        for example in streaming_data:
            training_examples.append(example)
            if len(training_examples) >= limit:
                break

        # Analyze collected data
        if training_examples:
            sample_example = training_examples[0]
            feature_count = len(sample_example.get('features', {}))

            # Get some basic statistics
            risk_scores = [ex.get('risk_score', 0) for ex in training_examples]
            avg_risk_score = sum(risk_scores) / len(risk_scores) if risk_scores else 0

            clusters = set(ex.get('export_metadata', {}).get('cluster_id', 'unknown')
                          for ex in training_examples)
            namespaces = set(ex.get('export_metadata', {}).get('namespace', 'unknown')
                           for ex in training_examples)

            return {
                "success": True,
                "message": f"Successfully collected {len(training_examples)} training examples",
                "data_summary": {
                    "examples_collected": len(training_examples),
                    "feature_count": feature_count,
                    "avg_risk_score": round(avg_risk_score, 2),
                    "unique_clusters": len(clusters),
                    "unique_namespaces": len(namespaces),
                    "clusters": list(clusters),
                    "namespaces": list(namespaces)
                },
                "sample_features": list(sample_example.get('features', {}).keys())[:10],
                "filters_used": filters
            }
        else:
            return {
                "success": True,
                "message": "No training examples found with current filters",
                "data_summary": {
                    "examples_collected": 0,
                    "feature_count": 0
                },
                "filters_used": filters
            }

    except HTTPException:
        raise
    except ImportError as e:
        logger.error(f"Central API components not available: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Central API integration components not available"
        )
    except Exception as e:
        logger.error(f"Sample collection from Central failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Sample collection failed: {str(e)}"
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