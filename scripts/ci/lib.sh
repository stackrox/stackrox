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

# set_ci_shared_export() - for openshift-ci and GHA this is state shared between steps.
set_ci_shared_export() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: set_ci_shared_export <env-name> <env-value>"
    fi

    ci_export "$@"

    local env_name="$1"
    local env_value="$2"

    echo "export ${env_name}=${env_value}" >> "${SHARED_DIR:-/tmp}/shared_env"
    echo "${env_name}=${env_value}" >> "${GITHUB_ENV:-/dev/null}"
}

ci_exit_trap() {
    local exit_code="$?"
    info "Executing a general purpose exit trap for CI"
    echo "Exit code is: ${exit_code}"

    if [[ "${exit_code}" == "0" ]]; then
        set_ci_shared_export JOB_DISPATCH_OUTCOME "${OUTCOME_PASSED}"
    elif [[ "${exit_code}" == "130" ]]; then
        set_ci_shared_export JOB_DISPATCH_OUTCOME "${OUTCOME_CANCELED}"
    else
        set_ci_shared_export JOB_DISPATCH_OUTCOME "${OUTCOME_FAILED}"
    fi

    post_process_test_results "${JOB_SLACK_FAILURE_ATTACHMENTS}" "${JOB_JUNIT2JIRA_SUMMARY_FILE}"

    while [[ -e /tmp/hold ]]; do
        info "Holding this job for debug"
        sleep 60
    done

    handle_dangling_processes

    gate_flaky_tests "${exit_code}"
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
    require_environment "ROX_ADMIN_PASSWORD"
    # TODO(ROX-28673): Temporary reset the serve name to fix the CI:
    roxctl -s "" -e "${API_ENDPOINT}" \
        central debug dump --output-dir "${output_dir}" \
        --insecure-skip-tls-verify
    ls -l "${output_dir}"
}

process_central_metrics() {
    info "Processing metrics from central debug dump"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: process_central_metrics <debug_dump_dir>"
    fi

    local output_dir="$1"

    local metrics_output
    local csv_output
    local debug_dump_zip
    metrics_output="$(mktemp --suffix=.prom)"
    csv_output="$(mktemp --suffix=.csv)"
    # shellcheck disable=SC2012
    debug_dump_zip="$(ls -t "${output_dir}"/*.zip | head -1)"
    unzip -p "${debug_dump_zip}" metrics-1 > "${metrics_output}"

    get_prometheus_metrics_parser

    # We need a link to repository. In case it's not part of job spec (e.g., periodic`s)
    # we will fallback to short commit
    base_link="$(echo "$JOB_SPEC" | jq ".refs.base_link | select( . != null )" -r)"
    calculated_base_link="https://github.com/stackrox/stackrox/commit/$(make --quiet --no-print-directory shortcommit)"

    local metadata="build_tag=${STACKROX_BUILD_TAG:-none},build_id=${BUILD_ID:-none},orchestrator_flavor=${ORCHESTRATOR_FLAVOR:-PROW},job_name=${JOB_NAME:-missing},base_link=${base_link:-$calculated_base_link}"
    prometheus-metric-parser single \
        --format csv \
        --file "${metrics_output}" \
        --labels "${metadata}" \
        > "${csv_output}"

    setup_gcp
    save_central_metrics "${csv_output}"
}

get_prometheus_metrics_parser() {
    local parserBin
    local parserDir
    parserBin=$(make prometheus-metric-parser -C "$ROOT" --silent | tail -1)
    parserDir=$(dirname "${parserBin}")
    export PATH="$parserDir":$PATH
    prometheus-metric-parser help
}

get_central_diagnostics() {
    info "Getting central diagnostics"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: get_central_diagnostics <output_dir>"
    fi

    local output_dir="$1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_ADMIN_PASSWORD"
    # TODO(ROX-28673): Temporary reset the serve name to fix the CI:
    roxctl -s "" -e "${API_ENDPOINT}" \
        central debug download-diagnostics --output-dir "${output_dir}" \
        --insecure-skip-tls-verify
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
    registry="$(registry_from_branding "$brand")"

    local tag
    tag="$(make --quiet --no-print-directory tag)"

    registry_rw_login "$registry"
    for image in "${main_image_set[@]}"; do
        retry 5 true \
          "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:${tag}" "$architectures" | cat
        if [[ "$push_context" == "merge-to-master" ]]; then
            retry 5 true \
              "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:latest" "$architectures" | cat
        fi
    done

    # Push manifest lists for scanner and collector for amd64 only
    local amd64_image_set=("scanner" "scanner-db" "scanner-slim" "scanner-db-slim" "collector")
    for image in "${amd64_image_set[@]}"; do
        retry 5 true \
          "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:${tag}" "amd64" | cat
    done
}

registry_from_branding() {
    local branding="$1"
    if [[ "$branding" == "STACKROX_BRANDING" ]]; then
        registry="quay.io/stackrox-io"
    elif [[ "$branding" == "RHACS_BRANDING" ]]; then
        registry="quay.io/rhacs-eng"
    else
        die "$branding is not a supported branding"
    fi
    echo "$registry"
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

    _push_main_image_set() {
        local registry="$1"
        local tag="$2"

        for image in "${main_image_set[@]}"; do
            retry 5 true \
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

    local registry
    registry="$(registry_from_branding "$brand")"

    local tag
    tag="$(make --quiet --no-print-directory tag)"

    registry_rw_login "$registry"

    _tag_main_image_set "$tag" "$registry" "$tag-$arch"
    _push_main_image_set "$registry" "$tag-$arch"

    if [[ "$push_context" == "merge-to-master" ]]; then
        _tag_main_image_set "$tag" "$registry" "latest-${arch}"
        _push_main_image_set "$registry" "latest-${arch}"
    fi
}

push_scanner_image_manifest_lists() {
    info "Pushing scanner-v4 and scanner-v4-db images as manifest lists"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: push_scanner_image_manifest_lists <registry> <architectures (CSV)>"
    fi

    local registry="$1"
    local architectures="$2"
    local scanner_image_set=("scanner-v4" "scanner-v4-db")

    local tag
    tag="$(make --quiet --no-print-directory -C scanner tag)"
    registry_rw_login "$registry"
    for image in "${scanner_image_set[@]}"; do
        retry 5 true \
          "$SCRIPTS_ROOT/scripts/ci/push-as-multiarch-manifest-list.sh" "${registry}/${image}:${tag}" "$architectures" | cat
    done
}

push_scanner_image_set() {
    info "Pushing scanner-v4 and scanner-v4-db images"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: push_scanner_image_set <registry> <arch>"
    fi

    local registry="$1"
    local arch="$2"

    local scanner_image_set=("scanner-v4" "scanner-v4-db")

    _push_scanner_image_set() {
        local registry="$1"
        local tag="$2"

        for image in "${scanner_image_set[@]}"; do
            retry 5 true \
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

    local tag
    tag="$(make --quiet --no-print-directory -C scanner tag)"

    registry_rw_login "$registry"

    _tag_scanner_image_set "$tag" "$registry" "$tag-$arch"
    _push_scanner_image_set "$registry" "$tag-$arch"
}

registry_rw_login() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: registry_rw_login <registry>"
    fi

    local registry="$1"

    case "$registry" in
        quay.io/rhacs-eng)
            _login() {
                # shellcheck disable=SC2317
                docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io
            }
            retry 5 true _login
            ;;
        quay.io/stackrox-io)
            _login() {
                # shellcheck disable=SC2317
                docker login -u "$QUAY_STACKROX_IO_RW_USERNAME" --password-stdin <<<"$QUAY_STACKROX_IO_RW_PASSWORD" quay.io
            }
            retry 5 true _login
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
            _login() {
                # shellcheck disable=SC2317
                docker login -u "$QUAY_RHACS_ENG_RO_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RO_PASSWORD" quay.io
            }
            retry 5 true _login
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

    local brand="$1"
    local arch="$2"

    local registry
    registry="$(registry_from_branding "$brand")"

    _retag() {
        retry 5 true "$SCRIPTS_ROOT/scripts/ci/pull-retag-push.sh" "$1" "$2"
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

    registry_rw_login "${registry}"

    _retag "${registry}/scanner:${scanner_version}"    "${registry}/scanner:${main_tag}-${arch}"
    _retag "${registry}/scanner-db:${scanner_version}" "${registry}/scanner-db:${main_tag}-${arch}"
    _retag "${registry}/scanner-slim:${scanner_version}"    "${registry}/scanner-slim:${main_tag}-${arch}"
    _retag "${registry}/scanner-db-slim:${scanner_version}" "${registry}/scanner-db-slim:${main_tag}-${arch}"

    _retag "${registry}/collector:${collector_version}"      "${registry}/collector:${main_tag}-${arch}"
}

