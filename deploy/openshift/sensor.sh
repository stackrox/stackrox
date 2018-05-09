#!/usr/bin/env bash

set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh

if [[ -z $CLUSTER ]]; then
    read -p "Enter cluster name to create: " CLUSTER
fi
echo "CLUSTER set to $CLUSTER"

CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "CLUSTER_API_ENDPOINT set to $CLUSTER_API_ENDPOINT"

LOCAL_API_ENDPOINT="${LOCAL_API_ENDPOINT:-localhost:8000}"

NAMESPACE=${NAMESPACE:-stackrox}
echo "NAMESPACE set to $NAMESPACE"

PREVENT_IMAGE_TAG=${PREVENT_IMAGE_TAG:-latest}
PREVENT_IMAGE=${PREVENT_IMAGE:-stackrox/prevent:$PREVENT_IMAGE_TAG}
echo "PREVENT_IMAGE set to $PREVENT_IMAGE"

if [[ -z $NON_INTERACTIVE ]]; then
  read -p "Review the above variables and hit enter to continue: "
fi

oc create ns "$NAMESPACE" || true

launch_sensor "$LOCAL_API_ENDPOINT" "$CLUSTER" "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "$NAMESPACE"
