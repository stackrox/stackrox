#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

shell_unit_tests() {
    info "Starting shell unit tests"
    make shell-unit-tests || touch FAIL
    
    info "Saving junit XML report"
    store_test_results shell-test-output reports

    [[ ! -f FAIL ]] || die "Unit tests failed"
}

shell_unit_tests "$*"
