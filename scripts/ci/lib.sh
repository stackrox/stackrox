#!/usr/bin/env bash

# A library of CI related reusable bash functions

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/metrics.sh
source "$SCRIPTS_ROOT/scripts/ci/metrics.sh"
# shellcheck source=../../scripts/ci/test_state.sh
source "$SCRIPTS_ROOT/scripts/ci/test_state.sh"

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

    finalize_job_record "${exit_code}" "false"

    (send_slack_notice_for_failures_on_merge "${exit_code}") || { echo "ERROR: Could not slack a test failure message"; }

    post_process_test_results

    while [[ -e /tmp/hold ]]; do
        info "Holding this job for debug"
        sleep 60
    done

    handle_dangling_processes
}

# handle_dangling_processes() - The OpenShift CI ci-operator will not complete a
# test job if there are processes remaining that were started by the job. While
# processes _should_ be cleaned up by their creators it is common that some are
# not, so this exists as a fail safe.
handle_dangling_processes() {
    info "Process state at exit:"
    ps -e -O ppid

    local psline this_pid pid
    ps -e -O ppid | while read -r psline; do
        # trim leading whitespace
        psline="$(echo "$psline" | xargs)"
        if [[ "$psline" =~ ^PID ]]; then
            # Ignoring header
            continue
        fi
        this_pid="$$"
        if [[ "$psline" =~ ^$this_pid ]]; then
            echo "Ignoring self: $psline"
            continue
        fi
        # shellcheck disable=SC1087
        if [[ "$psline" =~ [[:space:]]$this_pid[[:space:]] ]]; then
            echo "Ignoring child: $psline"
            continue
        fi
        if [[ "$psline" =~ entrypoint|defunct ]]; then
            echo "Ignoring ci-operator entrypoint or defunct process: $psline"
            continue
        fi
        echo "A candidate to kill: $psline"
        pid="$(echo "$psline" | cut -d' ' -f1)"
        echo "Will kill $pid"
        kill "$pid" || {
            echo "Error killing $pid"
        }
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
        ci_export MAIN_IMAGE_TAG "$(make --quiet --no-print-directory tag)"
    fi

    ci_export ROX_PRODUCT_BRANDING "RHACS_BRANDING"
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

push_image_manifest_lists() {
    info "Pushing main, roxctl and central-db images as manifest lists"

    if [[ "$#" -ne 3 ]]; then
        die "missing arg. usage: push_image_manifest_lists <push_context> <brand> <architectures (CSV)>"
    fi

    local push_context="$1"
    local brand="$2"
    local architectures="$3"

    local main_image_set=("main" "roxctl" "central-db")

    local registry
    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        registry="quay.io/stackrox-io"
    elif [[ "$brand" == "RHACS_BRANDING" ]]; then
        registry="quay.io/rhacs-eng"
    else
        die "$brand is not a supported brand"
    fi

    local tag
    tag="$(make --quiet --no-print-directory tag)"

    registry_rw_login "$registry"
    for image in "${main_image_set[@]}"; do
        "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:${tag}" "$architectures" | cat
        if [[ "$push_context" == "merge-to-master" ]]; then
            "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:latest" "$architectures" | cat
        fi
    done

    # Push manifest lists for scanner and collector for amd64 only
    local amd64_image_set=("scanner" "scanner-db" "scanner-slim" "scanner-db-slim" "collector" "collector-slim")
    for image in "${amd64_image_set[@]}"; do
        "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:${tag}" "amd64" | cat
    done
}

push_main_image_set() {
    info "Pushing main, roxctl and central-db images"

    if [[ "$#" -ne 3 ]]; then
        die "missing arg. usage: push_main_image_set <push_context> <brand> <arch>"
    fi

    local push_context="$1"
    local brand="$2"
    local arch="$3"

    local main_image_set=("main" "roxctl" "central-db")
    if is_OPENSHIFT_CI; then
        local main_image_srcs=("$MAIN_IMAGE" "$ROXCTL_IMAGE" "$CENTRAL_DB_IMAGE")
        oc registry login
    fi

    _push_main_image_set() {
        local registry="$1"
        local tag="$2"

        for image in "${main_image_set[@]}"; do
            docker push "${registry}/${image}:${tag}" | cat
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
    tag="$(make --quiet --no-print-directory tag)"
    for registry in "${destination_registries[@]}"; do
        registry_rw_login "$registry"

        if is_OPENSHIFT_CI; then
            _mirror_main_image_set "$registry" "$tag"
        else
            _tag_main_image_set "$tag" "$registry" "$tag-$arch"
            _push_main_image_set "$registry" "$tag-$arch"
        fi
        if [[ "$push_context" == "merge-to-master" ]]; then
            if is_OPENSHIFT_CI; then
                _mirror_main_image_set "$registry" "latest-${arch}"
            else
                _tag_main_image_set "$tag" "$registry" "latest-${arch}"
                _push_main_image_set "$registry" "latest-${arch}"
            fi
        fi
    done
}

push_scanner_image_manifest_lists() {
    info "Pushing scanner-v4 and scanner-v4-db images as manifest lists"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_scanner_image_manifest_lists <architectures (CSV)>"
    fi

    local architectures="$1"
    local scanner_image_set=("scanner-v4" "scanner-v4-db")
    local registries=("quay.io/rhacs-eng" "quay.io/stackrox-io")

    local tag
    tag="$(make --quiet --no-print-directory -C scanner tag)"
    for registry in "${registries[@]}"; do
        registry_rw_login "$registry"
        for image in "${scanner_image_set[@]}"; do
            "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:${tag}" "$architectures" | cat
        done
    done
}

push_scanner_image_set() {
    info "Pushing scanner-v4 and scanner-v4-db images"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_scanner_image_set <arch>"
    fi

    local arch="$1"

    local scanner_image_set=("scanner-v4" "scanner-v4-db")

    _push_scanner_image_set() {
        local registry="$1"
        local tag="$2"

        for image in "${scanner_image_set[@]}"; do
            docker push "${registry}/${image}:${tag}" | cat
        done
    }

    _tag_scanner_image_set() {
        local local_tag="$1"
        local registry="$2"
        local remote_tag="$3"

        for image in "${scanner_image_set[@]}"; do
            docker tag "stackrox/${image}:${local_tag}" "${registry}/${image}:${remote_tag}"
        done
    }

    local registries=("quay.io/rhacs-eng" "quay.io/stackrox-io")

    local tag
    tag="$(make --quiet --no-print-directory -C scanner tag)"
    for registry in "${registries[@]}"; do
        registry_rw_login "$registry"

        _tag_scanner_image_set "$tag" "$registry" "$tag-$arch"
        _push_scanner_image_set "$registry" "$tag-$arch"
    done
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

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: push_matching_collector_scanner_images <brand> <arch>"
    fi

    if is_OPENSHIFT_CI; then
        oc registry login
    fi

    local brand="$1"
    local arch="$2"

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

    if [[ "$arch" != "amd64" ]]; then
        echo "Skipping rebundling for non-amd64 arch"
        exit 0
    fi

    local main_tag
    main_tag="$(make --quiet --no-print-directory tag)"
    local scanner_version
    scanner_version="$(make --quiet --no-print-directory scanner-tag)"
    local collector_version
    collector_version="$(make --quiet --no-print-directory collector-tag)"

    for target_registry in "${target_registries[@]}"; do
        registry_rw_login "${target_registry}"

        _retag_or_mirror "${source_registry}/scanner:${scanner_version}"    "${target_registry}/scanner:${main_tag}-${arch}"
        _retag_or_mirror "${source_registry}/scanner-db:${scanner_version}" "${target_registry}/scanner-db:${main_tag}-${arch}"
        _retag_or_mirror "${source_registry}/scanner-slim:${scanner_version}"    "${target_registry}/scanner-slim:${main_tag}-${arch}"
        _retag_or_mirror "${source_registry}/scanner-db-slim:${scanner_version}" "${target_registry}/scanner-db-slim:${main_tag}-${arch}"

        _retag_or_mirror "${source_registry}/collector:${collector_version}"      "${target_registry}/collector:${main_tag}-${arch}"
        _retag_or_mirror "${source_registry}/collector:${collector_version}-slim" "${target_registry}/collector-slim:${main_tag}-${arch}"
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
            if is_in_PR_context && ! pr_has_label "ci-build-race-condition-debug"; then
                echo "ERROR: Your PR is missing the \"ci-build-race-condition-debug\" label."
                echo "ERROR: This label is required to build the images for $CI_JOB_NAME."
                # Quietly continue to allow labels added after tests start.
                # Otherwise this message will surface in the Prow log when
                # images timeout out below.
            fi
            ;;
        *)
            reqd_images=("main" "roxctl")
            ;;
    esac

    if [[ "${ROX_POSTGRES_DATASTORE:-}" == "true" ]] && [[ ! " ${reqd_images[*]} " =~ " central-db " ]]; then
        reqd_images+=("central-db")
    fi

    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR:-}" == "true" ]]; then
        reqd_images+=("stackrox-operator" "stackrox-operator-bundle" "stackrox-operator-index")
    fi

    info "Will poll for: ${reqd_images[*]}"

    local tag
    tag="$(make --quiet --no-print-directory tag)"
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

    touch "${STATE_IMAGES_AVAILABLE}"
}

