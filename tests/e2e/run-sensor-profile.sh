#!/usr/bin/env bash

# Runs sensor profiling

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

KUBE_BURNER_URL="https://github.com/cloud-bulldozer/kube-burner/releases/download/v1.7.8/kube-burner-V1.7.8-linux-x86_64.tar.gz"

fetch_kube_burner() {
    wget "$KUBE_BURNER_URL"
    tar xvf ./kube-burner-V1.7.8-linux-x86_64.tar.gz
}

profile_test() {
    local pprof_zip_output="$1"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"
    require_environment "COMPARISON_METRICS"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox

    info "Running Sensor Profiling"

    local pprof_dir
    pprof_dir=$(dirname "${pprof_zip_output}")

    fetch_kube_burner
    kube-burner ocp cluster-density-v2 --churn=false --local-indexing=true --iterations=20 --timeout=30m

    # Once kube-burner is done, fetch the pprof
    "$ROOT/scale/profiler/pprof.sh" "${pprof_dir}" "${API_ENDPOINT}" 1
    zip -r "${pprof_zip_output}" "${pprof_dir}"
}

profile_test "$@"
