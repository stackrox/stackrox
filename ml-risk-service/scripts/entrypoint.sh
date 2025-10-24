#!/bin/bash
set -e

# Default values
GRPC_PORT=${GRPC_PORT:-8080}
HEALTH_PORT=${HEALTH_PORT:-8081}
MODEL_FILE=${MODEL_FILE:-""}
CONFIG_FILE=${CONFIG_FILE:-"/app/config/feature_config.yaml"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}

# Setup logging
export PYTHONPATH=/app

# Start the service
echo "Starting ML Risk Service..."
echo "gRPC Port: $GRPC_PORT"
echo "Health Port: $HEALTH_PORT"
echo "Model File: $MODEL_FILE"
echo "Config File: $CONFIG_FILE"
echo "Log Level: $LOG_LEVEL"

# Generate protobuf code if needed (in production, this would be pre-generated)
echo "Ensuring protobuf code is available..."

# Start the gRPC server
exec python -m src.api.grpc_server \
    --config "$CONFIG_FILE" \
    --port "$GRPC_PORT" \
    --workers 10 \
    ${MODEL_FILE:+--model "$MODEL_FILE"}