# Multi-stage Dockerfile for ML Risk Service
FROM python:3.11-slim as builder

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies needed for building
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    libc6-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Production stage
FROM python:3.11-slim as production

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/app/src

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libgomp1 \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r mlrisk \
    && useradd -r -g mlrisk -s /bin/false mlrisk

# Copy Python packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Create app directory structure
WORKDIR /app
RUN mkdir -p /app/models /app/training /app/reports /app/logs \
    && chown -R mlrisk:mlrisk /app

# Copy application code
COPY src/ /app/src/
COPY training/ /app/training/
COPY proto/ /app/proto/

# Copy configuration
COPY src/config/feature_config.yaml /app/config/

# Create entrypoint script
RUN cat > /app/entrypoint.sh << 'EOF'
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
EOF

RUN chmod +x /app/entrypoint.sh \
    && chown mlrisk:mlrisk /app/entrypoint.sh

# Create health check script
RUN cat > /app/healthcheck.sh << 'EOF'
#!/bin/bash
# Health check script for container
HEALTH_PORT=${HEALTH_PORT:-8081}

# Check if health endpoint responds
curl -f "http://localhost:$HEALTH_PORT/health" || exit 1

# Check if ready endpoint responds
curl -f "http://localhost:$HEALTH_PORT/ready" || exit 1

echo "Health check passed"
EOF

RUN chmod +x /app/healthcheck.sh

# Switch to non-root user
USER mlrisk

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD /app/healthcheck.sh

# Default entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Labels
LABEL maintainer="StackRox ML Team" \
      description="ML Risk Ranking Service for StackRox" \
      version="1.0.0"