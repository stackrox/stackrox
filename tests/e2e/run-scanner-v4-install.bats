#!/usr/bin/env bats

# Runs Scanner V4 installation tests using the Bats testing framework.
#
# NOTE: For debugging purposes you can run this test suite locally against a remote cluster. For example:
#
#   ABORT_ON_FAILURE=true ORCHESTRATOR_FLAVOR=[openshift|k8s] BATS_CORE_ROOT=$HOME/bats-core \
#     bats --report-formatter junit --print-output-on-failure --show-output-of-passing-tests ./tests/e2e/run-scanner-v4-install.bats
#
#   (you need to point $BATS_CORE_ROOT to your directory containing checkouts of:
#     https://github.com/bats-core/bats-core
#     https://github.com/bats-core/bats-assert)
#

set -euo pipefail

if [[ -z "${REAL_TIME_TEST_OUTPUT:-}" ]] && test -t 0; then
    # Use real-time test output by default when executed on a terminal.
    export REAL_TIME_TEST_OUTPUT="true"
fi

outfd=1
if [[ "${REAL_TIME_TEST_OUTPUT:-}" == "true" ]]; then
    # Bats expects output to be printed unconditionally on fd 3.
    outfd=3
fi
export outfd

init() {
    set -euo pipefail
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
}

initialized="false"
begin_timestamp=""
current_label=""
post_processor_pid=""

_begin() {
    # In case it is convenient for a given test-case, _begin() can take care of the initialization.
    if [[ $initialized = "false" ]]; then
        init
        initialized="true"
    fi

    local label="${1:-}"
    current_label="$label"

    # Save original stdout and stderr fds as 4 and 5.
    exec 4>&1 5>&2
    # Connect new stdout and stderr fds with a pipe to post-processor.
    # In any case, output will be written to $outfd, which is either the original stdout
    # or 3 (for Bats real-time output).
    exec \
        1> >(bash -c "post_process_output '$label'" > "/dev/fd/$outfd") \
        2>&1
    post_processor_pid="$!"
    begin_timestamp=$(date +%s)
}

_end() {
    local end_timestamp=$(date +%s)
    local test_identifier=$(test_identifier_from_description "${BATS_TEST_DESCRIPTION:-}")

    emit_timing_data "$test_identifier" "$current_label" "$begin_timestamp" "$end_timestamp"
    # Close post-processing stdout and stderr and restore from original fds.
    exec 1>&- 2>&- 1>&4 2>&5
    wait "$post_processor_pid" || echo "Failed to wait for output post processor (PID ${post_processor_pid})."
    post_processor_pid=""
    current_label=""
    begin_timestamp=""
}

emit_timing_data() {
    local test="$1"
    local step="$2"
    local t0="$3"
    local t1="$4"
    local seconds_spent=$((t1 - t0))

    cat <<EOT
TIMING_DATA: {"test": "$test", "step": "$step", "seconds_spent": $seconds_spent}
EOT
}

# Combined _end() and _begin() for convenience.
_step() {
    _end
    _begin "$1"
}

export TEST_SUITE_ABORTED="false"

export test_suite_begin_timestamp=""