check_rhacs_eng_image_exists() {
    local name="$1"
    local tag="$2"

    if [[ "$name" =~ stackrox-operator-(bundle|index) ]]; then
        tag="$(echo "v${tag}" | sed 's,x,0,')"
    elif [[ "$name" == "stackrox-operator" ]]; then
        tag="${tag//x/0}"
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


check_scanner_version() {
    if ! is_release_version "$(make --quiet --no-print-directory scanner-tag)"; then
        echo "::error::Scanner tag does not look like a release tag. Please update SCANNER_VERSION file before releasing."
        exit 1
    fi
}

check_collector_version() {
    if ! is_release_version "$(make --quiet --no-print-directory collector-tag)"; then
        echo "::error::Collector tag does not look like a release tag. Please update COLLECTOR_VERSION file before releasing."
        exit 1
    fi
}

publish_roxctl() {
 if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: publish_roxctl <tag>"
    fi

    local tag="$1"

    echo "Push roxctl to gs://sr-roxc & gs://rhacs-openshift-mirror-src/assets" >> "${GITHUB_STEP_SUMMARY}"

    local temp_dir
    temp_dir="$(mktemp -d)"
    "${SCRIPTS_ROOT}/scripts/ci/roxctl-publish/prepare.sh" . "${temp_dir}"
    "${SCRIPTS_ROOT}/scripts/ci/roxctl-publish/publish.sh" "${temp_dir}" "${tag}" "gs://sr-roxc"
    "${SCRIPTS_ROOT}/scripts/ci/roxctl-publish/publish.sh" "${temp_dir}" "${tag}" "gs://rhacs-openshift-mirror-src/assets"
}

push_helm_charts() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: push_helm_charts <tag>"
    fi

    local tag="$1"

    echo "Publish Helm charts to github repository stackrox/release-artifacts and create a PR" >> "${GITHUB_STEP_SUMMARY}"

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
    if grep -q "${tag}" RELEASED_VERSIONS; then
        echo "Skip RELEASED_VERSIONS file change, already up to date ..." >> "${GITHUB_STEP_SUMMARY}"
    else
        echo "Update RELEASED_VERSIONS file ..." >> "${GITHUB_STEP_SUMMARY}"
        echo "${collector_version} ${tag}  # Rox release ${tag} by ${username} at $(date)" \
            >>RELEASED_VERSIONS
        gitbot add RELEASED_VERSIONS
        gitbot commit -m "Automatic update of RELEASED_VERSIONS file for Rox release ${tag}"
        gitbot push origin "${branch_name}"
    fi

    PRs=$(gh pr list -s open \
            --head "${branch_name}" \
            --json number \
            --jq length)
    if [ "$PRs" -eq 0 ]; then
        echo "Create a PR for collector to add this release to its RELEASED_VERSIONS file" >> "${GITHUB_STEP_SUMMARY}"
        gh pr create \
            --title "Update RELEASED_VERSIONS for StackRox release ${tag}" \
            --body "Add entry into the RELEASED_VERSIONS file" >> "${GITHUB_STEP_SUMMARY}"
    fi
    popd
}

