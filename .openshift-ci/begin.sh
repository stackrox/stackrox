#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

info "Start of CI handling"

openshift_ci_mods
openshift_ci_import_creds

create_job_record "${JOB_NAME:-missing}"

if [[ -z "${SHARED_DIR:-}" ]]; then
    echo "ERROR: There is no SHARED_DIR for step env sharing"
    exit 0 # not fatal but worth highlighting
fi

if [[ "${JOB_NAME:-}" =~ -ocp-4- ]]; then
    info "Setting worker node type and count for OCP 4 jobs"
    # https://github.com/stackrox/automation-flavors/blob/e6daf10b7df49fc003584790e25def036b2a3b0b/openshift-4/entrypoint.sh#L76
    set_ci_shared_export WORKER_NODE_COUNT 2
    set_ci_shared_export WORKER_NODE_TYPE e2-standard-8
fi

if [[ "${JOB_NAME:-}" =~ -gke-perf-scale- ]]; then
    info "Setting node type and count for GKE perf scale jobs"
    set_ci_shared_export NUM_NODES 9
    set_ci_shared_export MACHINE_TYPE n1-standard-8
fi

if [[ "${JOB_NAME:-}" =~ ocp-4-.*-perf-scale- ]]; then
    info "Setting worker node type and count for OCP perf scale jobs"
    set_ci_shared_export WORKER_NODE_COUNT 9
    set_ci_shared_export WORKER_NODE_TYPE n1-standard-8
fi
