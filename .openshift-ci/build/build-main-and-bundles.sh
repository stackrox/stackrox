#!/usr/bin/env bash

# Execute all build steps required to create the main image bundle.tar.gz and
# scripts used in image/rhel/Dockerfile

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

build_cli() {
    info "Building roxctl"

    make cli
    validate_expected_go_version
}

build_go_binaries() {
    info "Building Go binaries & swagger docs"

    if pr_has_label "ci-release-build"; then
        ci_export GOTAGS release
    fi

    if pr_has_label "ci-race-tests"; then
        RACE=true make main-build-nodeps
    else
        make main-build-nodeps
    fi

    make swagger-docs
}

UI_BUILD_LOG="/tmp/ui_build_log.txt"

background_build_ui() {
    info "Building the UI in the background"

    (make -C ui build > "$UI_BUILD_LOG" 2>&1)&
    ui_build_pid=$!
}

wait_for_ui_build() {
    info "Waiting for the UI build to complete..."

    local ui_build_exit=0

    wait "$ui_build_pid" || {
        ui_build_exit="$?"
    }

    cat "$UI_BUILD_LOG"

    if [[ "$ui_build_exit" != "0" ]]; then
        info "The UI build failed with exit $ui_build_exit"
        exit "$ui_build_exit"
    fi
}

make_stackrox_data() {
    info "Making /stackrox-data"

    mkdir /stackrox-data
    cp -r docs/build /stackrox-data/product-docs
    # Basic sanity check: are the docs in the right place?
    ls /stackrox-data/product-docs/index.html

    "$ROOT/image/fetch-stackrox-data.sh"

    mkdir -p /stackrox-data/docs/api/v1
    cp image/docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json
}

create_main_bundle_and_scripts() {
    info "Creating main bundle.tar.gz"

    if [[ -z "${DEBUG_BUILD:-}" ]]; then
        if [[ "$(git rev-parse --abbrev-ref HEAD)" =~ "-debug" ]]; then
            DEBUG_BUILD="yes"
        else
            DEBUG_BUILD="no"
        fi
    fi

    DEBUG_BUILD="${DEBUG_BUILD}" \
       "$ROOT/image/rhel/create-bundle.sh" image "local" "local" image/rhel
}

create_central_db_bundle() {
    "$ROOT/image/postgres/create-bundle.sh" image/postgres image/postgres "true"
}

cleanup_image() {
    if [[ -z "${OPENSHIFT_BUILD_NAME:-}" ]]; then
        info "This is not an OpenShift build, will not reduce the image"
        return
    fi

    info "Reducing the image size"

    # Save for image/roxctl.Dockerfile
    cp image/bin/roxctl-linux .

    set +e
    rm -rf /go/{bin,pkg}
    rm -rf /root/{.cache,.npm}
    rm -rf /usr/local/share/.cache
    rm -rf .git
    rm -rf bin
    rm -rf docs
    rm -rf image/{THIRD_PARTY_NOTICES,bin,ui}
    rm -rf ui/build ui/node_modules ui/**/node_modules
    set -e

    # Restore for image/roxctl.Dockerfile
    mkdir -p image/bin
    mv roxctl-linux image/bin/
}

build_main_and_bundles() {
    # TODO(RS-509) Submodules are not initialized in migration but they should
    # be in the 'src' delivered by OSCI in the final env so this can then be removed.
    git submodule update --init

    "$ROOT/status.sh" || true

    openshift_ci_mods

    "$ROOT/status.sh" || true

    background_build_ui
    build_cli
    build_go_binaries
    wait_for_ui_build

    info "Making THIRD_PARTY_NOTICES"
    make ossls-notice

    info "Copying binaries for image/"
    mkdir -p image/bin
    make copy-binaries-to-image-dir

    info "Building docs"
    make -C docs

    make_stackrox_data

    create_main_bundle_and_scripts
    create_central_db_bundle

    cleanup_image
}

build_main_and_bundles
