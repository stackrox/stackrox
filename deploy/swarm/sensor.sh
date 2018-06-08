#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $SWARM_DIR/launch.sh
source $SWARM_DIR/env.sh

launch_sensor "$SWARM_DIR" "$PREVENT_IMAGE" "$CLUSTER" "$CLUSTER_API_ENDPOINT" "$LOCAL_API_ENDPOINT" "$PREVENT_DISABLE_REGISTRY_AUTH"
