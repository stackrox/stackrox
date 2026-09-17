#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_test_bin() {
    info "Making test-bin"

    if command -v roxctl &>/dev/null; then
        info "roxctl already available: $(command -v roxctl)"
    else
        make cli_host-arch upgrader
        make cli-install
    fi
}

make_test_bin "$*"
