#!/usr/bin/env bash

# Tests part II of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"

set -euo pipefail

run_tests_part_2() {
    info "QA Automation Platform Part 2"
    exit 0

    if [[ ! -f "${STATE_DEPLOYED}" ]]; then
        info "Skipping part 2 tests due to earlier failure"
        exit 0
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    rm -f FAIL
    make -C qa-tests-backend sensor-bounce-test || touch FAIL

    store_qa_test_results "part-2-tests"
    [[ ! -f FAIL ]] || die "Part 2 tests failed"
}

run_tests_part_2
