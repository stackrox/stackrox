#!/usr/bin/env bash

# A library of CI related reusable bash functions

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

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

ci_exit_trap() {
    local exit_code="$?"
    info "Executing a general purpose exit trap for CI"
    echo "Exit code is: ${exit_code}"

    (send_slack_notice_for_failures_on_merge "${exit_code}") || { echo "ERROR: Could not slack a test failure message"; }

    while [[ -e /tmp/hold ]]; do
        info "Holding this job for debug"
        sleep 60
    done
}

create_exit_trap() {
    trap ci_exit_trap EXIT
}

setup_deployment_env() {
    info "Setting up the deployment environment"

    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: setup_deployment_env <docker-login> <use-websocket>"
    fi

    local docker_login="$1"
    local use_websocket="$2"

    if [[ "$docker_login" == "true" ]]; then
        registry_ro_login "quay.io/rhacs-eng"
    fi

    if [[ "$use_websocket" == "true" ]]; then
        ci_export CLUSTER_API_ENDPOINT "wss://central.stackrox:443"
    fi

    ci_export REGISTRY_USERNAME "$QUAY_RHACS_ENG_RO_USERNAME"
    ci_export REGISTRY_PASSWORD "$QUAY_RHACS_ENG_RO_PASSWORD"
    if [[ -z "${MAIN_IMAGE_TAG:-}" ]]; then
        ci_export MAIN_IMAGE_TAG "$(make --quiet tag)"
    fi

    REPO=rhacs-eng
    ci_export MAIN_IMAGE_REPO "quay.io/$REPO/main"
    ci_export CENTRAL_DB_IMAGE_REPO "quay.io/$REPO/central-db"
    ci_export COLLECTOR_IMAGE_REPO "quay.io/$REPO/collector"
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

    roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" --insecure-skip-tls-verify central debug dump --output-dir "${output_dir}"
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
    info "Pushing main, roxctl and central-db images"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: push_main_image_set <push_context> <brand>"
    fi

    local push_context="$1"
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
            oc_image_mirror "${main_image_srcs[$idx]}" "${registry}/${image}:${tag}"
            (( idx++ )) || true
        done
    }

    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        local destination_registries=("quay.io/stackrox-io")
    elif [[ "$brand" == "RHACS_BRANDING" ]]; then
        local destination_registries=("quay.io/rhacs-eng")
    else
        die "$brand is not a supported brand"
    fi

    local tag
    tag="$(make --quiet tag)"
    for registry in "${destination_registries[@]}"; do
        registry_rw_login "$registry"

        if is_OPENSHIFT_CI; then
            _mirror_main_image_set "$registry" "$tag"
        else
            _tag_main_image_set "$tag" "$registry" "$tag"
            _push_main_image_set "$registry" "$tag"
        fi
        if [[ "$push_context" == "merge-to-master" ]]; then
            if is_OPENSHIFT_CI; then
                _mirror_main_image_set "$registry" "latest"
            else
                _tag_main_image_set "$tag" "$registry" "latest"
                _push_main_image_set "$registry" "latest"
            fi
        fi
    done
}

push_operator_image_set() {
    info "Pushing stackrox-operator, stackrox-operator-bundle and stackrox-operator-index images"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: push_operator_image_set <push_context> <brand>"
    fi

    local push_context="$1"
    local brand="$2"

    local operator_image_set=("stackrox-operator" "stackrox-operator-bundle" "stackrox-operator-index")
    if is_OPENSHIFT_CI; then
        local operator_image_srcs=("$OPERATOR_IMAGE" "$OPERATOR_BUNDLE_IMAGE" "$OPERATOR_BUNDLE_INDEX_MAGE")
        oc registry login
    fi

    _push_operator_image_set() {
        local registry="$1"
        local tag="$2"

        local v
        for image in "${operator_image_set[@]}"; do
            if [[ "${image}" != "stackrox-operator" ]]; then
                # Only the bundle and index image tags have the v prefix.
                v="v"
            else
                v=""
            fi
            "$SCRIPTS_ROOT/scripts/ci/push-as-manifest-list.sh" "${registry}/${image}:${v}${tag}" | cat
        done
    }

    _tag_operator_image_set() {
        local local_tag="$1"
        local registry="$2"
        local remote_tag="$3"

        local v
        for image in "${operator_image_set[@]}"; do
            if [[ "${image}" != "stackrox-operator" ]]; then
                # Only the bundle and index image tags have the v prefix.
                v="v"
            else
                v=""
            fi
            docker tag "stackrox/${image}:${local_tag}" "${registry}/${image}:${v}${remote_tag}"
        done
    }

    _mirror_operator_image_set() {
        local registry="$1"
        local tag="$2"

        local idx=0
        local v
        for image in "${operator_image_set[@]}"; do
            if [[ "${image}" != "stackrox-operator" ]]; then
                # Only the bundle and index image tags have the v prefix.
                v="v"
            else
                v=""
            fi
            oc image mirror "${operator_image_srcs[$idx]}" "${registry}/${image}:${v}${tag}"
            (( idx++ )) || true
        done
    }

    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        local destination_registries=("quay.io/stackrox-io")
    elif [[ "$brand" == "RHACS_BRANDING" ]]; then
        local destination_registries=("quay.io/rhacs-eng")
    else
        die "$brand is not a supported brand"
    fi

    local tag
    tag="$(make --quiet -C operator tag)"
    for registry in "${destination_registries[@]}"; do
        registry_rw_login "$registry"

        if is_OPENSHIFT_CI; then
            _mirror_operator_image_set "$registry" "$tag"
        else
            _tag_operator_image_set "$tag" "$registry" "$tag"
            _push_operator_image_set "$registry" "$tag"
        fi
        if [[ "$push_context" == "merge-to-master" ]]; then
            if is_OPENSHIFT_CI; then
                _mirror_operator_image_set "$registry" "latest"
            else
                _tag_operator_image_set "$tag" "$registry" "latest"
                _push_operator_image_set "$registry" "latest"
            fi
        fi
    done
}

