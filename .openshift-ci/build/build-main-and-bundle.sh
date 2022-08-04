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

    local main_build_args="${MAIN_BUILD_ARGS:-}"

    if pr_has_label "ci-race-tests" || [[ "${RACE_CONDITION_DEBUG:-}" == "true" ]]; then
        main_build_args="${main_build_args} RACE=true"
    fi

    if [[ -n "${main_build_args}" ]]; then
        info "Building main with args: ${main_build_args}"
    fi

    # shellcheck disable=SC2086
    make ${main_build_args} main-build-nodeps

    make swagger-docs
}

UI_BUILD_LOG="/tmp/ui_build_log.txt"

background_build_ui() {
    info "Building the UI in the background"

    (retry 3 false make -C ui build > "$UI_BUILD_LOG" 2>&1)&
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

cleanup_image() {
    if [[ -z "${OPENSHIFT_BUILD_NAME:-}" ]]; then
        info "This is not an OpenShift build, will not reduce the image"
        return
    fi

    info "Reducing the image size"

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
}

build_main_and_bundles() {
    # avoid a -dirty tag
    info "Reset to remove Dockerfile modification by OpenShift CI"
    git restore .
    git status

    openshift_ci_mods

    info "Make the main image Dockerfile"
    make "$ROOT/image/rhel/Dockerfile.gen"

    background_build_ui
    build_cli
    build_go_binaries
    wait_for_ui_build

    info "Making THIRD_PARTY_NOTICES"
    make ossls-notice

    info "Copying binaries for image/"
    mkdir -p image/bin
    make copy-binaries-to-image-dir
    cp bin/linux/roxctl image/roxctl/roxctl-linux

    info "Building docs"
    make -C docs

    make_stackrox_data

    create_main_bundle_and_scripts

    cleanup_image
}

build_main_and_bundles
