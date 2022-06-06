#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

ui_unit_tests() {
    info "Starting UI unit tests"
    make ui-test || touch FAIL
    
    info "Saving junit XML report"
    store_test_results ui/test-results/reports reports

    [[ ! -f FAIL ]] || die "Unit tests failed"
}

ui_unit_tests "$*"
