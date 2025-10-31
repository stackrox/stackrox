# ML Risk Service for StackRox

A machine learning service for ranking deployment risk using configurable features extracted from deployment and image information. The system runs as a separate pod alongside Central and provides explainable risk rankings with comprehensive model management capabilities.

## Overview

This service implements a machine learning-based approach to risk assessment that:

- **Ranks deployments** based on configurable risk features
- **Extracts features** from deployment and image data matching StackRox risk multipliers
- **Provides explanations** using SHAP values for feature importance
- **Learns from data** to improve risk assessment over time
- **Reproduces existing risk scores** initially for seamless migration
- **Runs as separate pod** for independent scaling and updates
- **Manages model lifecycle** with versioning, deployment, and drift monitoring
- **Integrates seamlessly** with StackRox Central via REST API

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   StackRox      │    │   ML Risk        │    │   Model Storage │
│   Central       │◄──►│   Service        │◄──►│                 │
│                 │    │                  │    │ - Local FS      │
│ - Deployments   │    │ - Feature        │    │ - Google Cloud  │
│ - Images        │    │   Extraction     │    │ - Versioning    │
│ - Policies      │    │ - ML Model       │    │ - Metadata      │
│ - Risk Manager  │    │ - REST API       │    └─────────────────┘
│ - ML Scorer     │    │ - Explanations   │              │
└─────────────────┘    │ - Model Registry │    ┌─────────────────┐
         │              │ - Drift Monitor  │◄──►│   Training      │
         │              └──────────────────┘    │   Pipeline      │
         │                        │             │                 │
         │              ┌──────────────────┐    │ - Data Loader   │
         │              │   Management     │    │ - Model Trainer │
         └─────────────►│   API            │    │ - Validation    │
                        │ - Model Deploy   │    │ - Auto Deploy   │
                        │ - Health Check   │    └─────────────────┘
                        │ - Drift Reports  │
                        └──────────────────┘
```

## Features

### 1. Configurable Feature Extraction

The service extracts features that mirror StackRox's existing risk multipliers:

**Deployment Features:**
- Policy violations (count and severity)
- Process baseline violations
- Host access (network, PID, IPC)
- Container security (privileged containers)
- Port exposure and network reachability
- Service account permissions
- Deployment configuration

**Image Features:**
- Vulnerability counts by severity
- CVSS scores (average and maximum)
- Component counts (total and risky)
- Image age and metadata
- Base image information

### 2. Machine Learning Models

- **RandomForest Regressor**: Primary model for risk scoring (extensible architecture)
- **Sklearn Models**: Fallback support
- **Feature Normalization**: Matches StackRox's normalization
- **Group-based Ranking**: Ranks deployments within clusters

### 3. Explainable AI

- **SHAP Values**: Feature importance for individual predictions
- **Global Importance**: Model-wide feature significance
- **Category Analysis**: Risk by feature category
- **Human Explanations**: Natural language summaries

### 4. Model Management & Lifecycle

- **Model Registry**: Centralized model versioning and metadata
- **Automated Deployment**: Deploy models with validation and rollback
- **Health Monitoring**: Continuous model performance tracking
- **Drift Detection**: Monitor model performance degradation
- **A/B Testing**: Compare model versions in production
- **Model Lineage**: Track model evolution and relationships

### 5. Storage Backends

- **Local Filesystem**: Default storage for development and small deployments
- **Google Cloud Storage**: Scalable cloud storage with encryption support
- **Model Compression**: Automatic compression to reduce storage costs
- **Backup & Recovery**: Automated backup with configurable retention
- **Encryption**: At-rest encryption for sensitive models

### 6. Training Pipeline

- **Baseline Reproduction**: Recreates existing StackRox scores
- **JSON Data Input**: Flexible training data format
- **Validation Framework**: Ensures model quality
- **Incremental Learning**: Updates with new data
- **Auto-Deployment**: Automatic model deployment based on quality metrics

## Quick Start

### Prerequisites

- **UV Package Manager**: Install UV for fast, reliable Python package management
  ```bash
  curl -LsSf https://astral.sh/uv/install.sh | sh
  ```
- **Docker & Docker Compose**: For containerized deployment
- **Make**: For build automation

### Local Development Setup

1. **Clone repository and setup environment:**
   ```bash
   git clone <repository>
   cd ml-risk-service

   # Set up development environment with UV
   make setup-dev
   ```

2. **Generate sample data and train model:**
   ```bash
   make generate-sample-data
   make train-model
   ```

3. **Run tests and checks:**
   ```bash
   make check  # Runs lint, typecheck, and tests
   ```

### Using Docker Compose (Recommended for Integration Testing)

1. **Build and start services:**
   ```bash
   make docker-build
   make compose-up
   ```

2. **Access services:**
   - REST API: `http://localhost:8090`
   - API Docs: `http://localhost:8090/docs`
   - Health/Metrics: `http://localhost:8081`
   - Grafana Dashboard: `http://localhost:3000` (admin/admin)

