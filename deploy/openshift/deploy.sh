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

launch_central "$K8S_DIR" "$PREVENT_IMAGE"

launch_sensor "$K8S_DIR" "$CLUSTER" "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$RUNTIME_SUPPORT"
