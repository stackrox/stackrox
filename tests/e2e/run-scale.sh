#!/usr/bin/env bash

# Runs scale tests. Formerly CircleCI gke-api-scale-tests and gke-postgres-api-scale-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/tests/e2e/lib.sh"
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

scale_test() {
    info "Starting scale test"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"
    require_environment "COMPARISON_METRICS"

    export_test_environment

    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox_in_scale_mode

    run_scale_test
}

deploy_stackrox_in_scale_mode() {
    "$ROOT/deploy/k8s/deploy.sh"
    
    DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}" \
    get_central_basic_auth_creds

    "$ROOT/scale/launch_workload.sh" scale-test

    wait_for_api
}

run_scale_test() {
    info "Running scale test"

    mkdir /tmp/scale-tests/pprof
    # 45 min run so that we are confident that the run has completely finished
    "$ROOT/scale/profiler/pprof.sh" /tmp/scale-tests/pprof "${API_ENDPOINT}" 45
    zip -r /tmp/scale-tests/pprof.zip /tmp/scale-tests/pprof

    local debug_dump_dir="./debug-dump-scale-test"
    get_central_debug_dump "${debug_dump_dir}"

    get_prometheus_metrics_parser

    compare_with_stored_metrics "${debug_dump_dir}"
}

get_prometheus_metrics_parser() {
      go install github.com/stackrox/prometheus-metric-parser@latest
      prometheus-metric-parser help
}

compare_with_stored_metrics() {
    local debug_dump_dir="$1"
    local gs_path="gs://stackrox-ci-metrics/${COMPARISON_METRICS}"
    local compare_with

    compare_with=$(gsutil ls "${gs_path}"/stackrox_debug\* | sort | tail -1)
    echo "Using ${compare_with} as metrics for comparison"
    mkdir /tmp/metrics
    gsutil cp "${compare_with}" /tmp/metrics
    compare_with=$(find /tmp/metrics -maxdepth 1 | sort | tail -1)

    local this_run
    this_run=$(echo "${debug_dump_dir}"/stackrox_debug*.zip)
    echo "Comparing with ${this_run}"
    local compare_cmd="${PWD}/scripts/ci/compare-debug-metrics.sh"

    pushd /tmp/metrics
    "${compare_cmd}" "${compare_with}" "${this_run}" || true
    popd
}

scale_test
