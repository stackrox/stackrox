#!/usr/bin/env bats

# Runs Scanner V4 installation tests using the Bats testing framework.

init() {
    ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")"/../.. && pwd)"
    export ROOT

    if [[ "${CI:-}" != "true" ]]; then
        # Some friendly environment checks
        if [[ -z "${BATS_CORE_ROOT:-}" ]]; then
            echo "WARNING: You better set \$BATS_CORE_ROOT before executing this test suite." >&3
            exit 1
        fi
    fi

    # shellcheck source=../../scripts/ci/lib.sh
    source "$ROOT/scripts/ci/lib.sh"
    # shellcheck source=../../scripts/ci/gcp.sh
    source "$ROOT/scripts/ci/gcp.sh"
    # shellcheck source=../../scripts/ci/sensor-wait.sh
    source "$ROOT/scripts/ci/sensor-wait.sh"
    # shellcheck source=../../tests/e2e/lib.sh
    source "$ROOT/tests/e2e/lib.sh"
    # shellcheck source=../../tests/scripts/setup-certs.sh
    source "$ROOT/tests/scripts/setup-certs.sh"
    load "$ROOT/scripts/test_helpers.bats"

    require_environment "ORCHESTRATOR_FLAVOR"
}

export TEST_SUITE_ABORTED="false"

setup_file() {
    init

    # Use
    #   export CHART_BASE="/rhacs"
    #   export DEFAULT_IMAGE_REGISTRY="quay.io/rhacs-eng"
    # for the RHACS flavor.
    export CHART_BASE=""
    export DEFAULT_IMAGE_REGISTRY="quay.io/stackrox-io"

    export CURRENT_MAIN_IMAGE_TAG=${CURRENT_MAIN_IMAGE_TAG:-} # Setting a tag can be useful for local testing.
    export EARLIER_CHART_VERSION="4.3.0"
    export EARLIER_MAIN_IMAGE_TAG=$EARLIER_CHART_VERSION
    export USE_LOCAL_ROXCTL=true
    export ROX_PRODUCT_BRANDING=RHACS_BRANDING
    export CI=${CI:-false}
    OS="$(uname | tr '[:upper:]' '[:lower:]')"
    export OS
    export ORCH_CMD="${ROOT}/scripts/retry-kubectl.sh"
    export SENSOR_HELM_MANAGED=true
    export CENTRAL_CHART_DIR="${ROOT}/deploy/${ORCHESTRATOR_FLAVOR}/central-deploy/chart"
    export SENSOR_CHART_DIR="${ROOT}/deploy/${ORCHESTRATOR_FLAVOR}/sensor-deploy/chart"
    if [[ "${ORCHESTRATOR_FLAVOR:-}" == "openshift" ]]; then
      export ROX_OPENSHIFT_VERSION=4
    fi

    # Prepare earlier Helm chart version.
    if [[ -z "${CHART_REPOSITORY:-}" ]]; then
        CHART_REPOSITORY=$(mktemp -d "helm-charts.XXXXXX" -p /tmp)
    fi
    if [[ ! -e "${CHART_REPOSITORY}/.git" ]]; then
        git clone --depth 1 -b main https://github.com/stackrox/helm-charts "${CHART_REPOSITORY}"
    fi
    export CHART_REPOSITORY

    # Download and use earlier version of roxctl without Scanner V4 support
    # We will just hard-code a pre-4.4 version here.
    if [[ -z "${EARLIER_ROXCTL_PATH:-}" ]]; then
        EARLIER_ROXCTL_PATH=$(mktemp -d "early_roxctl.XXXXXX" -p /tmp)
    fi
    echo "EARLIER_ROXCTL_PATH=$EARLIER_ROXCTL_PATH"
    export EARLIER_ROXCTL_PATH
    if [[ ! -e "${EARLIER_ROXCTL_PATH}/roxctl" ]]; then
        curl --retry 5 --retry-connrefused -sL "https://mirror.openshift.com/pub/rhacs/assets/${EARLIER_MAIN_IMAGE_TAG}/bin/${OS}/roxctl" --output "${EARLIER_ROXCTL_PATH}/roxctl"
        chmod +x "${EARLIER_ROXCTL_PATH}/roxctl"
    fi

    export CUSTOM_CENTRAL_NAMESPACE=${CUSTOM_CENTRAL_NAMESPACE:-stackrox-central}
    export CUSTOM_SENSOR_NAMESPACE=${CUSTOM_SENSOR_NAMESPACE:-stackrox-sensor}

    export MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(make --quiet --no-print-directory -C "${ROOT}" tag)}"
    info "Using MAIN_IMAGE_TAG=$MAIN_IMAGE_TAG"

    # Taken from operator/Makefile
    export OPERATOR_VERSION_TAG=${OPERATOR_VERSION_TAG:-}
    if [[ -z "${OPERATOR_VERSION_TAG}" && -n "${MAIN_IMAGE_TAG:-}" ]]; then
        OPERATOR_VERSION_TAG=$(echo "${MAIN_IMAGE_TAG}" | sed -E 's@^(([[:digit:]]+\.)+)x(-)?@\10\3@g')
    fi

    # Configure a timeout for a single test. After 30m of runtime a test will be marked as failed
    # (and we will hopefully receive helpful logs for analysing the situation).
    # Without a timeout it might happen that the pod running the tests is simply killed and we won't
    # have any logs for investigation the situation.
    export BATS_TEST_TIMEOUT=200 # Seconds
}