### Using Kubernetes

1. **Deploy to Kubernetes:**
   ```bash
   make k8s-deploy
   ```

2. **Check status:**
   ```bash
   make k8s-status
   ```

3. **Port forward for testing:**
   ```bash
   make k8s-port-forward
   ```

## Configuration

### Environment Variables

#### Core Service Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `REST_PORT` | 8090 | REST API service port |
| `HEALTH_PORT` | 8081 | Health check port |
| `LOG_LEVEL` | INFO | Logging level |
| `CONFIG_FILE` | `/app/config/feature_config.yaml` | Feature configuration |
| `MODEL_FILE` | - | Pre-trained model file |

#### Model Storage Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `ROX_ML_MODEL_STORAGE_BACKEND` | `local` | Storage backend (`local`, `gcs`) |
| `ROX_ML_MODEL_STORAGE_BASE_PATH` | `/app/models` | Base path for model storage |
| `ROX_ML_MODEL_BACKUP_ENABLED` | `false` | Enable automatic model backup |
| `ROX_ML_MODEL_BACKUP_FREQUENCY` | `daily` | Backup frequency (`hourly`, `daily`, `weekly`) |
| `ROX_ML_MODEL_ENCRYPTION_ENABLED` | `false` | Enable encryption at rest |
| `ROX_ML_MODEL_COMPRESSION_ENABLED` | `true` | Enable model compression |
| `ROX_ML_MODEL_RETENTION_DAYS` | `0` | Model retention in days (0 = forever) |
| `ROX_ML_MODEL_VERSIONING_ENABLED` | `true` | Enable model versioning |
| `ROX_ML_MAX_MODEL_VERSIONS` | `10` | Maximum versions per model |

#### Google Cloud Storage Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `ROX_ML_GCS_PROJECT_ID` | - | GCS project ID |
| `ROX_ML_GCS_CREDENTIALS_PATH` | - | Path to GCS service account credentials |
| `ROX_ML_GCS_BUCKET_NAME` | - | GCS bucket name for model storage |

#### Model Deployment Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `ROX_ML_MODEL_AUTO_DEPLOY_ENABLED` | `false` | Enable automatic model deployment |
| `ROX_ML_MODEL_DEPLOYMENT_THRESHOLD` | `0.85` | Quality threshold for auto-deployment |
| `ROX_ML_MODEL_HEALTH_CHECK_ENABLED` | `true` | Enable model health monitoring |
| `ROX_ML_MODEL_HEALTH_CHECK_INTERVAL` | `5m` | Health check interval |
| `ROX_ML_MODEL_DRIFT_DETECTION_ENABLED` | `false` | Enable drift detection |
| `ROX_ML_MODEL_DRIFT_THRESHOLD` | `0.1` | Drift alert threshold |

