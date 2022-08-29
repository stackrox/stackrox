#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

style_checks() {
    info "Starting style-checks"
    make style || touch FAIL

    info "Saving junit XML report"
    mkdir -p junit-reports
    cp -a report.xml "junit-reports/" || true
    store_test_results junit-reports reports

    [[ ! -f FAIL ]] || die "Style checks failed"
}

style_checks "$@"
