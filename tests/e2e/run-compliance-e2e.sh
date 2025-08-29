#!/usr/bin/env bash

# Runs compliance e2e tests.

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

test_compliance_e2e() {
    info "Starting compliance v2 e2e tests"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"

    export_test_environment

    export SENSOR_HELM_DEPLOY=true

    setup_deployment_env false false
    remove_existing_stackrox_resources
    remove_compliance_operator_resources
    setup_default_TLS_certs

    # Compliance v2 requires the Compliance Operator to be installed.
    install_the_compliance_operator
    deploy_stackrox

    run_compliance_e2e_tests
}

run_compliance_e2e_tests() {
    info "Running compliance v2 e2e tests"

    rm -f FAIL
    make -C tests compliance-v2-tests || touch FAIL

    store_test_results "tests/compliance-v2-tests-results" "compliance-v2-tests-results"

    if is_OPENSHIFT_CI; then
        cp -a compliance/test-results/artifacts/* "${ARTIFACT_DIR}/" || true
    fi

    [[ ! -f FAIL ]] || die "Compliance e2e tests failed"
}

test_compliance_e2e