#### Central API Integration
| Variable | Default | Description |
|----------|---------|-------------|
| `TRAINING_CENTRAL_ENDPOINT` | - | Central endpoint for training data collection |
| `TRAINING_CENTRAL_API_TOKEN` | - | API token for training Central authentication |
| `PREDICTION_CENTRAL_ENDPOINT` | - | Central endpoint for prediction validation |
| `PREDICTION_CENTRAL_API_TOKEN` | - | API token for prediction Central authentication |

**Note:** The ML Risk Service can connect to two separate Central instances:
- **Training Central**: Used to collect deployment data for model training
- **Prediction Central**: Used to validate model predictions against real risk scores

Both use the same authentication configuration format (API token or mTLS). Configuration is managed via environment variables or `src/config/feature_config.yaml`.

**Example usage:**
```bash
# Training Central (for model training)
export TRAINING_CENTRAL_ENDPOINT="https://central.training.example.com"
export TRAINING_CENTRAL_API_TOKEN="your-training-api-token"

# Prediction Central (for validation)
export PREDICTION_CENTRAL_ENDPOINT="https://central.production.example.com"
export PREDICTION_CENTRAL_API_TOKEN="your-prediction-api-token"
```

**Breaking Change:** Previous versions used `CENTRAL_API_TOKEN` and `CENTRAL_ENDPOINT`. These have been renamed to `TRAINING_CENTRAL_*` for consistency with the dual-Central architecture.

### Feature Configuration

Edit `src/config/feature_config.yaml` to configure:

- **Feature weights**: Adjust importance of different risk factors
- **Model parameters**: RandomForest hyperparameters (extensible for future algorithms)
- **Normalization settings**: Saturation and max values
- **Training settings**: Batch size, iterations, early stopping

Example:
```yaml
features:
  deployment:
    policy_violations:
      enabled: true
      weight: 1.0
      normalize_saturation: 50
      max_value: 4.0
    host_network:
      enabled: true
      weight: 0.7
```

## API Reference

### REST API Endpoints

The service provides a comprehensive REST API for all operations. Access the interactive API documentation at `http://localhost:8090/docs` when the service is running.

#### Model Registry
- `GET /api/v1/ml/models` - List all models
- `GET /api/v1/ml/models/{modelId}` - List versions for a model
- `GET /api/v1/ml/models/{modelId}/versions/{version}` - Get specific model
- `DELETE /api/v1/ml/models/{modelId}/versions/{version}` - Delete model version

#### Model Deployment
- `GET /api/v1/ml/deployment/current` - Get currently deployed model
- `POST /api/v1/ml/deployment` - Deploy a model version
- `POST /api/v1/ml/deployment/rollback` - Rollback to previous version

#### Model Management
- `POST /api/v1/ml/models/{modelId}/versions/{version}/promote` - Promote to ready status
- `POST /api/v1/ml/models/{modelId}/versions/{version}/deprecate` - Mark as deprecated
- `POST /api/v1/ml/models/{modelId}/versions/{version}/reload` - Hot reload model
- `GET /api/v1/ml/models/list` - List models from ML service storage
- `GET /api/v1/ml/models/status/{status}` - Filter models by status

#### Model Analytics
- `GET /api/v1/ml/models/{modelId}/versions/{version}/lineage` - Get model lineage
- `GET /api/v1/ml/models/{modelId}/versions/compare?version1=v1&version2=v2` - Compare versions
- `GET /api/v1/ml/models/{modelId}/metrics/{metric}/history` - Get metric history
- `GET /api/v1/ml/models/{modelId}/versions/{version}/validate` - Validate for production

#### Training & Health
- `POST /api/v1/ml/training/trigger` - Trigger model training
- `GET /api/v1/ml/training/status` - Get training status
- `GET /api/v1/ml/health` - Basic model health
- `GET /api/v1/ml/health/detailed?include_trends=true&trend_hours=24` - Detailed health
- `GET /api/v1/ml/stats` - Registry statistics

