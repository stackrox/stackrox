#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh

export NAMESPACE="${NAMESPACE:-stackrox}"
echo "Kubernetes namespace set to $NAMESPACE"
oc create ns "$NAMESPACE" || true

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"
echo

export PREVENT_IMAGE="stackrox/prevent:${PREVENT_IMAGE_TAG:-latest}"

launch_central "$K8S_DIR" "$PREVENT_IMAGE" "$NAMESPACE"

launch_sensor "localhost:8000" "remote" "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$NAMESPACE"

