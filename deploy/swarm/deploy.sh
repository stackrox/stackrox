#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $SWARM_DIR/launch.sh

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.prevent_net:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export NAMESPACE="stackrox"
export CLUSTER="remote"

launch_central "$SWARM_DIR" "$PREVENT_IMAGE" "$NAMESPACE" "$LOCAL_API_ENDPOINT" "$PREVENT_DISABLE_REGISTRY_AUTH"

launch_sensor "$SWARM_DIR" "$PREVENT_IMAGE" "$CLUSTER" "$NAMESPACE" "$CLUSTER_API_ENDPOINT" "$LOCAL_API_ENDPOINT" "$PREVENT_DISABLE_REGISTRY_AUTH"