push_docs_image() {
    info "Pushing the docs image: $PIPELINE_DOCS_IMAGE"

    if ! is_OPENSHIFT_CI; then
        die "Only supported in OpenShift CI"
    fi

    oc registry login
    local docs_tag
    docs_tag="$(make --quiet docs-tag)"

    local registries=("quay.io/rhacs-eng" "quay.io/stackrox-io")

    for registry in "${registries[@]}"; do
        registry_rw_login "$registry"
        oc_image_mirror "$PIPELINE_DOCS_IMAGE" "${registry}/docs:$docs_tag"
        oc_image_mirror "$PIPELINE_DOCS_IMAGE" "${registry}/docs:$(make --quiet tag)"
    done
}

push_race_condition_debug_image() {
    info "Pushing the -race image: $MAIN_RCD_IMAGE"

    if ! is_OPENSHIFT_CI; then
        die "Only supported in OpenShift CI"
    fi

    oc registry login

    local registry="quay.io/rhacs-eng"
    registry_rw_login "$registry"
    oc_image_mirror "$MAIN_RCD_IMAGE" "${registry}/main:$(make --quiet tag)-rcd"
}

registry_rw_login() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: registry_rw_login <registry>"
    fi

    local registry="$1"

    case "$registry" in
        quay.io/rhacs-eng)
            docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io
            ;;
        quay.io/stackrox-io)
            docker login -u "$QUAY_STACKROX_IO_RW_USERNAME" --password-stdin <<<"$QUAY_STACKROX_IO_RW_PASSWORD" quay.io
            ;;
        *)
            die "Unsupported registry login: $registry" 
    esac
}

registry_ro_login() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: registry_ro_login <registry>"
    fi

    local registry="$1"

    case "$registry" in
        quay.io/rhacs-eng)
            docker login -u "$QUAY_RHACS_ENG_RO_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RO_PASSWORD" quay.io
            ;;
        *)
            die "Unsupported registry login: $registry" 
    esac
}

push_matching_collector_scanner_images() {
    info "Pushing collector & scanner images tagged with main-version to quay.io/rhacs-eng"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_matching_collector_scanner_images <brand>"
    fi

    if is_OPENSHIFT_CI; then
        oc registry login
    fi

    local brand="$1"

    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        local source_registry="quay.io/stackrox-io"
        local target_registries=( "quay.io/stackrox-io" )
    elif [[ "$brand" == "RHACS_BRANDING" ]]; then
        local source_registry="quay.io/rhacs-eng"
        local target_registries=( "quay.io/rhacs-eng" )
    else
        die "$brand is not a supported brand"
    fi

    _retag_or_mirror() {
        if is_OPENSHIFT_CI; then
            oc_image_mirror "$1" "$2"
        else
            "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "$1" "$2"
        fi
    }

    local main_tag
    main_tag="$(make --quiet tag)"
    local scanner_version
    scanner_version="$(make --quiet scanner-tag)"
    local collector_version
    collector_version="$(make --quiet collector-tag)"

    for target_registry in "${target_registries[@]}"; do
        registry_rw_login "${target_registry}"

        _retag_or_mirror "${source_registry}/scanner:${scanner_version}"    "${target_registry}/scanner:${main_tag}"
        _retag_or_mirror "${source_registry}/scanner-db:${scanner_version}" "${target_registry}/scanner-db:${main_tag}"
        _retag_or_mirror "${source_registry}/scanner-slim:${scanner_version}"    "${target_registry}/scanner-slim:${main_tag}"
        _retag_or_mirror "${source_registry}/scanner-db-slim:${scanner_version}" "${target_registry}/scanner-db-slim:${main_tag}"

        _retag_or_mirror "${source_registry}/collector:${collector_version}"      "${target_registry}/collector:${main_tag}"
        _retag_or_mirror "${source_registry}/collector:${collector_version}-slim" "${target_registry}/collector-slim:${main_tag}"
    done
}

oc_image_mirror() {
    retry 5 true oc image mirror "$1" "$2"
}

