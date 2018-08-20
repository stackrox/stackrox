#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $SWARM_DIR/launch.sh
source $SWARM_DIR/env.sh

export CLUSTER=${CLUSTER:-remote}
echo "CLUSTER set to $CLUSTER"

launch_central "$SWARM_DIR" "$PREVENT_IMAGE" "$PREVENT_DISABLE_REGISTRY_AUTH"

launch_sensor "$SWARM_DIR" "$PREVENT_IMAGE" "$CLUSTER" "$CLUSTER_API_ENDPOINT" "$PREVENT_DISABLE_REGISTRY_AUTH"

