#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/e2e/vm-scanning-lib.sh
source "$ROOT/tests/e2e/vm-scanning-lib.sh"

test_vm_scanning_e2e() {
    local output_dir="${1:-vm-scanning-tests-results}"

    info "Starting VM scanning e2e tests"

    export_test_environment
    setup_deployment_env true false
    ensure_vm_scanning_cluster_prereqs
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_optional_e2e_components

    ensure_virtctl_binary

    deploy_stackrox

    cd "$ROOT"
    rm -f FAIL
    # Run VM scanning preflight tests.
    make -C tests TESTFLAGS="-race -p 1 -timeout 90m" vm-scanning-tests || touch FAIL
    store_test_results "tests/vm-scanning-tests-results" "$output_dir"
    [[ ! -f FAIL ]] || die "VM scanning e2e tests failed"
}

test_vm_scanning_e2e "${1:-vm-scanning-tests-results}"
