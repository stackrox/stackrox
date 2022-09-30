#!/usr/bin/env bash

# A library of CI related reusable bash functions

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

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
    ci_export MAIN_IMAGE_REPO "quay.io/$REPO/main"
    ci_export CENTRAL_DB_IMAGE_REPO "quay.io/$REPO/central-db"
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

push_main_image_set() {
    info "Pushing main and roxctl images"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: push_main_image_set <branch> <brand>"
    fi

    local branch="$1"
    local brand="$2"

    local main_image_set=("main" "roxctl" "central-db")
    if is_OPENSHIFT_CI; then
        local main_image_srcs=("$MAIN_IMAGE" "$ROXCTL_IMAGE" "$CENTRAL_DB_IMAGE")
        oc registry login
    fi

    _push_main_image_set() {
        local registry="$1"
        local tag="$2"

        for image in "${main_image_set[@]}"; do
            "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "${registry}/${image}:${tag}" | cat
        done
    }

    _tag_main_image_set() {
        local local_tag="$1"
        local registry="$2"
        local remote_tag="$3"

        for image in "${main_image_set[@]}"; do
            docker tag "stackrox/${image}:${local_tag}" "${registry}/${image}:${remote_tag}"
        done
    }

    _mirror_main_image_set() {
        local registry="$1"
        local tag="$2"

        local idx=0
        for image in "${main_image_set[@]}"; do
            oc image mirror "${main_image_srcs[$idx]}" "${registry}/${image}:${tag}"
            (( idx++ )) || true
        done
    }

    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        require_environment "QUAY_STACKROX_IO_RW_USERNAME"
        require_environment "QUAY_STACKROX_IO_RW_PASSWORD"

        docker login -u "$QUAY_STACKROX_IO_RW_USERNAME" --password-stdin <<<"$QUAY_STACKROX_IO_RW_PASSWORD" quay.io

        local destination_registries=("quay.io/stackrox-io")
    elif [[ "$brand" == "RHACS_BRANDING" ]]; then
        require_environment "QUAY_RHACS_ENG_RW_USERNAME"
        require_environment "QUAY_RHACS_ENG_RW_PASSWORD"

        docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io

        local destination_registries=("quay.io/rhacs-eng")
    else
        die "$brand is not a supported brand"
    fi

    local tag
    tag="$(make --quiet tag)"
    for registry in "${destination_registries[@]}"; do
        if is_OPENSHIFT_CI; then
            _mirror_main_image_set "$registry" "$tag"
        else
            _tag_main_image_set "$tag" "$registry" "$tag"
            _push_main_image_set "$registry" "$tag"
        fi
        if [[ "$branch" == "master" ]]; then
            if is_OPENSHIFT_CI; then
                _mirror_main_image_set "$registry" "latest"
            else
                _tag_main_image_set "$tag" "$registry" "latest"
                _push_main_image_set "$registry" "latest"
            fi
        fi
    done
}

push_matching_collector_scanner_images() {
    info "Pushing collector & scanner images tagged with main-version to quay.io/rhacs-eng"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_matching_collector_scanner_images <brand>"
    fi

    local brand="$1"

    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        require_environment "QUAY_STACKROX_IO_RW_USERNAME"
        require_environment "QUAY_STACKROX_IO_RW_PASSWORD"

        docker login -u "$QUAY_STACKROX_IO_RW_USERNAME" --password-stdin <<<"$QUAY_STACKROX_IO_RW_PASSWORD" quay.io

        local source_registry="quay.io/stackrox-io"
        local target_registries=( "quay.io/stackrox-io" )
    elif [[ "$brand" == "RHACS_BRANDING" ]]; then
        require_environment "QUAY_RHACS_ENG_RW_USERNAME"
        require_environment "QUAY_RHACS_ENG_RW_PASSWORD"

        docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io

        local source_registry="quay.io/rhacs-eng"
        local target_registries=( "quay.io/rhacs-eng" )
    else
        die "$brand is not a supported brand"
    fi

    local main_tag
    main_tag="$(make --quiet tag)"
    local scanner_version
    scanner_version="$(make --quiet scanner-tag)"
    local collector_version
    collector_version="$(make --quiet collector-tag)"

    for target_registry in "${target_registries[@]}"; do
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "${source_registry}/scanner:${scanner_version}"    "${target_registry}/scanner:${main_tag}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "${source_registry}/scanner-db:${scanner_version}" "${target_registry}/scanner-db:${main_tag}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "${source_registry}/scanner-slim:${scanner_version}"    "${target_registry}/scanner-slim:${main_tag}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "${source_registry}/scanner-db-slim:${scanner_version}" "${target_registry}/scanner-db-slim:${main_tag}"

        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "${source_registry}/collector:${collector_version}"      "${target_registry}/collector:${main_tag}"
        "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "${source_registry}/collector:${collector_version}-slim" "${target_registry}/collector-slim:${main_tag}"
    done
}