poll_for_system_test_images() {
    info "Polling for images required for system tests"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: poll_for_system_test_images <seconds to wait>"
    fi

    local time_limit="$1"

    require_environment "QUAY_RHACS_ENG_BEARER_TOKEN"

    # Require images based on the job
    case "$CI_JOB_NAME" in
        *-operator-e2e-tests)
            reqd_images=("stackrox-operator" "stackrox-operator-bundle" "stackrox-operator-index" "main")
            ;;
        *-race-condition-qa-e2e-tests)
            reqd_images=("main-rcd" "roxctl")
            ;;
        *-postgres-*)
            reqd_images=("main" "roxctl" "central-db")
            ;;
        *)
            reqd_images=("main" "roxctl")
            ;;
    esac

    info "Will poll for: ${reqd_images[*]}"

    local tag
    tag="$(make --quiet tag)"
    local start_time
    start_time="$(date '+%s')"

    while true; do
        local all_exist=true
        for image in "${reqd_images[@]}"
        do
            if ! check_rhacs_eng_image_exists "$image" "$tag"; then
                info "$image does not exist"
                all_exist=false
                break
            fi
        done

        if $all_exist; then
            info "All images exist"
            break
        fi
        if (( $(date '+%s') - start_time > time_limit )); then
           die "ERROR: Timed out waiting for images after ${time_limit} seconds"
        fi
        sleep 60
    done
}

check_rhacs_eng_image_exists() {
    local name="$1"
    local tag="$2"

    if [[ "$name" =~ stackrox-operator-(bundle|index) ]]; then
        tag="$(echo "v${tag}" | sed 's,x,0,')"
    elif [[ "$name" == "stackrox-operator" ]]; then
        tag="$(echo "${tag}" | sed 's,x,0,')"
    elif [[ "$name" == "main-rcd" ]]; then
        name="main"
        tag="${tag}-rcd"
    fi

    local url="https://quay.io/api/v1/repository/rhacs-eng/$name/tag?specificTag=$tag"
    info "Checking for $name using $url"
    local check
    check=$(curl --location -sS -H "Authorization: Bearer ${QUAY_RHACS_ENG_BEARER_TOKEN}" "$url")
    echo "$check"
    [[ "$(jq -r '.tags | first | .name' <<<"$check")" == "$tag" ]]
}

check_docs() {
    info "Check docs version"

    if [[ "$#" -lt 1 ]]; then
        die "missing arg. usage: check_docs <tag>"
    fi

    local tag="$1"

    [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]] || {
        info "Skipping step as this is not a release or RC build"
        return 0
    }

    local release_version="${BASH_REMATCH[1]}"
    local expected_content_branch="rhacs-docs-${release_version}"
    local actual_content_branch
    actual_content_branch="$(git config -f .gitmodules submodule.docs/content.branch)"
    [[ "$actual_content_branch" == "$expected_content_branch" ]] || {
        echo >&2 "ERROR: Expected docs/content submodule to point to branch ${expected_content_branch}, got: ${actual_content_branch}"
        return 1
    }

    git submodule update --remote docs/content
    git diff --exit-code HEAD || {
        echo >&2 "ERROR: The docs/content submodule is out of date for the ${expected_content_branch} branch; please run"
        echo >&2 "  git submodule update --remote docs/content"
        echo >&2 "and commit the result."
        return 1
    }

    info "The docs version is as expected"
}

check_scanner_and_collector_versions() {
    info "Check on builds that COLLECTOR_VERSION and SCANNER_VERSION are release versions"

    local release_mismatch=0
    if ! is_release_version "$(make --quiet collector-tag)"; then
        echo >&2 "ERROR: Collector tag does not look like a release tag. Please update COLLECTOR_VERSION file before releasing."
        release_mismatch=1
    fi
    if ! is_release_version "$(make --quiet scanner-tag)"; then
        echo >&2 "ERROR: Scanner tag does not look like a release tag. Please update SCANNER_VERSION file before releasing."
        release_mismatch=1
    fi

    if [[ "$release_mismatch" == "1" ]]; then
        return 1
    fi

    info "The scanner and collector versions are release versions"
}

push_release() {
    info "Push release artifacts"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_release <tag>"
    fi

    local tag="$1"

    info "Push roxctl to gs://sr-roxc & gs://rhacs-openshift-mirror-src/assets"

    setup_gcp

    local temp_dir
    temp_dir="$(mktemp -d)"
    "${SCRIPTS_ROOT}/scripts/ci/roxctl-publish/prepare.sh" . "${temp_dir}"
    "${SCRIPTS_ROOT}/scripts/ci/roxctl-publish/publish.sh" "${temp_dir}" "${tag}" "gs://sr-roxc"
    "${SCRIPTS_ROOT}/scripts/ci/roxctl-publish/publish.sh" "${temp_dir}" "${tag}" "gs://rhacs-openshift-mirror-src/assets"

    info "Publish Helm charts to github repository stackrox/release-artifacts and create a PR"

    local central_services_chart_dir
    local secured_cluster_services_chart_dir
    central_services_chart_dir="$(mktemp -d)"
    secured_cluster_services_chart_dir="$(mktemp -d)"
    roxctl helm output central-services --image-defaults=stackrox.io --output-dir "${central_services_chart_dir}/stackrox"
    roxctl helm output central-services --image-defaults=rhacs --output-dir "${central_services_chart_dir}/rhacs"
    roxctl helm output central-services --image-defaults=opensource --output-dir "${central_services_chart_dir}/opensource"
    roxctl helm output secured-cluster-services --image-defaults=stackrox.io --output-dir "${secured_cluster_services_chart_dir}/stackrox"
    roxctl helm output secured-cluster-services --image-defaults=rhacs --output-dir "${secured_cluster_services_chart_dir}/rhacs"
    roxctl helm output secured-cluster-services --image-defaults=opensource --output-dir "${secured_cluster_services_chart_dir}/opensource"
    "${SCRIPTS_ROOT}/scripts/ci/publish-helm-charts.sh" "${tag}" "${central_services_chart_dir}" "${secured_cluster_services_chart_dir}"
}