poll_for_system_test_images() {
    info "Polling for images required for system tests"

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: poll_for_system_test_images <seconds to wait>"
    fi

    local time_limit="$1"

    require_environment "QUAY_RHACS_ENG_BEARER_TOKEN"

    local image_list
    image_list="$(mktemp)"
    populate_stackrox_image_list "${image_list}"
    info "Will poll for: $(awk '{print $1}' "${image_list}")"

    local start_time
    start_time="$(date '+%s')"

    local tag
    local image
    while read -r image tag
    do
        while ! check_rhacs_eng_image_exists "$image" "$tag"
        do
            info "$image does not exist"
            if (( $(date '+%s') - start_time > time_limit )); then
                check_build_workflows "$(get_commit_sha)"
                die "ERROR: Timed out waiting for images after ${time_limit} seconds"
            fi
            sleep 60
        done
    done < "$image_list"

    info "All images exist."
    touch "${STATE_IMAGES_AVAILABLE}"
}

# Image prefetch is broken into two sets:
# - prebuilt: test fixture images that already exist prior to CI job execution.
#   Prefetch for these images can start as soon as the cluster is ready.
# - system: the product images under test 'stackrox-images'. Prefetch for these
#   should not start until the images are built to avoid backoff noise in the
#   prefetcher logs.

image_prefetcher_await_msg_prefix="Waiting for pre-fetcher"

image_prefetcher_prebuilt_start() {
    info "Starting image pre-fetcher of pre-built images (NOT built for current commit)..."
    junit_wrap image-prefetcher-prebuilt-start \
               "Start image pre-fetcher for pre-built images (NOT built for current commit)." \
               "See log for error details." \
               _image_prefetcher_prebuilt_start
}

image_prefetcher_system_start() {
    info "Starting image pre-fetcher of system images (built for current commit)..."
    junit_wrap image-prefetcher-system-start \
               "Start image pre-fetcher for system images (built for current commit)." \
               "See log for error details." \
               _image_prefetcher_system_start
}

_image_prefetcher_prebuilt_start() {
    case "$CI_JOB_NAME" in
    *qa-e2e-tests)
        image_prefetcher_start_set qa-e2e
        # Override the default image pull policy for containers with quay.io
        # images to rely on prefetched images. This helps ensure that the static
        # prefect list stays up to date with additions.
        ci_export "IMAGE_PULL_POLICY_FOR_QUAY_IO" "Never"
        ;;
    # TODO(ROX-20508): for operaror-e2e jobs, pre-fetch images of the release from which operator upgrade test starts.
    *)
        info "No pre-built image prefetching is currently performed for: ${CI_JOB_NAME}."
        ;;
    esac
}

_image_prefetcher_system_start() {
    case "$CI_JOB_NAME" in
    # ROX-24818: GKE is excluded from system image prefetch as it causes
    # flakes in test.
    *-operator-e2e-tests|*ocp*qa-e2e-tests)
        image_prefetcher_start_set stackrox-images
        ;;
    # Enabling scanner V4 installation tests as well, even though they also run on GKE,
    # for gathering some more data points for CI reliability.
    *-scanner-v4-install-tests)
        image_prefetcher_start_set stackrox-images
        ;;
    *)
        info "No system image prefetching is performed for: ${CI_JOB_NAME}."
        ;;
    esac
}

image_prefetcher_start_set() {
    local ns="prefetch-images"
    local name="$1"

    make image-prefetcher-deploy-bin
    local image_prefetcher_deploy_bin
    image_prefetcher_deploy_bin="$(make print-image-prefetcher-deploy-bin)"
    local image_prefetcher_version
    image_prefetcher_version="$(go -C tools/test list -m -f '{{.Version}}' github.com/stackrox/image-prefetcher/deploy)"

    info "Using ${image_prefetcher_deploy_bin} ${image_prefetcher_version} for image prefetch deployment"

    local manifest
    manifest=$(mktemp)

    case "${ORCHESTRATOR_FLAVOR}" in
    k8s)
        flavor=vanilla
        ;;
    openshift)
        flavor=ocp
        ;;
    *)
        die "unsupported ORCHESTRATOR: ${ORCHESTRATOR_FLAVOR}"
        ;;
    esac

    # daemonset, etc
    ${image_prefetcher_deploy_bin} \
        --version="${image_prefetcher_version}" \
        --k8s-flavor="$flavor" \
        --secret=stackrox \
        --collect-metrics \
        "$name" > "$manifest"

    # image list
    local image_list
    image_list=$(mktemp)
    populate_prefetcher_image_list "$name" "${image_list}"
    echo "---" >> "$manifest"
    kubectl create --dry-run=client -o yaml --namespace=$ns configmap "$name" --from-file="images.txt=$image_list" >> "$manifest"

    # pull secret
    REGISTRY_PASSWORD="${QUAY_RHACS_ENG_RO_PASSWORD}" \
    REGISTRY_USERNAME="${QUAY_RHACS_ENG_RO_USERNAME}" \
    NAMESPACE=$ns \
      make -C operator stackrox-image-pull-secret

    # apply configmap, daemonset etc
    retry 5 true kubectl apply --namespace=$ns -f "$manifest"
    info "Image pre-fetcher is now running in the background. Its status will be checked later (look for message starting with ${image_prefetcher_await_msg_prefix}). Proceeding with other tasks in the meantime."
    rm -f "$image_list" "$manifest"
}