test_case_no=0

setup() {
    [[ "${TEST_SUITE_ABORTED}" == "true" ]] && return 1
    init
    set -euo pipefail

    export_test_environment
    if [[ "$CI" = "true" ]]; then
        setup_gcp
        setup_deployment_env false false
    fi

    if [[ "${SKIP_INITIAL_TEARDOWN:-}" != "true" ]] && (( test_case_no == 0 )); then
        # executing initial teardown to begin test execution in a well-defined state
        remove_existing_stackrox_resources "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}" "stackrox"
    fi
    if [[ ${TEARDOWN_ONLY:-} == "true" ]]; then
        echo "Only tearing down resources, exiting now..."
        exit 0
    fi

    test_case_no=$(( test_case_no + 1))

    export ROX_SCANNER_V4=true

    # By default we will use CRS-based cluster registration in this test suite, but there are some
    # specific tests which require CRS to be switched off (upgrade tests involving an old Helm chart, e.g.).
    export ROX_DEPLOY_SENSOR_WITH_CRS=true
}

describe_pods_in_namespace() {
    local namespace="$1"
    info "==============================="
    info "Pods in namespace ${namespace}:"
    info "==============================="
    "${ORCH_CMD}" </dev/null -n "${namespace}" get pods || true
    echo
    "${ORCH_CMD}" </dev/null -n "${namespace}" get pods -o name | while read -r pod_name; do
      echo "** DESCRIBING POD: ${namespace}/${pod_name}:"
      "${ORCH_CMD}" </dev/null -n "${namespace}" describe "${pod_name}" || true
      echo
      echo "** LOGS FOR POD: ${namespace}/${pod_name}:"
      "${ORCH_CMD}" </dev/null -n "${namespace}" logs "${pod_name}" || true
      echo

    done
}

describe_deployments_in_namespace() {
    local namespace="$1"
    info "====================================="
    info "Deployments in namespace ${namespace}:"
    info "====================================="
    "${ORCH_CMD}" </dev/null -n "${namespace}" get deployments || true
    echo
    "${ORCH_CMD}" </dev/null -n "${namespace}" get deployments -o name | while read -r name; do
      echo "** DESCRIBING DEPLOYMENT: ${namespace}/${name}:"
      "${ORCH_CMD}" </dev/null -n "${namespace}" describe "${name}" || true
    done
}