mark_collector_release() {
    info "Create a PR for collector to add this release to its RELEASED_VERSIONS file"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: mark_collector_release <tag>"
    fi

    local tag="$1"
    local username="roxbot"

    info "Check out collector source code"

    mkdir -p /tmp/collector
    git -C /tmp clone --depth=2 --no-single-branch https://github.com/stackrox/collector.git

    info "Create a branch for the PR"

    collector_version="$(cat COLLECTOR_VERSION)"
    pushd /tmp/collector || exit
    gitbot(){
        git -c "user.name=RoxBot" -c "user.email=roxbot@stackrox.com" \
            -c "url.https://${GITHUB_TOKEN}:x-oauth-basic@github.com/.insteadOf=https://github.com/" \
            "${@}"
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
    export CIRCLE_USERNAME=roxbot
    export CIRCLE_PULL_REQUEST="https://prow.ci.openshift.org/view/gs/origin-ci-test/logs/${JOB_NAME}/${BUILD_ID}"
    export GITHUB_TOKEN_FOR_PRS="${GITHUB_TOKEN}"
    /scripts/create_update_pr.sh "${branch_name}" collector "Update RELEASED_VERSIONS" "Add entry into the RELEASED_VERSIONS file"
    popd
}

is_tagged() {
    local tags
    tags="$(git tag --contains)"
    [[ -n "$tags" ]]
}

is_nightly_run() {
    [[ "${CIRCLE_TAG:-}" =~ -nightly- ]]
}

is_in_PR_context() {
    if is_CIRCLECI && [[ -n "${CIRCLE_PULL_REQUEST:-}" ]]; then
        return 0
    elif is_OPENSHIFT_CI && [[ -n "${PULL_NUMBER:-}" ]]; then
        return 0
    elif is_OPENSHIFT_CI && [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
        # bin, test-bin, images
        local pull_request
        pull_request=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].number' 2>&1) || return 1
        [[ "$pull_request" =~ ^[0-9]+$ ]] && return 0
    fi

    return 1
}

get_PR_number() {
    if is_CIRCLECI && [[ -n "${CIRCLE_PULL_REQUEST:-}" ]]; then
        echo "${CIRCLE_PULL_REQUEST}"
        return 0
    elif is_OPENSHIFT_CI && [[ -n "${PULL_NUMBER:-}" ]]; then
        echo "${PULL_NUMBER}"
        return 0
    elif is_OPENSHIFT_CI && [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
        # bin, test-bin, images
        local pull_request
        pull_request=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].number' 2>&1) || {
            echo 2>&1 "ERROR: Could not determine a PR number"
            return 1
        }
        if [[ "$pull_request" =~ ^[0-9]+$ ]]; then
            echo "$pull_request"
            return 0
        fi
    fi

    echo 2>&1 "ERROR: Could not determine a PR number"

    return 1
}

is_openshift_CI_rehearse_PR() {
    [[ "$(get_repo_full_name)" == "openshift/release" ]]
}

