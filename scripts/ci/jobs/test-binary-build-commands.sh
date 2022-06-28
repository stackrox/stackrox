#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_test_bin() {
    info "Making test-bin"

    info "Current Status:"
    "$ROOT/status.sh" || true

    make cli upgrader
    install_built_roxctl_in_gopath
}

make_test_bin "$*"
