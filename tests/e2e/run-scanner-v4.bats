#!/usr/bin/env bats

# Runs Scanner V4 tests using the Bats testing framework.

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
    # shellcheck source=../../tests/e2e/lib.sh
    source "$ROOT/tests/e2e/lib.sh"
    # shellcheck source=../../tests/scripts/setup-certs.sh
    source "$ROOT/tests/scripts/setup-certs.sh"
    load "$ROOT/scripts/test_helpers.bats"

    require_environment "ORCHESTRATOR_FLAVOR"
}

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
    export ORCH_CMD=kubectl
    export SENSOR_HELM_MANAGED=true

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
        curl --retry 5 -sL "https://mirror.openshift.com/pub/rhacs/assets/${EARLIER_MAIN_IMAGE_TAG}/bin/${OS}/roxctl" --output "${EARLIER_ROXCTL_PATH}/roxctl"
        chmod +x "${EARLIER_ROXCTL_PATH}/roxctl"
    fi

    export CUSTOM_CENTRAL_NAMESPACE=${CUSTOM_CENTRAL_NAMESPACE:-stackrox-central}
    export CUSTOM_SENSOR_NAMESPACE=${CUSTOM_SENSOR_NAMESPACE:-stackrox-sensor}

    export MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(make --quiet --no-print-directory -C "${ROOT}" tag)}"
    info "Using MAIN_IMAGE_TAG=$MAIN_IMAGE_TAG"

    # Taken from operator/Makefile
    export OPERATOR_VERSION_TAG=${OPERATOR_VERSION_TAG:-}
    if [[ -z "${OPERATOR_VERSION_TAG}" && -n "${MAIN_IMAGE_TAG:-}" ]]; then
        OPERATOR_VERSION_TAG=$(echo "${MAIN_IMAGE_TAG}" | sed -E 's@^(([[:digit:]]+\.)+)x(-)?@\10\3@g' | sed -E 's@^3.0.([[:digit:]]+\.[[:digit:]]+)(-)?@3.\1\2@g')
    fi

    setup_default_TLS_certs
}

test_case_no=0

setup() {
    init
    set -euo pipefail

    export_test_environment
    if [[ "$CI" = "true" ]]; then
        setup_gcp
        setup_deployment_env false false
    fi

    if (( test_case_no == 0 )); then
        # executing initial teardown to begin test execution in a well-defined state
        teardown
    fi
    if [[ ${TEARDOWN_ONLY:-} == "true" ]]; then
        echo "Only tearing down resources, exiting now..."
        exit 0
    fi

    test_case_no=$(( test_case_no + 1))

    export ROX_SCANNER_V4=true
}

describe_pods_in_namespace() {
    local namespace="$1"
    info "==============================="
    info "Pods in namespace ${namespace}:"
    info "==============================="
    "${ORCH_CMD}" -n "${namespace}" get pods
    echo
    "${ORCH_CMD}" -n "${namespace}" get pods -o name | while read -r pod_name; do
      echo "** DESCRIBING POD: ${namespace}/${pod_name}:"
      "${ORCH_CMD}" -n "${namespace}" describe "${pod_name}"
      echo
      echo "** LOGS FOR POD: ${namespace}/${pod_name}:"
      "${ORCH_CMD}" -n "${namespace}" logs "${pod_name}"
      echo

    done
}

describe_deployments_in_namespace() {
    local namespace="$1"
    info "====================================="
    info "Deployments in namespace ${namespace}:"
    info "====================================="
    "${ORCH_CMD}" -n "${namespace}" get deployments
    echo
    "${ORCH_CMD}" -n "${namespace}" get deployments -o name | while read -r name; do
      echo "** DESCRIBING DEPLOYMENT: ${namespace}/${name}:"
      "${ORCH_CMD}" -n "${namespace}" describe "${name}"
    done
}


