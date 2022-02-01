#!/usr/bin/env bash

set -euo pipefail

# A library of CI related reusable bash functions

set +u
SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
set -u

source "$SCRIPTS_ROOT/scripts/lib.sh"

ci_export() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: ci_export <env-name> <env-value>"
    fi

    local env_name="$1"
    local env_value="$2"

    if is_CIRCLECI && command -v cci-export > /dev/null; then
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
        docker login -u  "${QUAY_RHACS_ENG_RO_USERNAME}" --password-stdin <<<"${QUAY_RHACS_ENG_RO_PASSWORD}" quay.io
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
    require_environment "TEST_HOST_OS"
    require_environment "GOPATH"

    local roxctl="$SCRIPTS_ROOT/bin/$TEST_HOST_OS/roxctl"

    require_executable "$roxctl" "roxctl should be built"

    cp "$roxctl" "$GOPATH/bin/roxctl"
}