teardown() {
    if [[ "${TEST_SUITE_ABORTED}" == "true" ]]; then
        echo "Skipping teardown due to previous failure." >&3
        return
    fi

    if [[ "${BATS_TEST_COMPLETED:-}" != "1" ]]; then
        # Previous test failed.
        if [[ "${ABORT_ON_FAILURE:-}" == "true" ]]; then
            TEST_SUITE_ABORTED="true"
            echo "Aborting due to test failure." >&3
            return 1
        fi
    fi

    local central_namespace=""
    local sensor_namespace=""

    if "${ORCH_CMD}" </dev/null get ns "stackrox" >/dev/null 2>&1; then
        central_namespace="stackrox"
        sensor_namespace="stackrox"
    fi
    if "${ORCH_CMD}" </dev/null get ns "${CUSTOM_CENTRAL_NAMESPACE}" >/dev/null 2>&1; then
        central_namespace="${CUSTOM_CENTRAL_NAMESPACE}"
    fi
    if "${ORCH_CMD}" </dev/null get ns "${CUSTOM_SENSOR_NAMESPACE}" >/dev/null 2>&1; then
        sensor_namespace="${CUSTOM_SENSOR_NAMESPACE}"
    fi

    "$ROOT/scripts/ci/collect-service-logs.sh" "${central_namespace}" \
      "${SCANNER_V4_LOG_DIR}/${BATS_TEST_NUMBER}-${BATS_TEST_NAME}"

    if [[ "${central_namespace}" != "${sensor_namespace}" && -n "${sensor_namespace}" ]]; then
      "$ROOT/scripts/ci/collect-service-logs.sh" "${sensor_namespace}" \
        "${SCANNER_V4_LOG_DIR}/${BATS_TEST_NUMBER}-${BATS_TEST_NAME}"
    fi

    if [[ -z "${BATS_TEST_COMPLETED:-}" && -z "${BATS_TEST_SKIPPED}" && -n "${central_namespace}" ]]; then
        # Test did not "complete" and was not skipped. Collect some analysis data.
        describe_pods_in_namespace "${central_namespace}"
        describe_deployments_in_namespace "${central_namespace}"

        if [[ "${central_namespace}" != "${sensor_namespace}" && -n "${sensor_namespace}" ]]; then
            describe_pods_in_namespace "${sensor_namespace}"
            describe_deployments_in_namespace "${sensor_namespace}"
        fi
    fi

    run remove_existing_stackrox_resources "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}" "stackrox"
}

teardown_file() {
    remove_earlier_roxctl_binary
}

@test "Try timeout" {
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    export ROX_DEPLOY_SENSOR_WITH_CRS=false

    echo stdout msg
    echo stderr msg >&2
    sleep 300
}

verify_no_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    verify_no_scannerV4_indexer_deployed "$namespace"
    verify_no_scannerV4_matcher_deployed "$namespace"
}

verify_no_scannerV4_indexer_deployed() {
    local namespace=${1:-stackrox}
    run "${ORCH_CMD}" </dev/null -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "scanner-v4-indexer"
}

verify_no_scannerV4_matcher_deployed() {
    local namespace=${1:-stackrox}
    run "${ORCH_CMD}" </dev/null -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "scanner-v4-matcher"
}

# TODO: For now, Scanner v2 is expected to run in parallel.
# This must be removed when Scanner v2 will be phased out.
verify_scannerV2_deployed() {
    local namespace=${1:-stackrox}
    info "Waiting for Scanner V2 deployment to appear in namespace ${namespace}..."
    wait_for_object_to_appear "$namespace" deploy/scanner-db 600
    wait_for_object_to_appear "$namespace" deploy/scanner 300
    info "** Scanner V2 is deployed in namespace ${namespace}"
}

verify_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    verify_scannerV4_indexer_deployed "$namespace"
    verify_scannerV4_matcher_deployed "$namespace"
}

