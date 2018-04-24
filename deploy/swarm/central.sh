#!/usr/bin/env bash
set -e

SWARM_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $SWARM_DIR/launch.sh

if [[ -z $LOCAL_API_ENDPOINT ]]; then
    read -p "Enter local api endpoint: " LOCAL_API_ENDPOINT
fi
echo "LOCAL_API_ENDPOINT set to $LOCAL_API_ENDPOINT"

export PREVENT_IMAGE="stackrox/prevent:${PREVENT_IMAGE_TAG:-latest}"
echo "PREVENT_IMAGE set to $PREVENT_IMAGE"

NAMESPACE=${NAMESPACE:-stackrox}
echo "NAMESPACE set to $NAMESPACE"

read -p "Review the above variables and hit enter to continue: "

launch_central "$SWARM_DIR" "$PREVENT_IMAGE" "$NAMESPACE" "$LOCAL_API_ENDPOINT" "$PREVENT_DISABLE_REGISTRY_AUTH"