setup_file() {
    test_suite_begin_timestamp=$(date +%s)
    _begin "setup-file"


    cat <<'EOT'
    _    ____ ____    ___           _        _ _       _   _               _____         _
   / \  / ___/ ___|  |_ _|_ __  ___| |_ __ _| | | __ _| |_(_) ___  _ __   |_   _|__  ___| |_ ___
  / _ \| |   \___ \   | || '_ \/ __| __/ _` | | |/ _` | __| |/ _ \| '_ \    | |/ _ \/ __| __/ __|
 / ___ \ |___ ___) |  | || | | \__ \ || (_| | | | (_| | |_| | (_) | | | |   | |  __/\__ \ |_\__ \
/_/   \_\____|____/  |___|_| |_|___/\__\__,_|_|_|\__,_|\__|_|\___/|_| |_|   |_|\___||___/\__|___/

EOT

    bats_require_minimum_version 1.5.0
    require_environment "ORCHESTRATOR_FLAVOR"

    # Use
    #   export CHART_BASE="/rhacs"
    #   export DEFAULT_IMAGE_REGISTRY="quay.io/rhacs-eng"
    # for the RHACS flavor.
    export CHART_BASE=""
    export DEFAULT_IMAGE_REGISTRY="quay.io/stackrox-io"

    export MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(make --quiet --no-print-directory -C "${ROOT}" tag)}"
    info "Using MAIN_IMAGE_TAG=$MAIN_IMAGE_TAG"

    export CURRENT_MAIN_IMAGE_TAG=${CURRENT_MAIN_IMAGE_TAG:-} # Setting a tag can be useful for local testing.
    export EARLIER_CHART_VERSION="4.6.0"
    export EARLIER_MAIN_IMAGE_TAG=$EARLIER_CHART_VERSION
    export USE_LOCAL_ROXCTL="${USE_LOCAL_ROXCTL:-true}"
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
    if [[ -z "${ROX_ADMIN_PASSWORD:-}" ]]; then
        ROX_ADMIN_PASSWORD="$(openssl rand -base64 20 | tr -d '/=+')"
    fi
    export ROX_ADMIN_PASSWORD

    if ! command -v roxctl >/dev/null; then
        die "roxctl not found, please make sure it can be resolved via PATH."
    fi

    local roxctl_version
    roxctl_version="$(roxctl version)"

    # Prepare earlier Helm chart version.
    if [[ -z "${CHART_REPOSITORY:-}" ]]; then
        CHART_REPOSITORY=$(mktemp -d "helm-charts.XXXXXX" -p /tmp)
    fi
    if [[ ! -e "${CHART_REPOSITORY}/.git" ]]; then
        echo "Cloning released Helm charts into ${CHART_REPOSITORY}..."
        git clone --quiet --depth 1 -b main https://github.com/stackrox/helm-charts "${CHART_REPOSITORY}"
    fi
    export CHART_REPOSITORY

    if [[ -n "$MAIN_IMAGE_TAG" ]] && [[ "$roxctl_version" != "$MAIN_IMAGE_TAG" ]]; then
        info "MAIN_IMAGE_TAG ($MAIN_IMAGE_TAG) does not match roxctl version ($roxctl_version)."
    fi

    # Download and use earlier version of roxctl without Scanner V4 support
    # We will just hard-code a pre-4.4 version here.
    if [[ -z "${EARLIER_ROXCTL_PATH:-}" ]]; then
        EARLIER_ROXCTL_PATH=$(mktemp -d "early_roxctl.XXXXXX" -p /tmp)
    fi
    echo "For tests requiring an older roxctl version, $EARLIER_ROXCTL_PATH will be used."
    export EARLIER_ROXCTL_PATH
    if [[ ! -e "${EARLIER_ROXCTL_PATH}/roxctl" ]]; then
        local ROXCTL_URL="https://mirror.openshift.com/pub/rhacs/assets/${EARLIER_MAIN_IMAGE_TAG}/bin/${OS}/roxctl"
        echo "Downloading roxctl from ${ROXCTL_URL}..."
        curl --retry 5 --retry-connrefused -sL "$ROXCTL_URL" --output "${EARLIER_ROXCTL_PATH}/roxctl"
        chmod +x "${EARLIER_ROXCTL_PATH}/roxctl"
    fi

    export CUSTOM_CENTRAL_NAMESPACE=${CUSTOM_CENTRAL_NAMESPACE:-stackrox-central}
    export CUSTOM_SENSOR_NAMESPACE=${CUSTOM_SENSOR_NAMESPACE:-stackrox-sensor}

    # Taken from operator/Makefile
    export OPERATOR_VERSION_TAG=${OPERATOR_VERSION_TAG:-}
    if [[ -z "${OPERATOR_VERSION_TAG}" && -n "${MAIN_IMAGE_TAG:-}" ]]; then
        OPERATOR_VERSION_TAG=$(echo "${MAIN_IMAGE_TAG}" | sed -E 's@^(([[:digit:]]+\.)+)x(-)?@\10\3@g')
    fi
    echo "Using OPERATOR_VERSION_TAG=${OPERATOR_VERSION_TAG}"

    setup_default_TLS_certs

    # Configure a timeout for a single test. After 30m of runtime a test will be marked as failed
    # (and we will hopefully receive helpful logs for analysing the situation).
    # Without a timeout it might happen that the pod running the tests is simply killed and we won't
    # have any logs for investigation the situation.
    export BATS_TEST_TIMEOUT=1800 # Seconds

   if [[ -z "${HEAD_HELM_CHART_CENTRAL_SERVICES_DIR:-}" ]]; then
        HEAD_HELM_CHART_CENTRAL_SERVICES_DIR=$(mktemp -d)
        echo "Rendering fresh central-services Helm chart and writing to ${HEAD_HELM_CHART_CENTRAL_SERVICES_DIR}..."
        roxctl helm output central-services \
            --debug --debug-path="${ROOT}/image" \
            --output-dir="${HEAD_HELM_CHART_CENTRAL_SERVICES_DIR}" --remove
        export HEAD_HELM_CHART_CENTRAL_SERVICES_DIR
    fi

   if [[ -z "${HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR:-}" ]]; then
        HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR=$(mktemp -d)
        echo "Rendering fresh secured-cluster-services Helm chart and writing to ${HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR}..."
        roxctl helm output secured-cluster-services \
            --debug --debug-path="${ROOT}/image" \
            --output-dir="${HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR}" --remove
        export HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR
    fi

    # For installation testing we don't need to deploy collector per-node, it is sufficient to deploy
    # collector on a single node.
    local _worker; _worker=$(select_worker_node)

    echo "Patching node ${_worker} to include label run-collector=true"
    "${ORCH_CMD}" </dev/null label node "$_worker" run-collector=true
    echo "collector will only be scheduled on node ${_worker}"

    _end
}

select_worker_node() {
    local select_filter="true"
    if [[ "$ORCHESTRATOR_FLAVOR" == "openshift" ]]; then
        select_filter=".metadata.labels[\"node-role.kubernetes.io/worker\"] != null"
    fi

    "${ORCH_CMD}" </dev/null get nodes -o json | jq -r ".items | map(select(${select_filter})) | map(.metadata.name) | sort | first"
}

