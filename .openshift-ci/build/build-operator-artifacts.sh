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

    info "Downloading go modules"
    # We need to download dependencies before invoking the `manifests` target (depended on by `bundle`).
    # Otherwise we get the cryptic "Missing value for flag: -I" error when running protoc.
    go mod download

    info "Preparing bundle sources and smuggled status.sh file"
    # TODO(ROX-12346): get rid of the SILENT= once we gain some confidence (after release 3.72?)
    make -C operator bundle bundle-post-process smuggled-status-sh SILENT=

    info "Making a copy of the built bundle sources in a magically named directory that will be used instead of the bundle image."
    # The hacked opm tool will first see if a directory named as the reference exists, and if so, use its content as if it's an unpacked image of that name.
    # TODO(ROX-12347): get rid of or upstream this hack in a nicer way
    # Because the hack needs the directory to be named exactly the same as the image specification, this cannot be placed
    # in the "build/" directory.
    bundle_source_parent="operator/$(make --quiet default-image-registry)"
    mkdir -p "${bundle_source_parent}"
    cp -a operator/build/bundle "${bundle_source_parent}/stackrox-operator-bundle:v$(make --quiet --no-print-directory -C operator tag)"

    info "Preparing bundle index sources"
    # TODO(ROX-12346): get rid of the SILENT= once we gain some confidence (after release 3.72?)
    make -C operator index-build SKIP_INDEX_DOCKER_BUILD=--skip-build SILENT=
}

build_operator_bundle_and_binary