image_prefetcher_prebuilt_await() {
    if [[ "${IMAGE_PREFETCH_DISABLED:-false}" == "true" ]]; then
        return
    fi

    info "${image_prefetcher_await_msg_prefix} of pre-built images to complete..."
    junit_wrap image-prefetcher-prebuilt-await \
               "Waiting for pre-fetcher of pre-built images to complete." \
               "See log for error details." \
               _image_prefetcher_prebuilt_await
}

image_prefetcher_system_await() {
    if [[ "${IMAGE_PREFETCH_DISABLED:-false}" == "true" ]]; then
        return
    fi

    info "${image_prefetcher_await_msg_prefix} of system images to complete..."
    junit_wrap image-prefetcher-system-await \
               "Waiting for pre-fetcher of system images to complete." \
               "See log for error details." \
               _image_prefetcher_system_await
}

_image_prefetcher_prebuilt_await() {
    case "$CI_JOB_NAME" in
    *qa-e2e-tests)
        image_prefetcher_await_set qa-e2e
        ;;
    # TODO(ROX-20508): for operaror-e2e jobs, pre-fetch images of the release from which operator upgrade test starts.
    *)
        info "No pre-built image prefetching is currently performed for: ${CI_JOB_NAME}. Nothing to wait for."
        ;;
    esac
}

_image_prefetcher_system_await() {
    case "$CI_JOB_NAME" in
    # ROX-24818: GKE is excluded from system image prefetch as it causes
    # flakes in test.
    *-operator-e2e-tests|*ocp*qa-e2e-tests)
        image_prefetcher_await_set stackrox-images
        ;;
    # Enabling scanner V4 installation tests as well, even though they also run on GKE,
    # for gathering some more data points for CI reliability.
    *-scanner-v4-install-tests)
        image_prefetcher_await_set stackrox-images
        ;;
    *)
        info "No system image prefetching is performed for: ${CI_JOB_NAME}. Nothing to wait for."
        ;;
    esac
}

