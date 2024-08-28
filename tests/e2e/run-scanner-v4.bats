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
        OPERATOR_VERSION_TAG=$(echo "${MAIN_IMAGE_TAG}" | sed -E 's@^(([[:digit:]]+\.)+)x(-)?@\10\3@g' | sed -E 's@^3.0.([[:digit:]]+\.[[:digit:]]+)(-)?@3.\1\2@g')
    fi

    setup_default_TLS_certs

    # Configure a timeout for a single test. After 30m of runtime a test will be marked as failed
    # (and we will hopefully receive helpful logs for analysing the situation).
    # Without a timeout it might happen that the pod running the tests is simply killed and we won't
    # have any logs for investigation the situation.
    export BATS_TEST_TIMEOUT=1800 # Seconds
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

@test "Upgrade from old Helm chart to HEAD Helm chart with Scanner v4 enabled" {
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    local main_image_tag="${MAIN_IMAGE_TAG}"

    # Deploy earlier version without Scanner V4.
    local _CENTRAL_CHART_DIR_OVERRIDE="${CHART_REPOSITORY}${CHART_BASE}/${EARLIER_CHART_VERSION}/central-services"
    info "Deploying StackRox services using chart ${_CENTRAL_CHART_DIR_OVERRIDE}"

    if [[ -n "${EARLIER_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG"
    fi
    local _ROX_CENTRAL_EXTRA_HELM_VALUES_FILE
    _ROX_CENTRAL_EXTRA_HELM_VALUES_FILE=$(mktemp)
    cat <<EOF >"$_ROX_CENTRAL_EXTRA_HELM_VALUES_FILE"
central:
  persistence:
    none: true
EOF
    ROX_CENTRAL_EXTRA_HELM_VALUES_FILE="${_ROX_CENTRAL_EXTRA_HELM_VALUES_FILE}" CENTRAL_CHART_DIR_OVERRIDE="${_CENTRAL_CHART_DIR_OVERRIDE}" _deploy_stackrox

    # Upgrade to HEAD chart without explicit disabling of Scanner v4.
    info "Upgrading StackRox using HEAD Helm chart"
    MAIN_IMAGE_TAG="${main_image_tag}"

    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true

    _deploy_stackrox

    # Verify that Scanner v2 and v4 are up.
    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    verify_deployment_scannerV4_env_var_set "stackrox" "sensor"
}

@test "Fresh installation of HEAD Helm chart with Scanner V4 disabled and enabling it later" {
    info "Installing StackRox using HEAD Helm chart with Scanner v4 disabled and enabling it later"
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    ROX_SCANNER_V4=false _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_no_scannerV4_deployed "stackrox"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "central"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "sensor"

    SENSOR_SCANNER_V4_SUPPORT=true HELM_REUSE_VALUES=true _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    verify_deployment_scannerV4_env_var_set "stackrox" "sensor"

    # Deactivate Scanner V4 for both releases.
    info "Disabling Scanner V4 for Central"
    helm upgrade -n stackrox stackrox-central-services "${CENTRAL_CHART_DIR}" --reuse-values --set scannerV4.disable=true
    info "Disabling Scanner V4 for SecuredCluster"
    helm upgrade -n stackrox stackrox-secured-cluster-services "${SENSOR_CHART_DIR}" --reuse-values --set scannerV4.disable=true
    sleep 30 # Give the deployments time to terminate.

    verify_no_scannerV4_deployed "stackrox"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "central"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "sensor"

}

@test "Fresh installation of HEAD Helm chart with Scanner v4 enabled" {
    info "Installing StackRox using HEAD Helm chart with Scanner v4 enabled"
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=helm
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true

    _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    verify_deployment_scannerV4_env_var_set "stackrox" "sensor"
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
    verify_deployment_scannerV4_env_var_set "$central_namespace" "central"
    verify_scannerV4_indexer_deployed "$sensor_namespace"
    verify_deployment_scannerV4_env_var_set "$sensor_namespace" "sensor"

    # Deactivate Scanner V4 for both releases.
    helm upgrade -n "${central_namespace}" stackrox-central-services "${CENTRAL_CHART_DIR}" --reuse-values --set scannerV4.disable=true
    helm upgrade -n "${sensor_namespace}" stackrox-secured-cluster-services "${SENSOR_CHART_DIR}" --reuse-values --set scannerV4.disable=true
    sleep 30 # Give the deployments time to terminate.

    verify_no_scannerV4_deployed "${central_namespace}"
    verify_no_scannerV4_deployed "${sensor_namespace}"
    run ! verify_deployment_scannerV4_env_var_set "${central_namespace}" "central"
    run ! verify_deployment_scannerV4_env_var_set "${sensor_namespace}" "sensor"

}

@test "[Manifest Bundle] Fresh installation without Scanner V4, adding Scanner V4 later" {
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="false"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_HELM_DEPLOY="false"
    export GENERATE_SCANNER_DEPLOYMENT_BUNDLE="true"
    local scanner_bundle="${ROOT}/deploy/${ORCHESTRATOR_FLAVOR}/scanner-deploy"

    _deploy_stackrox

    verify_scannerV2_deployed
    verify_no_scannerV4_deployed
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "central"

    assert [ -d "${scanner_bundle}" ]
    assert [ -d "${scanner_bundle}/scanner-v4" ]

    echo "Deploying Scanner V4..."
    if [[ -x "${scanner_bundle}/scanner-v4/scripts/setup.sh" ]]; then
        "${scanner_bundle}/scanner-v4/scripts/setup.sh"
    fi
    "${ORCH_CMD}" </dev/null apply -R -f "${scanner_bundle}/scanner-v4"

    verify_scannerV4_deployed
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
}

@test "[Operator] Fresh installation with Scanner V4 enabled" {
    if [[ "${ORCHESTRATOR_FLAVOR:-}" != "openshift" ]]; then
        skip "This test is currently only supported on OpenShift"
    fi
    if [[ "${ENABLE_OPERATOR_TESTS:-}" != "true" ]]; then
        skip "Operator tests disabled. Set ENABLE_OPERATOR_TESTS=true to enable them."
    fi
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="true"
    # shellcheck disable=SC2030,SC2031
    export DEPLOY_STACKROX_VIA_OPERATOR="true"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_SUPPORT=true
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true

    VERSION="${OPERATOR_VERSION_TAG}" deploy_stackrox_operator
    _deploy_stackrox

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    verify_deployment_scannerV4_env_var_set "stackrox" "sensor"
}

@test "[Operator] Fresh multi-namespace installation with Scanner V4 enabled" {
    if [[ "${ORCHESTRATOR_FLAVOR:-}" != "openshift" ]]; then
        skip "This test is currently only supported on OpenShift"
    fi
    if [[ "${ENABLE_OPERATOR_TESTS:-}" != "true" ]]; then
        skip "Operator tests disabled. Set ENABLE_OPERATOR_TESTS=true to enable them."
    fi

    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="true"
    # shellcheck disable=SC2030,SC2031
    export DEPLOY_STACKROX_VIA_OPERATOR="true"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_SUPPORT=true
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_V4_SUPPORT=true
    VERSION="${OPERATOR_VERSION_TAG}" deploy_stackrox_operator
    _deploy_stackrox "" "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"

    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"
}

@test "[Operator] Upgrade multi-namespace installation" {
    if [[ "${ORCHESTRATOR_FLAVOR:-}" != "openshift" ]]; then
        skip "This test is currently only supported on OpenShift"
    fi
    if [[ "${ENABLE_OPERATOR_TESTS:-}" != "true" ]]; then
        skip "Operator tests disabled. Set ENABLE_OPERATOR_TESTS=true to enable them."
    fi

    # shellcheck disable=SC2030,SC2031
    export DEPLOY_STACKROX_VIA_OPERATOR="true"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_SCANNER_SUPPORT=true

    # Install old version of the operator & deploy StackRox.
    (
      VERSION="${OPERATOR_VERSION_TAG}" make -C operator deploy-previous-via-olm
      #  cd operator
      #  ./hack/olm-operator-install.sh stackrox-operator quay.io/rhacs-eng/stackrox-operator 4.3.0 4.3.0
    )
    ROX_SCANNER_V4="false" _deploy_stackrox "" "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"

    # Upgrade operator
    info "Upgrading StackRox Operator to version ${OPERATOR_VERSION_TAG}..."
    VERSION="${OPERATOR_VERSION_TAG}" make -C operator upgrade-via-olm
    info "Waiting for new rhacs-operator pods to become ready"
    # Give the old pods some time to terminate, otherwise we can end up
    # in a situation where the old pods are just about to terminate and this
    # would confuse the kubectl wait invocation below, which notices pods
    # vanishing while actually waiting for them to become ready.
    sleep 60
    "${ORCH_CMD}" </dev/null -n stackrox-operator wait --for=condition=Ready --timeout=3m pods -l app=rhacs-operator

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_no_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"
    verify_no_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

    wait_until_central_validation_webhook_is_ready "${CUSTOM_CENTRAL_NAMESPACE}"

    # Enable Scanner V4 on central side.
    info "Patching Central"
    "${ORCH_CMD}" </dev/null -n "${CUSTOM_CENTRAL_NAMESPACE}" \
      patch Central stackrox-central-services --type=merge --patch-file=<(cat <<EOT
spec:
  scannerV4:
    scannerComponent: Enabled
    indexer:
      scaling:
        autoScaling: Disabled
        replicas: 1
      resources:
        requests:
          cpu: "400m"
          memory: "1500Mi"
        limits:
          cpu: "1000m"
          memory: "2Gi"
    matcher:
      scaling:
        autoScaling: Disabled
        replicas: 1
      resources:
        requests:
          cpu: "400m"
          memory: "5Gi"
        limits:
          cpu: "1000m"
          memory: "5500Mi"
    db:
      resources:
        requests:
          cpu: "400m"
          memory: "2Gi"
        limits:
          cpu: "1000m"
          memory: "2500Mi"
EOT
    )

    info "Waiting for central to come back up after patching CR for activating Scanner V4"
    sleep 60
    "${ORCH_CMD}" </dev/null -n "${CUSTOM_CENTRAL_NAMESPACE}" wait --for=condition=Ready pods -l app=central || true

    info "Patching SecuredCluster"
    # Enable Scanner V4 on secured-cluster side
    "${ORCH_CMD}" </dev/null -n "${CUSTOM_SENSOR_NAMESPACE}" \
      patch SecuredCluster stackrox-secured-cluster-services --type=merge --patch-file=<(cat <<EOT
spec:
  scannerV4:
    scannerComponent: AutoSense
    indexer:
      scaling:
        autoScaling: Disabled
        replicas: 1
      resources:
        requests:
          cpu: "400m"
          memory: "1500Mi"
        limits:
          cpu: "1000m"
          memory: "2Gi"
    db:
      resources:
        requests:
          cpu: "200m"
          memory: "2Gi"
        limits:
          cpu: "1000m"
          memory: "2500Mi"
EOT
    )

    info "Waiting for sensor to come back up after patching CR for activating Scanner V4"
    sleep 60
    "${ORCH_CMD}" </dev/null -n "${CUSTOM_SENSOR_NAMESPACE}" wait --for=condition=Ready pods -l app=sensor || true

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"
    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

    # Test disabling of Scanner V4.
    info "Disabling Scanner V4 for Central"
    "${ORCH_CMD}" </dev/null -n "${CUSTOM_CENTRAL_NAMESPACE}" \
      patch Central stackrox-central-services --type=merge --patch-file=<(cat <<EOT
spec:
  scannerV4:
    scannerComponent: Disabled
EOT
    )

    info "Disabling Scanner V4 for SecuredCluster"
    "${ORCH_CMD}" </dev/null -n "${CUSTOM_SENSOR_NAMESPACE}" \
      patch SecuredCluster stackrox-secured-cluster-services --type=merge --patch-file=<(cat <<EOT
spec:
  scannerV4:
    scannerComponent: Disabled
EOT
    )

    sleep 2m # Give the operator some time to reconcile and the deployments to terminate.
    verify_no_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_no_scannerV4_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

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
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
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
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "central"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "sensor"

    info "Upgrading StackRox using HEAD deployment bundles"
    _deploy_stackrox

    verify_scannerV2_deployed
    verify_scannerV4_deployed
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "sensor" # no Scanner V4 support in Sensor with roxctl
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
