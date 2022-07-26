#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

style_checks() {
    info "Starting style-checks"
    make style
}

style_checks "$@"
