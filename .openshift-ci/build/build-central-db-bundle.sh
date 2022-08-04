#!/usr/bin/env bash

# Execute the build steps required to create the central-db image bundle.tar.gz and
# scripts used in image/rhel/Dockerfile

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

create_central_db_bundle() {
    info "Creating central-db bundle.tar.gz"

    "$ROOT/image/postgres/create-bundle.sh" image/postgres image/postgres "true"
}

build_central-db-bundle() {
    # avoid a -dirty tag
    info "Reset to remove Dockerfile modification by OpenShift CI"
    git restore .
    git status

    openshift_ci_mods

    info "Make the central-db image Dockerfile"
    make "$ROOT/image/postgres/Dockerfile.gen"

    create_central_db_bundle
}

build_central-db-bundle
