#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_bin() {
    info "Making bin"

    go mod download
}

make_bin "$*"