teardown_file() {
    local test_suite_end_timestamp=$(date +%s)
    _begin "teardown-file"
    emit_timing_data "" "test-suite" "$test_suite_begin_timestamp" "$test_suite_end_timestamp"
    _end
}

test_case_no=0

setup() {
    [[ "${TEST_SUITE_ABORTED}" == "true" ]] && return 1

    _begin "setup-test-env"

    echo "Executing Test: $BATS_TEST_DESCRIPTION"
    export_test_environment
    if [[ "$CI" = "true" ]]; then
        setup_gcp
        setup_deployment_env false false
    fi

    _step "pre-test-tear-down"

    if [[ "${SKIP_INITIAL_TEARDOWN:-}" != "true" ]] && (( test_case_no == 0 )); then
        # executing teardown to begin test execution in a well-defined state
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

    echo "Finished test setup."
    _end
}

# CRD needs to be owned by Helm if upgrading to 4.8+ from 4.7.x via Helm
apply_crd_ownership_for_upgrade() {
    local namespace=$1
    echo "Making sure that SecurityPolicies CRD has the correct metadata..."
    "${ORCH_CMD}" </dev/null annotate crd/securitypolicies.config.stackrox.io meta.helm.sh/release-name=stackrox-central-services || true
    "${ORCH_CMD}" </dev/null annotate crd/securitypolicies.config.stackrox.io meta.helm.sh/release-namespace="$namespace" || true
    "${ORCH_CMD}" </dev/null label crd/securitypolicies.config.stackrox.io app.kubernetes.io/managed-by=Helm || true
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
    _begin "post-test-tear-down"

    if [[ "${TEST_SUITE_ABORTED}" == "true" ]]; then
        echo "Skipping teardown due to previous failure." >&3
        return
    fi

    if [[ "${BATS_TEST_COMPLETED:-}" != "1" ]]; then
        # Previous test failed.
        echo "FAILED: Test \"$BATS_TEST_DESCRIPTION\" failed. Look above for the test steps that have led to this failure."
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

    echo "Using central namespace: ${central_namespace}"
    echo "Using sensor namespace: ${sensor_namespace}"

    if [[ -n "${SCANNER_V4_LOG_DIR:-}" ]]; then
        "$ROOT/scripts/ci/collect-service-logs.sh" "${central_namespace}" \
            "${SCANNER_V4_LOG_DIR}/${BATS_TEST_NUMBER}-${BATS_TEST_NAME}"

        if [[ "${central_namespace}" != "${sensor_namespace}" && -n "${sensor_namespace}" ]]; then
            "$ROOT/scripts/ci/collect-service-logs.sh" "${sensor_namespace}" \
                "${SCANNER_V4_LOG_DIR}/${BATS_TEST_NUMBER}-${BATS_TEST_NAME}"
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

    run remove_existing_stackrox_resources "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}" "stackrox"
    echo "Post-test teardown complete."

    _end
}

@test "Upgrade from old Helm chart to HEAD Helm chart with Scanner v4 enabled" {
    init

    local main_image_tag="${MAIN_IMAGE_TAG}"

    # Deploy earlier version without Scanner V4.
    local old_central_chart="${CHART_REPOSITORY}${CHART_BASE}/${EARLIER_CHART_VERSION}/central-services"
    local old_sensor_chart="${CHART_REPOSITORY}${CHART_BASE}/${EARLIER_CHART_VERSION}/secured-cluster-services"

    _begin "deploy-old-central"
    info "Deploying StackRox central-services using chart ${old_central_chart}"
    deploy_central_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$EARLIER_MAIN_IMAGE_TAG" "$old_central_chart" \
        -f <(cat <<EOT
central:
  adminPassword:
    value: "$ROX_ADMIN_PASSWORD"
EOT
    )

    _step "verify-scanner-V4-not-deployed"
    verify_scannerV2_deployed "$CUSTOM_CENTRAL_NAMESPACE"
    verify_no_scannerV4_deployed "$CUSTOM_CENTRAL_NAMESPACE"

    _begin "upgrade-to-HEAD-central"
    deploy_central_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$MAIN_IMAGE_TAG" "" \
        --reuse-values

    _step "verify-scanner-V4-not-deployed"
    verify_scannerV2_deployed "$CUSTOM_CENTRAL_NAMESPACE"
    verify_no_scannerV4_deployed "$CUSTOM_CENTRAL_NAMESPACE"

    _begin "enable-scanner-V4-in-central"
    deploy_central_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$MAIN_IMAGE_TAG" "" \
        --reuse-values \
        --set scannerV4.disable=false

    _step "verify-scanners-deployed"
    verify_scannerV2_deployed "$CUSTOM_CENTRAL_NAMESPACE"
    verify_scannerV4_deployed "$CUSTOM_CENTRAL_NAMESPACE"
    verify_deployment_scannerV4_env_var_set "$CUSTOM_CENTRAL_NAMESPACE" "central"

    _begin "deploy-old-sensor"
    info "Deploying StackRox secured-cluster-services using chart ${old_sensor_chart}"
    local central_endpoint="$(get_central_endpoint "$CUSTOM_CENTRAL_NAMESPACE")"
    local secured_cluster_name="$(get_cluster_name)"
    deploy_sensor_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$CUSTOM_SENSOR_NAMESPACE" \
        "$EARLIER_MAIN_IMAGE_TAG" "$old_sensor_chart" \
        "$secured_cluster_name" "$ROX_ADMIN_PASSWORD" "$central_endpoint"

    _step "verify-scanner-V4-not-deployed"
    verify_no_scannerV4_deployed "$CUSTOM_SENSOR_NAMESPACE"

    _step "upgrade-to-HEAD-sensor"
    deploy_sensor_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$CUSTOM_SENSOR_NAMESPACE" "" "" "" "" ""

    _step "verify-scanner-not-deployed"
    verify_no_scannerV4_deployed "$CUSTOM_SENSOR_NAMESPACE"

    _begin "enable-scanner-V4-in-secured-cluster"
    # Without creating the scanner-db-password secret manually Scanner V2 doesn't come up.
    # Let's just reuse an existing password for this for simplicity.
    "$ORCH_CMD" </dev/null -n "$CUSTOM_SENSOR_NAMESPACE" create secret generic scanner-db-password \
        --from-file=password=<(echo "$ROX_ADMIN_PASSWORD")

    deploy_sensor_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$CUSTOM_SENSOR_NAMESPACE" "" "" "" "" "" \
        --set scannerV4.disable=false \
        --set scanner.disable=false

    _step "verify-scanner-V4-deployed"
    verify_scannerV4_indexer_deployed "$CUSTOM_SENSOR_NAMESPACE"

    _end
}

