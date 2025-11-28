#!/bin/bash
# Health check script for container
REST_PORT=${REST_PORT:-8090}

# Check if health endpoint responds
curl -f "http://localhost:$REST_PORT/api/v1/health" || exit 1

# Check if ready endpoint responds (but don't fail if model isn't loaded yet)
curl -s "http://localhost:$REST_PORT/api/v1/health/ready" > /dev/null || echo "Warning: Service not fully ready"

echo "Health check passed"