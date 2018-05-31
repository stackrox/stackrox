#!/usr/bin/env bash
set -e

export ROX_CENTRAL_DASHBOARD_PORT="${ROX_CENTRAL_DASHBOARD_PORT:-8000}"
echo "Central dashboard port set to $ROX_CENTRAL_DASHBOARD_PORT"

export LOCAL_API_ENDPOINT="${LOCAL_API_ENDPOINT:-"localhost:$ROX_CENTRAL_DASHBOARD_PORT"}"
echo "Local StackRox Prevent endpoint set to $LOCAL_API_ENDPOINT"

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export PREVENT_IMAGE_TAG=${PREVENT_IMAGE_TAG:-$(git describe --tags --abbrev=10 --dirty)}
export PREVENT_IMAGE=${PREVENT_IMAGE:-stackrox/prevent:$PREVENT_IMAGE_TAG}
echo "PREVENT_IMAGE set to $PREVENT_IMAGE"

export NAMESPACE=${NAMESPACE:-stackrox}
echo "Kubernetes namespace set to $NAMESPACE"