@test "Fresh installation of HEAD Helm charts and toggling Scanner V4" {
    local password_setting=$(cat <<EOT
central:
  adminPassword:
    value: "$ROX_ADMIN_PASSWORD"
EOT
    )
    local secured_cluster_name="$(get_cluster_name)"

    ######################
    _begin "deploying-head-central"
    info "Deploying central-services using HEAD chart"
    deploy_central_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$MAIN_IMAGE_TAG" "" \
        -f <(echo "$password_setting")
    local central_endpoint="$(get_central_endpoint "$CUSTOM_CENTRAL_NAMESPACE")"

    ######################
    _begin "deploying-head-sensor"
    info "Deploying secured-cluster-services using HEAD chart"
    deploy_sensor_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$CUSTOM_SENSOR_NAMESPACE" \
        "$MAIN_IMAGE_TAG" "" \
        "$secured_cluster_name" "$ROX_ADMIN_PASSWORD" "$central_endpoint"

    ######################
    _step "verifying-central-scanners-deployed"
    verify_scannerV2_deployed "$CUSTOM_CENTRAL_NAMESPACE"
    verify_scannerV4_deployed "$CUSTOM_CENTRAL_NAMESPACE"
    verify_deployment_scannerV4_env_var_set "$CUSTOM_CENTRAL_NAMESPACE" "central"

    ######################
    _step "verifying-sensor-scanners-deployed"
    verify_scannerV2_deployed "$CUSTOM_SENSOR_NAMESPACE"
    verify_scannerV4_indexer_deployed "$CUSTOM_SENSOR_NAMESPACE"
    run verify_deployment_scannerV4_env_var_set "$CUSTOM_SENSOR_NAMESPACE" "sensor"

    ######################
    _begin "disabling-central-scanner-v4"
    info "Disabling Scanner V4 for central-services"
    deploy_central_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$MAIN_IMAGE_TAG" "" \
        --reuse-values --set scannerV4.disable=true

    ######################
    _step "disabling-sensor-scanners"
    info "Disabling Scanner V4 for secured-cluster-services"
    deploy_sensor_with_helm "$CUSTOM_CENTRAL_NAMESPACE" "$CUSTOM_SENSOR_NAMESPACE" "" "" "" "" "" \
        --set scannerV4.disable=true --set scanner.disable=true

    ######################
    _step "verifying-central-scanner-v4-not-deployed"
    info "Verifying Scanner V4 is getting removed for central-services"
    verify_deployment_deletion_with_timeout 4m "$CUSTOM_CENTRAL_NAMESPACE" scanner-v4-indexer scanner-v4-matcher scanner-v4-db
    run ! verify_deployment_scannerV4_env_var_set "$CUSTOM_CENTRAL_NAMESPACE" "central"

    ######################
    _step "verifying-sensor-scanner-v4-not-deployed"
    info "Verifying Scanner V4 is getting removed for secured-cluster-services"
    verify_deployment_deletion_with_timeout 4m "$CUSTOM_SENSOR_NAMESPACE" scanner-v4-indexer scanner-v4-db
    run ! verify_deployment_scannerV4_env_var_set "$CUSTOM_SENSOR_NAMESPACE" "sensor"

    _end
}

