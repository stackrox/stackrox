#!/usr/bin/env bash

set -euo pipefail

# A library of CI related reusable bash functions

set +u
SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
set -u

source "$SCRIPTS_ROOT/scripts/lib.sh"

ensure_CI() {
    if ! is_CI; then
        die "A CI environment is required."
    fi
}

ci_export() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: ci_export <env-name> <env-value>"
    fi

    local env_name="$1"
    local env_value="$2"

    if command -v cci-export >/dev/null; then
        cci-export "$env_name" "$env_value"
    else
        export "$env_name"="$env_value"
    fi
}

setup_deployment_env() {
    info "Setting up the deployment environment"

    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: setup_deployment_env <docker-login> <use-websocket>"
    fi

    local docker_login="$1"
    local use_websocket="$2"

    require_environment QUAY_RHACS_ENG_RO_USERNAME
    require_environment QUAY_RHACS_ENG_RO_PASSWORD

    if [[ "$docker_login" == "true" ]]; then
        docker login -u "${QUAY_RHACS_ENG_RO_USERNAME}" --password-stdin quay.io <<<"${QUAY_RHACS_ENG_RO_PASSWORD}"
    fi

    if [[ "$use_websocket" == "true" ]]; then
        ci_export CLUSTER_API_ENDPOINT "wss://central.stackrox:443"
    fi

    ci_export REGISTRY_USERNAME "$QUAY_RHACS_ENG_RO_USERNAME"
    ci_export REGISTRY_PASSWORD "$QUAY_RHACS_ENG_RO_PASSWORD"
    ci_export MAIN_IMAGE_TAG "$(make --quiet tag)"

    REPO=rhacs-eng
    ci_export MONITORING_IMAGE "quay.io/$REPO/monitoring:$(cat "$(git rev-parse --show-toplevel)/MONITORING_VERSION")"
    ci_export MAIN_IMAGE_REPO "quay.io/$REPO/main"
    ci_export COLLECTOR_IMAGE_REPO "quay.io/$REPO/collector"
    ci_export SCANNER_IMAGE "quay.io/$REPO/scanner:$(cat "$(git rev-parse --show-toplevel)/SCANNER_VERSION")"
    ci_export SCANNER_DB_IMAGE "quay.io/$REPO/scanner-db:$(cat "$(git rev-parse --show-toplevel)/SCANNER_VERSION")"
}

install_built_roxctl_in_gopath() {
    require_environment "GOPATH"

    local bin_os
    if is_darwin; then
        bin_os="darwin"
    elif is_linux; then
        bin_os="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    local roxctl="$SCRIPTS_ROOT/bin/$bin_os/roxctl"

    require_executable "$roxctl" "roxctl should be built"

    cp "$roxctl" "$GOPATH/bin/roxctl"
}

get_central_debug_dump() {
    info "Getting a central debug dump"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: get_central_debug_dump <output_dir>"
    fi

    local output_dir="$1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central debug dump --output-dir "${output_dir}"
    ls -l "${output_dir}"
}

get_central_diagnostics() {
    info "Getting central diagnostics"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: get_central_diagnostics <output_dir>"
    fi

    local output_dir="$1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central debug download-diagnostics --output-dir "${output_dir}" --insecure-skip-tls-verify
    ls -l "${output_dir}"
}