verify_scannerV4_indexer_deployed() {
    local namespace=${1:-stackrox}
    info "Waiting for Scanner V4 Indexer to appear in namespace ${namespace}..."
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-db 600
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-indexer 300
    wait_for_ready_pods "${namespace}" "scanner-v4-db" 600
    wait_for_ready_pods "${namespace}" "scanner-v4-indexer" 600
    info "** Scanner V4 Indexer is deployed in namespace ${namespace}"
}

verify_scannerV4_matcher_deployed() {
    local namespace=${1:-stackrox}
    info "Waiting for Scanner V4 Matcher to appear in namespace ${namespace}..."
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-db 600
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-matcher 300
    wait_for_ready_pods "${namespace}" "scanner-v4-db" 600
    wait_for_ready_pods "${namespace}" "scanner-v4-matcher" 600
    info "** Scanner V4 Matcher is deployed in namespace ${namespace}"
}

verify_deployment_scannerV4_env_var_set() {
    local namespace=${1:-stackrox}
    local deployment=${2:-central}
    local deployment_env_vars
    local scanner_v4_value

    deployment_env_vars="$("${ORCH_CMD}" </dev/null -n "${namespace}" get deploy/"${deployment}" -o jsonpath="{.spec.template.spec.containers[?(@.name=='${deployment}')].env}")"
    scanner_v4_value="$(echo "${deployment_env_vars}" | jq -r '.[] | select(.name == "ROX_SCANNER_V4").value')"

    if [[ "${scanner_v4_value}" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# We are using our own deploy function, because we want to have the flexibility to patch down resources
# after deployment. Without this we are only able to special-case local deployments and CI deployments,
# but not, for example, manual testing on Infra.
#
# shellcheck disable=SC2120
_deploy_stackrox() {
    local tls_client_certs=${1:-}
    local central_namespace=${2:-stackrox}
    local sensor_namespace=${3:-stackrox}

    _deploy_central "${central_namespace}"
    # shellcheck disable=SC2031
    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR}" != "true" && "${HELM_REUSE_VALUES:-}" != "true" ]]; then
      # In case we are reusing existing Helm values we should not export new
      # central credentials into the environment.
      DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}" export_central_basic_auth_creds
    fi
    wait_for_api "${central_namespace}"
    setup_client_TLS_certs "${tls_client_certs}"
    record_build_info "${central_namespace}"

    _deploy_sensor "${sensor_namespace}" "${central_namespace}"
    echo "Sensor deployed. Waiting for sensor to be up"
    sensor_wait "${sensor_namespace}"

    # Bounce collectors to avoid restarts on initial module pull
    "${ORCH_CMD}" </dev/null -n "${sensor_namespace}" delete pod -l app=collector --grace-period=0

    sensor_wait "${sensor_namespace}"

    wait_for_collectors_to_be_operational "${sensor_namespace}"

    touch "${STATE_DEPLOYED}"
}

# shellcheck disable=SC2120
_deploy_central() {
    local central_namespace=${1:-stackrox}
    deploy_central "${central_namespace}"
    patch_down_central "${central_namespace}"
}

patch_down_central() {
    local central_namespace="$1"
    # shellcheck disable=SC2031
    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR:-}" != "true" ]]; then
        patch_down_central_directly "${central_namespace}"
    fi
}

patch_down_central_directly() {
   local central_namespace="$1"

    if "${ORCH_CMD}" </dev/null -n "${central_namespace}" get hpa scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${central_namespace}" patch "hpa/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi

    if "${ORCH_CMD}" </dev/null -n "${central_namespace}" get hpa scanner-v4-matcher >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${central_namespace}" patch "hpa/scanner-v4-matcher" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi
    if "${ORCH_CMD}" </dev/null -n "${central_namespace}" get deploy/scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${central_namespace}" patch "deploy/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  replicas: 1
EOF
        )
    fi
    if "${ORCH_CMD}" </dev/null -n "${central_namespace}" get deploy/scanner-v4-matcher >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${central_namespace}" patch "deploy/scanner-v4-matcher" --patch-file <(cat <<EOF