@test "Fresh installation of HEAD Helm charts with Scanner V4 enabled in multi-namespace mode" {
    init

    local central_namespace="$CUSTOM_CENTRAL_NAMESPACE"
    local sensor_namespace="$CUSTOM_SENSOR_NAMESPACE"

    _begin "deploy-stackrox"

    info "Installing StackRox using HEAD Helm chart with Scanner V4 enabled in multi-namespace mode"

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

    verify_deployment_deletion_with_timeout 4m "stackrox" scanner-v4-indexer scanner-v4-matcher scanner-v4-db
    run ! verify_deployment_scannerV4_env_var_set "${central_namespace}" "central"
    run ! verify_deployment_scannerV4_env_var_set "${sensor_namespace}" "sensor"

    _end
}

@test "[Manifest Bundle] Fresh installation without Scanner V4, adding Scanner V4 later" {
    init

    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    # shellcheck disable=SC2030,SC2031
    export ROX_SCANNER_V4="false"
    # shellcheck disable=SC2030,SC2031
    export SENSOR_HELM_DEPLOY="false"
    export GENERATE_SCANNER_DEPLOYMENT_BUNDLE="true"
    local scanner_bundle="${ROOT}/deploy/${ORCHESTRATOR_FLAVOR}/scanner-deploy"

    _begin "deploy-stackrox"

    _deploy_stackrox

    _step "verify"

    verify_scannerV2_deployed
    verify_no_scannerV4_deployed
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "central"

    _step "deploy-scanner-v4"

    assert [ -d "${scanner_bundle}" ]
    assert [ -d "${scanner_bundle}/scanner-v4" ]

    echo "Deploying Scanner V4..."
    if [[ -x "${scanner_bundle}/scanner-v4/scripts/setup.sh" ]]; then
        "${scanner_bundle}/scanner-v4/scripts/setup.sh"
    fi
    "${ORCH_CMD}" </dev/null apply -R -f "${scanner_bundle}/scanner-v4"

    verify_scannerV4_deployed
    verify_deployment_scannerV4_env_var_set "stackrox" "central"

    _end
}

@test "[Operator] Fresh installation with Scanner V4 enabled" {
    init

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

    _begin "deploy-stackrox"

    VERSION="${OPERATOR_VERSION_TAG}" deploy_stackrox_operator
    _deploy_stackrox

    _step "verify"

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    verify_deployment_scannerV4_env_var_set "stackrox" "sensor"

    _end
}

@test "[Operator] Fresh multi-namespace installation with Scanner V4 enabled" {
    init

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

    _begin "deploy-stackrox"

    VERSION="${OPERATOR_VERSION_TAG}" deploy_stackrox_operator
    _deploy_stackrox "" "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}"

    _step "verify"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"

    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

    _end
}

@test "[Operator] Upgrade multi-namespace installation" {
    init

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

    _begin "deploy-stackrox"

    # Install old version of the operator & deploy StackRox.
    VERSION="${OPERATOR_VERSION_TAG}" make -C operator deploy-previous-via-olm
    ROX_SCANNER_V4="false" _deploy_stackrox "" "${CUSTOM_CENTRAL_NAMESPACE}" "${CUSTOM_SENSOR_NAMESPACE}"

    _step "verify"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"

    _step "upgrade-operator"

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

    _step "verify"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_no_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"
    verify_no_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

    wait_until_central_validation_webhook_is_ready "${CUSTOM_CENTRAL_NAMESPACE}"

    _step "patch-central"

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

    _step "patch-secured-cluster"

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

    _step "verify"

    verify_scannerV2_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_scannerV4_deployed "${CUSTOM_CENTRAL_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"
    verify_scannerV2_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_scannerV4_indexer_deployed "${CUSTOM_SENSOR_NAMESPACE}"
    verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

    _step "disable-scanner-v4"

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

    _step "verify"

    verify_deployment_deletion_with_timeout 4m "${CUSTOM_CENTRAL_NAMESPACE}" scanner-v4-indexer scanner-v4-matcher scanner-v4-db
    verify_deployment_deletion_with_timeout 4m "${CUSTOM_SENSOR_NAMESPACE}" scanner-v4-indexer scanner-v4-db
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_CENTRAL_NAMESPACE}" "central"
    run ! verify_deployment_scannerV4_env_var_set "${CUSTOM_SENSOR_NAMESPACE}" "sensor"

    _end
}

@test "Fresh installation using roxctl with Scanner V4 enabled" {
    _begin "deploy-stackrox"

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

    _step "verify"

    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
    verify_deployment_scannerV4_env_var_set "stackrox" "central"

    _end
}

@test "Upgrade from old version without Scanner V4 to HEAD with Scanner V4 enabled" {
    _begin "deploy-stackrox"

    if [[ "$CI" = "true" ]]; then
        setup_default_TLS_certs
    fi

    # Install using roxctl deployment bundles
    # shellcheck disable=SC2030,SC2031
    export OUTPUT_FORMAT=""
    info "Using roxctl executable ${EARLIER_ROXCTL_PATH}/roxctl for generating pre-Scanner V4 deployment bundles"
    PATH="${EARLIER_ROXCTL_PATH}:${PATH}" MAIN_IMAGE_TAG="${EARLIER_MAIN_IMAGE_TAG}" ROX_SCANNER_V4=false _deploy_stackrox

    _step "verify"

    verify_scannerV2_deployed
    verify_no_scannerV4_deployed
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "central"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "sensor"

    _step "upgrade-stackrox"

    info "Upgrading StackRox using HEAD deployment bundles"
    ROX_SCANNER_V4=true _deploy_stackrox

    _step "verify"

    verify_scannerV2_deployed
    verify_scannerV4_deployed
    verify_deployment_scannerV4_env_var_set "stackrox" "central"
    run ! verify_deployment_scannerV4_env_var_set "stackrox" "sensor" # no Scanner V4 support in Sensor with roxctl

    _end
}