push_main_and_roxctl_images() {
    info "Pushing main and roxctl images"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_main_and_roxctl_images <branch>"
    fi

    require_environment "DOCKER_IO_PUSH_USERNAME"
    require_environment "DOCKER_IO_PUSH_PASSWORD"
    require_environment "QUAY_RHACS_ENG_RW_USERNAME"
    require_environment "QUAY_RHACS_ENG_RW_PASSWORD"

    local branch="$1"

    docker login -u "$DOCKER_IO_PUSH_USERNAME" --password-stdin <<<"$DOCKER_IO_PUSH_PASSWORD" docker.io
    "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "docker.io/stackrox/main:$(make --quiet tag)" | cat
    "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "docker.io/stackrox/roxctl:$(make --quiet tag)" | cat
    "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "docker.io/stackrox/central-db:$(make --quiet tag)" | cat
    if [[ "$branch" == "master" ]]; then
        docker tag "docker.io/stackrox/main:$(make --quiet tag)" docker.io/stackrox/main:latest
        "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" docker.io/stackrox/main:latest

        docker tag "docker.io/stackrox/roxctl:$(make --quiet tag)" docker.io/stackrox/roxctl:latest
        "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" docker.io/stackrox/roxctl:latest

        docker tag "docker.io/stackrox/central-db:$(make --quiet tag)" docker.io/stackrox/central-db:latest
        "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" docker.io/stackrox/central-db:latest
    fi

    QUAY_REPO="rhacs-eng"
    docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io
    "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "quay.io/$QUAY_REPO/main:$(make --quiet tag)" | cat
    "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "quay.io/$QUAY_REPO/roxctl:$(make --quiet tag)" | cat
    "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "quay.io/$QUAY_REPO/central-db:$(make --quiet tag)" | cat
    if [[ "$branch" == "master" ]]; then
        docker tag "quay.io/$QUAY_REPO/main:$(make --quiet tag)" "quay.io/$QUAY_REPO/main:latest"
        "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "quay.io/$QUAY_REPO/main:latest"

        docker tag "quay.io/$QUAY_REPO/roxctl:$(make --quiet tag)" "quay.io/$QUAY_REPO/roxctl:latest"
        "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "quay.io/$QUAY_REPO/roxctl:latest"

        docker tag "quay.io/$QUAY_REPO/central-db:$(make --quiet tag)" "quay.io/$QUAY_REPO/central-db:latest"
        "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "quay.io/$QUAY_REPO/central-db:latest"
    fi
}

push_matching_collector_scanner_images() {
    info "Pushing collector & scanner images tagged with main-version to docker.io/stackrox and quay.io/rhacs-eng"

    require_environment "DOCKER_IO_PUSH_USERNAME"
    require_environment "DOCKER_IO_PUSH_PASSWORD"
    require_environment "QUAY_RHACS_ENG_RW_USERNAME"
    require_environment "QUAY_RHACS_ENG_RW_PASSWORD"

    docker login -u "$DOCKER_IO_PUSH_USERNAME" --password-stdin <<<"$DOCKER_IO_PUSH_PASSWORD" docker.io
    docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io

    MAIN_TAG="$(make --quiet tag)"
    SCANNER_VERSION="$(make --quiet scanner-tag)"
    COLLECTOR_VERSION="$(make --quiet collector-tag)"

    REGISTRIES=( "docker.io/stackrox" "quay.io/rhacs-eng" )
    for TARGET_REGISTRY in "${REGISTRIES[@]}"; do
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "quay.io/rhacs-eng/scanner:${SCANNER_VERSION}"    "${TARGET_REGISTRY}/scanner:${MAIN_TAG}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "quay.io/rhacs-eng/scanner-db:${SCANNER_VERSION}" "${TARGET_REGISTRY}/scanner-db:${MAIN_TAG}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "quay.io/rhacs-eng/scanner-slim:${SCANNER_VERSION}"    "${TARGET_REGISTRY}/scanner-slim:${MAIN_TAG}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "quay.io/rhacs-eng/scanner-db-slim:${SCANNER_VERSION}" "${TARGET_REGISTRY}/scanner-db-slim:${MAIN_TAG}"

        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "quay.io/rhacs-eng/collector:${COLLECTOR_VERSION}"      "${TARGET_REGISTRY}/collector:${MAIN_TAG}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "quay.io/rhacs-eng/collector:${COLLECTOR_VERSION}-slim" "${TARGET_REGISTRY}/collector-slim:${MAIN_TAG}"
    done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
