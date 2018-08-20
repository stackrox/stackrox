#!/usr/bin/env bash

set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh
source $K8S_DIR/env.sh

export RUNTIME_SUPPORT=${RUNTIME_SUPPORT:-false}
echo "RUNTIME_SUPPORT set to $RUNTIME_SUPPORT"

launch_sensor "$LOCAL_API_ENDPOINT" "$CLUSTER" "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$RUNTIME_SUPPORT"
