#!/usr/bin/env bash

# Tests part II of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/tests/e2e/lib.sh"

set -euo pipefail

run_tests_part_2() {
    info "QA Automation Platform Part 2"
    info "Not running tests"
}

run_tests_part_2
