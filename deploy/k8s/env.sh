#!/usr/bin/env bash
set -e

export ORCH="k8s"
if [[ -n "${CI:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    export ORCH_CMD="${SCRIPT_DIR}/../../scripts/retry-kubectl.sh"
else
    export ORCH_CMD="kubectl"
fi
export ORCH_FULLNAME="kubernetes"
export CLUSTER_TYPE="KUBERNETES_CLUSTER"

export ADMISSION_CONTROLLER_POD_EVENTS="${ADMISSION_CONTROLLER_POD_EVENTS:-true}"
echo "ADMISSION_CONTROLLER_POD_EVENTS set to ${ADMISSION_CONTROLLER_POD_EVENTS}"