image_prefetcher_await_set() {
    local ns="prefetch-images"
    local name="$1"
    local extra_fields='{"build_id": "'"${BUILD_ID:-}"'", "job_name": "'"${JOB_NAME:-}"'", "orchestrator": "'"${ORCHESTRATOR_FLAVOR:-}"'", "build_tag": "'"${STACKROX_BUILD_TAG:-}"'"}'

    info "Waiting for image prefetcher set ${name} to complete..."
    if kubectl rollout status daemonset "$name" -n "$ns" --timeout 15m; then
        info "All images in the set are now pre-fetched."
    else
        info "WARNING: Pre-fetching failed to complete in time."
        info "To investigate closer, go to https://console.cloud.google.com/bigquery and run a query such as:"
        local query
        query=$(mktemp)
        cat > "${query}" <<- EOM

            SELECT started_at, duration_ms, image, error
            FROM \`acs-san-stackroxci.ci_metrics.stackrox_image_prefetches\`
            WHERE error IS NOT NULL AND
            $(echo "${extra_fields}" | jq -r '[to_entries | .[] | select(.value != "") | (.key + "=\"" + .value + "\"")] | join(" AND ")')
            ORDER BY started_at DESC LIMIT 1000

EOM
        cat "${query}"
        info "Note: The data is imported into the table periodically: https://github.com/stackrox/stackrox/actions/workflows/batch-load-test-metrics.yml"

        if [[ -n ${ARTIFACT_DIR:-} ]]; then
            local prefetcher_help="$ARTIFACT_DIR/image-pre-fetcher-${name}-failure-summary.html"
            cat > "${prefetcher_help}" <<- EOM
                <html>
                <head>
                <title>Image pre-fetcher ${name} failure</title>
                <style>
                  body { color: #e8e8e8; background-color: #424242; font-family: "Roboto", "Helvetica", "Arial", sans-serif }
                  a { color: #ff8caa }
                  a:visited { color: #ff8caa }
                </style>
                </head>
                <body>

                Waiting for image prefetcher set ${name} to complete timed out.<br>
                To investigate closer, go to <a target="_blank" href="https://console.cloud.google.com/bigquery">BigQuery</a> and run a query such as the following:
                <br>
                <pre>
EOM
            cat >> "${prefetcher_help}" "${query}"
            cat >> "${prefetcher_help}" <<- EOM
                </pre>
                Note: The data is imported into the table <a target="_blank" href="https://github.com/stackrox/stackrox/actions/workflows/batch-load-test-metrics.yml">periodically</a>.
                <br><br>
                </body>
                </html>
EOM
        fi
        rm -f "${query}"
    fi
    info "Now retrieving prefetcher metrics..."
    local attempt=0
    local service="service/${name}-metrics"
    while [[ -z $(kubectl -n "${ns}" get "${service}" -o jsonpath="{.status.loadBalancer.ingress}" 2>/dev/null) ]]; do
        if [ "$attempt" -lt "60" ]; then
            info "Waiting for ${service} to obtain endpoint ..."
            ((attempt++))
            sleep 10
        else
            info "Something is wrong with the ${service} service. See the following 'describe' output."
            kubectl -n "${ns}" describe "${service}" || true
            die "Timeout waiting for ${service} to obtain endpoint!"
        fi
    done
    local endpoint
    endpoint="$(kubectl -n "${ns}" get "${service}" -o json | service_get_endpoint)"
    local fetcher_metrics
    fetcher_metrics="$(mktemp --suffix=.csv)"
    local fetcher_metrics_json
    fetcher_metrics_json="$(mktemp --suffix=.json)"
    local metrics_url="http://${endpoint}:8080/metrics"
    if ! curl --silent --show-error --fail --retry 3 --retry-connrefused "${metrics_url}" > "${fetcher_metrics_json}"; then
        die "Failed to fetch prefetcher metrics from ${metrics_url}"
    fi
    # See the stackrox_image_prefetches table definition in https://github.com/stackrox/automation-iac/blob/main/resources/testing/stackrox-ci/metrics.tf
    # for the order of columns.
    if ! jq --raw-output \
      --argjson cols '["attempt_id", "started_at", "image", "duration_ms", "node", "size_bytes", "error", "build_id", "job_name", "orchestrator", "build_tag"]' \
      --argjson extra "${extra_fields}" \
      'map(.started_at = (.started_at | todate) | ($extra+.) as $row | $cols | map($row[.])) as $rows | $cols, $rows[] | @csv' \
      "${fetcher_metrics_json}" > "${fetcher_metrics}"; then
        info "WARNING: Failed to convert image prefetcher metrics to CSV with extra fields ${extra_fields}"
        info "Dumping the input JSON file:"
        jq . < "${fetcher_metrics_json}"
        die "Failed to convert image prefetcher metrics to CSV, aborting."
    fi
    rm -f "${fetcher_metrics_json}"

    if save_image_prefetches_metrics "${fetcher_metrics}"; then
        info "Image pre-fetcher metrics retrieved and saved."
    else
        info "WARNING: failed to save image pre-fetcher metrics."
    fi
    rm -f "${fetcher_metrics}"
}

service_get_endpoint() {
    jq -r '.status.loadBalancer.ingress // error("List of ingress points of LB " + .metadata.name + " is empty.") | .[0] | .hostname // .ip'
}

populate_prefetcher_image_list() {
    local name="$1"
    local image_list="$2"

    case "$name" in
    stackrox-images)
        local image_tag_list
        image_tag_list=$(mktemp)
        populate_stackrox_image_list "${image_tag_list}"
        # convert format from "basename tag" to "quay.io/.../basename:tag" expected by pre-fetcher
        awk '{print "quay.io/rhacs-eng/" $1 ":" $2}' "$image_tag_list" > "$image_list"
        rm -f "$image_tag_list"
        ;;
    qa-e2e)
        cp "$SCRIPTS_ROOT/qa-tests-backend/scripts/images-to-prefetch.txt" "$image_list"
        ;;
    *)
        die "ERROR: An unsupported image prefetcher target was requested: $name"
        ;;
    esac
}

populate_stackrox_image_list() {
    local image_list="$1"

    local tag
    tag="$(make --quiet --no-print-directory tag)"
    local operator_metadata_tag
    operator_metadata_tag="$(echo "v${tag}" | sed 's,x,0,')"
    local operator_controller_tag="${tag//x/0}"

    # Require images based on the job
    case "$CI_JOB_NAME" in
        *-operator-e2e-tests)
            cat >> "${image_list}" << END
stackrox-operator ${operator_controller_tag}
stackrox-operator-bundle ${operator_metadata_tag}
stackrox-operator-index ${operator_metadata_tag}
main ${tag}
central-db ${tag}
collector ${tag}
scanner ${tag}
scanner-db ${tag}
scanner-v4 ${tag}
scanner-v4-db ${tag}
END
            ;;
        *-race-condition-qa-e2e-tests)
            cat >> "${image_list}" << END
central-db ${tag}
main ${tag}-rcd
roxctl ${tag}
END
            if is_in_PR_context && ! pr_has_label "ci-build-race-condition-debug"; then
                echo "ERROR: Your PR is missing the \"ci-build-race-condition-debug\" label."
                echo "ERROR: This label is required to build the images for $CI_JOB_NAME."
                # Quietly continue to allow labels added after tests start.
                # Otherwise this message will surface in the Prow log when
                # images timeout out below.
            fi
            ;;
        *-scanner-v4-install-tests)
            cat >> "${image_list}" << END
stackrox-operator ${operator_controller_tag}
stackrox-operator-bundle ${operator_metadata_tag}
stackrox-operator-index ${operator_metadata_tag}
main ${tag}
central-db ${tag}
collector ${tag}
scanner ${tag}
scanner-db ${tag}
scanner-v4 ${tag}
scanner-v4-db ${tag}
roxctl ${tag}
END
            ;;
        *)
            cat >> "${image_list}" << END
central-db ${tag}
main ${tag}
roxctl ${tag}
END
            ;;
    esac

    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR:-}" == "true" ]]; then
            cat >> "${image_list}" << END
stackrox-operator ${operator_controller_tag}
stackrox-operator-bundle ${operator_metadata_tag}
stackrox-operator-index ${operator_metadata_tag}
END
    fi

    # Remove duplicates.
    unique="$(mktemp)"
    sort -u "${image_list}" > "${unique}"
    cat "${unique}" > "${image_list}"
    rm -f "${unique}"
}

check_rhacs_eng_image_exists() {
    local name="$1"
    local tag="$2"

    local url="https://quay.io/api/v1/repository/rhacs-eng/$name/tag?specificTag=$tag"
    info "Checking for $name using $url"
    local check
    local extra_args=()
    local public_images=("stackrox-operator-index")
    if [[ -n "${QUAY_RHACS_ENG_BEARER_TOKEN:-}" ]]; then
        extra_args+=("-H" "Authorization: Bearer ${QUAY_RHACS_ENG_BEARER_TOKEN}")
    else
        # shellcheck disable=SC2076
        if [[ ! " ${public_images[*]} " =~ " ${name} " ]]; then
            info "Warning: Checking for image existence without QUAY_RHACS_ENG_BEARER_TOKEN is not supported for image ${name}:${tag}"
            info "Warning: It is only supported for the following public image repositories: ${public_images[*]}"
        fi
    fi
    check=$(curl --location -sS "${extra_args[@]}" "$url")
    echo "$check"
    [[ "$(jq -r '.tags | first | .name' <<<"$check")" == "$tag" ]]
}

check_build_workflows() {
    local commit_sha="$1"

    {
        echo
        info "GitHub Actions workflow status for build.yaml:"
        check-workflow-run \
            --workflow=build.yaml \
            --head-SHA="${commit_sha}"

        echo
        info "GitHub Actions workflow status for scanner-build.yaml:"
        check-workflow-run \
            --workflow=scanner-build.yaml \
            --head-SHA="${commit_sha}"
    } | tee "${STATE_BUILD_RESULTS}" || true
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
    "${SCRIPTS_ROOT}/scripts/ci/artifacts-publish/prepare-roxctl.sh" . "${temp_dir}"
    "${SCRIPTS_ROOT}/scripts/ci/artifacts-publish/publish.sh" "${temp_dir}" "${tag}" "gs://sr-roxc"
    "${SCRIPTS_ROOT}/scripts/ci/artifacts-publish/publish.sh" "${temp_dir}" "${tag}" "gs://rhacs-openshift-mirror-src/assets"
}

publish_openapispec() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: publish_openapispec <tag>"
    fi

    local tag="$1"

    echo "Push OpenAPI spec to gs://rhacs-openshift-mirror-src/assets" >> "${GITHUB_STEP_SUMMARY}"

    local temp_dir
    temp_dir="$(mktemp -d)"
    "${SCRIPTS_ROOT}/scripts/ci/artifacts-publish/prepare-openapispec.sh" "${temp_dir}" "${tag}"
    "${SCRIPTS_ROOT}/scripts/ci/artifacts-publish/publish.sh" "${temp_dir}" "${tag}" "gs://rhacs-openshift-mirror-src/openapi-spec"
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
    roxctl helm output central-services --image-defaults=rhacs --output-dir "${central_services_chart_dir}/rhacs"
    roxctl helm output central-services --image-defaults=opensource --output-dir "${central_services_chart_dir}/opensource"
    roxctl helm output secured-cluster-services --image-defaults=rhacs --output-dir "${secured_cluster_services_chart_dir}/rhacs"
    roxctl helm output secured-cluster-services --image-defaults=opensource --output-dir "${secured_cluster_services_chart_dir}/opensource"
    "${SCRIPTS_ROOT}/scripts/ci/publish-helm-charts.sh" "${tag}" "${central_services_chart_dir}" "${secured_cluster_services_chart_dir}"
}

mark_collector_release() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: mark_collector_release <tag>"
    fi

    local tag="$1"
    local username="${GITHUB_USERNAME}"

    info "Check out collector source code"

    mkdir -p /tmp/collector
    git -C /tmp clone --depth=2 --no-single-branch https://github.com/stackrox/collector.git

    info "Create a branch for the PR"

    collector_version="$(cat COLLECTOR_VERSION)"
    pushd /tmp/collector || exit
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
    if grep -qF "${tag}" RELEASED_VERSIONS; then
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

gitbot() {
    git -c "user.name=${GITHUB_USERNAME}" \
        -c "user.email=${GITHUB_EMAIL}" \
        -c "url.https://${GITHUB_TOKEN}:x-oauth-basic@github.com/.insteadOf=https://github.com/" \
        "${@}"
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

get_branch_name() {
    # Returns the PR branch name (sometimes that's called PR source branch), e.g. 'johndoe/ROX-23456-fix-branch-name'.
    # For non-PRs, returns branch name where the commit happened, e.g. 'master'.
    if is_OPENSHIFT_CI; then
        # Prow variables doc: https://docs.prow.k8s.io/docs/jobs/#job-environment-variables
        if [[ -n "${PULL_HEAD_REF:-}" ]]; then
            # presubmit runs
            echo "${PULL_HEAD_REF}"
        elif [[ -n "${PULL_BASE_REF:-}" ]]; then
            # postsubmit and batch runs
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
            die "Expected PULL_HEAD_REF or PULL_BASE_REF or CLONEREFS_OPTIONS"
        fi
    elif is_GITHUB_ACTIONS; then
        # GHA doc: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/store-information-in-variables#default-environment-variables
        local ref="${GITHUB_HEAD_REF:-${GITHUB_REF_NAME:-}}"
        if [[ -z "${ref}" ]]; then
            die "Expected GITHUB_HEAD_REF or GITHUB_REF_NAME"
        fi
        echo "${ref}"
    else
        die "unsupported"
    fi
}

get_repo_full_name() {
    if is_GITHUB_ACTIONS; then
        [[ -n "${GITHUB_REPOSITORY:-}" ]] || die "expect: GITHUB_REPOSITORY"
        echo "${GITHUB_REPOSITORY}"
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
    elif is_GITHUB_ACTIONS; then
        echo "${GITHUB_SHA}"
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
        info "Warning: checking for a label in a non PR context: details: ${pr_details}, exitstatus: ${exitstatus}"
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
        return 0
    fi
    if [[ -e "${_PR_DETAILS_CACHE_FILE}" ]]; then
        _PR_DETAILS="$(cat "${_PR_DETAILS_CACHE_FILE}")"
        echo "${_PR_DETAILS}"
        return 0
    fi

    _not_a_PR() {
        echo "This does not appear to be a PR context" >&2
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
            echo "Expect a JOB_SPEC or CLONEREFS_OPTIONS" >&2
            exit 2
        fi
        [[ "${pull_request}" == "null" ]] && _not_a_PR
    elif is_GITHUB_ACTIONS; then
        pull_request="$(jq -r .pull_request.number "${GITHUB_EVENT_PATH}")" || _not_a_PR
        [[ "${pull_request}" == "null" ]] && _not_a_PR
        org="${GITHUB_REPOSITORY_OWNER}"
        repo="${GITHUB_REPOSITORY#*/}"
    else
        echo "Unsupported CI" >&2
        exit 2
    fi

    local headers url pr_details

    headers=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        headers+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi

    url="https://api.github.com/repos/${org}/${repo}/pulls/${pull_request}"

    if ! pr_details=$(curl --retry 5 --retry-connrefused -sS "${headers[@]}" "${url}"); then
        echo "Github API error: $pr_details, exit code: $?" >&2
        exit 2
    fi

    if [[ "$(jq .id <<<"$pr_details")" == "null" ]]; then
        # A valid PR response is expected at this point
        echo "Invalid response from GitHub: $pr_details" >&2
        exit 2
    fi
    _PR_DETAILS="$pr_details"
    echo "$pr_details" | tee "${_PR_DETAILS_CACHE_FILE}"
}

openshift_ci_mods() {
    info "BEGIN OpenShift CI mods"

    openshift_ci_debug

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
    BUILD_TAG="$(git describe --exact-match --tags HEAD)" || echo "Warning: Cannot get tag"
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

# openshift_ci_debug() - store useful state (env & git) to help debug CI.
# NOTE: only run before any creds are imported to the environment.
openshift_ci_debug() {
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

    if [[ "${JOB_NAME:-}" =~ -eks- ]]; then
        # Explicitly set AWS creds from the stackrox-stackrox-e2e-tests vault to
        # override any from other vaults e.g. automation-flavors.
        AWS_ACCESS_KEY_ID="$(cat /tmp/vault/stackrox-stackrox-e2e-tests/AWS_ACCESS_KEY_ID)"
        AWS_SECRET_ACCESS_KEY="$(cat /tmp/vault/stackrox-stackrox-e2e-tests/AWS_SECRET_ACCESS_KEY)"
        export AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY
        aws sts get-caller-identity | jq -r '.Arn'
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

remove_qa_test_results() {
    rm -rf qa-tests-backend/build/test-results
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

    mkdir -p "$dest"
    cp -a "$from" "$dest" || true # (best effort)
}

post_process_test_results() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: post_process_test_results <slack attachments file.json> <summary output.json>"
    fi

    if ! is_OPENSHIFT_CI; then
        return 0
    fi

    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "ERROR: ARTIFACT_DIR is not set which is expected in openshift CI"
        return 0
    fi

    local slack_attachments_file="$1"
    local summary_file="$2"
    local csv_output
    local extra_args=()
    local base_link
    local calculated_base_link
    local create_jiras
    local jira_project="ROX"
    local prow_job_link

    set +u
    {
        info "Post processing junit records to JIRA issues, BigQuery metrics and Slack attachments as appropriate"

        prow_job_link="$(make_prow_job_link)"

        if is_in_PR_context; then
            if pr_has_label "ci-test-junit-processing"; then
                create_jiras="true"
            else
                create_jiras="false"
            fi
            jira_project="RS"
        else
            if [[ "${PULL_BASE_REF:-unknown}" =~ ^release ]]; then
                create_jiras="false"
            elif [[ "${JOB_NAME:-unknown}" =~ interop ]]; then
                create_jiras="false"
            else
                create_jiras="true"
            fi
        fi

        if [[ "${create_jiras}" == "false" ]]; then
            extra_args=(--dry-run)
            info "Will use junit2jira to create CSV for BigQuery input"
        else
            info "Will create JIRA issues for junit failures found in ${ARTIFACT_DIR}"
        fi

        csv_output="$(mktemp --suffix=.csv)"
        # We need a link to repository. In case it's not part of job spec (e.g., periodic`s)
        # we will fallback to short commit
        base_link="$(echo "$JOB_SPEC" | jq ".refs.base_link | select( . != null )" -r)"
        calculated_base_link="https://github.com/stackrox/stackrox/commit/$(make --quiet --no-print-directory shortcommit)"
        curl --retry 5 --retry-connrefused -SsfL https://github.com/stackrox/junit2jira/releases/download/v0.0.24/junit2jira -o junit2jira && \
        chmod +x junit2jira && \
        ./junit2jira \
            -base-link "${base_link:-$calculated_base_link}" \
            -build-id "${BUILD_ID}" \
            -build-link "${prow_job_link}" \
            -build-tag "${STACKROX_BUILD_TAG}" \
            -csv-output "${csv_output}" \
            -jira-project "${jira_project}" \
            -job-name "${JOB_NAME}" \
            -junit-reports-dir "${ARTIFACT_DIR}" \
            -orchestrator "${ORCHESTRATOR_FLAVOR:-PROW}" \
            -threshold 10 \
            -html-output "$ARTIFACT_DIR/junit2jira-summary.html" \
            -slack-output "${slack_attachments_file}" \
            -summary-output "${summary_file}" \
            "${extra_args[@]}"

        save_test_metrics "${csv_output}"
    } || true
    set -u
}

