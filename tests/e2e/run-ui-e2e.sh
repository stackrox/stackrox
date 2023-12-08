#!/usr/bin/env bash

# Runs ui/ e2e tests. Formerly CircleCI gke-ui-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

test_ui_e2e() {
    info "Starting UI e2e tests"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"

    export_test_environment

    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox
    deploy_optional_e2e_components

    run_ui_e2e_tests
}

run_ui_e2e_tests() {
    info "Running UI e2e tests"

    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        local hostname
        if [[ "${API_HOSTNAME}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            info "Getting hostname from IP: ${API_HOSTNAME}"
            hostname=$("$ROOT/tests/e2e/get_hostname.py" "${API_HOSTNAME}")
        else
            hostname="${API_HOSTNAME}"
        fi
        info "Hostname for central-lb alias: ${hostname}"
        echo "central-lb ${hostname}" > /tmp/hostaliases
        export HOSTALIASES=/tmp/hostaliases
        export UI_BASE_URL="https://central-lb:443"
    elif [[ "${LOAD_BALANCER}" == "route" ]]; then
        die "unsupported LOAD_BALANCER ${LOAD_BALANCER}"
    else
        export UI_BASE_URL="https://localhost:${LOCAL_PORT}"
    fi

    make -C ui test-e2e || touch FAIL

    store_test_results "ui/test-results/reports" "reports"
    collect_coverage

    if is_OPENSHIFT_CI; then
        cp -a ui/test-results/artifacts/* "${ARTIFACT_DIR}/" || true
    fi

    [[ ! -f FAIL ]] || die "UI e2e tests failed"
}

collect_coverage() {
    {
        curl -Os https://uploader.codecov.io/latest/linux/codecov
        chmod +x codecov
        ./codecov --flags "ui-e2e-tests"
    } || echo "Uploading coverage failed!"
}

test_ui_e2e
