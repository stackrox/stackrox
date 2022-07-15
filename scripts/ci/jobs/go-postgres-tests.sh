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

    make go-postgres-unit-tests
}

go_postgres_unit_tests "$*"
