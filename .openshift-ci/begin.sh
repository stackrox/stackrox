#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

openshift_ci_debug

info "It shall begin"

if [[ -z "${SHARED_DIR:-}" ]]; then
    echo "ERROR: There is no SHARED_DIR for step env sharing"
    exit 0 # not fatal but worth highlighting
fi

if [[ "${JOB_NAME:-}" =~ -ocp-4- ]]; then
    info "Setting worker node type and count for OCP 4 jobs"
    echo "WORKER_NODE_COUNT=2" | tee -a "${SHARED_DIR}/shared_env"
    echo "WORKER_NODE_TYPE=e2-standard-8" | tee -a "${SHARED_DIR}/shared_env"
fi
