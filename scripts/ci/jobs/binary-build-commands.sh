#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_bin() {
    info "Making bin"

    go mod download

    mkdir -p /cache-saved
    local go_mod_cache
    go_mod_cache=$(go env GOMODCACHE)
    mv "$go_mod_cache" /cache-saved/GOMODCACHE
}

make_test_bin "$*"