get_base_ref() {
    if is_CIRCLECI; then
        echo "${CIRCLE_BRANCH}"
    elif is_OPENSHIFT_CI; then
        if [[ -n "${PULL_BASE_REF:-}" ]]; then
            # presubmit, postsubmit and batch runs
            # (ref: https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables)
            echo "${PULL_BASE_REF}"
        elif [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
            # periodics - CLONEREFS_OPTIONS exists in binary_build_commands and images.
            local base_ref
            base_ref="$(jq -r <<<"${CLONEREFS_OPTIONS}" '.refs[0].base_ref')" || die "invalid CLONEREFS_OPTIONS yaml"
            if [[ "$base_ref" == "null" ]]; then
                die "expect: base_ref in CLONEREFS_OPTIONS.refs[0]"
            fi
            echo "${base_ref}"
        else
            die "Expect PULL_BASE_REF or CLONEREFS_OPTIONS"
        fi
    else
        die "unsupported"
    fi
}

get_repo_full_name() {
    if is_CIRCLECI; then
        # CIRCLE_REPOSITORY_URL=git@github.com:stackrox/stackrox.git
        echo "${CIRCLE_REPOSITORY_URL:15:-4}"
    elif is_OPENSHIFT_CI; then
        if [[ -n "${REPO_OWNER:-}" ]]; then
            # presubmit, postsubmit and batch runs
            # (ref: https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables)
            [[ -n "${REPO_NAME:-}" ]] || die "expect: REPO_NAME"
            echo "${REPO_OWNER}/${REPO_NAME}"
        elif [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
            # periodics - CLONEREFS_OPTIONS exists in binary_build_commands and images.
            local org
            local repo
            org="$(jq -r <<<"${CLONEREFS_OPTIONS}" '.refs[0].org')" || die "invalid CLONEREFS_OPTIONS yaml"
            repo="$(jq -r <<<"${CLONEREFS_OPTIONS}" '.refs[0].repo')" || die "invalid CLONEREFS_OPTIONS yaml"
            if [[ "$org" == "null" ]] || [[ "$repo" == "null" ]]; then
                die "expect: org and repo in CLONEREFS_OPTIONS.refs[0]"
            fi
            echo "${org}/${repo}"
        else
            die "Expect REPO_OWNER/NAME or CLONEREFS_OPTIONS"
        fi
    else
        die "unsupported"
    fi
}

pr_has_label() {
    if [[ -z "${1:-}" ]]; then
        die "usage: pr_has_label <expected label> [<pr details>]"
    fi

    local expected_label="$1"
    local pr_details
    local exitstatus=0
    pr_details="${2:-$(get_pr_details)}" || exitstatus="$?"
    if [[ "$exitstatus" != "0" ]]; then
        info "Warning: checking for a label in a non PR context"
        return 1
    fi

    if is_openshift_CI_rehearse_PR; then
        pr_has_label_in_body "${expected_label}" "$pr_details"
    else
        jq '([.labels | .[].name]  // []) | .[]' -r <<<"$pr_details" | grep -qx "${expected_label}"
    fi
}

pr_has_label_in_body() {
    if [[ "$#" -ne 2 ]]; then
        die "usage: pr_has_label_in_body <expected label> <pr details>"
    fi

    local expected_label="$1"
    local pr_details="$2"

    [[ "$(jq -r '.body' <<<"$pr_details")" =~ \/label:[[:space:]]*$expected_label ]]
}

# get_pr_details() from GitHub and display the result. Exits 1 if not run in CI in a PR context.
_PR_DETAILS=""
get_pr_details() {
    local pull_request
    local org
    local repo

    if [[ -n "${_PR_DETAILS}" ]]; then
        echo "${_PR_DETAILS}"
        return
    fi

    _not_a_PR() {
        echo '{ "msg": "this is not a PR" }'
        exit 1
    }

    if is_CIRCLECI; then
        [ -n "${CIRCLE_PULL_REQUEST:-}" ] || _not_a_PR
        [ -n "${CIRCLE_PROJECT_USERNAME}" ] || { echo "CIRCLE_PROJECT_USERNAME not found" ; exit 2; }
        [ -n "${CIRCLE_PROJECT_REPONAME}" ] || { echo "CIRCLE_PROJECT_REPONAME not found" ; exit 2; }
        pull_request="${CIRCLE_PULL_REQUEST##*/}"
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
            echo "Expect a JOB_SPEC or CLONEREFS_OPTIONS"
            exit 2
        fi
        [[ "${pull_request}" == "null" ]] && _not_a_PR
    elif is_GITHUB_ACTIONS; then
        pull_request="$(jq -r .pull_request.number "${GITHUB_EVENT_PATH}")" || _not_a_PR
        org="${GITHUB_REPOSITORY_OWNER}"
        repo="${GITHUB_REPOSITORY#*/}"
    else
        echo "Expect Circle or OpenShift CI"
        exit 2
    fi

    headers=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        headers+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi

    url="https://api.github.com/repos/${org}/${repo}/pulls/${pull_request}"
    pr_details=$(curl --retry 5 -sS "${headers[@]}" "${url}")
    if [[ "$(jq .id <<<"$pr_details")" == "null" ]]; then
        # A valid PR response is expected at this point
        echo "Invalid response from GitHub: $pr_details"
        exit 2
    fi
    _PR_DETAILS="$pr_details"
    echo "$pr_details"
}

GATE_JOBS_CONFIG="$SCRIPTS_ROOT/scripts/ci/gate-jobs-config.json"

gate_job() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: gate_job <job>"
    fi

    local job="$1"
    local job_config
    job_config="$(jq -r .\""$job"\" "$GATE_JOBS_CONFIG")"

    info "Will determine whether to run: $job"

    if [[ "$job_config" == "null" ]]; then
        info "$job will run because there is no gating criteria for $job"
        return
    fi

    local pr_details
    local exitstatus=0
    pr_details="$(get_pr_details)" || exitstatus="$?"

    if [[ "$exitstatus" == "0" ]]; then
        if is_openshift_CI_rehearse_PR; then
            gate_openshift_release_rehearse_job "$job" "$pr_details"
        else
            gate_pr_job "$job_config" "$pr_details"
        fi
    elif [[ "$exitstatus" == "1" ]]; then
        gate_merge_job "$job_config"
    else
        die "Could not determine if this is a PR versus a merge"
    fi
}

get_var_from_job_config() {
    local var_name="$1"
    local job_config="$2"

    local value
    value="$(jq -r ."$var_name" <<<"$job_config")"
    if [[ "$value" == "null" ]]; then
        die "$var_name is not defined in this jobs config"
    fi
    if [[ "${value:0:1}" == "[" ]]; then
        value="$(jq -cr '.[]' <<<"$value")"
    fi
    echo "$value"
}

gate_pr_job() {
    local job_config="$1"
    local pr_details="$2"

    local run_with_labels=()
    local skip_with_label
    local run_with_changed_path
    local changed_path_to_ignore
    local run_with_labels_from_json
    run_with_labels_from_json="$(get_var_from_job_config run_with_labels "$job_config")"
    if [[ -n "${run_with_labels_from_json}" ]]; then
        mapfile -t run_with_labels <<<"${run_with_labels_from_json}"
    fi
    skip_with_label="$(get_var_from_job_config skip_with_label "$job_config")"
    run_with_changed_path="$(get_var_from_job_config run_with_changed_path "$job_config")"
    changed_path_to_ignore="$(get_var_from_job_config changed_path_to_ignore "$job_config")"

    if [[ -n "$skip_with_label" ]]; then
        if pr_has_label "${skip_with_label}" "${pr_details}"; then
            info "$job will not run because the PR has label $skip_with_label"
            exit 0
        fi
    fi

    for run_with_label in "${run_with_labels[@]}"; do
        if pr_has_label "${run_with_label}" "${pr_details}"; then
            info "$job will run because the PR has label $run_with_label"
            return
        fi
    done

    if [[ -n "${run_with_changed_path}" || -n "${changed_path_to_ignore}" ]]; then
        local diff_base
        if is_CIRCLECI; then
            diff_base="$(git merge-base HEAD origin/master)"
            echo "Determined diff-base as ${diff_base}"
            echo "Master SHA: $(git rev-parse origin/master)"
        elif is_OPENSHIFT_CI; then
            diff_base="$(jq -r '.refs[0].base_sha' <<<"$CLONEREFS_OPTIONS")"
            echo "Determined diff-base as ${diff_base}"
            [[ "${diff_base}" != "null" ]] || die "Could not find base_sha in CLONEREFS_OPTIONS: $CLONEREFS_OPTIONS"
        else
            die "unsupported"
        fi
        echo "Diffbase diff:"
        { git diff --name-only "${diff_base}" | cat ; } || true
        ignored_regex="${changed_path_to_ignore}"
        [[ -n "$ignored_regex" ]] || ignored_regex='$^' # regex that matches nothing
        match_regex="${run_with_changed_path}"
        [[ -n "$match_regex" ]] || match_regex='^.*$' # grep -E -q '' returns 0 even on empty input, so we have to specify some pattern
        if grep -E -q "$match_regex" < <({ git diff --name-only "${diff_base}" || echo "???" ; } | grep -E -v "$ignored_regex"); then
            info "$job will run because paths matching $match_regex (and not matching ${ignored_regex}) had changed."
            return
        fi
    fi

    info "$job will be skipped"
    exit 0
}

gate_merge_job() {
    local job_config="$1"

    local run_on_master
    local run_on_tags
    run_on_master="$(get_var_from_job_config run_on_master "$job_config")"
    run_on_tags="$(get_var_from_job_config run_on_tags "$job_config")"

    local base_ref
    base_ref="$(get_base_ref)" || {
        info "Warning: error running get_base_ref():"
        echo "${base_ref}"
        info "will continue with tests."
    }

    if [[ "${base_ref}" == "master" && "${run_on_master}" == "true" ]]; then
        info "$job will run because this is master and run_on_master==true"
        return
    fi

    if is_tagged && [[ "${run_on_tags}" == "true" ]]; then
        info "$job will run because the head of this branch is tagged and run_on_tags==true"
        return
    fi

    info "$job will be skipped - neither master/run_on_master or tagged/run_on_tags"
    exit 0
}

# gate_openshift_release_rehearse_job() - use the PR description to indicate if
# the pj-rehearse job should run for configured jobs.
gate_openshift_release_rehearse_job() {
    local job="$1"
    local pr_details="$2"

    if [[ "$(jq -r '.body' <<<"$pr_details")" =~ open.the.gate.*$job ]]; then
        info "$job will run because the gate was opened"
        return
    fi

    cat << _EOH_
$job will be skipped. If you want to run a gated job during openshift/release pj-rehearsal 
update the PR description with:
open the gate: $job
_EOH_
    exit 0
}

openshift_ci_mods() {
    info "BEGIN OpenShift CI mods"

    info "Env A-Z dump:"
    env | sort | grep -E '^[A-Z]' || true

    info "Git log:"
    git log --oneline --decorate -n 20 || true

    info "Recent git refs:"
    git for-each-ref --format='%(creatordate) %(refname)' --sort=creatordate | tail -20

    info "Current Status:"
    "$ROOT/status.sh" || true

    # For ci_export(), override BASH_ENV from stackrox-test with something that is writable.
    BASH_ENV=$(mktemp)
    export BASH_ENV

    # These are not set in the binary_build_commands or image build envs.
    export CI=true
    export OPENSHIFT_CI=true

    if is_in_PR_context && ! is_openshift_CI_rehearse_PR; then
        local sha
        sha=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].sha') || echo "WARNING: Cannot find pull sha"
        if [[ -n "${sha:-}" ]] && [[ "$sha" != "null" ]]; then
            info "Will checkout SHA to match PR: $sha"
            git checkout "$sha"
        else
            echo "WARNING: Could not determine a SHA for this PR, ${sha:-}"
        fi
    fi

    # Provide Circle CI vars that are commonly used
    export CIRCLE_JOB="${JOB_NAME:-${OPENSHIFT_BUILD_NAME}}"
    CIRCLE_TAG="$(git tag --sort=creatordate --contains | tail -1)" || echo "Warning: Cannot get tag"
    export CIRCLE_TAG

    # For gradle
    export GRADLE_USER_HOME="${HOME}"

    handle_nightly_runs

    info "Status after mods:"
    "$ROOT/status.sh" || true

    STACKROX_BUILD_TAG=$(make --quiet tag)
    export STACKROX_BUILD_TAG

    info "END OpenShift CI mods"
}