gate_flaky_tests() {
    local exit_code="$1"

    if [[ "${exit_code}" == "0" ]]; then
        exit "${exit_code}"
    fi

    # Gating flaky tests is enabled only on PRs.
    if ! is_in_PR_context; then
        exit "${exit_code}"
    fi

    # Prepare flakechecker
    curl --retry 5 --retry-connrefused -SsfL https://github.com/stackrox/junit2jira/releases/download/v0.0.24/flakechecker -o /tmp/flakechecker || exit "${exit_code}"
    chmod +x /tmp/flakechecker
    setup_gcp || echo "setup_gcp called"

    # Run flakechecker for failed test
    local config_file="${SCRIPTS_ROOT}/scripts/ci/flakechecker/flake-config.yml"
    if /tmp/flakechecker -config-file "${config_file}" -job-name "${JOB_NAME}" -junit-reports-dir "${ARTIFACT_DIR}"; then
      # Flakechecker exits successfully IF AND ONLY IF it finds a NON-EMPTY set of test failures in the
      # JUnit report, AND ALL these test failures are found to be known flaky tests defined in flake-config.yml file.
      # And the recent failure ratio for found failed tests is below threshold defined in flake-config.yml.
      #
      # In this case, we change the overall exit code of the job to success, hoping that the failure was only
      # due to the tests flaky behavior (and not some other issue in the test job).
      exit 0
    else
      # Flakechecker fails in case it identified test failures which do not qualify for suppression, OR
      # in case there were no test failures at all in the JUnit report (a sign of problems elsewhere in the test job).
      #
      # In this case we keep the exit code as it was.
      exit "${exit_code}"
    fi
}

