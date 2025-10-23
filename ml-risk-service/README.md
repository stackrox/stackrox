# ML Risk Service for StackRox

A machine learning service for ranking deployment risk using configurable features extracted from deployment and image information. The system runs as a separate pod alongside Central and provides explainable risk rankings.

## Overview

This service implements a machine learning-based approach to risk assessment that:

- **Ranks deployments** based on configurable risk features
- **Extracts features** from deployment and image data matching StackRox risk multipliers
- **Provides explanations** using SHAP values for feature importance
- **Learns from data** to improve risk assessment over time
- **Reproduces existing risk scores** initially for seamless migration
- **Runs as separate pod** for independent scaling and updates

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   StackRox      │    │   ML Risk        │    │   Training      │
│   Central       │◄──►│   Service        │◄──►│   Pipeline      │
│                 │    │                  │    │                 │
│ - Deployments   │    │ - Feature        │    │ - Data Loader   │
│ - Images        │    │   Extraction     │    │ - Model Trainer │
│ - Policies      │    │ - ML Model       │    │ - Validation    │
│ - Risk Manager  │    │ - gRPC API       │    │                 │
└─────────────────┘    │ - Explanations   │    └─────────────────┘
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

- **LightGBM Ranker**: Primary model for learning-to-rank
- **Sklearn Models**: Fallback support
- **Feature Normalization**: Matches StackRox's normalization
- **Group-based Ranking**: Ranks deployments within clusters

### 3. Explainable AI

- **SHAP Values**: Feature importance for individual predictions
- **Global Importance**: Model-wide feature significance
- **Category Analysis**: Risk by feature category
- **Human Explanations**: Natural language summaries

### 4. Training Pipeline

- **Baseline Reproduction**: Recreates existing StackRox scores
- **JSON Data Input**: Flexible training data format
- **Validation Framework**: Ensures model quality
- **Incremental Learning**: Updates with new data

## Quick Start

### Using Docker Compose (Recommended for Development)

1. **Clone and build:**
   ```bash
   git clone <repository>
   cd ml-risk-service
   make docker-build
   ```

2. **Generate sample data and train model:**
   ```bash
   make generate-sample-data
   make train-model
   ```

3. **Start services:**
   ```bash
   make compose-up
   ```

4. **Access services:**
   - ML Service gRPC: `localhost:8080`
   - Health/Metrics: `localhost:8081`
   - Grafana Dashboard: `localhost:3000` (admin/admin)

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

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_PORT` | 8080 | gRPC service port |
| `HEALTH_PORT` | 8081 | Health check port |
| `LOG_LEVEL` | INFO | Logging level |
| `CONFIG_FILE` | `/app/config/feature_config.yaml` | Feature configuration |
| `MODEL_FILE` | - | Pre-trained model file |

### Feature Configuration

Edit `src/config/feature_config.yaml` to configure:

- **Feature weights**: Adjust importance of different risk factors
- **Model parameters**: LightGBM hyperparameters
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

### gRPC Service

The service provides the following gRPC methods:

#### GetDeploymentRisk
Get risk score for a single deployment with feature explanations.

#### GetBatchDeploymentRisk
Get risk scores for multiple deployments efficiently.

#### TrainModel
Train the model with new data.

#### GetModelHealth
Get model health and performance metrics.

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

```bash
make dev-setup
```

### Running Tests

```bash
make test
make lint
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
export ROX_ML_RISK_SERVICE_ENABLED=true
export ROX_ML_RISK_SERVICE_ENDPOINT=ml-risk-service:8080
export ROX_ML_RISK_MODE_ENABLED=true
```

### Integration Modes

1. **Disabled** (default): Use traditional risk scoring only
2. **Augmented**: Combine traditional and ML scores
3. **Replacement**: Use ML scores exclusively (with fallback)

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

## Monitoring

### Metrics

The service exposes Prometheus metrics:

- `ml_risk_predictions_total` - Total predictions made
- `ml_risk_prediction_duration_seconds` - Prediction latency
- `ml_risk_model_loaded` - Model loaded status
- `ml_risk_memory_usage_bytes` - Memory usage
- `ml_risk_cpu_usage_percent` - CPU usage

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

2. **High memory usage**
   - Reduce model complexity in config
   - Limit training data size
   - Adjust container memory limits

3. **Poor prediction accuracy**
   - Validate training data quality
   - Check feature normalization
   - Review baseline reproduction metrics

4. **gRPC connection failures**
   - Verify network policies allow traffic
   - Check service discovery configuration
   - Validate TLS settings

### Debugging

```bash
# Check pod logs
make k8s-logs

# Get shell in pod
make k8s-shell

# Check service status
curl http://localhost:8081/status

# Test prediction (requires grpcurl)
make test-prediction
```

## Production Deployment

### Resource Requirements

- **CPU**: 500m request, 2000m limit
- **Memory**: 1Gi request, 4Gi limit
- **Storage**: 10Gi for models and data

### Security Considerations

- Runs as non-root user
- Read-only root filesystem
- Network policies restrict traffic
- No privilege escalation
- Drops all capabilities

### Scaling

- Horizontal: Multiple replicas with shared model storage
- Vertical: Increase CPU/memory for larger models
- Model storage: Use ReadWriteMany PVC for model sharing

### Backup and Recovery

- Model files stored in persistent volumes
- Training data backed up separately
- Configuration in ConfigMaps and version control

## Contributing

1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Run `make ci-pipeline` before submitting

## License

[StackRox License]