get_central_endpoint() {
    local namespace="$1"
    local central_ip="$("${ORCH_CMD}" -n "$CUSTOM_CENTRAL_NAMESPACE" </dev/null get service central-loadbalancer \
        -o json | service_get_endpoint)"
    echo "${central_ip}:443"
}

get_cluster_name() {
    local prefix="${1:-sc}"
    echo "${prefix}-${RANDOM}"
}


verify_deployment_deletion() {
    local deployment_names_file="$1"; shift
    local namespace="$1"; shift
    local deployment_names="$@"

    echo "Waiting for the following deployments in namespace ${namespace} to be deleted: $deployment_names"

    echo "$deployment_names" | tr ' ' '\n' >> "$deployment_names_file"

    local deleted
    while [[ -s "$deployment_names_file" ]]; do
        active_deployments=$("${ORCH_CMD}" </dev/null -n "$namespace" get deployments -o json)
        deployment_names=$(cat "$deployment_names_file")

        for deployment_name in $deployment_names; do
            deleted=false
            if ! jq -e ".items[] | select (.metadata.name == \"$deployment_name\")" <<< "$active_deployments" > /dev/null 2>&1; then
                deleted=true
            elif jq -e ".items[] | select (.metadata.name == \"$deployment_name\") | .metadata.deletionTimestamp" > /dev/null 2>&1 <<<"$active_deployments"; then
                deleted=true
            fi
            if [[ "$deleted" == "true" ]]; then
                echo "Deployment ${namespace}/$deployment_name deleted."
                sed -ie "/^${deployment_name}$/d" "$deployment_names_file"
            fi
        done
        sleep 1
    done

    echo "All deployments deleted."
}
export -f verify_deployment_deletion

verify_deployment_deletion_with_timeout() {
    local timeout_duration="$1"; shift
    local deployment_names_file; deployment_names_file=$(mktemp)

    local ret=0
    timeout "$timeout_duration" bash -c "verify_deployment_deletion \"$deployment_names_file\" $*" || ret=$?
    rm -f "$deployment_names_file"

    case $ret in
    0)
        ;;
    124)
        echo "Waiting for deployment deletion of deployments timed out."
        return 1
        ;;
    125|126|127|137)
        echo "Waiting for deployment deletion failed with unexpected exit code $ret."
        return 1
        ;;
    *)
        echo "deployment deletion failed with exit code $ret."
        return 1
        ;;
    esac
}

verify_no_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    verify_no_scannerV4_indexer_deployed "$namespace"
    verify_no_scannerV4_matcher_deployed "$namespace"
}

verify_no_scannerV4_indexer_deployed() {
    local namespace=${1:-stackrox}
    echo "Verifying that scanner V4 indexer is not deployed"
    run "${ORCH_CMD}" </dev/null -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "scanner-v4-indexer"
}

