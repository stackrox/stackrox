#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

go_postgres_unit_tests() {
    info "Starting go postgres unit tests"
}

go_postgres_unit_tests "$*"
