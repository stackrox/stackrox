#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $K8S_DIR/launch.sh
source $K8S_DIR/env.sh

if [[ -z $NON_INTERACTIVE ]]; then
  read -p "Review the above variables and hit enter to continue: "
fi
kubectl create ns "$NAMESPACE" || true

launch_central "$ROX_CENTRAL_DASHBOARD_PORT" "$LOCAL_API_ENDPOINT" "$K8S_DIR" "$PREVENT_IMAGE" "$NAMESPACE"