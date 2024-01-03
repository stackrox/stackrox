#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

ui_component_tests() {
    info "Starting UI component tests"
    make ui-component-tests || touch FAIL

    store_test_results "ui/apps/platform/cypress/test-results/reports" "reports"

    if is_OPENSHIFT_CI; then
        cp -a ui/apps/platform/cypress/test-results/artifacts/* "${ARTIFACT_DIR}/" || true
    fi

    [[ ! -f FAIL ]] || die "UI component tests failed"
}

ui_component_tests "$*"
