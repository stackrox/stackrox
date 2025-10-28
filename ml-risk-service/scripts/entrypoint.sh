#!/bin/bash
set -e

# Default values
GRPC_PORT=${GRPC_PORT:-8080}
HEALTH_PORT=${HEALTH_PORT:-8081}
REST_PORT=${REST_PORT:-8090}
MODEL_FILE=${MODEL_FILE:-""}
CONFIG_FILE=${CONFIG_FILE:-"/app/config/feature_config.yaml"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}
ENABLE_REST=${ENABLE_REST:-"true"}

# Setup logging and Python environment
export PYTHONPATH=/app
export PATH="/app/.venv/bin:$PATH"

# Verify Python environment
echo "Python environment check:"
echo "Python executable: $(which python)"
echo "Python version: $(python --version)"
echo "Virtual environment: $(/app/.venv/bin/python -c 'import sys; print(sys.prefix)')"

# Start the service
echo "Starting ML Risk Service..."
echo "gRPC Port: $GRPC_PORT"
echo "Health Port: $HEALTH_PORT"
echo "REST Port: $REST_PORT"
echo "Model File: $MODEL_FILE"
echo "Config File: $CONFIG_FILE"
echo "Log Level: $LOG_LEVEL"
echo "Enable REST API: $ENABLE_REST"

# Generate protobuf code if needed (in production, this would be pre-generated)
echo "Ensuring protobuf code is available..."

# Function to start gRPC server
start_grpc_server() {
    echo "Starting gRPC server on port $GRPC_PORT..."
    echo "Using Python: $(which python)"
    /app/.venv/bin/python -m src.api.grpc_server \
        --config "$CONFIG_FILE" \
        --port "$GRPC_PORT" \
        --workers 10 \
        ${MODEL_FILE:+--model "$MODEL_FILE"} &
    GRPC_PID=$!
    echo "gRPC server started with PID $GRPC_PID"
}

# Function to start REST server
start_rest_server() {
    echo "Starting REST API server on port $REST_PORT..."
    echo "Using Python: $(which python)"
    /app/.venv/bin/python -m src.api.rest_server \
        --config "$CONFIG_FILE" \
        --host "0.0.0.0" \
        --port "$REST_PORT" \
        --log-level "$LOG_LEVEL" &
    REST_PID=$!
    echo "REST API server started with PID $REST_PID"
}

# Handle shutdown gracefully
shutdown() {
    echo "Shutting down ML Risk Service..."
    if [ ! -z "$GRPC_PID" ]; then
        echo "Stopping gRPC server (PID $GRPC_PID)..."
        kill -TERM "$GRPC_PID" 2>/dev/null || true
    fi
    if [ ! -z "$REST_PID" ] && [ "$ENABLE_REST" = "true" ]; then
        echo "Stopping REST API server (PID $REST_PID)..."
        kill -TERM "$REST_PID" 2>/dev/null || true
    fi
    wait
    echo "ML Risk Service stopped"
    exit 0
}

# Set up signal handlers
trap shutdown SIGTERM SIGINT

# Start servers
start_grpc_server

if [ "$ENABLE_REST" = "true" ]; then
    start_rest_server
fi

echo "ML Risk Service fully started"
echo "- gRPC API: http://localhost:$GRPC_PORT"
if [ "$ENABLE_REST" = "true" ]; then
    echo "- REST API: http://localhost:$REST_PORT"
    echo "- API Docs: http://localhost:$REST_PORT/docs"
fi

# Wait for all background processes
wait