"""
Training endpoints for ML Risk Service REST API.
"""

import logging
from typing import Dict, Any, Optional
from fastapi import APIRouter, HTTPException, Depends, status, Query

from src.api.schemas import (
    TrainModelRequest,
    TrainModelResponse,
    ErrorResponse
)
from src.services.training_service import TrainingService
from src.services.risk_service import RiskPredictionService
from src.training.data_loader import TrainingDataLoader

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

    This endpoint trains a new ML model using the provided training samples.
    The training process includes:
    - Data validation and preprocessing
    - Feature extraction and ranking dataset creation
    - Model training with cross-validation
    - Performance evaluation and feature importance analysis

    - **training_data**: List of training samples with features and target scores
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
                detail="Insufficient training data (minimum 10 samples required)"
            )

        if len(request.training_data) > 10000:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Training data too large (maximum 10,000 samples)"
            )

        logger.info(f"Starting model training with {len(request.training_data)} samples")
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
    num_examples: int = Query(100, ge=10, le=1000, description="Number of samples to generate"),
    training_service: TrainingService = Depends(get_training_service)
) -> Dict[str, Any]:
    """
    Generate sample training data for testing.

    This endpoint creates synthetic deployment data with realistic risk patterns
    for testing the training pipeline and API endpoints.

    - **num_examples**: Number of training samples to generate (10-1000)

    Returns the path to generated data file and statistics.
    """
    try:
        logger.info(f"Generating {num_examples} sample training samples")
        sample_file = training_service.generate_sample_training_data(num_examples)

        return {
            "success": True,
            "sample_file": sample_file,
            "examples_generated": num_examples,
            "message": f"Generated {num_examples} sample training samples"
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
                "Random Forest Regressor",
                "Extensible for future algorithms"
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
    limit: int = Query(10, description="Number of training samples to collect"),
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
        logger.info(f"Collecting {limit} training samples from Central (last {days_back} days)")

        # Collect sample data using streaming approach with limit
        training_samples = []
        streaming_data = data_loader.load_from_central_api_streaming_with_config(
            config_path=None,
            filters=filters
        )

        for example in streaming_data:
            training_samples.append(example)
            if len(training_samples) >= limit:
                break

        # Analyze collected data
        if training_samples:
            sample_example = training_samples[0]
            feature_count = len(sample_example.get('features', {}))

            # Get some basic statistics
            risk_scores = [ex.get('risk_score', 0) for ex in training_samples]
            avg_risk_score = sum(risk_scores) / len(risk_scores) if risk_scores else 0

            clusters = set(ex.get('export_metadata', {}).get('cluster_id', 'unknown')
                          for ex in training_samples)
            namespaces = set(ex.get('export_metadata', {}).get('namespace', 'unknown')
                           for ex in training_samples)

            return {
                "success": True,
                "message": f"Successfully collected {len(training_samples)} training samples",
                "data_summary": {
                    "samples_collected": len(training_samples),
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
                "message": "No training samples found with current filters",
                "data_summary": {
                    "samples_collected": 0,
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
    "/central/train-full",
    response_model=TrainModelResponse,
    responses={
        400: {"model": ErrorResponse, "description": "Invalid parameters or Central not enabled"},
        500: {"model": ErrorResponse, "description": "Training failed"}
    },
    summary="Train model with all data from Central API",
    description="Perform complete model training using data collected from Central's workloads API"
)
async def train_full_from_central(
    limit: Optional[int] = Query(None, ge=10, le=10000, description="Maximum training samples to collect"),
    include_inactive: bool = Query(False, description="Include inactive deployments"),
    severity_threshold: str = Query("MEDIUM_SEVERITY", description="Minimum severity threshold"),
    clusters: Optional[str] = Query(None, description="Comma-separated cluster IDs to filter"),
    namespaces: Optional[str] = Query(None, description="Comma-separated namespaces to filter"),
    config_override: Optional[str] = Query("", description="JSON configuration overrides"),
    test_mode: bool = Query(False, description="Use optimized settings for quick testing (limit=50)"),
    training_service: TrainingService = Depends(get_training_service),
    risk_service: RiskPredictionService = Depends(get_risk_service)
) -> TrainModelResponse:
    """
    Train a complete ML model using all available data from Central API.

    This endpoint performs a comprehensive training workflow:

    1. **Data Collection**: Streams ALL workload data from Central's `/v1/export/vuln-mgmt/workloads` endpoint
    2. **Data Processing**: Converts Central API format to training format with feature extraction
    3. **Model Training**: Uses existing training pipeline with cross-validation and evaluation
    4. **Model Storage**: Saves trained model with version management and metadata

    **Parameters:**
    - **limit**: Optional limit on training samples to prevent memory issues
    - **include_inactive**: Whether to include inactive/stopped deployments
    - **severity_threshold**: Minimum vulnerability severity to include (LOW_SEVERITY, MEDIUM_SEVERITY, HIGH_SEVERITY, CRITICAL_SEVERITY)
    - **clusters**: Optional comma-separated list of cluster IDs to filter (default: all clusters)
    - **namespaces**: Optional comma-separated list of namespaces to filter (default: all namespaces)
    - **config_override**: Optional JSON string with training configuration overrides
    - **test_mode**: Enable quick testing mode (sets limit=50 for fast validation)

    **Data Collection:**
    - Collects ALL deployments from Central (no time-based filtering)
    - Filter by cluster/namespace to focus on specific environments
    - Use limit parameter to cap the number of training samples

    **Use Cases:**
    - **Production Training**: Train models with complete enterprise workload data
    - **Scheduled Retraining**: Periodic model updates with all available data
    - **Environment-Specific Training**: Train models for specific clusters or namespaces
    - **Quick Testing**: Use test_mode=true for rapid validation of training pipeline

    **Note:** This operation may take several minutes for large datasets. The endpoint
    will return detailed training metrics, model version, and feature importance analysis.

    Returns comprehensive training results including metrics, model version, and feature importance.
    """
    try:
        from src.config.central_config import CentralConfig

        # Validate Central API is enabled
        config = CentralConfig()
        if not config.is_enabled():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Central API integration is not enabled in configuration"
            )

        # Validate configuration
        is_valid, issues = config.validate_configuration()
        if not is_valid:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Central API configuration invalid: {'; '.join(issues)}"
            )

        # Build filters for data collection (no date filtering - collect all deployments)
        filters = {
            'include_inactive': include_inactive,
            'severity_threshold': severity_threshold
        }

        # Override parameters for test mode
        if test_mode:
            logger.info("Test mode enabled - using optimized settings for quick validation")
            limit = 50  # Small sample for quick testing

        # Add optional filters
        if clusters:
            filters['clusters'] = [c.strip() for c in clusters.split(',') if c.strip()]
        if namespaces:
            filters['namespaces'] = [n.strip() for n in namespaces.split(',') if n.strip()]

        logger.info(f"Starting full model training from Central API")
        logger.info(f"Filters: {filters}")
        logger.info(f"Limit: {limit}")

        # Train model using Central API data
        response = training_service.train_model_from_central(
            filters=filters,
            limit=limit,
            config_override=config_override if config_override else None,
            risk_service=risk_service
        )

        if response.success:
            logger.info(f"Central API model training completed successfully. Version: {response.model_version}")
        else:
            logger.error(f"Central API model training failed: {response.error_message}")

        return response

    except HTTPException:
        raise
    except ImportError as e:
        logger.error(f"Central API components not available: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Central API integration components not available"
        )
    except Exception as e:
        logger.error(f"Central API training request failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Training failed: {str(e)}"
        )


@router.post(
    "/central/validate-predictions",
    response_model=Dict[str, Any],
    responses={
        400: {"model": ErrorResponse, "description": "Invalid parameters or prediction Central not configured"},
        500: {"model": ErrorResponse, "description": "Validation failed"}
    },
    summary="Validate predictions against prediction Central",
    description="Validate model predictions by comparing against actual risk scores from a prediction Central instance"
)
async def validate_predictions_from_central(
    model_id: str = Query("stackrox-risk-model", description="Model ID to validate"),
    model_version: Optional[str] = Query(None, description="Model version (defaults to latest)"),
    days_back: int = Query(7, ge=1, le=90, description="Number of days back to collect validation data"),
    limit: int = Query(50, ge=10, le=500, description="Number of deployments to validate"),
    include_inactive: bool = Query(False, description="Include inactive deployments"),
    severity_threshold: str = Query("MEDIUM_SEVERITY", description="Minimum severity threshold"),
    clusters: Optional[str] = Query(None, description="Comma-separated cluster IDs to filter"),
    namespaces: Optional[str] = Query(None, description="Comma-separated namespaces to filter"),
    risk_service: RiskPredictionService = Depends(get_risk_service)
) -> Dict[str, Any]:
    """
    Validate model predictions against actual risk scores from prediction Central.

    This endpoint performs prediction validation by:

    1. **Load Model**: Loads the specified trained model from storage
    2. **Connect to Prediction Central**: Uses PREDICTION_CENTRAL_* environment variables
    3. **Collect Deployments**: Pulls deployments from prediction Central
    4. **Run Predictions**: Generates risk score predictions for each deployment
    5. **Compare Scores**: Compares predicted vs actual risk scores from Central
    6. **Calculate Metrics**: Computes MAE, RMSE, correlation, NDCG, and accuracy metrics

    **Parameters:**
    - **model_id**: Model identifier to validate
    - **model_version**: Specific model version (defaults to latest if not specified)
    - **days_back**: Number of days back to collect validation data (1-90)
    - **limit**: Maximum number of deployments to validate (10-500)
    - **include_inactive**: Whether to include inactive/stopped deployments
    - **severity_threshold**: Minimum vulnerability severity to include
    - **clusters**: Optional comma-separated list of cluster IDs to filter
    - **namespaces**: Optional comma-separated list of namespaces to filter

    **Use Cases:**
    - **Model Validation**: Evaluate model performance on production data
    - **A/B Testing**: Compare different model versions
    - **Monitoring**: Track model accuracy over time
    - **Pre-Deployment**: Validate models before production deployment

    **Note:** This requires separate configuration for prediction Central instance using
    PREDICTION_CENTRAL_ENDPOINT and PREDICTION_CENTRAL_API_TOKEN environment variables.

    Returns validation metrics including MAE, RMSE, correlation, NDCG, and detailed prediction results.
    """
    try:
        from src.config.central_config import CentralConfig
        from src.clients.central_export_client import CentralExportClient
        from src.services.central_export_service import CentralExportService

        # Validate prediction Central configuration
        prediction_config = CentralConfig.from_prediction_env()

        if not prediction_config.is_enabled():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Prediction Central API integration is not enabled. "
                       "Set PREDICTION_CENTRAL_ENDPOINT and PREDICTION_CENTRAL_API_TOKEN environment variables."
            )

        # Validate configuration
        is_valid, issues = prediction_config.validate_configuration()
        if not is_valid:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Prediction Central API configuration invalid: {'; '.join(issues)}"
            )

        # Get the current model
        model = risk_service.model
        if model is None or model.model is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="No trained model available. Train a model first using /training/central/train-full"
            )

        # Create prediction Central client
        client_config = prediction_config.get_client_config()
        auth_config = client_config.pop('authentication')
        endpoint = client_config.pop('endpoint')

        prediction_client = CentralExportClient(
            endpoint=endpoint,
            auth_token=auth_config.get('token', ''),
            config=client_config
        )

        # Build filters for validation data collection
        filters = {
            'days_back': days_back,
            'include_inactive': include_inactive,
            'severity_threshold': severity_threshold
        }

        # Add optional filters
        if clusters:
            filters['clusters'] = [c.strip() for c in clusters.split(',') if c.strip()]
        if namespaces:
            filters['namespaces'] = [n.strip() for n in namespaces.split(',') if n.strip()]

        logger.info(f"Starting prediction validation against prediction Central")
        logger.info(f"Model: {model_id} version: {model_version or 'latest'}")
        logger.info(f"Filters: {filters}")
        logger.info(f"Limit: {limit}")

        # Create a temporary Central export service for training Central (to access validate_predictions method)
        # We don't actually use the training Central client - just need the service class
        from src.config.central_config import create_central_client_from_config
        training_client = create_central_client_from_config()
        training_service = CentralExportService(client=training_client, config={})

        # Run validation
        validation_results = training_service.validate_predictions(
            model=model,
            prediction_client=prediction_client,
            filters=filters,
            limit=limit
        )

        # Clean up
        prediction_client.close()
        training_client.close()

        # Prepare response
        response = {
            "success": True,
            "model_id": model_id,
            "model_version": model.model_version or "unknown",
            "validation_summary": {
                "total_samples": validation_results['total_samples'],
                "successful_predictions": validation_results['successful_predictions'],
                "failed_predictions": validation_results['failed_predictions'],
                "success_rate": round(
                    validation_results['successful_predictions'] / validation_results['total_samples'] * 100, 2
                ) if validation_results['total_samples'] > 0 else 0
            },
            "metrics": {
                "mae": round(validation_results.get('mae', 0.0), 4),
                "rmse": round(validation_results.get('rmse', 0.0), 4),
                "correlation": round(validation_results.get('correlation', 0.0), 4),
                "ndcg": round(validation_results.get('ndcg', 0.0), 4),
                "within_30_percent": round(validation_results.get('within_30_percent', 0.0), 2),
                "mean_actual_score": round(validation_results.get('mean_actual_score', 0.0), 4),
                "mean_predicted_score": round(validation_results.get('mean_predicted_score', 0.0), 4)
            },
            "filters_used": filters,
            "prediction_central_endpoint": prediction_config.get_endpoint(),
            "predictions": validation_results.get('predictions', [])[:20]  # Return first 20 for brevity
        }

        logger.info(f"Validation complete: MAE={response['metrics']['mae']:.4f}, "
                   f"Correlation={response['metrics']['correlation']:.4f}, "
                   f"NDCG={response['metrics']['ndcg']:.4f}")

        return response

    except HTTPException:
        raise
    except ImportError as e:
        logger.error(f"Central API components not available: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Central API integration components not available"
        )
    except Exception as e:
        logger.error(f"Prediction validation request failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Validation failed: {str(e)}"
        )