make_prow_job_link() {
    local prow_job_link="https://prow.ci.openshift.org/view/gs/origin-ci-test/"
    if is_in_PR_context; then
        prow_job_link+="pr-logs/pull/stackrox_stackrox/${PULL_NUMBER}/"
    else
        prow_job_link+="logs/"
    fi
    prow_job_link+="$JOB_NAME/$BUILD_ID"
    echo "${prow_job_link}"
}

# There are currently two openshift-ci steps where junit failures are summarized for slack.
JOB_SLACK_FAILURE_ATTACHMENTS="${SHARED_DIR:-/tmp}/job-slack-failure-attachments.json"
END_SLACK_FAILURE_ATTACHMENTS="/tmp/end-slack-failure-attachments.json"
JOB_JUNIT2JIRA_SUMMARY_FILE="${SHARED_DIR:-/tmp}/job-junit2jira-summary.json"
END_JUNIT2JIRA_SUMMARY_FILE="/tmp/end-junit2jira-summary.json"

send_slack_failure_summary() {
    if ! is_OPENSHIFT_CI || is_nightly_run; then
        return 0
    fi

    if [[ "${PULL_BASE_REF:-unknown}" =~ ^release ]]; then
        info "Skipping slack message for release branches"
        return 0
    fi

    if [[ "${JOB_TYPE:-unknown}" == "periodic" ]]; then
        info "Skipping slack message for periodics (scheduled prow jobs)"
        return 0
    fi

    if is_system_test_without_images; then
        # Avoid multiple slack messages from the e2e tests waiting for images.
        info "Skipping slack message for a system test failure when images were not found"
        return 0
    fi

    # Check env required for the job link and slack an error message when
    # undefined.
    _slack_check_env "BUILD_ID"
    _slack_check_env "JOB_NAME"
    local prow_job_link
    prow_job_link="$(make_prow_job_link)"

    local webhook_url="${TEST_FAILURES_NOTIFY_WEBHOOK}"

    _slack_check_env "PULL_BASE_SHA"
    local commit_sha="${PULL_BASE_SHA}"

    if is_in_PR_context; then
        if pr_has_label "ci-test-junit-processing"; then
            # Send to #acs-slack-ci-integration-testing when testing the
            # JUNIT -> Jira, BigQuery, Slack pipeline.
            webhook_url="${SLACK_CI_INTEGRATION_TESTING_WEBHOOK}"
            commit_sha="${PULL_PULL_SHA}"
        else
            info "Skipping slack message for PRs"
            return 0
        fi
    fi

    local org repo

    if [[ -n "${JOB_SPEC:-}" ]]; then
        org=$(jq -r <<<"$JOB_SPEC" '.refs.org')
        repo=$(jq -r <<<"$JOB_SPEC" '.refs.repo')
    elif [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
        org=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].org')
        repo=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].repo')
    else
        _send_slack_error "Expect a JOB_SPEC or CLONEREFS_OPTIONS"
        return 1
    fi

    if [[ "$org" == "null" ]] || [[ "$repo" == "null" ]]; then
        _send_slack_error "Could not determine org and/or repo"
        return 1
    fi

    local commit_details_url="https://api.github.com/repos/${org}/${repo}/commits/${commit_sha}"
    local exitstatus=0
    local commit_details
    commit_details=$(curl --retry 5 --retry-connrefused -sS "${commit_details_url}") || exitstatus="$?"
    if [[ "$exitstatus" != "0" ]]; then
        _send_slack_error "Cannot get commit details: ${commit_details}"
        return 1
    fi

    _slack_check_env "JOB_NAME_SAFE"
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
        _send_slack_error "Error parsing the commit details: ${commit_details}"
        return 1
    fi

    local mention_author=""
    _set_mention_author

    local slack_attachments=""
    _make_slack_failure_attachments

    local slack_mention=""
    _make_slack_mention

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
                "text": "*Commit:* <\($commit_url)|\($commit_msg)>\n*Repo:* \($repo)\n*Author:* \($author_name)\($slack_mention)\n*Log:* \($prow_job_link)"
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

    local payload
    if ! payload="$(jq --null-input \
                       --arg job_name "$job_name" \
                       --arg commit_url "$commit_url" \
                       --arg commit_msg "$commit_msg" \
                       --arg repo "$repo" \
                       --arg author_name "$author_name" \
                       --arg slack_mention "$slack_mention" \
                       --arg prow_job_link "$prow_job_link" \
                       --argjson slack_attachments "$slack_attachments" \
                       "$body")"; then
        _send_slack_error "Error formatting slack message: [${slack_attachments}/${payload}/$?]"
        return 1
    fi

    echo -e "About to post:\n$payload"

    local post_output
    if ! post_output="$(echo "$payload" | \
                        curl --location --silent --show-error --fail \
                             --data @- --header 'Content-Type: application/json' \
                             "$webhook_url")"; then
        _send_slack_error "Error posting to Slack: [${post_output}/$?]"
        return 1
    fi
}

