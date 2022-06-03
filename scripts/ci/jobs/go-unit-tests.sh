#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

go_unit_tests() {
    info "Starting go unit tests (GOTAGS=${GOTAGS})"
    if [[ "$GOTAGS" == "release" ]]; then
        export ROX_IMAGE_FLAVOR=stackrox.io
    else
        export ROX_IMAGE_FLAVOR=development_build
    fi
    make go-unit-tests GOTAGS="${GOTAGS}" || touch FAIL
    
    info "Saving junit XML report"
    make generate-junit-reports || touch FAIL
    store_test_results junit-reports reports

    [[ ! -f FAIL ]] || die "Unit tests failed"
}

go_unit_tests "$*"
