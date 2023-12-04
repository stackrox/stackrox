#!/usr/bin/env bash

# Runs Scanner V4 tests.

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

scannerV4_test() {
    info "Starting Scanner V4 test"
    local pprof_zip_output="$1"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox_in_scannerV4_mode

    run_scannerV4_test "$pprof_zip_output"
}

deploy_stackrox_in_scannerV4_mode() {
    "$ROOT/deploy/k8s/deploy.sh"
    
    DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}" \
    export_central_basic_auth_creds

    "$ROOT/scale/launch_workload.sh" scale-test

    wait_for_api

    touch "${STATE_DEPLOYED}"
}

run_scale_test() {
    info "Running scale test"

    local pprof_zip_output="$1"
    local pprof_dir
    pprof_dir=$(dirname "${pprof_zip_output}")

    mkdir -p "${pprof_dir}"
    # 45 min run so that we are confident that the run has completely finished.
    "$ROOT/scale/profiler/pprof.sh" "${pprof_dir}" "${API_ENDPOINT}" 45
    zip -r "${pprof_zip_output}" "${pprof_dir}"

    local debug_dump_dir="/tmp/scale-test-debug-dump"
    get_central_debug_dump "${debug_dump_dir}"

    get_prometheus_metrics_parser

    compare_with_stored_metrics "${debug_dump_dir}"

    if is_nightly_run && [[ -n "${STORE_METRICS:-}" ]]; then
        store_metrics "${debug_dump_dir}"
    fi
}

get_prometheus_metrics_parser() {
    go install github.com/stackrox/prometheus-metric-parser@latest
    prometheus-metric-parser help
}

compare_with_stored_metrics() {
    local debug_dump_dir="$1"
    local gs_path="gs://stackrox-ci-scale-test-results/${COMPARISON_METRICS}"
    local baseline_source
    local baseline_dir="/tmp/scale-test-baseline-metrics"
    local baseline_metrics
    local baseline_metrics_basename
    local this_run_metrics
    local compare_cmd="${PWD}/scripts/ci/compare-debug-metrics.sh"
    local comparison_output="comparison.html"
    local compare_wd="/tmp/scale-test-comparison"

    baseline_source=$(gsutil ls "${gs_path}"/stackrox_debug\* | sort | tail -1)
    info "Using ${baseline_source} as metrics for comparison"
    mkdir -p "${baseline_dir}"
    gsutil cp "${baseline_source}" "${baseline_dir}"
    baseline_metrics=$(find "${baseline_dir}" -maxdepth 1 | sort | tail -1)
    baseline_metrics_basename="$(basename "${baseline_metrics}")"

    this_run_metrics=$(echo "${debug_dump_dir}"/stackrox_debug*.zip)

    info "Comparing with ${this_run_metrics}"

    mkdir -p "${compare_wd}"
    pushd "${compare_wd}"
    # The compare script will error if any of the metrics have drifted by an
    # error threshold. At present that is not sufficient to fail the entire
    # scale-test so we ignore it with || true.
    "${compare_cmd}" "${baseline_metrics}" "${this_run_metrics}" "${comparison_output}" || true
    store_as_spyglass_artifact "${comparison_output}" "${baseline_metrics_basename}"
    popd
}

store_metrics() {
    local debug_dump_dir="$1"
    local this_run_metrics
    local gs_path="gs://stackrox-ci-scale-test-results/${STORE_METRICS}"

    this_run_metrics=$(echo "${debug_dump_dir}"/stackrox_debug*.zip)
    gsutil cp "${this_run_metrics}" "${gs_path}"

    unzip -d "${debug_dump_dir}"/stackrox_debug "${this_run_metrics}"
    prometheus-metric-parser single --file="${debug_dump_dir}"/stackrox_debug/metrics-2 \
        --format=gcp-monitoring --labels='Test=ci-scale-test,ClusterFlavor=gke' \
        --project-id=acs-san-stackroxci --timestamp="$(date -u +"%s")"
}

store_as_spyglass_artifact() {
    local comparison_output="$1"
    local metrics_name="$2"

    artifact_file="$ARTIFACT_DIR/scale-comparison-with-baseline-summary.html"

    cat > "$artifact_file" <<- HEAD
<html>
    <head>
        <title>Scale test comparison with baseline: ${metrics_name}</title>
    </head>
    <body>
    <pre style="background: #fff;">
HEAD

    cat "$comparison_output" >> "$artifact_file"

    cat >> "$artifact_file" <<- FOOT
    </pre>
</html>
FOOT
}

scale_test "$@"