openshift_ci_import_creds() {
    shopt -s nullglob
    for cred in /tmp/secret/**/[A-Z]*; do
        export "$(basename "$cred")"="$(cat "$cred")"
    done
    for cred in /tmp/vault/**/[A-Z]*; do
        export "$(basename "$cred")"="$(cat "$cred")"
    done
}

unset_namespace_env_var() {
    # NAMESPACE is injected by OpenShift CI for the cluster that is running the
    # tests but this can have side effects for stackrox tests due to its use as
    # the default namespace e.g. with helm.
    if [[ -n "${NAMESPACE:-}" ]]; then
        export OPENSHIFT_CI_NAMESPACE="$NAMESPACE"
        unset NAMESPACE
    fi
}

openshift_ci_e2e_mods() {
    unset_namespace_env_var

    # The incoming KUBECONFIG is for the openshift/release cluster and not the
    # e2e test cluster.
    if [[ -n "${KUBECONFIG:-}" ]]; then
        info "There is an incoming KUBECONFIG in ${KUBECONFIG}"
        export OPENSHIFT_CI_KUBECONFIG="$KUBECONFIG"
    fi
    KUBECONFIG="$(mktemp)"
    info "KUBECONFIG set: ${KUBECONFIG}"
    export KUBECONFIG

    # KUBERNETES_{PORT,SERVICE} env values interact with commandline kubectl tests
    if env | grep -e ^KUBERNETES_; then
        local envfile
        envfile="$(mktemp)"
        info "Will clear ^KUBERNETES_ env"
        env | grep -e ^KUBERNETES_ | cut -d= -f1 | awk '{ print "unset", $1 }' > "$envfile"
        # shellcheck disable=SC1090
        source "$envfile"
    fi
}

