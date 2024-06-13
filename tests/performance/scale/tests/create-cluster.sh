#!/usr/bin/env bash
set -eou pipefail

export INFRA_NAME=$1

export ARTIFACTS_DIR="/tmp/artifacts-${INFRA_NAME}"

infractl create openshift-4-perf-scale "${INFRA_NAME}" --arg master-node-type=n2-standard-16 --description "Perf testing cluster" --download-dir="${ARTIFACT_DIR}"

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"
