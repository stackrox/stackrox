#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

shell_unit_tests() {
    info "Starting shell unit tests"
    make shell-unit-tests
    
    info "Saving junit XML report"
    store_test_results shell-test-output reports
}

shell_unit_tests "$*"