#### Drift Monitoring
- `GET /api/v1/ml/drift/report?model_id=X&version=Y&period_hours=24` - Get drift report
- `GET /api/v1/ml/drift/alerts?severity=medium` - Get active drift alerts
- `POST /api/v1/ml/drift/baseline` - Set drift baseline

### Health Endpoints

- `GET /health` - Basic health check
- `GET /ready` - Readiness check (model loaded)
- `GET /metrics` - Prometheus metrics
- `GET /status` - Detailed status information

## Training Data Format

Training data should be provided as JSON:

```json
{
  "deployments": [
    {
      "deployment": {
        "id": "deployment-1",
        "name": "web-app",
        "namespace": "production",
        "replicas": 3,
        "host_network": false,
        "containers": [...]
      },
      "images": [
        {
          "id": "image-1",
          "name": {"registry": "docker.io", "remote": "nginx"},
          "scan": {
            "components": [...]
          }
        }
      ],
      "alerts": [
        {
          "policy": {"name": "High Risk Policy", "severity": "HIGH_SEVERITY"}
        }
      ],
      "current_risk_score": 2.5
    }
  ]
}
```

## Development

### Prerequisites

- Python 3.11+
- Docker
- Kubernetes (optional)
- Make

### Setup Development Environment

The project uses [UV](https://github.com/astral-sh/uv) for fast, reliable Python package management.

```bash
# Basic setup (core dependencies only)
make setup

# Development setup (includes dev tools: pytest, black, mypy, etc.)
make setup-dev

# Full ML setup (includes scikit-multilearn, shap, matplotlib)
make setup-ml
```

### Running Tests and Checks

```bash
# Run all checks (recommended)
make check

# Individual commands
make test         # Run pytest with coverage
make lint         # Run flake8 and pylint
make typecheck    # Run mypy type checking
make format       # Format code with black and isort
```

### Package Management with UV

```bash
# Add new dependency
uv add requests

# Add development dependency
uv add --dev pytest-mock

# Add optional ML dependency
uv add --optional ml matplotlib

# Sync dependencies
uv sync

# Update dependencies
uv lock --upgrade
```

### Training a Model

```bash
# Generate sample data
make generate-sample-data

# Train model
python -c "
from training.train_pipeline import TrainingPipeline
pipeline = TrainingPipeline()
results = pipeline.run_full_pipeline('sample_training_data.json')
print(f'Training completed: {results[\"success\"]}')
"
```

### Local Development with Docker

```bash
# Build and run locally
make docker-build
make docker-run

# Check logs
make docker-logs

# Test health
make health-check

# Stop
make docker-stop
```

## StackRox Integration

### Environment Variables for Central

Configure Central to use the ML service:

```bash
# Core ML Service Configuration
export ROX_ML_RISK_SERVICE_ENABLED=true
export ROX_ML_RISK_SERVICE_ENDPOINT=ml-risk-service:8080
export ROX_ML_RISK_SERVICE_TLS=false
export ROX_ML_RISK_SERVICE_TIMEOUT=30s

# Model Storage Configuration
export ROX_ML_MODEL_STORAGE_BACKEND=gcs
export ROX_ML_GCS_PROJECT_ID=your-project-id
export ROX_ML_GCS_BUCKET_NAME=stackrox-ml-models
export ROX_ML_MODEL_VERSIONING_ENABLED=true

# Model Deployment Configuration
export ROX_ML_MODEL_AUTO_DEPLOY_ENABLED=true
export ROX_ML_MODEL_DEPLOYMENT_THRESHOLD=0.85
export ROX_ML_MODEL_HEALTH_CHECK_ENABLED=true
export ROX_ML_MODEL_DRIFT_DETECTION_ENABLED=true
```

### Integration Modes

1. **Disabled** (default): Use traditional risk scoring only
2. **Augmented**: Combine traditional and ML scores
3. **Replacement**: Use ML scores exclusively (with fallback)

### ML Scorer Integration

The ML scorer integrates seamlessly with StackRox's deployment risk assessment:

```go
// Create ML scorer
mlScorer := deployment.NewMLScorer()

// Score a deployment using ML
risk := mlScorer.Score(ctx, deployment, imageRisks)

// Check ML status
if deployment.IsMLEnabled() {
    health, err := deployment.GetMLHealthStatus(ctx)
    // Handle health status
}
```

### Risk Manager Integration

The ML service integrates with StackRox's risk manager:

```go
// Create manager with ML integration
manager := risk.CreateManagerBasedOnConfig(datastore, scorer)

// Check if ML is enabled
if mlManager, ok := manager.(*risk.ManagerWithML); ok {
    health, err := mlManager.GetMLHealthStatus(ctx)
    // Handle health status
}
```

### Model Management Integration

Central provides REST API endpoints for model management:

```bash
# List models
curl -X GET "https://central.stackrox.io/api/v1/ml/models"

# Deploy a model
curl -X POST "https://central.stackrox.io/api/v1/ml/deployment" \
  -H "Content-Type: application/json" \
  -d '{"model_id": "stackrox-risk-model", "version": "v1.2.3"}'

# Get drift report
curl -X GET "https://central.stackrox.io/api/v1/ml/drift/report?period_hours=24"
```

## Monitoring

### Metrics

The service exposes Prometheus metrics:

#### Core Performance Metrics
- `ml_risk_predictions_total` - Total predictions made
- `ml_risk_prediction_duration_seconds` - Prediction latency
- `ml_risk_batch_predictions_total` - Batch predictions made
- `ml_risk_feature_extraction_duration_seconds` - Feature extraction time

#### Model Management Metrics
- `ml_risk_model_loaded` - Model loaded status
- `ml_risk_model_deployments_total` - Model deployment count
- `ml_risk_model_rollbacks_total` - Model rollback count
- `ml_risk_model_validation_duration_seconds` - Model validation time
- `ml_risk_model_size_bytes` - Current model size
- `ml_risk_model_age_seconds` - Age of current model

#### Health & Performance Metrics
- `ml_risk_health_checks_total` - Health check count
- `ml_risk_health_check_duration_seconds` - Health check latency
- `ml_risk_memory_usage_bytes` - Memory usage
- `ml_risk_cpu_usage_percent` - CPU usage
- `ml_risk_storage_operations_total` - Storage operation count
- `ml_risk_storage_errors_total` - Storage error count

#### Drift Monitoring Metrics
- `ml_risk_drift_score` - Current drift score
- `ml_risk_drift_alerts_total` - Total drift alerts
- `ml_risk_drift_checks_total` - Drift check count
- `ml_risk_drift_baseline_updates_total` - Baseline update count

### Grafana Dashboard

A sample Grafana dashboard is included in `monitoring/grafana/dashboards/`.

### Health Monitoring

The service provides comprehensive health checks:

- **Liveness**: Basic service health
- **Readiness**: Model loaded and ready
- **Detailed Status**: Full system status with metrics

## Troubleshooting

### Common Issues

1. **Model not loading**
   - Check `MODEL_FILE` environment variable
   - Verify model file exists and is readable
   - Check logs for training errors
   - Validate storage backend configuration
   - Ensure GCS credentials are correctly configured

2. **Storage backend issues**
   - For GCS: Verify `ROX_ML_GCS_PROJECT_ID` and credentials
   - Check bucket permissions and existence
   - Validate network connectivity to cloud storage
   - Review storage backend logs

3. **Model deployment failures**
   - Check model validation status before deployment
   - Verify model meets deployment threshold requirements
   - Review model registry health status
   - Check for sufficient resources

4. **High memory usage**
   - Reduce model complexity in config
   - Limit training data size
   - Adjust container memory limits
   - Enable model compression
   - Review model versioning retention settings

5. **Poor prediction accuracy**
   - Validate training data quality
   - Check feature normalization
   - Review baseline reproduction metrics
   - Monitor drift detection alerts
   - Compare with previous model versions

6. **REST API connection failures**
   - Verify network policies allow HTTP traffic
   - Check service discovery configuration
   - Validate TLS/SSL settings
   - Review timeout configurations
   - Check Central to ML service connectivity

7. **Drift detection issues**
   - Ensure drift baseline is properly set
   - Check drift threshold configuration
   - Validate incoming data quality
   - Review drift alert frequency settings

8. **Performance issues**
   - Monitor prediction latency metrics
   - Check feature extraction performance
   - Review batch processing efficiency
   - Validate storage I/O performance

### Debugging

```bash
# Check pod logs
make k8s-logs

# Get shell in pod
make k8s-shell

# Check service status
curl http://localhost:8081/status

# Test prediction via REST API
make test-prediction
```

## Production Deployment

### Resource Requirements

#### ML Risk Service Pod
- **CPU**: 500m request, 2000m limit
- **Memory**: 1Gi request, 4Gi limit
- **Storage**: 10Gi for local models and cache

#### Model Storage (GCS Recommended)
- **Storage**: Unlimited cloud storage
- **IOPS**: High-performance storage for model loading
- **Bandwidth**: Sufficient for model downloads and uploads
- **Backup**: Automatic versioning and retention

### Security Considerations

#### Container Security
- Runs as non-root user (UID 1001)
- Read-only root filesystem
- Network policies restrict traffic
- No privilege escalation
- Drops all capabilities
- Uses distroless base image

#### Data Security
- Model encryption at rest (configurable)
- TLS for all HTTP communications
- Service account authentication for GCS
- Secrets management for credentials
- Audit logging for model access

#### Network Security
- mTLS between Central and ML service
- Network policies restrict pod-to-pod communication
- Ingress/egress rules for cloud storage access
- VPC peering for cloud deployments

### Scaling

#### Horizontal Scaling
- Multiple ML service replicas
- Shared model storage (GCS/ReadWriteMany PVC)
- Load balancing via Kubernetes service
- Auto-scaling based on CPU/memory usage

#### Vertical Scaling
- Increase CPU/memory for larger models
- Optimize JVM heap size for model loading
- Adjust batch sizes for processing

#### Storage Scaling
- Use cloud storage (GCS) for unlimited capacity
- Implement model compression to reduce storage costs
- Configure retention policies for automatic cleanup
- Monitor storage costs and usage

### High Availability

#### Service Availability
- Deploy multiple replicas across availability zones
- Use pod disruption budgets
- Health checks and readiness probes
- Graceful shutdown handling

#### Data Availability
- Cloud storage with multi-region replication
- Model backup and versioning
- Disaster recovery procedures
- Monitoring and alerting

### Backup and Recovery

#### Model Backup
- Automatic backup to cloud storage
- Configurable retention policies
- Point-in-time recovery capabilities
- Cross-region replication for disaster recovery

#### Configuration Backup
- ConfigMaps and Secrets in version control
- Helm charts for reproducible deployments
- Infrastructure as Code (Terraform/Pulumi)
- Environment-specific configurations

#### Monitoring and Alerting
- Prometheus metrics collection
- Grafana dashboards for visualization
- Alertmanager for critical alerts
- Log aggregation and analysis

### Performance Optimization

#### Model Performance
- Model quantization and compression
- Batch prediction optimization
- Feature caching strategies
- Warm-up procedures for new models

#### Storage Performance
- Use SSD storage for local caching
- Optimize cloud storage configuration
- Implement model pre-loading
- Monitor storage I/O patterns

#### Network Performance
- Optimize HTTP connection pooling
- Use efficient serialization formats (JSON)
- Implement request caching where appropriate
- Monitor network latency and throughput

## Contributing

1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Run `make ci-pipeline` before submitting

## License

[StackRox License]