poll_for_opensource_images() {
    info "Polling for opensource images required for system tests"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: poll_for_opensource_images <seconds to wait>"
    fi

    local time_limit="$1"

    local tag
    tag="$(make --quiet tag)"
    local start_time
    start_time="$(date '+%s')"

    _image_exists() {
        local name="$1"
        local url="https://quay.io/v2/stackrox-io/$name/manifests/$tag"
        info "Checking for $name using $url"
        curl -sS --head --fail "$url"
    }

    while true; do
        if _image_exists "main" && _image_exists "roxctl" && _image_exists "central-db"; then
            info "All images exist"
            break
        fi
        if (( $(date '+%s') - start_time > time_limit )); then
           die "Timed out waiting for images after ${time_limit} seconds"
        fi
        sleep 60
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

    # RS-487: create_update_pr.sh needs to be fixed so it is not Circle CI dependent.
    require_environment "CIRCLE_USERNAME"
    /scripts/create_update_pr.sh "${branch_name}" collector "Update RELEASED_VERSIONS" "Add entry into the RELEASED_VERSIONS file"
}

pr_has_label() {
    if [[ -z "${1:-}" ]]; then
        die "usage: pr_has_label <expected label>"
    fi

    local expected_label="$1"
    get_pr_details | jq '([.labels | .[].name]  // []) | .[]' -r | grep -qx "${expected_label}"
}

get_pr_details() {
    local pull_request
    local org
    local repo

    if is_CIRCLECI; then
        [ -n "${CIRCLE_PULL_REQUEST}" ] || { echo "Not on a PR, ignoring label overrides"; exit 3; }
        [ -n "${CIRCLE_PROJECT_USERNAME}" ] || { echo "CIRCLE_PROJECT_USERNAME not found" ; exit 2; }
        [ -n "${CIRCLE_PROJECT_REPONAME}" ] || { echo "CIRCLE_PROJECT_REPONAME not found" ; exit 2; }
        pull_request="${CIRCLE_PULL_REQUEST}"
        org="${CIRCLE_PROJECT_USERNAME}"
        repo="${CIRCLE_PROJECT_REPONAME}"
    elif is_OPENSHIFT_CI; then
        if [[ -n "${JOB_SPEC:-}" ]]; then
            pull_request=$(jq -r <<<"$JOB_SPEC" '.refs.pulls[0].number')
            org=$(jq -r <<<"$JOB_SPEC" '.refs.org')
            repo=$(jq -r <<<"$JOB_SPEC" '.refs.repo')
        elif [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
            pull_request=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].number')
            org=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].org')
            repo=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].repo')
        else
            die "not supported"
        fi
    else
        die "not supported"
    fi

    headers=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        headers+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi

    url="https://api.github.com/repos/${org}/${repo}/pulls/${pull_request}"
    curl -sS "${headers[@]}" "${url}"
}

openshift_ci_mods() {
    # For ci_export(), override BASH_ENV from stackrox-test with something that is writable.
    BASH_ENV=$(mktemp)
    export BASH_ENV

    # These are not set in the binary_build_commands or image build envs.
    export CI=true
    export OPENSHIFT_CI=true

    # For gradle
    export GRADLE_USER_HOME="${HOME}"
}

validate_expected_go_version() {
    info "Validating the expected go version against what was used to build roxctl"

    roxctl_go_version="$(roxctl version --json | jq '.GoVersion' -r)"
    expected_go_version="$(cat EXPECTED_GO_VERSION)"
    if [[ "${roxctl_go_version}" != "${expected_go_version}" ]]; then
        echo "Got unexpected go version ${roxctl_go_version} (wanted ${expected_go_version})"
        exit 1
    fi

    # Ensure that the Go version is up-to-date in go.mod as well.
    # Note that the patch version is not specified in go.mod.
    [[ "${expected_go_version}" =~ ^go(1\.[0-9]{2})(\.[0-9]+)?$ ]]
    go_version="${BASH_REMATCH[1]}"

    # TODO(ROX-8056): temporarily suspend the following check. The source needs to be go1.16 compatible
    # due to OSBS build constraints, but we don't want everyone to revert their local toolchain
    # to that.
    go_version="1.16" # hardcode. To be removed once the above is fixed.

    go mod edit -go "${go_version}"
    git diff --exit-code -- go.mod
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
