#!/usr/bin/env bash

# Execute all build steps required to create the operator bundle.tar.gz and
# scripts used in image/rhel/Dockerfile

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

build_operator_bundle_and_binary() {
    # avoid a -dirty tag
    info "Reset to remove Dockerfile modification by OpenShift CI"
    git restore .
    git status

    info "Current Status:"
    "$ROOT/status.sh" || true

    openshift_ci_mods

    make -C operator bundle bundle-post-process build
}

build_operator_bundle_and_binary
