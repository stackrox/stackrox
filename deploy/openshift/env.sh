#!/usr/bin/env bash
set -e

export ORCH="openshift"
if [[ -n "${CI:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    export KUBECTL="oc"
    export ORCH_CMD="${SCRIPT_DIR}/../../scripts/retry-kubectl.sh"
else
    export ORCH_CMD="oc"
fi
export ORCH_FULLNAME="openshift"

export ADMISSION_CONTROLLER_POD_EVENTS="${ADMISSION_CONTROLLER_POD_EVENTS:-false}"
echo "ADMISSION_CONTROLLER_POD_EVENTS set to ${ADMISSION_CONTROLLER_POD_EVENTS}"

export ROX_OPENSHIFT_VERSION="${ROX_OPENSHIFT_VERSION:-4}"
echo "ROX_OPENSHIFT_VERSION set to ${ROX_OPENSHIFT_VERSION}"

export CLUSTER_TYPE="OPENSHIFT_CLUSTER"
if [[ "${ROX_OPENSHIFT_VERSION:-}" == "4" ]]; then
    export CLUSTER_TYPE="OPENSHIFT4_CLUSTER"
fi
