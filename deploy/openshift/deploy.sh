#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh
source $K8S_DIR/env.sh

export CLUSTER=${CLUSTER:-remote}
echo "CLUSTER set to $CLUSTER"

export RUNTIME_SUPPORT=${RUNTIME_SUPPORT:-false}
echo "RUNTIME_SUPPORT set to $RUNTIME_SUPPORT"

launch_central "$ROX_CENTRAL_DASHBOARD_PORT" "$LOCAL_API_ENDPOINT" "$K8S_DIR" docker-registry.default.svc:5000 "$PREVENT_IMAGE_TAG" "$NAMESPACE"

launch_sensor "$LOCAL_API_ENDPOINT" "$CLUSTER" docker-registry.default.svc:5000 "$PREVENT_IMAGE_TAG" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$NAMESPACE"

