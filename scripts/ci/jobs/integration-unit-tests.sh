#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

integration_unit_tests() {
    info "Starting integration unit tests"
    # Retry a few times if the tests fail since they can be flaky.
    local success=0
    local max_tries=5
    for i in $(seq 1 "${max_tries}"); do
        if make integration-unit-tests; then
            success=1
            break
        fi
        echo "Retrying (failed ${i} times)"
    done
    if [[ "${success}" == 0 ]]; then
        echo "Failed after ${max_tries} tries"
        touch FAIL
    fi
    
    info "Saving junit XML report"
    make generate-junit-reports || touch FAIL
    store_test_results junit-reports reports

    [[ ! -f FAIL ]] || die "Unit tests failed"
}

integration_unit_tests "$*"
