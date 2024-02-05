#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Runs sensor integration tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/e2e/run.sh
source "$ROOT/tests/e2e/run.sh"

test_sensor() {
    info "Starting sensor integration tests"

    require_environment "KUBECONFIG"

    export_test_environment

    export SENSOR_HELM_DEPLOY=true
    export ROX_ACTIVE_VULN_REFRESH_INTERVAL=1m
    export ROX_NETPOL_FIELDS=true

    test_preamble
    setup_deployment_env false false
    # shellcheck disable=SC2119
    remove_existing_stackrox_resources
    # Deploy stackrox (this is needed for the reconnect tests)
    setup_default_TLS_certs
    deploy_stackrox
    setup_ktunnel "$ROOT" "$ROOT/ktunnel.log"

    # fetch the certificates
    "$ROOT/tools/local-sensor/scripts/fetch-certs.sh"

    rm -f FAIL

    info "Sensor k8s integration tests"
    make sensor-integration-test || touch FAIL
    info "Saving junit XML report"
    make generate-junit-reports || touch FAIL
    kill -2 "$KTUNNEL_PID"
    store_test_results junit-reports reports
    store_test_results "test-output/test.log" "sensor-integration"
    store_test_results "$ROOT/ktunnel.log" "ktunnel"
    [[ ! -f FAIL ]] || die "sensor-integration e2e tests failed"
}

setup_ktunnel() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: setup-ktunnel <base-dir> <log-file>"
    fi
    local base_dir="$1"
    local ktunnel_log_name="$2"

    local out_path="$base_dir/ktunnel_out"
    mkdir "$out_path"
    [[ ! -f "$out_path" ]] || die "sensor-integration e2e failed: failed to create output directory for ktunnel"

    local ktunnel_version="1.6.1"
    local ktunnel_arch
    ktunnel_arch=$(uname -m)
    local ktunnel_file_name="$base_dir/ktunnel.tar.gz"

    curl -o "$ktunnel_file_name" -L https://github.com/omrikiei/ktunnel/releases/download/v"$ktunnel_version"/ktunnel_"$ktunnel_version"_Darwin_"$ktunnel_arch".tar.gz
    [[ -f "$ktunnel_file_name" ]] || die "sensor-integration 2e2 failed: ktunnel installation failed"
    tar xvzf "$ktunnel_file_name" -C "$out_path"

    # Patch colelctor to use the ktunnel sensor
    kubectl -n stackrox set env ds/collector GRPC_SERVER=local-sensor.stackrox.svc:8443 ROX_ADVERTISED_ENDPOINT=local-sensor.stackrox.svc:8443
    # Start ktunnel
    "$out_path"/ktunnel -n stackrox expose local-sensor 8443:8443 > "$ktunnel_log_name" 2>&1 &
    KTUNNEL_PID=$!
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_sensor "$*"
fi