operator_e2e_test_setup() {
    # TODO(ROX-11901): pass the brand explicitly from the CI config file rather than hardcode here
    registry_ro_login "quay.io/rhacs-eng"
    export ROX_PRODUCT_BRANDING="RHACS_BRANDING"

    # $NAMESPACE is set by OpenShift CI, but confuses `operator-sdk scorecard` which runs against
    # a completely different cluster, where this namespace does not even exist.
    # Note that even though unsetting the variable turns out not to be sufficient for `operator-sdk scorecard`
    # (still gets the namespace from *somewhere*), we're keeping this here as it might affect other tools.
    unset_namespace_env_var
}

handle_nightly_runs() {
    if ! is_OPENSHIFT_CI; then
        die "Only for OpenShift CI"
    fi

    if ! is_in_PR_context; then
        info "Debug:"
        echo "JOB_NAME: ${JOB_NAME:-}"
        echo "JOB_NAME_SAFE: ${JOB_NAME_SAFE:-}"
    fi

    local nightly_tag_prefix
    nightly_tag_prefix="$(git describe --tags --abbrev=0 --exclude '*-nightly-*')-nightly-"
    if ! is_in_PR_context && [[ "${JOB_NAME_SAFE:-}" =~ ^nightly- ]]; then
        ci_export CIRCLE_TAG "${nightly_tag_prefix}$(date '+%Y%m%d')"
    elif is_in_PR_context && pr_has_label "simulate-nightly-run"; then
        local sha
        if [[ -n "${PULL_PULL_SHA:-}" ]]; then
            sha="${PULL_PULL_SHA}"
        else
            sha=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].sha') || die "Cannot find pull sha"
            [[ "$sha" != "null" ]] || die "Cannot find pull sha"
        fi
        ci_export CIRCLE_TAG "${nightly_tag_prefix}${sha:0:8}"
    fi
}

handle_nightly_binary_version_mismatch() {
    if ! is_OPENSHIFT_CI; then
        die "Only for OpenShift CI"
    fi

    if is_in_PR_context || ! [[ "${JOB_NAME_SAFE:-}" =~ ^nightly- ]]; then
        return 0
    fi

    # JOB_NAME_SAFE is not set in test_binary_build_commands context for
    # periodics, so the roxctl produced in that step will cause deploy.sh to
    # fail.

    if ! is_in_PR_context; then
        info "Debug:"
        echo "JOB_NAME: ${JOB_NAME:-}"
        echo "JOB_NAME_SAFE: ${JOB_NAME_SAFE:-}"
    fi

    info "Correcting binary versions for nightly e2e tests"
    echo "Current roxctl is: $(command -v roxctl || true), version: $(roxctl version || true)"

    if ! [[ "$(roxctl version || true)" =~ nightly-$(date '+%Y%m%d') ]]; then
        make cli-build upgrader
        install_built_roxctl_in_gopath
        echo "Replacement roxctl is: $(command -v roxctl || true), version: $(roxctl version || true)"
    fi
}

