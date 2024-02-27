#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "${ROOT}/scripts/ci/lib.sh"

set -euo pipefail

REPORTS_DIR=$(mktemp -d)
FAILED=0

bats \
    --print-output-on-failure \
    --verbose-run \
    --report-formatter junit \
    --output "${REPORTS_DIR}" \
    "${ROOT}/tests/e2e/run-scanner-v4.bats" || FAILED=1

info "Saving junit XML report..."
store_test_results "${REPORTS_DIR}" reports

exit "${FAILED}"
