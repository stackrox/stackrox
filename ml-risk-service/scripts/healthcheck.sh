#!/bin/bash
# Health check script for container
HEALTH_PORT=${HEALTH_PORT:-8081}

# Check if health endpoint responds
curl -f "http://localhost:$HEALTH_PORT/health" || exit 1

# Check if ready endpoint responds
curl -f "http://localhost:$HEALTH_PORT/ready" || exit 1

echo "Health check passed"