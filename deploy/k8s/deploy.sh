#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh
source $K8S_DIR/env.sh


launch_central "$K8S_DIR" "$MAIN_IMAGE"

launch_sensor "$K8S_DIR" "$CLUSTER" "$MAIN_IMAGE" "$CLUSTER_API_ENDPOINT" "$RUNTIME_SUPPORT"

