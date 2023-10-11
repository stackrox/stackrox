#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Runs operator e2e tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$ROOT/scripts/lib.sh"
source "$ROOT/scripts/ci/lib.sh"

test_operator_e2e() {
    info "Starting operator e2e tests"

    require_environment "KUBECONFIG"

    read -r -d '' kuttl_help <<- _EO_KUTTL_HELP_ || true
See log and/or kuttl JUnit output for error details.
Reading operator/tests/TROUBLESHOOTING_E2E_TESTS.md may also be helpful.
_EO_KUTTL_HELP_

    local FAILED=0

    info "Fetching kuttl binary"
    junit_wrap fetch-kuttl \
               "Download kuttl binary." \
               "See log for error details." \
               "make" "-C" "operator" "kuttl"

    info "Deploying operator"
    junit_wrap deploy-previous-operator \
               "Deploy previously released version of the operator." \
               "${kuttl_help}" \
               "make" "-C" "operator" "deploy-previous-via-olm"

    info "Executing operator upgrade test"
    junit_wrap test-upgrade \
               "Test operator upgrade from previously released version to the current one." \
               "${kuttl_help}" \
               "make" "-C" "operator" "test-upgrade" || FAILED=1
    store_test_results "operator/build/kuttl-test-artifacts-upgrade" "kuttl-test-artifacts-upgrade"
    if junit_contains_failure "$(stored_test_results "kuttl-test-artifacts-upgrade")"; then
        # Prevent double-reporting
        remove_junit_record test-upgrade
    fi
    [[ $FAILED = 0 ]] || die "operator upgrade tests failed"

    info "Executing operator e2e tests"
    junit_wrap test-e2e \
               "Run operator E2E tests." \
               "${kuttl_help}" \
               "make" "-C" "operator" "test-e2e-deployed" || FAILED=1
    store_test_results "operator/build/kuttl-test-artifacts" "kuttl-test-artifacts"
    if junit_contains_failure "$(stored_test_results "kuttl-test-artifacts")"; then
        # Prevent double-reporting
        remove_junit_record test-e2e
    fi
    [[ $FAILED = 0 ]] || die "operator e2e tests failed"

    info "Executing Operator Bundle Scorecard tests"
    junit_wrap bundle-test-image \
                "Run scorecard tests." \
                "See log for error details." \
                "$ROOT/operator/scripts/retry.sh" "4" "2" \
                "make" "-C" "operator" "bundle-test-image"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_operator_e2e "$*"
fi
