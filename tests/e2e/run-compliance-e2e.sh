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
    info "Starting compliance e2e tests"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"

    export_test_environment

    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox
    # This is what installs the Compliance Operator if the
    # INSTALL_COMPLIANCE_OPERATOR environment variable is set to "true".
    deploy_optional_e2e_components

    run_compliance_e2e_tests
}

run_compliance_e2e_tests() {
    info "Running compliance e2e tests"

    make -C tests compliance-tests || touch FAIL

    store_test_results "compliance/test-results/reports" "reports"

    if is_OPENSHIFT_CI; then
        cp -a compliance/test-results/artifacts/* "${ARTIFACT_DIR}/" || true
    fi

    [[ ! -f FAIL ]] || die "Compliance e2e tests failed"
}

test_compliance_e2e
