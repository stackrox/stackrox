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
    setup_deployment_env true false # with docker login, no websockets
    ensure_vm_scanning_cluster_prereqs
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_optional_e2e_components

    # TEMPORARY — revert this commit before merge.
    # Skip ConsoleCLIDownload so CI always exercises the GitHub virtctl path
    # (digest skip + latest release) instead of waiting for a CNV OOM flake.
    if is_CI; then
        warn "TEMPORARY: skipping ConsoleCLIDownload virtctl install; using GitHub release only. Revert before merge."
        _install_virtctl_from_kubevirt_release \
            || die "TEMPORARY: GitHub virtctl install failed (forced path; revert this commit before merge)"
    elif ! ensure_virtctl_binary; then
        die "Secure virtctl download failed. Refusing insecure curl -k fallback outside CI. Set VIRTCTL_PATH or fix cluster ingress trust material."
    fi

    deploy_stackrox

    cd "$ROOT"
    rm -f FAIL
    # Run the full VM scanning e2e suite.
    make -C tests TESTFLAGS="-race -p 1 -timeout 90m" vm-scanning-tests || touch FAIL
    store_test_results "tests/vm-scanning-tests-results" "$output_dir/vm-e2e-tests"
    [[ ! -f FAIL ]] || die "VM scanning e2e tests failed"
}

test_vm_scanning_e2e "${1:-vm-scanning-tests-results}"
