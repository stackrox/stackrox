#!/bin/bash
set -e

# Default values
HEALTH_PORT=${HEALTH_PORT:-8081}
REST_PORT=${REST_PORT:-8090}
MODEL_FILE=${MODEL_FILE:-""}
CONFIG_FILE=${CONFIG_FILE:-"/app/config/feature_config.yaml"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}

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
echo "Health Port: $HEALTH_PORT"
echo "REST Port: $REST_PORT"
echo "Model File: $MODEL_FILE"
echo "Config File: $CONFIG_FILE"
echo "Log Level: $LOG_LEVEL"

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
    if [ ! -z "$REST_PID" ]; then
        echo "Stopping REST API server (PID $REST_PID)..."
        kill -TERM "$REST_PID" 2>/dev/null || true
    fi
    wait
    echo "ML Risk Service stopped"
    exit 0
}

# Set up signal handlers
trap shutdown SIGTERM SIGINT

# Start REST server
start_rest_server

echo "ML Risk Service fully started"
echo "- REST API: http://localhost:$REST_PORT"
echo "- API Docs: http://localhost:$REST_PORT/docs"
echo "- Health: http://localhost:$HEALTH_PORT"

# Wait for all background processes
wait