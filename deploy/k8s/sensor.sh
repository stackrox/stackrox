#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh
source $COMMON_DIR/k8sbased.sh
source $COMMON_DIR/env.sh
source $K8S_DIR/env.sh

if [[ -z $CLUSTER ]]; then
    read -p "Enter cluster name to create: " CLUSTER
fi
echo "CLUSTER set to $CLUSTER"

if [[ -z "${ROX_ADMIN_PASSWORD}" ]]; then
    export ROX_ADMIN_PASSWORD="${ROX_PASSWORD:-}"
fi
if [[ -z "$ROX_ADMIN_PASSWORD" && -f "${K8S_DIR}/central-deploy/password" ]]; then
	export ROX_ADMIN_PASSWORD="$(cat ${K8S_DIR}/central-deploy/password)"
fi

launch_sensor "$K8S_DIR"
