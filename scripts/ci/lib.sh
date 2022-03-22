#!/usr/bin/env bash

set -euo pipefail

# A library of CI related reusable bash functions

set +u
SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
set -u

source "$SCRIPTS_ROOT/scripts/lib.sh"

# Caution when editing: make sure groups would correspond to BASH_REMATCH use.
RELEASE_RC_TAG_BASH_REGEX='^([[:digit:]]+(\.[[:digit:]]+)*)(-rc\.[[:digit:]]+)?$'

is_release_version() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: is_release_version <version>"
    fi
    [[ "$1" =~ $RELEASE_RC_TAG_BASH_REGEX && -z "${BASH_REMATCH[3]}" ]]
}

is_RC_version() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: is_RC_version <version>"
    fi
    [[ "$1" =~ $RELEASE_RC_TAG_BASH_REGEX && -n "${BASH_REMATCH[3]}" ]]
}

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

check_docs() {
    info "Check docs version"

    if [[ "$#" -lt 1 ]]; then
        die "missing arg. usage: check_docs <tag>"
    fi

    local tag="$1"
    local only_run_on_releases="${2:-false}"

    [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]] || {
        info "Skipping step as this is not a release or RC build"
        exit 0
    }

    if [[ "$only_run_on_releases" == "true" ]]; then
        [[ -z "${BASH_REMATCH[3]}" ]] || {
            info "Skipping as this is an RC build"
            exit 0
        }
    fi

    local version="${BASH_REMATCH[1]}"
    local expected_content_branch="rhacs-docs-${version}"
    local actual_content_branch
    actual_content_branch="$(git config -f .gitmodules submodule.docs/content.branch)"
    [[ "$actual_content_branch" == "$expected_content_branch" ]] || {
        echo >&2 "Expected docs/content submodule to point to branch ${expected_content_branch}, got: ${actual_content_branch}"
        exit 1
    }

    git submodule update --remote docs/content
    git diff --exit-code HEAD || {
        echo >&2 "The docs/content submodule is out of date for the ${expected_content_branch} branch; please run"
        echo >&2 "  git submodule update --remote docs/content"
        echo >&2 "and commit the result."
        exit 1
    }

    info "The docs version is as expected"
    exit 0
}

check_scanner_and_collector() {
    info "Check on release builds that COLLECTOR_VERSION and SCANNER_VERSION are release"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: check_scanner_and_collector <fail-on-rc>"
    fi

    local fail_on_rc="$1"
    local main_release_like=0
    local main_rc=0
    local main_tag
    main_tag="$(make --quiet tag)"
    if is_release_version "$main_tag"; then
        main_release_like=1
    fi
    if is_RC_version "$main_tag"; then
        main_release_like=1
        main_rc=1
    fi

    local release_mismatch=0
    if ! is_release_version "$(make --quiet collector-tag)" && [[ "$main_release_like" == "1" ]]; then
        echo >&2 "Collector tag does not look like a release tag. Please update COLLECTOR_VERSION file before releasing."
        release_mismatch=1
    fi
    if ! is_release_version "$(make --quiet scanner-tag)" && [[ "$main_release_like" == "1" ]]; then
        echo >&2 "Scanner tag does not look like a release tag. Please update SCANNER_VERSION file before releasing."
        release_mismatch=1
    fi

    if [[ "$release_mismatch" == "1" && ( "$main_rc" == "0" || "$fail_on_rc" == "true" ) ]]; then
        # Note that the script avoids doing early exits in order for the most of its logic to be executed drung
        # regular pipeline runs so that it does not get rusty by the time of the release.
        exit 1
    fi
}

mark_collector_release() {
    info "Create a PR for collector to add this release to its RELEASED_VERSIONS file"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: mark_collector_release <tag> <username>"
    fi

    ensure_CI

    local tag="$1"
    local username="$2"

    if ! is_release_version "$tag"; then
        die "A release version is required. Got $tag"
    fi

    ssh-keyscan -H github.com >> ~/.ssh/known_hosts

    info "Check out collector source code"

    mkdir -p /tmp/collector
    git -C /tmp clone --depth=2 --no-single-branch git@github.com:stackrox/collector.git

    info "Create a branch for the PR"

    collector_version="$(cat COLLECTOR_VERSION)"
    cd /tmp/collector || exit
    gitbot(){
        git -c "user.name=RoxBot" -c "user.email=roxbot@stackrox.com" "${@}"
    }
    gitbot checkout master && gitbot pull

    branch_name="release-${tag}/update-RELEASED_VERSIONS"
    if gitbot fetch --quiet origin "${branch_name}"; then
        gitbot checkout "${branch_name}"
        gitbot pull --quiet --set-upstream origin "${branch_name}"
    else
        gitbot checkout -b "${branch_name}"
        gitbot push --set-upstream origin "${branch_name}"
    fi

    info "Update RELEASED_VERSIONS"

    # We need to make sure the file ends with a newline so as not to corrupt it when appending.
    [[ ! -f RELEASED_VERSIONS ]] || sed --in-place -e '$a'\\ RELEASED_VERSIONS
    echo "${collector_version} ${tag}  # Rox release ${tag} by ${username} at $(date)" \
        >>RELEASED_VERSIONS
    gitbot add RELEASED_VERSIONS
    gitbot commit -m "Automatic update of RELEASED_VERSIONS file for Rox release ${tag}"
    gitbot push origin "${branch_name}"

    # RS-487: These two env vars are required by /scripts/create_update_pr.sh which needs to get
    # resolved in stackrox/rox-ci-image.
    require_environment "CIRCLE_USERNAME"
    require_environment "CIRCLE_PULL_REQUEST"
    /scripts/create_update_pr.sh "${branch_name}" collector "Update RELEASED_VERSIONS" "Add entry into the RELEASED_VERSIONS file"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
