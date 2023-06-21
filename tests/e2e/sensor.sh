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
    remove_existing_stackrox_resources

    rm -f FAIL

    info "Sensor k8s integration tests"
    make sensor-integration-test || touch FAIL
    info "Saving junit XML report"
    make generate-junit-reports || touch FAIL
    store_test_results junit-reports reports
    store_test_results "test-output/test.log" "sensor-integration"
    [[ ! -f FAIL ]] || die "sensor-integration e2e tests failed"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_sensor "$*"
fi
