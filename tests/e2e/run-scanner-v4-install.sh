#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "${ROOT}/scripts/ci/lib.sh"

set -euo pipefail

export SCANNER_V4_LOG_DIR="$1"
export REAL_TIME_TEST_OUTPUT="true"
REPORTS_DIR=$(mktemp -d)
FAILED=0

echo "Worker node types for Scanner V4 installation tests:"
kubectl get nodes -o json | \
    jq -jr '.items[] | .metadata.name, ": ", .metadata.labels."beta.kubernetes.io/instance-type", "\n"'
echo

bats \
    --print-output-on-failure \
    --show-output-of-passing-tests \
    --report-formatter junit \
    --output "${REPORTS_DIR}" \
    "${ROOT}/tests/e2e/run-scanner-v4-install.bats" || FAILED=1

info "Saving junit XML report..."
store_test_results "${REPORTS_DIR}" reports

exit "${FAILED}"
