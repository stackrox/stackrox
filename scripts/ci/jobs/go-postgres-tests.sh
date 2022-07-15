#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

go_postgres_unit_tests() {
    info "Starting go postgres unit tests"

    initdb "${HOME}/data"
    pg_ctl -D "${HOME}/data" -l logfile -o "-k /tmp" start
    export PGHOST=/tmp
    createuser -s postgres

    make go-postgres-unit-tests || touch FAIL

    info "Saving junit XML report"
    make generate-junit-reports || touch FAIL
    store_test_results junit-reports reports

    [[ ! -f FAIL ]] || die "Unit tests failed"
}

go_postgres_unit_tests "$*"
