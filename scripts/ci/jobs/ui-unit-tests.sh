#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

ui_unit_tests() {
    info "Starting UI unit tests"
    make ui-test
    
    info "Saving junit XML report"
    store_test_results ui/test-results/reports reports
}

ui_unit_tests "$*"
