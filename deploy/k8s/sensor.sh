#!/usr/bin/env bash
# shellcheck disable=SC1091
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

# shellcheck source=../common/deploy.sh
source "$COMMON_DIR"/deploy.sh
# shellcheck source=../common/k8sbased.sh
source "$COMMON_DIR"/k8sbased.sh
# shellcheck source=../common/env.sh
source "$COMMON_DIR"/env.sh
# shellcheck source=./env.sh
source "$K8S_DIR"/env.sh

if [[ -z $CLUSTER ]]; then
    read -p -r "Enter cluster name to create: " CLUSTER
fi
echo "CLUSTER set to $CLUSTER"

if [[ -z "${ROX_ADMIN_PASSWORD}" ]]; then
    export ROX_ADMIN_PASSWORD="${ROX_PASSWORD:-}"
fi
if [[ -z "$ROX_ADMIN_PASSWORD" && -f "${K8S_DIR}/central-deploy/password" ]]; then
	# shellcheck disable=SC2086
	ROX_ADMIN_PASSWORD="$(cat ${K8S_DIR}/central-deploy/password)"
	export ROX_ADMIN_PASSWORD
fi

launch_sensor "$K8S_DIR"
