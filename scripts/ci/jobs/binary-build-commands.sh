#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_cache() {
    info "Making cache"

    go mod download
    cd ui
    yarn install
    cd ..
    mkdir -p /cache-saved
    touch /cache-saved/to-keep-openshift-release-happy.txt || true
}

make_cache "$*"
