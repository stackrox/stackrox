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
    info "Starting ScannerV4 test"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "ROX_SCANNER_V4_ENABLED"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox

    run_scannerV4_test
}

run_scannerV4_test() {
    info "Running scannerV4 test"
    info "Nothing yet..."
}

scannerV4_test "$@"
