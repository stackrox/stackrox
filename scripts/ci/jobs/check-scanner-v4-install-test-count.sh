#!/usr/bin/env bash
set -euo pipefail

# This script checks that the scanner-v4-install test suite contains exactly the expected number
# of test cases. The scanner-v4-install-tests CI job is slated for removal - its remaining cases
# are meant to be moved to the operator-e2e job (ROX-33131) - so new test cases must NOT be added
# to it.
#
# TODO(ROX-33131): remove this check together with the scanner-v4-install-tests job.

EXPECTED_TEST_COUNT=3
BATS_FILE="tests/e2e/run-scanner-v4-install.bats"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
cd "$ROOT"

if [[ ! -f "$BATS_FILE" ]]; then
    echo "ERROR: $BATS_FILE not found." >&2
    echo "If the scanner-v4-install-tests job was removed, remove this check as well." >&2
    exit 1
fi

test_count=$(grep -c '^[[:space:]]*@test' "$BATS_FILE" || true)

if [[ "$test_count" -ne "$EXPECTED_TEST_COUNT" ]]; then
    echo "ERROR: expected $EXPECTED_TEST_COUNT test cases in $BATS_FILE, found $test_count:" >&2
    grep '^[[:space:]]*@test' "$BATS_FILE" | sed 's/^[[:space:]]*@test //; s/ {$//' | sed 's/^/  - /' >&2
    echo "" >&2
    echo "The scanner-v4-install-tests job is slated for removal (ROX-33131)." >&2
    echo "Do not add new test cases to it; if a case needs to be dropped, update" >&2
    echo "EXPECTED_TEST_COUNT in scripts/ci/jobs/check-scanner-v4-install-test-count.sh." >&2
    exit 1
fi

echo "✓ $BATS_FILE contains exactly $EXPECTED_TEST_COUNT test cases."