teardown() {
    local central_namespace=""
    local sensor_namespace=""
    local namespaces=( )

    if "${ORCH_CMD}" get ns "stackrox" >/dev/null 2>&1; then
        central_namespace="stackrox"
        sensor_namespace="stackrox"
        namespaces=( "stackrox" "${namespaces[@]}" )
    fi
    if "${ORCH_CMD}" get ns "${CUSTOM_CENTRAL_NAMESPACE}" >/dev/null 2>&1; then
        central_namespace="${CUSTOM_CENTRAL_NAMESPACE}"
        namespaces=( "${central_namespace}" "${namespaces[@]}" )
    fi
    if "${ORCH_CMD}" get ns "${CUSTOM_SENSOR_NAMESPACE}" >/dev/null 2>&1; then
        sensor_namespace="${CUSTOM_SENSOR_NAMESPACE}"
        if [[ "${central_namespace}" != "${sensor_namespace}" ]]; then
            namespaces=( "${sensor_namespace}" "${namespaces[@]}" )
        fi
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

    run remove_existing_stackrox_resources "${namespaces[@]}"
}

teardown_file() {
    remove_earlier_roxctl_binary
}

@test "Upgrade from old Helm chart to HEAD Helm chart with Scanner v4 enabled" {
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    local main_image_tag="${MAIN_IMAGE_TAG}"

    # Deploy earlier version without Scanner V4.
    local _CENTRAL_CHART_DIR_OVERRIDE="${CHART_REPOSITORY}${CHART_BASE}/${EARLIER_CHART_VERSION}/central-services"
    info "Deplying StackRox services using chart ${_CENTRAL_CHART_DIR_OVERRIDE}"

    if [[ -n "${EARLIER_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG"
    fi
    CENTRAL_CHART_DIR_OVERRIDE="${_CENTRAL_CHART_DIR_OVERRIDE}" _deploy_stackrox

    # Upgrade to HEAD chart without explicit disabling of Scanner v4.
    info "Upgrading StackRox using HEAD Helm chart"
    MAIN_IMAGE_TAG="${main_image_tag}"

    _deploy_stackrox

    # Verify that Scanner v2 and v4 are up.
    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_central_scannerV4_env_var_set "stackrox"
}

@test "Fresh installation of HEAD Helm chart with Scanner V4 disabled and enabling it later" {
    info "Installing StackRox using HEAD Helm chart with Scanner v4 disabled and enabling it later"
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    ROX_SCANNER_V4=false _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_no_scannerV4_deployed "stackrox"
    run ! verify_central_scannerV4_env_var_set "stackrox"

    HELM_REUSE_VALUES=true _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_central_scannerV4_env_var_set "stackrox"
}

@test "Fresh installation of HEAD Helm chart with Scanner v4 enabled" {
    info "Installing StackRox using HEAD Helm chart with Scanner v4 enabled"
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_central_scannerV4_env_var_set "stackrox"
}

@test "Fresh installation of HEAD Helm charts with Scanner v4 enabled in multi-namespace mode" {
    local central_namespace="$CUSTOM_CENTRAL_NAMESPACE"
    local sensor_namespace="$CUSTOM_SENSOR_NAMESPACE"

    info "Installing StackRox using HEAD Helm chart with Scanner v4 enabled in multi-namespace mode"

    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_SUPPORT=true
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true
    _deploy_stackrox "" "$central_namespace" "$sensor_namespace"

    verify_scannerV2_deployed "$central_namespace"
    verify_scannerV4_deployed "$central_namespace"
    verify_central_scannerV4_env_var_set "$central_namespace"
    verify_scannerV4_indexer_deployed "$sensor_namespace"
}

@test "[Manifest Bundle] Fresh installation without Scanner V4, adding Scanner V4 later" {
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="false"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_HELM_DEPLOY="false"
    if [[ "${ORCHESTRATOR_FLAVOR:-}" == "openshift" ]]; then
      export ROX_OPENSHIFT_VERSION=4
    fi
    export GENERATE_SCANNER_DEPLOYMENT_BUNDLE="true"
    local scanner_bundle="${ROOT}/deploy/${ORCHESTRATOR_FLAVOR}/scanner-deploy"

    _deploy_stackrox

    verify_scannerV2_deployed
    verify_no_scannerV4_deployed
    run ! verify_central_scannerV4_env_var_set

    assert [ -d "${scanner_bundle}" ]
    assert [ -d "${scanner_bundle}/scanner-v4" ]

    echo "Deploying Scanner V4..."
    if [[ -x "${scanner_bundle}/scanner-v4/scripts/setup.sh" ]]; then
        "${scanner_bundle}/scanner-v4/scripts/setup.sh"
    fi
    ${ORCH_CMD} apply -R -f "${scanner_bundle}/scanner-v4"

    verify_scannerV4_deployed
    verify_central_scannerV4_env_var_set
}

@test "[Operator] Fresh installation with Scanner V4 enabled" {
    if [[ "${ORCHESTRATOR_FLAVOR:-}" != "openshift" ]]; then
        skip "This test is currently only supported on OpenShift"
    fi
    if [[ "${ENABLE_OPERATOR_TESTS:-}" != "true" ]]; then
        skip "Operator tests disabled. Set ENABLE_OPERATOR_TESTS=true to enable them."
    fi
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="true"
    # shellcheck disable=SC2030,SC2031
    export DEPLOY_STACKROX_VIA_OPERATOR="true"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_SUPPORT=true
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true
    _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
}

@test "[Operator] Fresh multi-namespace installation with Scanner V4 enabled" {
    if [[ "${ORCHESTRATOR_FLAVOR:-}" != "openshift" ]]; then
        skip "This test is currently only supported on OpenShift"
    fi
    if [[ "${ENABLE_OPERATOR_TESTS:-}" != "true" ]]; then
        skip "Operator tests disabled. Set ENABLE_OPERATOR_TESTS=true to enable them."
    fi

    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="true"
    # shellcheck disable=SC2030,SC2031
    export DEPLOY_STACKROX_VIA_OPERATOR="true"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_SUPPORT=true
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true
    _deploy_stackrox "" "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"

    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
}

@test "Fresh installation using roxctl with Scanner V4 enabled" {
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="true"
    if [[ "${ORCHESTRATOR_FLAVOR:-}" == "openshift" ]]; then
      export ROX_OPENSHIFT_VERSION=4
    fi
    # shellcheck disable=SC2030,SC2031
    export SENSOR_HELM_DEPLOY="false"

    _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_central_scannerV4_env_var_set "stackrox"
}

@test "Upgrade from old version without Scanner V4 support to the version which supports Scanner v4" {
    if [[ "$CI" = "true" ]]; then
        setup_default_TLS_certs
    fi

    # Install using roxctl deployment bundles
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    info "Using roxctl executable ${EARLIER_ROXCTL_PATH}/roxctl for generating pre-Scanner V4 deployment bundles"
    PATH="${EARLIER_ROXCTL_PATH}:${PATH}" MAIN_IMAGE_TAG="${EARLIER_MAIN_IMAGE_TAG}" _deploy_stackrox
    verify_scannerV2_deployed
    verify_no_scannerV4_deployed
    run ! verify_central_scannerV4_env_var_set "stackrox"

    info "Upgrading StackRox using HEAD deployment bundles"
    _deploy_stackrox

    verify_scannerV2_deployed
    verify_scannerV4_deployed
    verify_central_scannerV4_env_var_set "stackrox"
}

verify_no_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    verify_no_scannerV4_indexer_deployed "$namespace"
    verify_no_scannerV4_matcher_deployed "$namespace"
}

verify_no_scannerV4_indexer_deployed() {
    local namespace=${1:-stackrox}
    run kubectl -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "scanner-v4-indexer"
}

verify_no_scannerV4_matcher_deployed() {
    local namespace=${1:-stackrox}
    run kubectl -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "scanner-v4-matcher"
}

# TODO: For now, Scanner v2 is expected to run in parallel.
# This must be removed when Scanner v2 will be phased out.
verify_scannerV2_deployed() {
    local namespace=${1:-stackrox}
    wait_for_object_to_appear "$namespace" deploy/scanner 300
    wait_for_object_to_appear "$namespace" deploy/scanner-db 300
}

verify_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    verify_scannerV4_indexer_deployed "$namespace"
    verify_scannerV4_matcher_deployed "$namespace"
}