verify_no_scannerV4_matcher_deployed() {
    local namespace=${1:-stackrox}
    echo "Verifying that scanner V4 matcher is not deployed"
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

verify_no_scannerV2_deployed() {
    local namespace=${1:-stackrox}
    echo "Verifying that scanner V2 is not deployed"
    run "${ORCH_CMD}" </dev/null -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "^(scanner|scanner-db)$"
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

    echo "Looking for ROX_SCANNER_V4 environment variable being set in ${namespace}/${deployment}."

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
    export_central_cert "${central_namespace}"
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
    if [[ "${CI:-}" != "true" ]]; then
        info "Creating namespace and image pull secrets..."
        "${ORCH_CMD}" </dev/null create namespace "$central_namespace" || true
        "${ROOT}/deploy/common/pull-secret.sh" stackrox quay.io | "${ORCH_CMD}" -n "$central_namespace" apply -f -
    fi
    deploy_central "${central_namespace}"
    patch_down_central "${central_namespace}"
}

# shellcheck disable=SC2120
# deploy_central_with_helm "<central namespace>" "<main image tag>" "<helm chart dir overwrite>" [ additional args for helm ... ]
deploy_central_with_helm() {
    echo "Deploying central-services via Helm..."
    local central_namespace="$1"; shift
    local main_image_tag="$1"; shift
    local helm_chart_dir="$HEAD_HELM_CHART_CENTRAL_SERVICES_DIR"
    local use_default_chart="true"
    if [[ -n "$1" ]]; then
        # overwrite Helm chart path.
        helm_chart_dir="$1"
        use_default_chart="false"
    fi; shift

    if [[ "${CI:-}" != "true" ]]; then
        info "Creating namespace and image pull secrets..."
        echo "{ \"apiVersion\": \"v1\", \"kind\": \"Namespace\", \"metadata\": { \"name\": \"$central_namespace\" } }" | "${ORCH_CMD}" apply -f -
        "${ROOT}/deploy/common/pull-secret.sh" stackrox quay.io | "${ORCH_CMD}" -n "$central_namespace" apply -f -
    fi

    local image_overwrites=""
    local base_helm_values=""
    local upgrade="false"

    if [[ "$use_default_chart" == "true" ]]; then
        image_overwrites=$(cat <<EOT
image:
  registry: "$DEFAULT_IMAGE_REGISTRY"
central:
  db:
    image:
      tag: "$main_image_tag"
  image:
    tag: "$main_image_tag"

scannerV4:
  image:
    tag: "$main_image_tag"
  db:
    image:
      tag: "$main_image_tag"
EOT
        )
    fi

    command=("install" "--create-namespace")
    if helm list -n "$central_namespace" -o json | jq -e '.[] | select(.name == "stackrox-central-services")' > /dev/null 2>&1; then
        helm_generated_values_file=$(mktemp)
        "${ORCH_CMD}" -n "$central_namespace" get secrets -o json \
            </dev/null \
            | jq -r '.items[] | select(.metadata.name | startswith("stackrox-generated-")) | .data["generated-values.yaml"] | @base64d' \
            > "$helm_generated_values_file"
        command=("upgrade" "--install" "--reuse-values" "-f" "$helm_generated_values_file")
        upgrade="true"
        apply_crd_ownership_for_upgrade "$central_namespace"
    else
        base_helm_values=$(cat <<EOT
central:
  resources:
    requests:
      cpu: 500m
      memory: 2Gi
    limits:
      cpu: 2000m
      memory: 4Gi
  telemetry:
    enabled: false
  exposure:
    loadBalancer:
      enabled: true
  db:
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
      limits:
        cpu: 2000m
        memory: 4Gi

scanner:
  resources:
    requests:
      cpu: "500m"
      memory: "500Mi"
    limits:
      cpu: "2000m"
      memory: "2500Mi"
  dbResources:
    requests:
      cpu: "400m"
      memory: "512Mi"
    limits:
      cpu: "2000m"
      memory: "4Gi"
  replicas: 1
  autoscaling:
    disable: true

scannerV4:
  indexer:
    replicas: 1
    autoscaling:
      disable: true
  matcher:
    replicas: 1
    autoscaling:
      disable: true

allowNonstandardNamespace: true
EOT
        )
    fi

    echo "Deploying stackrox-central-services Helm chart \"${helm_chart_dir}\" into namespace ${central_namespace} with the following settings:"
    if [[ -n "$base_helm_values" ]]; then
        echo "base Helm values:"
        echo "$base_helm_values" | sed -e 's/^/  |/;'
    fi
    if [[ -n "$image_overwrites" ]]; then
        echo "image overwrites:"
        echo "$image_overwrites" | sed -e 's/^/  |/;'
    fi
    echo "additional arguments:"
    echo "  | $*"

    helm -n "${central_namespace}" "${command[@]}" \
        -f <(echo "$image_overwrites") \
        -f <(echo "$base_helm_values") \
        "$@" \
        stackrox-central-services "${helm_chart_dir}"

    if [[ "$upgrade" == "true" ]]; then
        # TODO(ROX-28903): For some reason pods don't always terminate smoothly after an upgrade.
        bounce_pods "${central_namespace}"
    fi

    echo "Waiting for API..."
    wait_for_api "${central_namespace}"

    if [[ "$upgrade" == "false" ]]; then
        echo "Setting up client TLS certs..."
        setup_client_TLS_certs ""
        echo "Recording build info..."
        record_build_info "${central_namespace}"
    fi
}

# shellcheck disable=SC2120
# deploy_sensor_with_helm "<central namespace>" "<sensor namespace>"
#   "<main image tag>" "<helm chart dir overwrite>"
#   "<cluster name>" "<central admin password>" "<central endpoint>" [ additional args for helm ... ]
deploy_sensor_with_helm() {
    echo "Deploying secured-cluster-services via Helm..."
    local central_namespace="$1"; shift
    local sensor_namespace="$1"; shift
    local main_image_tag="${1:-$MAIN_IMAGE_TAG}"; shift
    local helm_chart_dir="$HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR"
    local use_default_chart="true"
    if [[ -n "$1" ]]; then
        # overwrite Helm chart path.
        helm_chart_dir="$1"
        use_default_chart="false"
    fi; shift
    local cluster_name="$1"; shift
    local central_password="$1"; shift
    local central_endpoint="$1"; shift

    local image_overwrites=""
    local base_helm_values=""
    local upgrade="false"

    if [[ "$use_default_chart" == "true" ]]; then
        image_overwrites=$(cat <<EOT
image:
  registry: "$DEFAULT_IMAGE_REGISTRY"
  main:
    tag: "$main_image_tag"
  scannerV4:
    tag: "$main_image_tag"
  scannerV4DB:
    tag: "$main_image_tag"

EOT
        )
    fi

    command=("install" "--create-namespace")
    if helm list -n "$sensor_namespace" -o json | jq -e '.[] | select(.name == "stackrox-secured-cluster-services")' > /dev/null 2>&1; then
        command=("upgrade" "--install" "--reuse-values")
        upgrade="true"
    else
        # Later this will be replaced by CRS.
        echo "Retrieving init-bundle from Central..."
        init_bundle="$("${ORCH_CMD}" </dev/null -n "$central_namespace" exec deploy/central -- roxctl \
            --insecure-skip-tls-verify \
            -p "$central_password" \
            -e "central.${central_namespace}.svc:443" \
            central init-bundles generate "$cluster_name" --output=-)"

        if [[ "${CI:-}" != "true" ]]; then
            info "Creating image pull secrets..."
            "${ORCH_CMD}" </dev/null create namespace "$sensor_namespace" || true
            "${ROOT}/deploy/common/pull-secret.sh" stackrox quay.io | "${ORCH_CMD}" -n "$sensor_namespace" apply -f -
            "${ROOT}/deploy/common/pull-secret.sh" collector-stackrox quay.io | "${ORCH_CMD}" -n "$sensor_namespace" apply -f -
        fi
        base_helm_values=$(cat <<EOT
clusterName: "$cluster_name"
centralEndpoint: "$central_endpoint"

scanner:
  resources:
    requests:
      cpu: "500m"
      memory: "500Mi"
    limits:
      cpu: "2000m"
      memory: "2500Mi"
  dbResources:
    requests:
      cpu: "400m"
      memory: "512Mi"
    limits:
      cpu: "2000m"
      memory: "4Gi"
  replicas: 1
  autoscaling:
    disable: true

scannerV4:
  indexer:
    replicas: 1
    autoscaling:
      disable: true
  matcher:
    replicas: 1
    autoscaling:
      disable: true
  db:
    persistence:
      none: true

admissionControl:
  replicas: 1

collector:
  nodeSelector:
    run-collector: "true"

allowNonstandardNamespace: true
EOT
        )
    fi

    echo "Deploying stackrox-secured-cluster-services Helm chart \"${helm_chart_dir}\" into namespace ${sensor_namespace} with the following settings:"
    if [[ -n "$base_helm_values" ]]; then
        echo "base Helm values:"
        echo "$base_helm_values" | sed -e 's/^/  |/;'
    fi
    if [[ -n "$image_overwrites" ]]; then
        echo "image overwrites:"
        echo "$image_overwrites" | sed -e 's/^/  |/;'
    fi
    echo "additional arguments:"
    echo "  | $*"

    helm -n "${sensor_namespace}" "${command[@]}" \
        -f <(echo "$image_overwrites") \
        -f <(echo "$base_helm_values") \
        -f <(echo "$init_bundle") \
        "$@" \
        stackrox-secured-cluster-services "${helm_chart_dir}"

    if [[ "$upgrade" == "true" ]]; then
        # TODO(ROX-28903): For some reason pods don't always terminate smoothly after an upgrade.
        bounce_pods "${sensor_namespace}"
    fi

    echo "Sensor deployed. Waiting for sensor to be up"
    sensor_wait "${sensor_namespace}"
    wait_for_collectors_to_be_operational "${sensor_namespace}"
}

bounce_pods() {
    local namespace="$1"
    echo "Bouncing all workload pods..."
    "${ORCH_CMD}" </dev/null -n "$namespace" delete pod --all --force --grace-period=0
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
    if [[ "${CI:-}" != "true" ]]; then
        info "Creating image pull secrets..."
        "${ORCH_CMD}" </dev/null create namespace "$sensor_namespace" || true
        "${ROOT}/deploy/common/pull-secret.sh" stackrox quay.io | "${ORCH_CMD}" -n "$sensor_namespace" apply -f -
        "${ROOT}/deploy/common/pull-secret.sh" collector-stackrox quay.io | "${ORCH_CMD}" -n "$sensor_namespace" apply -f -
    fi
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

post_process_output() {
    local label="$1"
    local default_severity="INFO"
    local severity
    local msg

    while IFS="" read -r line; do
        # We post-process raw log lines which are emitted directly, e.g. using 'echo', as well as enriched log lines which have been emitted using 'info'.
        # If a line matches the format emitted by 'info', we only take the actual message payload to not duplicate the informational prefix, which would produce
        # lines of the form
        #
        #   INFO: <timestamp>: [<label>] INFO: <timestamp>: <message text>
        #
        # Instead we want all lines to be normalized to a form:
        #
        #   INFO: <timestamp>: [<label>] <message text>
        #
        severity="$default_severity"

        if [[ "$line" =~ ^(INFO|ERROR):(\ [[:alpha:]]+\ [[:alpha:]]+\ [[:digit:]]+\ [[:digit:]]{2}:[[:digit:]]{2}:[[:digit:]]{2}\ [[:alpha:]]+\ [[:digit:]]+:)?\ (.*) ]]; then
            severity="${BASH_REMATCH[1]}"
            msg="[${label}] ${BASH_REMATCH[3]}"
        else
            msg="[${label}] ${line}"
        fi
        echo "$severity: $(date): $msg"
    done
}
export -f post_process_output

test_identifier_from_description() {
    local identifier="$1"
    # Substitute all whitespaces with underscores
    identifier="${identifier// /_}"
    echo "$identifier"
}