_set_mention_author() {
    mention_author="false"

    # Mention the commit author if new JIRA issues were created
    if [[ -f "${JOB_JUNIT2JIRA_SUMMARY_FILE}" && \
        "$(jq -r '.newJIRAs' "${JOB_JUNIT2JIRA_SUMMARY_FILE}")" != "0" ]]; then
        mention_author="true"
    fi
    if [[ -f "${END_JUNIT2JIRA_SUMMARY_FILE}" && \
        "$(jq -r '.newJIRAs' "${END_JUNIT2JIRA_SUMMARY_FILE}")" != "0" ]]; then
        mention_author="true"
    fi
}

_make_slack_mention() {
    if [[ "${mention_author}" == "true" && "${author_login}" != "dependabot[bot]" ]]; then
        slack_mention="$("$SCRIPTS_ROOT"/scripts/ci/get-slack-user-id.sh "$author_login")"
        if [[ -n "$slack_mention" ]]; then
            slack_mention=", <@${slack_mention}>"
        else
            slack_mention=", _unable to resolve Slack user for GitHub login ${author_login}_"
        fi
    fi
}

_make_slack_failure_attachments() {
    info "Converting junit failures to slack attachments"

    slack_attachments=""
    if [[ ! -f "${JOB_SLACK_FAILURE_ATTACHMENTS}" ]]; then
        if [[ "${CREATE_CLUSTER_OUTCOME:-}" == "${OUTCOME_PASSED}" ]]; then
            slack_attachments+="$(_make_slack_failure_plain_text_block \
                "Could not parse junit in main test step. Check build logs for more information.")"
        fi
    else
        slack_attachments+="$(cat "${JOB_SLACK_FAILURE_ATTACHMENTS}")"
    fi
    if [[ ! -f "${END_SLACK_FAILURE_ATTACHMENTS}" ]]; then
        slack_attachments+="$(_make_slack_failure_plain_text_block \
            "Could not parse junit in final test step. Check build logs for more information.")"
    else
        slack_attachments+="$(cat "${END_SLACK_FAILURE_ATTACHMENTS}")"
    fi

    slack_attachments="$(echo "${slack_attachments}" | jq '.[]' | jq -s '.')"

    if [[ "$(echo "${slack_attachments}" | jq 'length')" == "0" ]]; then
        msg="No junit records were found for this failure. Check build logs \
and artifacts for more information. Consider adding an \
issue to improve CI to detect this failure pattern. (Add a CI_Fail_Better label)."
        slack_attachments="$(_make_slack_failure_plain_text_block "${msg}")"

        # Mention the commit author when the job failed with no JUNIT records
        mention_author="true"
    fi
}

_make_slack_failure_block() {
    # shellcheck disable=SC2016
    local body='
[
  {
    "color": "#bb2124",
    "blocks": [
      {
        "type": "section",
        "text": {
          "type": "\($section_type)",
          "text": "\($content)"
        }
      }
    ]
  }
]
'
    jq --null-input \
       --arg section_type "$1" \
       --arg content "$2" \
       "$body"
}

_make_slack_failure_plain_text_block() {
    _make_slack_failure_block "plain_text" "$1"
}

_make_slack_failure_markdown_block() {
    _make_slack_failure_block "mrkdwn" "$1"
}

_send_slack_error() {
    echo "ERROR: $1"
    curl -XPOST -d @- -H 'Content-Type: application/json' "${webhook_url}" << __EOM__
{ "text": "*An error occurred dealing with a job failure:*\n\t- Job: ${prow_job_link}.\n\t- $1." }
__EOM__
}

_slack_check_env() {
    (
        set +u
        if [[ -z "$(eval echo "\$$1")" ]]; then
            _send_slack_error "An expected environment variable is unset/empty: $1"
            return 1
        fi
    )
}

slack_workflow_failure() {
    # shellcheck disable=SC2153
    local github_context="${GITHUB_CONTEXT}"
    local webhook_url="${TEST_FAILURES_NOTIFY_WEBHOOK}"

    if is_in_PR_context; then
        if pr_has_label "ci-test-github-action-slack-messages"; then
            # Send to #acs-slack-ci-integration-testing when testing.
            webhook_url="${SLACK_CI_INTEGRATION_TESTING_WEBHOOK}"
        else
            info "Skipping slack message for PRs"
            return 0
        fi
    fi

    local workflow_name commit_msg commit_url repo author_name author_login repo_url run_id
    workflow_name=$(jq -r <<<"${github_context}" '.workflow')
    event_name=$(jq -r <<<"${github_context}" '.event_name')
    if [[ "${event_name}" == "push" ]]; then
        commit_msg=$(jq -r <<<"${github_context}" '.event.head_commit.message')
        commit_msg="${commit_msg%%$'\n'*}" # use first line of commit msg
        commit_url=$(jq -r <<<"${github_context}" '.event.head_commit.url')
        author_name=$(jq -r <<<"${github_context}" '.event.head_commit.author.name')
        author_login=$(jq -r <<<"${github_context}" '.event.head_commit.author.username')
        repo_url=$(jq -r <<<"${github_context}" '.event.repository.url')
    else
        commit_msg="This is a test slack message"
        commit_url=$(jq -r <<<"${github_context}" '.event.pull_request.diff_url')
        author_name=$(jq -r <<<"${github_context}" '.actor')
        author_login=$(jq -r <<<"${github_context}" '.actor')
        repo_url=$(jq -r <<<"${github_context}" '.event.pull_request.base.repo.html_url')
    fi
    repo=$(jq -r <<<"${github_context}" '.repository')
    run_id=$(jq -r <<<"${github_context}" '.run_id')

    # If global "mention_author" is set use that value.
    local mention_author="${mention_author:-true}"
    local slack_mention=""
    _make_slack_mention

    local attachments=""
    local job_name job_url
    IFS=$'\n'
    for job in $(gh run view --jq '.jobs[] | select(.conclusion == "failure")' --json 'jobs' -R "${repo}" "${run_id}" | jq -sc '.[]')
    do
        job_name=$(jq -r <<<"${job}" '.name')
        job_url=$(jq -r <<<"${job}" '.url')
        attachments+="$(_make_slack_failure_markdown_block "Job: <${job_url}|${job_name}>")"
    done
    attachments="$(echo "${attachments}" | jq '.[]' | jq -s '.')"

    # shellcheck disable=SC2016
    local body='
{
    "text": "\($workflow_name) failed.
Commit: \($commit_msg).
Author: \($author_name)\($slack_mention).",
    "blocks": [
        {
            "type": "header",
            "text": {
                "type": "plain_text",
                "text": "\($workflow_name) failed."
            }
        },
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": "
*Commit:* <\($commit_url)|\($commit_msg)>.
*Repo:* \($repo).
*Author:* \($author_name)\($slack_mention).
*Workflow:* \($repo_url)/actions/runs/\($run_id)"
            }
        },
        {
            "type": "divider"
        }
    ],
    "attachments": $attachments
}
'

    local payload
    payload="$(jq --null-input \
        --arg workflow_name "${workflow_name}" \
        --arg commit_url "$commit_url" \
        --arg commit_msg "$commit_msg" \
        --arg repo "$repo" \
        --arg author_name "$author_name" \
        --arg slack_mention "$slack_mention" \
        --arg repo_url "$repo_url" \
        --arg run_id "$run_id" \
        --argjson attachments "$attachments" \
       "$body")"

    echo -e "About to post:\n$payload"

    echo "$payload" | curl --location --silent --show-error --fail \
         --data @- --header 'Content-Type: application/json' \
         "${webhook_url}"
}