spec:
  replicas: 1
EOF
        )
    fi
}

# shellcheck disable=SC2120
_deploy_sensor() {
    local sensor_namespace=${1:-stackrox}
    local central_namespace=${2:-stackrox}
    deploy_sensor "${sensor_namespace}" "${central_namespace}"
    patch_down_sensor "${sensor_namespace}"
}

patch_down_sensor() {
    local sensor_namespace="$1"
    # shellcheck disable=SC2031
    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR:-}" != "true" ]]; then
        patch_down_sensor_directly "${sensor_namespace}"
    fi
}

patch_down_sensor_directly() {
   local sensor_namespace="$1"

    if "${ORCH_CMD}" </dev/null -n "${sensor_namespace}" get hpa scanner >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${sensor_namespace}" patch "hpa/scanner" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi
    if "${ORCH_CMD}" </dev/null -n "${sensor_namespace}" get hpa scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${sensor_namespace}" patch "hpa/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi
    if "${ORCH_CMD}" </dev/null -n "${central_namespace}" get deploy/scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" </dev/null -n "${sensor_namespace}" patch "deploy/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  replicas: 1
EOF
        )
    fi
}

# This function tries to fix shortcomings of `kubectl wait`. Instead of (wrongly) caring about pods terminating
# in the beginning because the overall situation has not stabilized yet, this function only waits until *some*
# pod in the specified deployment becomes ready.
#
# Hopefully makes CI less flaky.
wait_for_ready_pods() {
    local namespace="${1}"
    local deployment="${2}"
    local timeout_seconds="${3:-300}" # 5 minutes

    local start_time
    start_time="$(date '+%s')"
    local start_time
    local deployment_json
    local num_replicas
    local num_ready_replicas
    local now

    echo "Waiting for pod within deployment ${namespace}/${deployment} to become ready in ${timeout_seconds} seconds"

    while true; do
      deployment_json="$("${ORCH_CMD}" </dev/null -n "${namespace}" get "deployment/${deployment}" -o json)"
      num_replicas="$(jq '.status.replicas // 0' <<<"${deployment_json}")"
      num_ready_replicas="$(jq '.status.readyReplicas // 0' <<<"${deployment_json}")"
      echo "${deployment} replicas: ${num_replicas}"
      echo "${deployment} readyReplicas: ${num_ready_replicas}"
      if (( num_ready_replicas >  0 )); then
        break
      fi
      now=$(date '+%s')
      if (( now - start_time > timeout_seconds)); then
        echo >&2 "Timed out after ${timeout_seconds} seconds while waiting for ready pods within deployment ${namespace}/${deployment}"
        "${ORCH_CMD}" </dev/null -n "${namespace}" get pod -o wide
        "${ORCH_CMD}" </dev/null -n "${namespace}" get deploy -o wide
        exit 1
      fi
      sleep 2
    done

    echo "Pod(s) within deployment ${namespace}/${deployment} ready."
}

remove_earlier_roxctl_binary() {
    if [[ -d "${EARLIER_ROXCTL_PATH}" ]]; then
      rm -f "${EARLIER_ROXCTL_PATH}/roxctl"
      rmdir "${EARLIER_ROXCTL_PATH}"
      echo "Removed earlier roxctl binary"
    fi
}

# Waits until ValidationWebhook is completely functional.
wait_until_central_validation_webhook_is_ready() {
    local central_namespace=$1

    info "Waiting for AdmissionWebhook to be functional by trying to patch Central in namespace ${central_namespace}..."
    sleep 1m
    patch_test_file=$(mktemp)
    cat >"${patch_test_file}" <<EOT
spec:
  customize:
    envVars:
      - name: IGNORE_THIS_PLEASE
        value: it-is-just-about-checking-validationhook-readiness
EOT
    retry 7 true "${ORCH_CMD}" </dev/null -n "${central_namespace}" patch Central stackrox-central-services --type=merge --patch-file="${patch_test_file}"
}
