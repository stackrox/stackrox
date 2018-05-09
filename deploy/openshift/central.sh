#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh

export NAMESPACE="${NAMESPACE:-stackrox}"
echo "Kubernetes namespace set to $NAMESPACE"

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"
echo

if [[ -z $NON_INTERACTIVE ]]; then
  read -p "Review the above variables and hit enter to continue: "
fi
oc create ns "$NAMESPACE" || true

launch_central "$K8S_DIR" "$PREVENT_IMAGE" "$NAMESPACE"