verify_scannerV4_indexer_deployed() {
    local namespace=${1:-stackrox}
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-db 300
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-indexer 300
    wait_for_ready_pods "${namespace}" "scanner-v4-db" 300
    wait_for_ready_pods "${namespace}" "scanner-v4-indexer" 120
}

verify_scannerV4_matcher_deployed() {
    local namespace=${1:-stackrox}
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-db 300
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-matcher 300
    wait_for_ready_pods "${namespace}" "scanner-v4-db" 300
    wait_for_ready_pods "${namespace}" "scanner-v4-matcher" 120
}

verify_central_scannerV4_env_var_set() {
    local namespace=${1:-stackrox}
    local central_env_vars
    local scanner_v4_value

    central_env_vars="$("${ORCH_CMD}" -n "${namespace}" get deploy/central -o jsonpath="{.spec.template.spec.containers[?(@.name=='central')].env}")"
    scanner_v4_value="$(echo "${central_env_vars}" | jq -r '.[] | select(.name == "ROX_SCANNER_V4").value')"

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

    VERSION="${OPERATOR_VERSION_TAG}" deploy_stackrox_operator

    _deploy_central "${central_namespace}"
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
    "${ORCH_CMD}" -n "${sensor_namespace}" delete pod -l app=collector --grace-period=0

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

    if "$ORCH_CMD" -n "${central_namespace}" get hpa scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${central_namespace}" patch "hpa/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi

    if "$ORCH_CMD" -n "${central_namespace}" get hpa scanner-v4-matcher >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${central_namespace}" patch "hpa/scanner-v4-matcher" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi
    if "${ORCH_CMD}" -n "${central_namespace}" get deploy/scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${central_namespace}" patch "deploy/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: indexer
          resources:
            requests:
              memory: "4300Mi"
              cpu: "1000m"
            limits:
              memory: "4600Mi"
              cpu: "1000m"
EOF
        )
    fi
    if "${ORCH_CMD}" -n "${central_namespace}" get deploy/scanner-v4-matcher >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${central_namespace}" patch "deploy/scanner-v4-matcher" --patch-file <(cat <<EOF
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: matcher
          resources:
            requests:
              memory: "2000Mi"
              cpu: "400m"
            limits:
              memory: "2000Mi"
              cpu: "6000m"
