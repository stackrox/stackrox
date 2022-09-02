#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

style_checks() {
    info "Starting style-checks"

    # Temp hack for missing tools
    source "$ROOT/scripts/ci/gcp.sh"
    source "$ROOT/tests/e2e/lib.sh"
    setup_gcp
    gsutil cp gs://roxci-artifacts/stackrox/tools/shellcheck-v0.8.0/shellcheck .
    install shellcheck /go/bin
    shellcheck || true
    gsutil cp gs://roxci-artifacts/stackrox/tools/xmlstarlet .
    install xmlstarlet /go/bin
    xmlstarlet || true

    make style || touch FAIL

    info "Saving junit XML report"
    mkdir -p junit-reports
    cp -a report.xml "junit-reports/" || true
    store_test_results junit-reports reports

    [[ ! -f FAIL ]] || die "Style checks failed"
}

style_checks "$@"
