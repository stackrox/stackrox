#!/usr/bin/env bash

# Runs ui/ e2e tests. Formerly CircleCI gke-ui-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/tests/e2e/lib.sh"
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

test_ui_e2e() {
    info "Starting UI e2e tests"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export_test_environment

    if is_OPENSHIFT_CI; then
        # TODO(RS-494) may provide roxctl
        make cli-linux
        install_built_roxctl_in_gopath
    fi

    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox

    run_ui_e2e_tests
}

run_ui_e2e_tests() {
    info "Running UI e2e tests"

    if [[ "${LOAD_BALANCER}" == "lb" ]]; then
        echo "central-lb ${API_HOSTNAME}" > /tmp/hostaliases
        export HOSTALIASES=/tmp/hostaliases
        export UI_BASE_URL="https://central-lb:443"
    else
        export UI_BASE_URL="https://localhost:${LOCAL_PORT}"
    fi

    make -C ui test-e2e || touch FAIL

    store_test_results "ui/test-results/reports" "reports"

    [[ ! -f FAIL ]] || die "UI e2e tests failed"
}

test_ui_e2e
