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

    go mod download
    make -C operator bundle bundle-post-process build SILENT=
    # TODO(porridge): a hack to get opm to build an index based on bundle directory before bundle image is pushed
    # The hacked opm tool will first see if a directory named as the reference exists, and if so, use its content as if it's an unpacked image of that name.
    mkdir -p operator/"$(make --quiet default-image-registry)"
    cp -a operator/build/bundle "operator/$(make --quiet default-image-registry)/stackrox-operator-bundle:v$(make --quiet --no-print-directory -C operator tag)"
    make -C operator index-build-no-base SKIP_INDEX_DOCKER_BUILD=--skip-build SILENT=
}

build_operator_bundle_and_binary
