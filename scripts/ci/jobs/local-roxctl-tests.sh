#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../tests/e2e/run.sh
source "$ROOT/tests/e2e/run.sh"

set -euo pipefail

local_roxctl_tests() {
    info "Starting local roxctl tests"

    MAIN_TAG=$(make --quiet --no-print-directory tag)
    export MAIN_TAG

    run_roxctl_bats_tests "roxctl-test-output" "local"  || touch FAIL

    info "Saving junit XML report"
    store_test_results roxctl-test-output reports

    [[ ! -f FAIL ]] || die "roxctl tests failed"
}

local_roxctl_tests "$*"