EOF
        )
    fi
    if "${ORCH_CMD}" -n "${central_namespace}" get deploy/scanner-v4-db >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${central_namespace}" patch "deploy/scanner-v4-db" --patch-file <(cat <<EOF
spec:
  template:
    spec:
      containers:
        - name: db
          resources:
            requests:
              memory: "500Mi"
              cpu: "300m"
            limits:
              memory: "1000Mi"
              cpu: "1000m"
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

    "${ORCH_CMD}" -n "${sensor_namespace}" patch "deploy/sensor" --patch-file <(cat <<EOF
spec:
  template:
    spec:
      containers:
        - name: sensor
          resources:
            requests:
              cpu: "1500m"
            limits:
              cpu: "2000m"
EOF
    )

    if "$ORCH_CMD" -n "${sensor_namespace}" get hpa scanner >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${sensor_namespace}" patch "hpa/scanner" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi
    if "$ORCH_CMD" -n "${sensor_namespace}" get hpa scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${sensor_namespace}" patch "hpa/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  minReplicas: 1
  maxReplicas: 1
EOF
        )
    fi
    if "${ORCH_CMD}" -n "${central_namespace}" get deploy/scanner-v4-db >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${sensor_namespace}" patch "deploy/scanner-v4-db" --patch-file <(cat <<EOF
spec:
  template:
    spec:
      containers:
        - name: db
          resources:
            requests:
              memory: "500Mi"
              cpu: "300m"
            limits:
              memory: "1000Mi"
              cpu: "1000m"
EOF
        )
    fi
    if "${ORCH_CMD}" -n "${central_namespace}" get deploy/scanner-v4-indexer >/dev/null 2>&1; then
        "${ORCH_CMD}" -n "${sensor_namespace}" patch "deploy/scanner-v4-indexer" --patch-file <(cat <<EOF
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: indexer
          resources:
            requests:
              memory: "4300Mi"
              cpu: "1000m"
            limits:
              memory: "4600Mi"
              cpu: "1000m"
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
      deployment_json="$("${ORCH_CMD}" -n "${namespace}" get "deployment/${deployment}" -o json)"
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
        "${ORCH_CMD}" -n "${namespace}" get pod -o wide
        "${ORCH_CMD}" -n "${namespace}" get deploy -o wide
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