# junit_wrap() - runs a command and creates a JUNIT record if the command
# succeeds or fails. Some output of the command is included in the failure
# JUNIT.
#
# WARNING: If this is used to wrap a bash function and not a separate binary
# or script file there are two side effects that may not be expected:
#
# 1. errexit is not propagated to the function context and the function will
#    continue on error. This is contrary to the typical approach from `set -e`
#    used throughout this repo.
# 2. exports are not propagated back to the calling context because the command
#    runs in a subshell.

junit_wrap() {
    if [[ "$#" -lt 4 ]]; then
        die "missing args. usage: junit_wrap <class> <description> <failure_message> <command> [ args ]"
    fi

    local class="$1"; shift
    local description="$1"; shift
    local failure_message="$1"; shift
    local command_output_file
    command_output_file="$(mktemp)"

    if "$@" 2>&1 | tee "${command_output_file}"; then
        save_junit_success "${class}" "${description}"
        rm -f "${command_output_file}"
    else
        local ret_code="$?"
        local failure_body=""
        if [[ -n "$failure_message" ]]; then
            failure_body="${failure_message}
"
        fi
        failure_body="${failure_body}Command output: $(tail --bytes=512 "${command_output_file}")"

        save_junit_failure "${class}" "${description}" "${failure_body}"
        rm -f "${command_output_file}"

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

_JUNIT_RESULT_SUCCESS="SUCCESS"
_JUNIT_RESULT_FAILURE="FAILURE"
_JUNIT_RESULT_SKIPPED="SKIPPED"

save_junit_success() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: save_junit_success <class> <description>"
    fi

    _save_junit_record "${_JUNIT_RESULT_SUCCESS}" "$@"
}

save_junit_failure() {
    if [[ "$#" -ne 3 ]]; then
        die "missing args. usage: save_junit_failure <class> <description> <details>"
    fi

    _save_junit_record "${_JUNIT_RESULT_FAILURE}" "$@"
}

save_junit_skipped() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: save_junit_skipped <class> <description>"
    fi

    _save_junit_record "${_JUNIT_RESULT_SKIPPED}" "$@"
}

remove_junit_record() {
    local class="$1"
    local junit_dir
    junit_dir="$(get_junit_misc_dir)"
    local junit_file="${junit_dir}/junit-${class}.xml"
    rm -f "${junit_file}"
}

_save_junit_record() {
    local disposition="$1"
    local class="$2"
    local description="$3"
    local details="${4:-}"

    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "Warning: save_junit_success() requires the \$ARTIFACT_DIR variable to be set"
        return
    fi

    local junit_dir
    junit_dir="$(get_junit_misc_dir)"
    mkdir -p "${junit_dir}/db"

    # base64 encode failure details to condense multilines
    if [[ $details != "SUCCESS" ]]; then
        details="$(base64 -w0 <<< "$details")"
    fi

    # record this instance
    local record_length=3
    local record="${junit_dir}/db/${class}.txt"
    {
        echo "${description}"
        echo "${disposition}"
        echo "${details}"
     } >> "${record}"

    local tests
    tests=$(( "$(wc -l < "${record}")" / "${record_length}" ))

    local failures=0
    local skipped=0
    local lines
    readarray -t lines < "${record}"
    while (( ${#lines[@]} ))
    do
        local result="${lines[1]}"
        if [[ "${result}" == "${_JUNIT_RESULT_FAILURE}" ]]; then
            failures=$(( failures+1 ))
        fi
        if [[ "${result}" == "${_JUNIT_RESULT_SKIPPED}" ]]; then
            skipped=$(( skipped+1 ))
        fi
        lines=( "${lines[@]:${record_length}}" )
    done

    local junit_file="${junit_dir}/junit-${class}.xml"

    cat << _EO_SUITE_HEADER_ > "${junit_file}"
<testsuite name="${class}" tests="${tests}" skipped="${skipped}" failures="${failures}" errors="0">
_EO_SUITE_HEADER_

    readarray -t lines < "${record}"
    while (( ${#lines[@]} ))
    do
        local description="${lines[0]}"
        local result="${lines[1]}"
        local details="${lines[2]}"

        # XML escape description
        description="${description//&/&amp;}"
        description="${description//\"/&quot;}"
        description="${description//\'/&#39;}"
        description="${description//</&lt;}"
        description="${description//>/&gt;}"

        cat << _EO_CASE_HEADER_ >> "${junit_file}"
        <testcase name="${description}" classname="${class}">
_EO_CASE_HEADER_

        if [[ "$result" == "${_JUNIT_RESULT_FAILURE}" ]]; then
            details="$(base64 --decode <<< "$details")"
        cat << _EO_FAILURE_ >> "${junit_file}"
            <failure><![CDATA[${details}]]></failure>
_EO_FAILURE_
        fi
        if [[ "$result" == "${_JUNIT_RESULT_SKIPPED}" ]]; then
        cat << _EO_SKIPPED_ >> "${junit_file}"
            <skipped/>
_EO_SKIPPED_
        fi

        echo "        </testcase>" >> "${junit_file}"

        lines=( "${lines[@]:3}" )
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

is_system_test_without_images() {
    case "${JOB_NAME:-missing}" in
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

    artifact_file="$ARTIFACT_DIR/cluster-version.html"

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