validate_expected_go_version() {
    info "Validating the expected go version against what was used to build roxctl"

    roxctl_go_version="$(roxctl version --json | jq '.GoVersion' -r)"
    expected_go_version="$(head -n 1 EXPECTED_GO_VERSION)"
    if [[ "${roxctl_go_version}" != "${expected_go_version}" ]]; then
        echo "Got unexpected go version ${roxctl_go_version} (wanted ${expected_go_version})"
        exit 1
    fi

    # Ensure that the Go version is up-to-date in go.mod as well.
    # Note that the patch version is not specified in go.mod.
    [[ "${expected_go_version}" =~ ^go(1\.[0-9]{2})(\.[0-9]+)?$ ]]
    go_version="${BASH_REMATCH[1]}"

    go mod edit -go "${go_version}"
    git diff --exit-code -- go.mod
}

store_qa_test_results() {
    if ! is_OPENSHIFT_CI; then
        return
    fi

    local to="${1:-qa-tests}"

    info "Copying qa-tests-backend results to $to"

    store_test_results qa-tests-backend/build/test-results/test "$to"
}

store_test_results() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: store_test_results <from> <to>"
    fi

    if ! is_OPENSHIFT_CI; then
        return
    fi

    local from="$1"
    local to="$2"

    info "Copying test results from $from to $to"

    local dest="${ARTIFACT_DIR}/junit-$to"

    cp -a "$from" "$dest" || true # (best effort)
}

send_slack_notice_for_failures_on_merge() {
    local exitstatus="${1:-}"

    if ! is_OPENSHIFT_CI || [[ "$exitstatus" == "0" ]] || is_in_PR_context || is_nightly_run; then
        return 0
    fi

    local tag
    tag="$(make --quiet tag)"
    if [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]]; then
        return 0
    fi

    local webhook_url="${TEST_FAILURES_NOTIFY_WEBHOOK}"

    local commit_details
    org=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].org') || return 1
    repo=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].repo') || return 1
    [[ "$org" != "null" ]] && [[ "$repo" != "null" ]] || return 1
    local commit_details_url="https://api.github.com/repos/${org}/${repo}/commits/${OPENSHIFT_BUILD_COMMIT}"
    commit_details=$(curl --retry 5 -sS "${commit_details_url}") || return 1

    local job_name="${JOB_NAME_SAFE#merge-}"

    local commit_msg
    commit_msg=$(jq -r <<<"$commit_details" '.commit.message') || return 1
    commit_msg="${commit_msg%%$'\n'*}" # use first line of commit msg
    local commit_url
    commit_url=$(jq -r <<<"$commit_details" '.html_url') || return 1
    local author
    author=$(jq -r <<<"$commit_details" '.commit.author.name') || return 1
    [[ "$commit_msg" != "null" ]] && [[ "$commit_url" != "null" ]] && [[ "$author" != "null" ]] || return 1

    local log_url="https://prow.ci.openshift.org/view/gs/origin-ci-test/logs/${JOB_NAME}/${BUILD_ID}"

    # shellcheck disable=SC2016
    local body='
{
    "text": "*Job Name:* \($job_name)",
    "blocks": [
		{
			"type": "header",
			"text": {
				"type": "plain_text",
				"text": "Prow job failure: \($job_name)"
			}
		},
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": "*Commit:* <\($commit_url)|\($commit_msg)>\n*Repo:* \($repo)\n*Author:* \($author)\n*Log:* \($log_url)"
            }
        },
		{
			"type": "divider"
		}
    ]
}
'

    echo "About to post:"
    jq --null-input --arg job_name "$job_name" --arg commit_url "$commit_url" --arg commit_msg "$commit_msg" \
       --arg repo "$repo" --arg author "$author" --arg log_url "$log_url" "$body"

    jq --null-input --arg job_name "$job_name" --arg commit_url "$commit_url" --arg commit_msg "$commit_msg" \
       --arg repo "$repo" --arg author "$author" --arg log_url "$log_url" "$body" | \
    curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url"
}

save_junit_failure() {
    if [[ "$#" -ne 3 ]]; then
        die "missing args. usage: save_junit_failure <class> <description> <details>"
    fi

    if [[ -z "${ARTIFACT_DIR}" ]]; then
        info "Warning: save_junit_failure() requires an ARTIFACT_DIR"
        return
    fi

    local class="$1"
    local description="$2"
    local details="$3"

    cat << EOF > "${ARTIFACT_DIR}/junit-${class}.xml"
<testsuite name="${class}" tests="1" skipped="0" failures="1" errors="0">
    <testcase name="${description}" classname="${class}">
        <failure>${details}</failure>
    </testcase>
</testsuite>
EOF
}

add_build_comment_to_pr() {
    info "Adding a comment with the build tag to the PR"

    # hub-comment is tied to Circle CI env
    local url
    url=$(get_pr_details | jq -r '.html_url')
    export CIRCLE_PULL_REQUEST="$url"

    local sha
    sha=$(get_pr_details | jq -r '.head.sha')
    sha=${sha:0:7}
    export _SHA="$sha"

    local tag
    tag=$(make tag)
    export _TAG="$tag"

    local tmpfile
    tmpfile=$(mktemp)
    cat > "$tmpfile" <<- EOT
Images are ready for the commit at {{.Env._SHA}}.

To use with deploy scripts, first \`export MAIN_IMAGE_TAG={{.Env._TAG}}\`.
EOT

    hub-comment -type build -template-file "$tmpfile"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