is_tagged() {
    local tags
    tags="$(git tag --contains)"
    [[ -n "$tags" ]]
}

is_nightly_run() {
    [[ "${BUILD_TAG:-}" =~ -nightly- ]] || [[ "${GITHUB_REF:-}" =~ nightly- ]]
}

is_in_PR_context() {
    if is_GITHUB_ACTIONS && [[ -n "${GITHUB_BASE_REF:-}" ]]; then
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
    if is_OPENSHIFT_CI && [[ -n "${PULL_NUMBER:-}" ]]; then
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
    elif is_GITHUB_ACTIONS; then
        local pull_request
        pull_request=$(jq --raw-output .pull_request.number "$GITHUB_EVENT_PATH") || {
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
    if is_OPENSHIFT_CI; then
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
    if is_GITHUB_ACTIONS; then
        [[ -n "${GITHUB_ACTION_REPOSITORY:-}" ]] || die "expect: GITHUB_ACTION_REPOSITORY"
        echo "${GITHUB_ACTION_REPOSITORY}"
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

get_commit_sha() {
    if is_OPENSHIFT_CI; then
        echo "${PULL_PULL_SHA:-${PULL_BASE_SHA}}"
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

# pr_has_pragma() - returns true if a pragma exists. A pragma is a key with
# value in the description body of a PR that influences how CI behaves.
# e.g. /pragma gk_release_channel:rapid.
pr_has_pragma() {
    if [[ "$#" -ne 1 ]]; then
        die "usage: pr_has_pragma <key>"
    fi

    local pr_details
    if ! pr_details="$(get_pr_details)"; then
        info "Warning: checking for a pragma in a non PR context"
        return 0
    fi

    local key_to_check="$1"
    [[ "$(jq -r '.body' <<<"$pr_details")" =~ \/pragma:[[:space:]]*$key_to_check: ]]
}

# pr_get_pragma() - outputs the pragma key value if it exists.
pr_get_pragma() {
    if [[ "$#" -ne 1 ]]; then
        die "usage: pr_get_pragma <key>"
    fi

    local pr_details
    if ! pr_details="$(get_pr_details)"; then
        echo ''
        return 0
    fi

    local key_to_check="$1"
    while IFS= read -r line; do
        if [[ "$line" =~ \/pragma:[[:space:]]*$key_to_check:[[:space:]]*(.+) ]]; then
            # shellcheck disable=SC2001
            echo "${BASH_REMATCH[1]}" | sed -e 's/[[:space:]]*$//'
        fi
    done <<< "$(jq -r '.body' <<<"$pr_details")"
}

# get_pr_details() from GitHub and display the result. Exits 1 if not run in CI in a PR context.
_PR_DETAILS=""
_PR_DETAILS_CACHE_FILE="/tmp/PR_DETAILS_CACHE.json"
get_pr_details() {
    local pull_request
    local org
    local repo

    if [[ -n "${_PR_DETAILS}" ]]; then
        echo "${_PR_DETAILS}"
        return
    fi
    if [[ -e "${_PR_DETAILS_CACHE_FILE}" ]]; then
        _PR_DETAILS="$(cat "${_PR_DETAILS_CACHE_FILE}")"
        echo "${_PR_DETAILS}"
        return
    fi

    _not_a_PR() {
        echo '{ "msg": "this is not a PR" }'
        exit 1
    }

    if is_OPENSHIFT_CI; then
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
        [[ "${pull_request}" == "null" ]] && _not_a_PR
        org="${GITHUB_REPOSITORY_OWNER}"
        repo="${GITHUB_REPOSITORY#*/}"
    else
        echo "Unsupported CI"
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
    echo "$pr_details" | tee "${_PR_DETAILS_CACHE_FILE}"
}

openshift_ci_mods() {
    info "BEGIN OpenShift CI mods"

    local debug="${ARTIFACT_DIR:-/tmp}/debug.txt"

    echo "Env A-Z dump:" > "${debug}"
    env | sort | grep -E '^[A-Z]' >> "${debug}" || true

    ensure_writable_home_dir

    # Prevent fatal error "detected dubious ownership in repository" from recent git.
    git config --global --add safe.directory "$(pwd)"

    echo "Git log:" >> "${debug}"
    git log --oneline --decorate -n 20 >> "${debug}" || true

    echo "Recent git refs:" >> "${debug}"
    git for-each-ref --format='%(creatordate) %(refname)' --sort=creatordate | tail -20 >> "${debug}"

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
        if [[ -n "${PULL_PULL_SHA:-}" ]]; then
            sha="${PULL_PULL_SHA}"
        else
            sha=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].sha') || echo "WARNING: Cannot find pull sha"
        fi
        if [[ -n "${sha:-}" ]] && [[ "$sha" != "null" ]]; then
            info "Will checkout SHA to match PR: $sha"
            git checkout "$sha"
            git submodule update
        else
            echo "WARNING: Could not determine a SHA for this PR, ${sha:-}"
        fi
    fi

    # Target a tag if HEAD is tagged.
    BUILD_TAG="$(git tag --sort=creatordate --contains | tail -1)" || echo "Warning: Cannot get tag"
    export BUILD_TAG

    # For gradle
    export GRADLE_USER_HOME="${HOME}"

    handle_nightly_runs

    info "Status after mods:"
    "$ROOT/status.sh" || true

    STACKROX_BUILD_TAG=$(make --quiet --no-print-directory tag)
    export STACKROX_BUILD_TAG

    info "END OpenShift CI mods"
}

ensure_writable_home_dir() {
    # Single step test jobs do not have HOME
    if [[ -z "${HOME:-}" ]] || ! touch "${HOME}/openshift-ci-write-test"; then
        info "HOME (${HOME:-unset}) is not set or not writeable, using mktemp dir"
        HOME=$( mktemp -d )
        export HOME
        info "HOME is now $HOME"
    fi
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

    local nightly_tag_prefix
    nightly_tag_prefix="$(git describe --tags --abbrev=0 --exclude '*-nightly-*')-nightly-"
    if ! is_in_PR_context && [[ "${JOB_NAME_SAFE:-}" =~ ^nightly- ]]; then
        ci_export BUILD_TAG "${nightly_tag_prefix}$(date '+%Y%m%d')"
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
        make cli_host-arch upgrader
        make cli-install
        echo "Replacement roxctl is: $(command -v roxctl || true), version: $(roxctl version || true)"
    fi
}

store_qa_test_results() {
    if ! is_OPENSHIFT_CI; then
        return
    fi

    local to="${1:-qa-tests}"

    info "Copying qa-tests-backend results to $to"

    for test_results in qa-tests-backend/build/test-results/*; do
        store_test_results "$test_results" "$to"
    done
}

stored_test_results() {
    echo "${ARTIFACT_DIR}/junit-$1"
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

    local dest
    dest="$(stored_test_results "$to")"

    cp -a "$from" "$dest" || true # (best effort)
}

post_process_test_results() {
    if ! is_OPENSHIFT_CI; then
        return 0
    fi

    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "ERROR: ARTIFACT_DIR is not set which is expected in openshift CI"
        return 0
    fi

    local csv_output
    local extra_args=()

    set +u
    {
        if is_in_PR_context || [[ "${PULL_BASE_REF:-unknown}" =~ ^release ]]; then
            info "Converting JUNIT found in ${ARTIFACT_DIR} to CSV"
            extra_args=(--dry-run)
        else
            info "Creating JIRA issues for failures found in ${ARTIFACT_DIR}"
        fi

        csv_output="$(mktemp --suffix=.csv)"

        curl --retry 5 -SsfL https://github.com/stackrox/junit2jira/releases/download/v0.0.11/junit2jira -o junit2jira && \
        chmod +x junit2jira && \
        ./junit2jira \
            -base-link "$(echo "$JOB_SPEC" | jq ".refs.base_link" -r)" \
            -build-id "${BUILD_ID}" \
            -build-link "https://prow.ci.openshift.org/view/gs/origin-ci-test/logs/$JOB_NAME/$BUILD_ID" \
            -build-tag "${STACKROX_BUILD_TAG}" \
            -csv-output "${csv_output}" \
            -job-name "${JOB_NAME}" \
            -junit-reports-dir "${ARTIFACT_DIR}" \
            -orchestrator "${ORCHESTRATOR_FLAVOR:-PROW}" \
            -threshold 5 \
            -html-output "$ARTIFACT_DIR/junit2jira-summary.html" \
            "${extra_args[@]}"

        info "Creating Big Query test records from ${csv_output}"
        bq load \
            --skip_leading_rows=1 \
            --allow_quoted_newlines \
            ci_metrics.stackrox_tests "${csv_output}"
    } || true
    set -u
}

send_slack_notice_for_failures_on_merge() {
    local exitstatus="${1:-}"

    if ! is_OPENSHIFT_CI || [[ "$exitstatus" == "0" ]] || is_in_PR_context || is_nightly_run; then
        return 0
    fi

    if [[ "${PULL_BASE_REF:-unknown}" =~ ^release ]]; then
        info "Skipping slack message for release branches"
        return 0
    fi

    if is_system_test_without_images; then
        # Avoid multiple slack messages from the e2e tests waiting for images.
        info "Skipping slack message for a system test failure when images were not found"
        return 0
    fi

    local webhook_url="${TEST_FAILURES_NOTIFY_WEBHOOK}"
    local log_url="https://prow.ci.openshift.org/view/gs/origin-ci-test/logs/${JOB_NAME:-missing}/${BUILD_ID:-missing}"

    function slack_error() {
        echo "ERROR: $1"
        curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url" << __EOM__
{ "text": "*An error occurred dealing with a test failure:*\n\t- Test: ${log_url}.\n\t- $1." }
__EOM__
    }

    function check_env() {
        (
            set +u
            if [[ -z "$(eval echo "\$$1")" ]]; then
                slack_error "An expected environment variable is unset/empty: $1"
                return 1
            fi
        )
    }

    if [[ -n "${JOB_SPEC:-}" ]]; then
        org=$(jq -r <<<"$JOB_SPEC" '.refs.org')
        repo=$(jq -r <<<"$JOB_SPEC" '.refs.repo')
    elif [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
        org=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].org')
        repo=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].repo')
    else
        slack_error "Expect a JOB_SPEC or CLONEREFS_OPTIONS"
        return 1
    fi

    if [[ "$org" == "null" ]] || [[ "$repo" == "null" ]]; then
        slack_error "Could not determine org and/or repo"
        return 1
    fi

    check_env "PULL_BASE_SHA"
    check_env "JOB_NAME_SAFE"
    check_env "JOB_NAME"
    check_env "BUILD_ID"

    local commit_details_url="https://api.github.com/repos/${org}/${repo}/commits/${PULL_BASE_SHA}"
    local exitstatus=0
    local commit_details
    commit_details=$(curl --retry 5 -sS "${commit_details_url}") || exitstatus="$?"
    if [[ "$exitstatus" != "0" ]]; then
        slack_error "Cannot get commit details: ${commit_details}"
        return 1
    fi

    local job_name="${JOB_NAME_SAFE#merge-}"

    local commit_msg
    commit_msg=$(jq -r <<<"$commit_details" '.commit.message') || exitstatus="$?"
    commit_msg="${commit_msg%%$'\n'*}" # use first line of commit msg
    local commit_url
    commit_url=$(jq -r <<<"$commit_details" '.html_url') || exitstatus="$?"
    local author_name
    author_name=$(jq -r <<<"$commit_details" '.commit.author.name') || exitstatus="$?"
    local author_login
    author_login=$(jq -r <<<"$commit_details" '.author.login') || exitstatus="$?"
    if [[ "$exitstatus" != "0" ]]; then
        slack_error "Error parsing the commit details: ${commit_details}"
        return 1
    fi

    local slack_mention
    slack_mention="$("$SCRIPTS_ROOT"/scripts/ci/get-slack-user-id.sh "$author_login")"
    if [[ -n "$slack_mention" ]]; then
      slack_mention="<@${slack_mention}>"
    else
      slack_mention="_unable to resolve Slack user for GitHub login ${author_login}_"
    fi

    info "Converting junit failures to slack attachments"

    local slack_attachments='
[
  {
    "color": "#bb2124",
    "blocks": [
      {
        "type": "section",
        "text": {
          "type": "plain_text",
          "text": "Could not parse junit files. Check build logs for more information."
        }
      }
    ]
  }
]
'
    if [[ -n "${ARTIFACT_DIR}" ]]; then
        if ! command -v junit-parse >/dev/null 2>&1; then
            get_junit_parse_cli || true
        fi
        if command -v junit-parse >/dev/null 2>&1; then
            local junit_file_names=()
            while IFS='' read -r line; do junit_file_names+=("$line"); done < <(find "${ARTIFACT_DIR}" -type f -name '*.xml' || true)
            local check_slack_attachments
            check_slack_attachments=$(junit-parse "${junit_file_names[@]}") || exitstatus="$?"
            if [[ "$exitstatus" == "0" ]]; then
                slack_attachments="$check_slack_attachments"
            fi
        fi
    fi

    # shellcheck disable=SC2016
    local body='
{
    "text": "Prow job failure: \($job_name)",
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
                "text": "*Commit:* <\($commit_url)|\($commit_msg)>\n*Repo:* \($repo)\n*Author:* \($author_name), \($slack_mention)\n*Log:* \($log_url)"
            }
        },
        {
            "type": "context",
            "elements": [
                {
                    "type": "mrkdwn",
                    "text": "You got tagged but have no idea why or what to do? Check <https://docs.google.com/document/d/1d5ga073jkv4CO1kAJqp8MPGpC6E1bwyrCGZ7S5wKg3w/edit#heading=h.li2pdsxtk1hu|this document>."
                }
            ]
        },
        {
            "type": "divider"
        }
    ],
    "attachments": $slack_attachments
}
'

    payload="$(jq --null-input \
      --arg job_name "$job_name" \
      --arg commit_url "$commit_url" \
      --arg commit_msg "$commit_msg" \
      --arg repo "$repo" \
      --arg author_name "$author_name" \
      --arg slack_mention "$slack_mention" \
      --arg log_url "$log_url" \
      --argjson slack_attachments "$slack_attachments" \
      "$body")"
    echo -e "About to post:\n$payload"

    echo "$payload" | curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url" || {
        slack_error "Error posting to Slack"
        return 1
    }
}

junit_wrap() {
    if [[ "$#" -lt 4 ]]; then
        die "missing args. usage: junit_wrap <class> <description> <failure_message> <command> [ args ]"
    fi

    local class="$1"; shift
    local description="$1"; shift
    local failure_message="$1"; shift
    local command_output=""

    if command_output="$("$@" 2>&1)"; then
        echo "${command_output}"
        save_junit_success "${class}" "${description}"
    else
        local ret_code="$?"
        echo "${command_output}"

        local failure_body=""
        if [[ -n "$failure_message" ]]; then
            failure_body="${failure_message}
"
        fi
        if [[ "${#command_output}" -gt 512 ]]; then
            command_output="...${command_output: -512}"
        fi
        failure_body="${failure_body}Command output: ${command_output}"

        save_junit_failure "${class}" "${description}" "${failure_body}"

        return ${ret_code}
    fi
}

junit_contains_failure() {
    local dir="$1"
    if [[ ! -d $dir ]]; then
        return 1
    fi
    # There should be few files in such dir, and they should have well-behaved names,
    # and "return" does not mix with piping to "while read", so we use a "for" over find.
    # shellcheck disable=SC2044
    for f in $(find "$dir" -type f -iname '*.xml'); do
        if grep -q '<failure ' "$f"; then
            return 0
        fi
    done
    return 1
}

get_junit_misc_dir() {
    echo "${ARTIFACT_DIR}/junit-misc"
}

save_junit_success() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: save_junit_success <class> <description>"
    fi

    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "Warning: save_junit_success() requires the \$ARTIFACT_DIR variable to be set"
        return
    fi

    save_junit_record "$@"
}

save_junit_failure() {
    if [[ "$#" -ne 3 ]]; then
        die "missing args. usage: save_junit_failure <class> <description> <details>"
    fi

    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "Warning: save_junit_failure() requires the \$ARTIFACT_DIR variable to be set"
        return
    fi

    save_junit_record "$@"
}

remove_junit_record() {
    local class="$1"
    local junit_dir
    junit_dir="$(get_junit_misc_dir)"
    local junit_file="${junit_dir}/junit-${class}.xml"
    rm -f "${junit_file}"
}

save_junit_record() {
    local class="$1"
    local description="$2"
    local details="${3:-SUCCESS}"

    local junit_dir
    junit_dir="$(get_junit_misc_dir)"
    mkdir -p "${junit_dir}/db"

    # base64 encode failure details to condense multilines
    if [[ $details != "SUCCESS" ]]; then
        details="$(base64 -w0 <<< "$details")"
    fi

    # record this instance
    local record="${junit_dir}/db/${class}.txt"
    echo "${description}" >> "${record}"
    echo "${details}" >> "${record}"

    local tests
    tests=$(( "$(wc -l < "${record}")" / 2 ))

    local failures=0
    local lines
    readarray -t lines < "${record}"
    while (( ${#lines[@]} ))
    do
        local details="${lines[1]}"
        if [[ "$details" != "SUCCESS" ]]; then
            failures=$(( failures+1 ))
        fi
        lines=( "${lines[@]:2}" )
    done

    local junit_file="${junit_dir}/junit-${class}.xml"

    cat << _EO_SUITE_HEADER_ > "${junit_file}"
<testsuite name="${class}" tests="${tests}" skipped="0" failures="${failures}" errors="0">
_EO_SUITE_HEADER_

    readarray -t lines < "${record}"
    while (( ${#lines[@]} ))
    do
        local description="${lines[0]}"
        local details="${lines[1]}"

        cat << _EO_CASE_HEADER_ >> "${junit_file}"
        <testcase name="${description}" classname="${class}">
_EO_CASE_HEADER_

        if [[ "$details" != "SUCCESS" ]]; then
            details="$(base64 --decode <<< "$details")"
        cat << _EO_FAILURE_ >> "${junit_file}"
            <failure><![CDATA[${details}]]></failure>
_EO_FAILURE_
        fi

        echo "        </testcase>" >> "${junit_file}"

        lines=( "${lines[@]:2}" )
    done

    echo "</testsuite>" >> "${junit_file}"
}

add_build_comment_to_pr() {
    info "Adding a comment with the build tag to the PR"

    local pr_details
    local exitstatus=0
    pr_details="$(get_pr_details)" || exitstatus="$?"
    if [[ "$exitstatus" != "0" ]]; then
        echo "DEBUG: Unable to get the PR details from GitHub: $exitstatus"
        echo "DEBUG: PR details: ${pr_details}"
        info "Will continue without commenting on the PR"
        return
    fi

    # hub-comment is tied to Circle CI env
    local url
    url=$(jq -r '.html_url' <<<"$pr_details")
    export CIRCLE_PULL_REQUEST="$url"

    local sha
    sha=$(jq -r '.head.sha' <<<"$pr_details")
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

get_junit_parse_cli() {
    go install github.com/stackrox/junit-parse@latest
}

is_system_test_without_images() {
    case "${CI_JOB_NAME:-missing}" in
        *-e2e-tests|*-upgrade-tests|*-version-compatibility-tests)
            [[ ! -f "${STATE_IMAGES_AVAILABLE}" ]]
            ;;
        *)
            false
            ;;
    esac
}

handle_gha_tagged_build() {
    if [[ -z "${GITHUB_REF:-}" ]]; then
        echo "No GITHUB_REF in env"
        exit 0
    fi
    echo "GITHUB_REF: ${GITHUB_REF}"
    if [[ "${GITHUB_REF:-}" =~ ^refs/tags/ ]]; then
        tag="${GITHUB_REF#refs/tags/*}"
        echo "This is a tagged build: $tag"
        echo "BUILD_TAG=$tag" >> "$GITHUB_ENV"
    else
        echo "This is not a tagged build"
    fi
}

slack_prow_notice() {
    info "Slack a notice that prow tests have started"

    if [[ "$#" -lt 1 ]]; then
        die "missing arg. usage: slack_prow_notice <tag>"
    fi

    local tag="$1"

    [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]] || is_nightly_run || {
        info "Skipping step as this is not a release, RC or nightly build"
        return 0
    }

    local build_url
    local webhook_url
    if [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]]; then
        local release
        release="$(get_release_stream "$tag")"
        build_url="https://prow.ci.openshift.org/?repo=stackrox%2Fstackrox&job=*release-$release*"
        if is_release_test_stream "$tag"; then
            # send to #acs-slack-integration-testing when testing the release process
            webhook_url="${SLACK_MAIN_WEBHOOK}"
        else
            # send to #acs-release-notifications
            webhook_url="${RELEASE_WORKFLOW_NOTIFY_WEBHOOK}"
        fi
    elif is_nightly_run; then
        build_url="https://prow.ci.openshift.org/?repo=stackrox%2Fstackrox&job=*stackrox*night*"
        # send to #acs-nightly-ci-runs
        webhook_url="${NIGHTLY_WORKFLOW_NOTIFY_WEBHOOK}"
    else
        die "unexpected"
    fi

    local github_url="https://github.com/stackrox/stackrox/releases/tag/$tag"

    jq -n \
    --arg build_url "$build_url" \
    --arg tag "$tag" \
    --arg github_url "$github_url" \
    '{"text": ":prow: Prow CI for tag <\($github_url)|\($tag)> started! Check the status of the tests under the following URL: \($build_url)"}' \
| curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url"
}

gather_debug_for_cluster_under_test() {
    highlight_cluster_versions
    record_cluster_info
}

highlight_cluster_versions() {
    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "No place for artifacts, skipping cluster version dump"
        return
    fi

    artifact_file="$ARTIFACT_DIR/cluster-version-summary.html"

    cat > "$artifact_file" <<- HEAD
<html>
    <head>
        <title>Cluster Versions</title>
        <style>
          body { color: #e8e8e8; background-color: #424242; font-family: "Roboto", "Helvetica", "Arial", sans-serif }
          a { color: #ff8caa }
          a:visited { color: #ff8caa }
        </style>
    </head>
    <body>
HEAD

    local nodes
    nodes="$(kubectl get nodes -o wide 2>&1 || true)"
    local kubectl_version
    kubectl_version="$(kubectl version -o json 2>&1 || true)"
    local oc_version
    oc_version="$(oc version -o json 2>&1 || true)"

    cat >> "$artifact_file" << DETAILS
      <h3>Nodes:</h3>
      kubectl get nodes -o wide
      <pre>$nodes</pre>
      <h3>Versions:</h3>
      kubectl version -o json
      <pre>$kubectl_version</pre>
      oc version -o json
      <pre>$oc_version</pre>
DETAILS

    cat >> "$artifact_file" <<- FOOT
    <br />
    <br />
  </body>
</html>
FOOT
}

record_cluster_info() {
    _record_cluster_info || {
        # Failure to gather metrics is not a test failure
        info "WARNING: Recording cluster info failed"
    }
}

_record_cluster_info() {
    info "Record some cluster info"

    # Assumes (a) there is a single cluster under test (cut_*) and (b) all nodes
    # in the cluster are homogeneous.

    # Product version. Currently used for OpenShift version. Could cover cloud
    # provider versions for example.
    local cut_product_version=""
    local oc_version
    oc_version="$(oc version -o json 2>&1 || true)"
    local openshiftVersion
    openshiftVersion=$(jq -r <<<"$oc_version" '.openshiftVersion')
    if [[ "$openshiftVersion" != "null" ]]; then
        cut_product_version="$openshiftVersion"
    fi

    # K8s version.
    local cut_k8s_version=""
    local kubectl_version
    kubectl_version="$(kubectl version -o json 2>&1 || true)"
    local serverGitVersion
    serverGitVersion=$(jq -r <<<"$kubectl_version" '.serverVersion.gitVersion')
    if [[ "$serverGitVersion" != "null" ]]; then
        cut_k8s_version="$serverGitVersion"
    fi

    # Node info: OS, Kernel & Container Runtime.
    local nodes
    nodes="$(kubectl get nodes -o json 2>&1 || true)"
    local osImage
    osImage=$(jq -r <<<"$nodes" '.items[0].status.nodeInfo.osImage')
    local cut_os_image=""
    if [[ "$osImage" != "null" ]]; then
        cut_os_image="$osImage"
    fi
    local kernelVersion
    kernelVersion=$(jq -r <<<"$nodes" '.items[0].status.nodeInfo.kernelVersion')
    local cut_kernel_version=""
    if [[ "$kernelVersion" != "null" ]]; then
        cut_kernel_version="$kernelVersion"
    fi
    local containerRuntimeVersion
    containerRuntimeVersion=$(jq -r <<<"$nodes" '.items[0].status.nodeInfo.containerRuntimeVersion')
    local cut_container_runtime_version=""
    if [[ "$containerRuntimeVersion" != "null" ]]; then
        cut_container_runtime_version="$containerRuntimeVersion"
    fi

    update_job_record \
      cut_product_version "$cut_product_version" \
      cut_k8s_version "$cut_k8s_version" \
      cut_os_image "$cut_os_image" \
      cut_kernel_version "$cut_kernel_version" \
      cut_container_runtime_version "$cut_container_runtime_version"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
