#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test utility functions

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$TEST_ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/lib.sh
source "$TEST_ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/test_state.sh
source "$TEST_ROOT/scripts/ci/test_state.sh"

export QA_TEST_DEBUG_LOGS="/tmp/qa-tests-backend-logs"

# shellcheck disable=SC2120
deploy_stackrox() {
    local tls_client_certs=${1:-}
    local central_namespace=${2:-stackrox}
    local sensor_namespace=${3:-stackrox}

    setup_podsecuritypolicies_config

    deploy_stackrox_operator

    deploy_central "${central_namespace}"

    export_central_basic_auth_creds
    wait_for_api "${central_namespace}"
    setup_client_TLS_certs "${tls_client_certs}"
    record_build_info "${central_namespace}"

    deploy_sensor "${sensor_namespace}" "${central_namespace}"
    echo "Sensor deployed. Waiting for sensor to be up"
    sensor_wait "${sensor_namespace}"

    # Bounce collectors to avoid restarts on initial module pull
    kubectl -n "${sensor_namespace}" delete pod -l app=collector --grace-period=0

    sensor_wait "${sensor_namespace}"

    wait_for_collectors_to_be_operational "${sensor_namespace}"

    touch "${STATE_DEPLOYED}"
}

# shellcheck disable=SC2120
deploy_stackrox_with_custom_central_and_sensor_versions() {
    if [[ "$#" -ne 2 ]]; then
        die "expected central chart version and sensor chart version as parameters in \
          deploy_stackrox_with_custom_central_and_sensor_versions: \
          deploy_stackrox_with_custom_central_and_sensor_versions <central chart version> <sensor chart version>"
    fi
    local central_version="$1"
    local sensor_version="$2"

    ci_export DEPLOY_STACKROX_VIA_OPERATOR "false"
    ci_export OUTPUT_FORMAT "helm"

    # Repo name can't be too long or `helm search repo [REPO_NAME] -l` cuts off part of the name and the regex below fails.
    helm_repo_name="tmp-srox-compat"
    helm repo add "${helm_repo_name}" https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource
    helm repo update

    current_tag="$(make tag --quiet --no-print-directory)"

    helm_charts="$(helm search repo "${helm_repo_name}" -l)"
    central_regex="${helm_repo_name}/stackrox-central-services[ \t]*.${central_version}[ \t]*.([0-9]+\.[0-9]+\.[0-9]+)"
    sensor_regex="${helm_repo_name}/stackrox-secured-cluster-services[ \t]*.${sensor_version}[ \t]*.([0-9]+\.[0-9]+\.[0-9]+)"

    charts_dir="$(mktemp -d ./charts-dir.XXXXXX)"

    # If the central version is the same as the current_tag, the default behavior of deploy_central() is correct for compatibility tests
    chart_name="stackrox-central-services"
    if  [[ $helm_charts =~ $central_regex ]]; then
        central_chart="${helm_repo_name}/${chart_name}"
        ci_export CENTRAL_CHART_DIR_OVERRIDE "${charts_dir}/${chart_name}"
        helm pull "${central_chart}" --version "${central_version}" --untar --untardir "${charts_dir}"
        echo "Pulled helm chart for ${chart_name} to ${CENTRAL_CHART_DIR_OVERRIDE}"
    elif [[ "$current_tag" != "${central_version}" ]]; then
        echo >&2 "${chart_name} helm chart for version ${central_version} not found in ${helm_repo_name} repo nor is it the current tag."
        exit 1
    fi

    # If the sensor version is the same as the current_tag the default behavior of deploy_sensor() is incorrect, because it will deploy
    # a sensor version to match the central version. In our tests we want to test current sensor vs older central too,
    # and since current sensor is not available in the repo either the chart is created here in the elif case.
    chart_name="stackrox-secured-cluster-services"
    if [[ $helm_charts =~ $sensor_regex ]]; then
        sensor_chart="${helm_repo_name}/${chart_name}"
        ci_export SENSOR_CHART_DIR_OVERRIDE "${charts_dir}/${chart_name}"
        helm pull "${sensor_chart}" --version "${sensor_version}" --untar --untardir "${charts_dir}"
        echo "Pulled helm chart for ${chart_name} to ${SENSOR_CHART_DIR_OVERRIDE}"
    elif [[ "$current_tag" == "${sensor_version}" ]]; then
        if [[ $(roxctl version) != "$current_tag" ]]; then
            echo >&2 "Reported roxctl version $(roxctl version) is different from requested tag ${current_tag}. It won't be possible to get helm charts for ${current_tag}. Please check test setup."
            exit 1
        fi
        ci_export SENSOR_CHART_DIR_OVERRIDE "${charts_dir}/${chart_name}"
        roxctl helm output secured-cluster-services --image-defaults=opensource --output-dir "${SENSOR_CHART_DIR_OVERRIDE}" --remove
        echo "Downloaded ${chart_name} helm chart for version ${sensor_version} to ${SENSOR_CHART_DIR_OVERRIDE}"
    else
        echo >&2 "${chart_name} helm chart for version ${sensor_version} not found in ${helm_repo_name} repo nor is it the latest tag."
        exit 1
    fi

    deploy_stackrox

    rm -rf "$charts_dir"

    helm repo remove "${helm_repo_name}"
    ci_export CENTRAL_CHART_DIR_OVERRIDE ""
    ci_export SENSOR_CHART_DIR_OVERRIDE ""
}

# export_test_environment() - Persist environment variables for the remainder of
# this context (context == whatever pod or VM this test is running in)
# Existing settings are maintained to allow override for different test flavors.
export_test_environment() {
    ci_export ADMISSION_CONTROLLER_UPDATES "${ADMISSION_CONTROLLER_UPDATES:-true}"
    ci_export ADMISSION_CONTROLLER "${ADMISSION_CONTROLLER:-true}"
    ci_export COLLECTION_METHOD "${COLLECTION_METHOD:-core_bpf}"
    ci_export DEPLOY_STACKROX_VIA_OPERATOR "${DEPLOY_STACKROX_VIA_OPERATOR:-false}"
    ci_export INSTALL_COMPLIANCE_OPERATOR "${INSTALL_COMPLIANCE_OPERATOR:-false}"
    ci_export LOAD_BALANCER "${LOAD_BALANCER:-lb}"
    ci_export LOCAL_PORT "${LOCAL_PORT:-443}"
    ci_export MONITORING_SUPPORT "${MONITORING_SUPPORT:-false}"
    ci_export SCANNER_SUPPORT "${SCANNER_SUPPORT:-true}"
    ci_export USE_MIDSTREAM_IMAGES "${USE_MIDSTREAM_IMAGES:-false}"
    ci_export REMOTE_CLUSTER_ARCH "${REMOTE_CLUSTER_ARCH:-x86_64}"

    ci_export ROX_BASELINE_GENERATION_DURATION "${ROX_BASELINE_GENERATION_DURATION:-1m}"
    ci_export ROX_NETWORK_BASELINE_OBSERVATION_PERIOD "${ROX_NETWORK_BASELINE_OBSERVATION_PERIOD:-2m}"
    ci_export ROX_QUAY_ROBOT_ACCOUNTS "${ROX_QUAY_ROBOT_ACCOUNTS:-true}"
    ci_export ROX_SYSLOG_EXTRA_FIELDS "${ROX_SYSLOG_EXTRA_FIELDS:-true}"
    ci_export ROX_VULN_MGMT_REPORTING_ENHANCEMENTS "${ROX_VULN_MGMT_REPORTING_ENHANCEMENTS:-true}"
    ci_export ROX_VULN_MGMT_WORKLOAD_CVES "${ROX_VULN_MGMT_WORKLOAD_CVES:-true}"
    ci_export ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL "${ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL:-true}"
    ci_export ROX_WORKLOAD_CVES_FIXABILITY_FILTERS "${ROX_WORKLOAD_CVES_FIXABILITY_FILTERS:-true}"
    ci_export ROX_SEND_NAMESPACE_LABELS_IN_SYSLOG "${ROX_SEND_NAMESPACE_LABELS_IN_SYSLOG:-true}"
    ci_export ROX_DECLARATIVE_CONFIGURATION "${ROX_DECLARATIVE_CONFIGURATION:-true}"
    ci_export ROX_MOVE_INIT_BUNDLES_UI "${ROX_MOVE_INIT_BUNDLES_UI:-true}"
    ci_export ROX_COMPLIANCE_ENHANCEMENTS "${ROX_COMPLIANCE_ENHANCEMENTS:-true}"
    ci_export ROX_ADMINISTRATION_EVENTS "${ROX_ADMINISTRATION_EVENTS:-true}"
    ci_export ROX_POLICY_CRITERIA_MODAL "${ROX_POLICY_CRITERIA_MODAL:-true}"
    ci_export ROX_TELEMETRY_STORAGE_KEY_V1 "DISABLED"
    ci_export ROX_SCANNER_V4_SUPPORT "${ROX_SCANNER_V4_SUPPORT:-true}"
    ci_export ROX_CLOUD_CREDENTIALS "${ROX_CLOUD_CREDENTIALS:-true}"
    ci_export ROX_SCANNER_V4 "${ROX_SCANNER_V4:-false}"
    ci_export ROX_CLOUD_SOURCES "${ROX_CLOUD_SOURCES:-true}"

    if is_in_PR_context && pr_has_label ci-fail-fast; then
        ci_export FAIL_FAST "true"
    fi
}

deploy_stackrox_operator() {
    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR}" == "false" ]]; then
        return
    fi

    export REGISTRY_PASSWORD="${QUAY_RHACS_ENG_RO_PASSWORD}"
    export REGISTRY_USERNAME="${QUAY_RHACS_ENG_RO_USERNAME}"

    if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
        info "Deploying ACS operator via midstream images"
        # Retrieving values from json map for operator and iib
        ocp_version=$(kubectl get clusterversion -o=jsonpath='{.items[0].status.desired.version}' | cut -d '.' -f 1,2)
        OPERATOR_VERSION=$(< operator/midstream/iib.json jq -r '.operator.version')
        VERSION=$(< operator/midstream/iib.json jq -r --arg version "$ocp_version" '.iibs[$version]')
        #Exporting the above vars
        export IMAGE_TAG_BASE="brew.registry.redhat.io/rh-osbs/iib"
        export OPERATOR_VERSION
        export VERSION

        make -C operator kuttl deploy-via-olm-midstream
    else
        info "Deploying ACS operator"
        ROX_PRODUCT_BRANDING=RHACS_BRANDING make -C operator kuttl deploy-via-olm
    fi
}

deploy_central() {
    local central_namespace=${1:-stackrox}
    info "Deploying central to namespace ${central_namespace}"

    # If we're running a nightly build or race condition check, then set CGO_CHECKS=true so that central is
    # deployed with strict checks
    if is_nightly_run || pr_has_label ci-race-tests || [[ "${CI_JOB_NAME:-}" =~ race-condition ]]; then
        ci_export CGO_CHECKS "true"
    fi

    if pr_has_label ci-race-tests || [[ "${CI_JOB_NAME:-}" =~ race-condition ]]; then
        ci_export IS_RACE_BUILD "true"
    fi

    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR}" == "true" ]]; then
        deploy_central_via_operator "${central_namespace}"
    else
        if [[ -z "${OUTPUT_FORMAT:-}" ]]; then
            if pr_has_label ci-helm-deploy; then
                ci_export OUTPUT_FORMAT helm
            fi
        fi

        DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"
        CENTRAL_NAMESPACE="${central_namespace}" "${ROOT}/${DEPLOY_DIR}/central.sh"
    fi
}

# shellcheck disable=SC2120
deploy_central_via_operator() {
    local central_namespace=${1:-stackrox}
    info "Deploying central via operator into namespace ${central_namespace}"

    make -C operator stackrox-image-pull-secret

    ROX_PASSWORD="$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c12 || true)"
    centralAdminPasswordBase64="$(echo "$ROX_PASSWORD" | base64)"

    centralDefaultTlsSecretKeyBase64="$(base64 -w0 < "${ROX_DEFAULT_TLS_KEY_FILE}")"
    centralDefaultTlsSecretCertBase64="$(base64 -w0 < "${ROX_DEFAULT_TLS_CERT_FILE}")"

    central_exposure_loadBalancer_enabled="false"
    central_exposure_route_enabled="false"
    if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
        # Load balancer not available for ppc64le/s390x
        LOAD_BALANCER="route"
    fi
    case "${LOAD_BALANCER}" in
    "lb") central_exposure_loadBalancer_enabled="true" ;;
    "route") central_exposure_route_enabled="true" ;;
    esac

    customize_envVars=""
    if [[ "${CGO_CHECKS:-}" == "true" ]]; then
        customize_envVars+=$'\n      - name: GODEBUG'
        customize_envVars+=$'\n        value: "2"'
        customize_envVars+=$'\n      - name: MUTEX_WATCHDOG_TIMEOUT_SECS'
        customize_envVars+=$'\n        value: "15"'
    fi
    customize_envVars+=$'\n      - name: ROX_BASELINE_GENERATION_DURATION'
    customize_envVars+=$'\n        value: '"${ROX_BASELINE_GENERATION_DURATION}"
    customize_envVars+=$'\n      - name: ROX_DEVELOPMENT_BUILD'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_NETWORK_BASELINE_OBSERVATION_PERIOD'
    customize_envVars+=$'\n        value: '"${ROX_NETWORK_BASELINE_OBSERVATION_PERIOD}"
    customize_envVars+=$'\n      - name: ROX_POSTGRES_DATASTORE'
    customize_envVars+=$'\n        value: "'"${ROX_POSTGRES_DATASTORE:-false}"'"'
    customize_envVars+=$'\n      - name: ROX_PROCESSES_LISTENING_ON_PORT'
    customize_envVars+=$'\n        value: "'"${ROX_PROCESSES_LISTENING_ON_PORT:-true}"'"'
    customize_envVars+=$'\n      - name: ROX_TELEMETRY_STORAGE_KEY_V1'
    customize_envVars+=$'\n        value: "'"${ROX_TELEMETRY_STORAGE_KEY_V1:-DISABLED}"'"'
    customize_envVars+=$'\n      - name: ROX_RISK_REPROCESSING_INTERVAL'
    customize_envVars+=$'\n        value: "15s"'
    customize_envVars+=$'\n      - name: ROX_COMPLIANCE_ENHANCEMENTS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_CLOUD_CREDENTIALS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_CLOUD_SOURCES'
    customize_envVars+=$'\n        value: "true"'

    CENTRAL_YAML_PATH="tests/e2e/yaml/central-cr.envsubst.yaml"
    # Different yaml for midstream images
    if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
        CENTRAL_YAML_PATH="tests/e2e/yaml/central-cr-midstream.envsubst.yaml"
    fi
    env - \
      centralAdminPasswordBase64="$centralAdminPasswordBase64" \
      centralDefaultTlsSecretKeyBase64="$centralDefaultTlsSecretKeyBase64" \
      centralDefaultTlsSecretCertBase64="$centralDefaultTlsSecretCertBase64" \
      central_exposure_loadBalancer_enabled="$central_exposure_loadBalancer_enabled" \
      central_exposure_route_enabled="$central_exposure_route_enabled" \
      customize_envVars="$customize_envVars" \
    envsubst \
      < "${CENTRAL_YAML_PATH}" | kubectl apply -n "${central_namespace}" -f -

    wait_for_object_to_appear "${central_namespace}" deploy/central 300
}

# shellcheck disable=SC2120
deploy_sensor() {
    local sensor_namespace=${1:-stackrox}
    local central_namespace=${2:-stackrox}

    info "Deploying sensor into namespace ${sensor_namespace} (central is expected in namespace ${central_namespace})"

    ci_export ROX_AFTERGLOW_PERIOD "15"

    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR}" == "true" ]]; then
        deploy_sensor_via_operator "${sensor_namespace}" "${central_namespace}"
    else
        if [[ "${OUTPUT_FORMAT:-}" == "helm" ]]; then
            echo "Deploying Sensor using Helm ..."
            ci_export SENSOR_HELM_DEPLOY "true"
            ci_export ADMISSION_CONTROLLER "true"
        else
            echo "Deploying sensor using kubectl ... "
            if [[ -n "${IS_RACE_BUILD:-}" ]]; then
                # builds with -race are slow at generating the sensor bundle
                # https://stack-rox.atlassian.net/browse/ROX-6987
                ci_export ROXCTL_TIMEOUT "60s"
            fi
        fi

        DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"
        CENTRAL_NAMESPACE="${central_namespace}" SENSOR_NAMESPACE="${sensor_namespace}" "${ROOT}/${DEPLOY_DIR}/sensor.sh"
    fi

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        # Sensor is CPU starved under OpenShift causing all manner of test failures:
        # https://stack-rox.atlassian.net/browse/ROX-5334
        # https://stack-rox.atlassian.net/browse/ROX-6891
        # et al.
        kubectl -n "${sensor_namespace}" set resources deploy/sensor -c sensor --requests 'cpu=2' --limits 'cpu=4'
    fi
}

# shellcheck disable=SC2120
deploy_sensor_via_operator() {
    local sensor_namespace=${1:-stackrox}
    local central_namespace=${2:-stackrox}
    info "Deploying sensor via operator into namespace ${sensor_namespace} (central is expected in namespace ${central_namespace})"

    kubectl -n "${central_namespace}" exec deploy/central -- \
    roxctl central init-bundles generate my-test-bundle \
        --insecure-skip-tls-verify \
        --password "$ROX_PASSWORD" \
        --output-secrets - \
    | kubectl -n "${sensor_namespace}" apply -f -

    if [[ -n "${COLLECTION_METHOD:-}" ]]; then
       echo "Overriding the product default collection method due to COLLECTION_METHOD variable: ${COLLECTION_METHOD}"
    else
       die "COLLECTION_METHOD not set"
    fi

    upper_case_collection_method="$(echo "$COLLECTION_METHOD" | tr '[:lower:]' '[:upper:]')"
    env - \
      collection_method="$upper_case_collection_method" \
    envsubst \
      < tests/e2e/yaml/secured-cluster-cr.envsubst.yaml | kubectl apply -n "${sensor_namespace}" -f -

    wait_for_object_to_appear "${sensor_namespace}" deploy/sensor 300
    wait_for_object_to_appear "${sensor_namespace}" ds/collector 300

    if [[ -n "${ROX_AFTERGLOW_PERIOD:-}" ]]; then
       kubectl -n "${sensor_namespace}" set env ds/collector ROX_AFTERGLOW_PERIOD="${ROX_AFTERGLOW_PERIOD}"
    fi

    if [[ -n "${ROX_PROCESSES_LISTENING_ON_PORT:-}" ]]; then
       kubectl -n "${sensor_namespace}" set env deployment/sensor ROX_PROCESSES_LISTENING_ON_PORT="${ROX_PROCESSES_LISTENING_ON_PORT}"
       kubectl -n "${sensor_namespace}" set env ds/collector ROX_PROCESSES_LISTENING_ON_PORT="${ROX_PROCESSES_LISTENING_ON_PORT}"
    fi
}

export_central_basic_auth_creds() {
    if [[ -f "${DEPLOY_DIR}/central-deploy/password" ]]; then
        info "Getting central basic auth creds from central-deploy/password"
        ROX_PASSWORD="$(cat "${DEPLOY_DIR}"/central-deploy/password)"
    elif [[ -n "${ROX_PASSWORD:-}" ]]; then
        info "Using existing ROX_PASSWORD env"
    else
        echo "Expected to find file ${DEPLOY_DIR}/central-deploy/password or ROX_PASSWORD env"
        exit 1
    fi

    ROX_USERNAME="admin"
    ci_export "ROX_USERNAME" "$ROX_USERNAME"
    ci_export "ROX_PASSWORD" "$ROX_PASSWORD"
}

deploy_optional_e2e_components() {
    info "Installing optional components used in E2E tests"

    if [[ "${INSTALL_COMPLIANCE_OPERATOR:-false}" == "true" ]]; then
        install_the_compliance_operator
    else
        info "Skipping the compliance operator install"
    fi
}

install_the_compliance_operator() {
    csv=$(oc get csv -n openshift-compliance -o json | jq ".items[] | select(.metadata.name | test(\"compliance-operator\")).metadata.name")
    if [[ $csv == "" ]]; then
        # Install from subscription, but point to the upstream images available
        # in https://github.com/complianceascode/compliance-operator/pkgs/container/compliance-operator.
        # Similar process as documented in https://docs.openshift.com/container-platform/latest/security/compliance_operator/compliance-operator-installation.html
        info "Installing the compliance operator"
        oc create -f "${ROOT}/tests/e2e/yaml/compliance-operator/namespace.yaml"
        oc create -f "${ROOT}/tests/e2e/yaml/compliance-operator/catalog-source.yaml"
        oc create -f "${ROOT}/tests/e2e/yaml/compliance-operator/operator-group.yaml"
        oc create -f "${ROOT}/tests/e2e/yaml/compliance-operator/subscription.yaml"
        oc create -f "${ROOT}/tests/e2e/yaml/compliance-operator/complianceRole.yaml"
        oc create -f "${ROOT}/tests/e2e/yaml/compliance-operator/complianceRoleBinding.yaml"
        wait_for_object_to_appear openshift-compliance deploy/compliance-operator
    else
        info "Reusing existing compliance operator deployment from $csv subscription"
    fi

    wait_for_profile_bundles_to_be_ready
    oc get csv -n openshift-compliance
}

setup_client_CA_auth_provider() {
    info "Set up client CA auth provider for endpoints_test.go"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"
    require_environment "CLIENT_CA_PATH"

    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        central userpki create test-userpki -r Analyst -c "$CLIENT_CA_PATH"
}

setup_generated_certs_for_test() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: setup_generated_certs_for_test <dir>"
    fi

    info "Setting up generated certs for test"

    local dir="$1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        sensor generate-certs remote --output-dir "$dir"
    [[ -f "$dir"/cluster-remote-tls.yaml ]]
    # Use the certs in future steps that will use client auth.
    # This will ensure that the certs are valid.
    sensor_tls_cert="$(kubectl create --dry-run=client -o json -f "$dir"/cluster-remote-tls.yaml | jq 'select(.metadata.name=="sensor-tls")')"
    for file in ca.pem sensor-cert.pem sensor-key.pem; do
        echo "${sensor_tls_cert}" | jq --arg filename "${file}" '.stringData[$filename]' -r > "$dir/${file}"
    done
}

setup_podsecuritypolicies_config() {
    info "Set POD_SECURITY_POLICIES variable based on kubernetes version"

    SUPPORTS_PSP=$(kubectl api-resources | grep "podsecuritypolicies" -c || true)
    if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
        ci_export "POD_SECURITY_POLICIES" "false"
        info "POD_SECURITY_POLICIES set to false"
    else
        ci_export "POD_SECURITY_POLICIES" "true"
        info "POD_SECURITY_POLICIES set to true"
    fi
}

# wait_for_collectors_to_be_operational() ensures that collector pods are able
# to load kernel objects and create network connections.
# shellcheck disable=SC2120
wait_for_collectors_to_be_operational() {
    local sensor_namespace=${1:-stackrox}
    info "Will wait for collectors to reach a ready state in namespace ${sensor_namespace}"

    local readiness_indicator="Successfully established GRPC stream for signals"
    local timeout=300
    local retry_interval=10

    local start_time
    start_time="$(date '+%s')"
    local all_ready="false"
    while [[ "$all_ready" == "false" ]]; do
        all_ready="true"
        for pod in $(kubectl -n "${sensor_namespace}" get pods -l app=collector -o json | jq -r '.items[].metadata.name'); do
            echo "Checking readiness of $pod"
            if kubectl -n "${sensor_namespace}" logs -c collector "${pod}" | grep "${readiness_indicator}" > /dev/null 2>&1; then
                echo "$pod is deemed ready"
            else
                info "$pod is not ready"
                kubectl -n "${sensor_namespace}" logs -c collector "$pod"
                all_ready="false"
                break
            fi
        done
        if (( $(date '+%s') - start_time > "$timeout" )); then
            echo "ERROR: Collector readiness check timed out after $timeout seconds"
            echo "Not all collector logs contain: $readiness_indicator"
            exit 1
        fi
        if [[ "$all_ready" == "false" ]]; then
            info "Found at least one unready collector pod, will check again in $retry_interval seconds"
            sleep "$retry_interval"
        fi
    done
}

# shellcheck disable=SC2120
patch_resources_for_test() {
    local central_namespace=${1:-stackrox}
    info "Patch the loadbalancer and netpol resources for endpoints test"

    require_environment "TEST_ROOT"
    require_environment "API_HOSTNAME"

    kubectl -n "${central_namespace}" patch svc central-loadbalancer --patch "$(cat "$TEST_ROOT"/tests/e2e/yaml/endpoints-test-lb-patch.yaml)"
    kubectl -n "${central_namespace}" apply -f "$TEST_ROOT/tests/e2e/yaml/endpoints-test-netpol.yaml"

    for target_port in 8080 8081 8082 8443 8444 8445 8446 8447 8448; do
        check_endpoint_availability "$target_port"
    done

    # Ensure the API is available as well after patching the load balancer.
    wait_for_api "$central_namespace"
}

check_endpoint_availability() {
    local target_port="$1"
    # shellcheck disable=SC2034
    for i in $(seq 1 20); do
        if echo "Endpoint check" 2>/dev/null > /dev/tcp/"${API_HOSTNAME}"/"${target_port}"; then
            return
        fi
        sleep 1
    done
    die "Port ${target_port} did not become reachable in time"
}

check_stackrox_logs() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: check_stackrox_logs <dir>"
    fi

    local dir="$1"

    if [[ ! -d "$dir/stackrox/pods" ]]; then
        die "StackRox logs were not collected. (Use ./scripts/ci/collect-service-logs.sh stackrox)"
    fi

    check_for_stackrox_OOMs "$dir"
    check_for_stackrox_restarts "$dir"
    check_for_errors_in_stackrox_logs "$dir"
}

check_for_stackrox_OOMs() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: check_for_stackrox_OOMs <dir>"
    fi

    local dir="$1"

    if [[ ! -d "$dir/stackrox/pods" ]]; then
        die "StackRox logs were not collected. (Use ./scripts/ci/collect-service-logs.sh stackrox)"
    fi

    local objects
    objects=$(ls "$dir"/stackrox/pods/*_object.json || true)
    if [[ -n "$objects" ]]; then
        for object in $objects; do
            local app_name
            # This wack jq slurp flag with the if statement is due to https://github.com/stedolan/jq/issues/1142
            if app_name=$(jq -ser 'if . == [] then null else .[] | select(.kind=="Pod") | .metadata.labels["app"] end' "$object"); then
                info "Checking $object for OOMKilled"
                if jq -e '. | select(.status.containerStatuses[].lastState.terminated.reason=="OOMKilled")' "$object" >/dev/null 2>&1; then
                    save_junit_failure "OOM Check" "Check for $app_name OOM kills" "A container of $app_name was OOM killed"
                else
                    save_junit_success "OOM Check" "Check for $app_name OOM kills"
                fi
            else
                echo "found $object that isn't a pod object"
            fi
        done
    fi
}

check_for_stackrox_restarts() {
    info "Checking for unexplained restarts by stackrox pods"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: check_for_stackrox_restarts <dir>"
    fi

    local dir="$1"

    if [[ ! -d "$dir/stackrox/pods" ]]; then
        die "StackRox logs were not collected. (Use ./scripts/ci/collect-service-logs.sh stackrox)"
    fi

    local previous_logs
    previous_logs=$(ls "$dir"/stackrox/pods/*-previous.log || true)
    if [[ -n "$previous_logs" ]]; then
        info "Pod restarts were found"
        local check_out=""
        # shellcheck disable=SC2086
        if ! check_out="$(scripts/ci/logcheck/check-restart-logs.sh "${CI_JOB_NAME}" $previous_logs)"; then
            pods=$(echo $check_out | grep "copied to Artifacts" | cut -d- -f1,3 | sort -u | tr '\n' ' ')
            save_junit_failure "Pod Restarts" "${pods}" "${check_out}"
            die "ERROR: Found at least one unexplained pod restart. ${check_out}"
        fi
        info "Restarts were considered benign"
        echo "${check_out}"
    else
        info "No pod restarts were found"
    fi

    save_junit_success "Pod Restarts" "Check for unexplained pod restart"
}

check_for_errors_in_stackrox_logs() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: check_for_errors_in_stackrox_logs <dir>"
    fi

    local dir="$1/stackrox/pods"

    if [[ ! -d "${dir}" ]]; then
        die "StackRox logs were not collected. (Use ./scripts/ci/collect-service-logs.sh stackrox)"
    fi

    local pod_objects=()
    _get_pod_objects "${dir}"
    _verify_item_count "${dir}" "${#pod_objects[*]}"

    declare -A podnames_by_app
    _group_pods_by_app_label

    # Check the logs for each app separately
    local app logs check_out summary
    local failure_found="false"
    LOGCHECK_SCRIPT="${LOGCHECK_SCRIPT:-scripts/ci/logcheck/check.sh}"
    for app in "${!podnames_by_app[@]}"; do
        logs="$(_get_logs_for_app "${app}")"
        # shellcheck disable=SC2086
        if [[ -n "${logs}" ]] && ! check_out="$(${LOGCHECK_SCRIPT} ${logs})"; then
            summary="$(summarize_check_output "${check_out}")"
            save_junit_failure "SuspiciousLog-${app}" "${summary}" "$check_out"
            failure_found="true"
        else
            save_junit_success "SuspiciousLog-${app}" "Suspicious entries in log file(s)"
        fi
    done
    if [[ "${failure_found}" == "true" ]]; then
        die "ERROR: Found at least one suspicious log file entry."
    fi
}

_get_pod_objects() {
    local dir="$1"

    local nullglob_setting
    nullglob_setting="$(shopt nullglob || true)"
    if [[ "${nullglob_setting}" =~ off ]]; then
        shopt -s nullglob
    fi

    local pod
    for pod in "${dir}"/*_object.json; do
        pod_objects[${#pod_objects[*]}]="${pod}"
    done

    if [[ "${nullglob_setting}" =~ off ]]; then
        shopt -u nullglob
    fi
}

_verify_item_count() {
    local dir="$1"
    local pod_object_count="$2"

    # ITEM_COUNT.txt is used to keep this function in sync with the file output
    # used by ./scripts/ci/collect-service-logs.sh

    if [[ ! -f "${dir}/ITEM_COUNT.txt" ]]; then
        die "ITEM_COUNT.txt is missing. (Check output from ./scripts/ci/collect-service-logs.sh"
    fi

    local item_count
    item_count="$(cat "${dir}/ITEM_COUNT.txt")"

    if [[ "${item_count}" != "${pod_object_count}" ]]; then
        die "The recorded number of items (${item_count}) differs from the objects found (${pod_object_count})"
    fi
}

_group_pods_by_app_label() {
    local pod_object app podname
    for pod_object in "${pod_objects[@]}"; do
        podname="$(jq -r '.metadata.name' < "${pod_object}")"
        if [[ -z "${podname}" || "${podname}" == "null" ]]; then
            die "ERROR: All pods should have a name! (check ${podname})"
        fi
        app="$(jq -r '.metadata.labels.app' < "${pod_object}")"
        if [[ -z "${app}" || "${app}" == "null" ]]; then
            info "Warning: All Stackrox pods should have an app label (check ${podname})"
            app="unknown"
        fi
        podnames_by_app[${app}]="${podnames_by_app[${app}]:-} ${podname}"
    done
}

_get_logs_for_app() {
    local app="$1"
    local logs=""
    for podname in ${podnames_by_app[${app}]}; do
        if this_logs="$(ls "${dir}/${podname}"*.log)"; then
            if [[ -z "${logs}" ]]; then
                logs="${this_logs}"
            else
                logs="${logs} ${this_logs}"
            fi
        fi
    done

    local filtered
    # shellcheck disable=SC2010,SC2086
    filtered=$(ls $logs | grep -Ev "(previous|_describe).log$" || true)

    echo "${filtered}"
}

summarize_check_output() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: summarize_check_output <output>"
    fi

    local MAX_SUMMARY_LENGTH=128
    local output="$1"

    output="$(
        echo "${output}" | \
        # The first line from check.sh is the first suspicious log found
        head -1 | \
        # Remove dates
        sed -r -e 's/[[:digit:]]{4}[/-][[:digit:]]{2}[/-][[:digit:]]{2}//g' | \
        # Remove time
        sed -r -e 's/[[:digit:]]{2}\:[[:digit:]]{2}\:[[:digit:]]{2}\.?[[:digit:]]*//g' | \
        # Replace images
        sed -r -e 's/(image ").*?"/\1__image__"/g' | \
        # Replace IDs
        sed -r -e 's/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/__ID__/g' | \
        # Replace pointers
        sed -r -e 's/0x[0-9a-f]+\??/_addr_/g' | \
        # Replace IPs + Ports
        sed -r -e 's/[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+\:?[[:digit:]]*/__ip[:port]__"/g' \
        || true
    )"

    if [[ "${#output}" -gt ${MAX_SUMMARY_LENGTH} ]]; then
        output="${output:0:${MAX_SUMMARY_LENGTH}}..."
    fi

    echo "${output}"
}

collect_and_check_stackrox_logs() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: collect_and_check_stackrox_logs <output-dir> <test_stage>"
    fi

    local dir="$1/$2"

    info "Will collect stackrox logs to $dir and check them"

    "$TEST_ROOT/scripts/ci/collect-service-logs.sh" stackrox "$dir"

    check_stackrox_logs "$dir"
}

# remove_existing_stackrox_resources() This exists for smoother repeat runs of
# system tests against the same cluster.
# shellcheck disable=SC2120
remove_existing_stackrox_resources() {
    info "Will remove any existing stackrox resources"
    local namespaces=( "$@" )
    local psps_supported=false
    local resource_types="cm,deploy,ds,rs,rc,networkpolicy,secret,svc,serviceaccount,pvc,role,rolebinding"
    local global_resource_types="pv,validatingwebhookconfigurations,clusterrole,clusterrolebinding"

    if [[ ${#namespaces[@]} == 0 ]]; then
        namespaces+=( "stackrox" )
    fi

    info "Tearing down StackRox resources for namespaces ${namespaces[*]}..."

    # Check API Server Capabilities.
    if kubectl api-resources -o name | grep -q "^securitycontextconstraints\.security\.openshift\.io$"; then
        resource_types="${resource_types},SecurityContextConstraints"
    fi
    if kubectl api-resources -o name | grep -q "^podsecuritypolicies\.policy$"; then
        psps_supported=true
        resource_types="${resource_types},psp"
    fi

    (
        if [[ "$psps_supported" = "true" ]]; then
            kubectl delete -R -f scripts/ci/psp --wait
        fi

        # midstream ocp specific
        if kubectl get ns stackrox-operator >/dev/null 2>&1; then
            kubectl -n stackrox-operator delete "$resource_types" -l "app=rhacs-operator" --wait
        fi
        kubectl delete --ignore-not-found ns stackrox-operator --wait

        for namespace in "${namespaces[@]}"; do
            if kubectl get ns "$namespace" >/dev/null 2>&1; then
                kubectl -n "$namespace" delete "$resource_types" -l "app.kubernetes.io/name=stackrox" --wait
            fi
            kubectl delete --ignore-not-found ns "$namespace" --wait
        done

        kubectl delete "${global_resource_types}" -l "app.kubernetes.io/name=stackrox" --wait

        helm list -o json | jq -r '.[] | .name' | while read -r name; do
            case "$name" in
                monitoring | central | scanner | sensor)
                    helm uninstall "$name"
                    ;;
            esac
        done

        kubectl get namespace -o name | grep -E '^namespace/qa' | while read -r namespace; do
            kubectl delete --wait "$namespace"
        done
    ) 2>&1 | sed -e 's/^/out: /' || true # (prefix output to avoid triggering prow log focus)
    info "Finished tearing down resources."
}

remove_compliance_operator_resources() {
    info "Will remove any existing compliance operator resources"
    if kubectl get crd compliancecheckresults.compliance.openshift.io; then
        (
            kubectl -n openshift-compliance delete ssb --all --wait --ignore-not-found=true
            # The profilebundles must be deleted before the csv. If not, the finalizers
            # will prevent the profilebundles from deleting because the CRDs are gone.
            kubectl -n openshift-compliance delete pb --all --wait --ignore-not-found=true
            kubectl -n openshift-compliance delete sub --all  --ignore-not-found=true
            kubectl -n openshift-compliance delete csv --all  --ignore-not-found=true
            kubectl -n openshift-compliance delete operatorgroup --all  --ignore-not-found=true
            kubectl -n openshift-marketplace delete catalogsource compliance-operator  --ignore-not-found=true
            kubectl delete namespace openshift-compliance --wait --ignore-not-found=true
        # (prefix output to avoid triggering prow log focus)
        ) 2>&1 | sed -e 's/^/out: /' || true
    fi
}

# shellcheck disable=SC2120
wait_for_api() {
    local central_namespace=${1:-stackrox}
    info "Waiting for Central to be ready in namespace ${central_namespace}"

    start_time="$(date '+%s')"
    max_seconds=${MAX_WAIT_SECONDS:-300}

    while true; do
        central_json="$(kubectl -n "${central_namespace}" get deploy/central -o json)"
        replicas="$(jq '.status.replicas' <<<"$central_json")"
        ready_replicas="$(jq '.status.readyReplicas' <<<"$central_json")"
        curr_time="$(date '+%s')"
        elapsed_seconds=$(( curr_time - start_time ))

        # Ready case
        if [[ "$replicas" == 1 && "$ready_replicas" == 1 ]]; then
            sleep 30
            break
        fi

        # Timeout case
        if (( elapsed_seconds > max_seconds )); then
            kubectl -n "${central_namespace}" get pod -o wide
            kubectl -n "${central_namespace}" get deploy -o wide
            echo >&2 "wait_for_api() timeout after $max_seconds seconds."
            exit 1
        fi

        # Otherwise report and retry
        echo "waiting ($elapsed_seconds/$max_seconds)"
        sleep 5
    done

    info "Central deployment is ready in namespace ${central_namespace}."
    info "Waiting for Central API endpoint"

    if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
        API_HOSTNAME=$(kubectl get routes/central -n "${central_namespace}" -o json | jq -r '.spec.host')
        API_PORT=443
    else
        API_HOSTNAME=localhost
        API_PORT=8000
        LOAD_BALANCER="${LOAD_BALANCER:-}"
        if [[ "${LOAD_BALANCER}" == "lb" ]]; then
            API_HOSTNAME=$(./scripts/k8s/get-lb-ip.sh "${central_namespace}")
            API_PORT=443
        fi
    fi
    API_ENDPOINT="${API_HOSTNAME}:${API_PORT}"
    PING_URL="https://${API_ENDPOINT}/v1/ping"
    info "PING_URL is set to ${PING_URL}"

    set +e
    NUM_SUCCESSES_IN_A_ROW=0
    SUCCESSES_NEEDED_IN_A_ROW=3
    # shellcheck disable=SC2034
    for i in $(seq 1 60); do
        pong="$(curl -sk --connect-timeout 5 --max-time 10 "${PING_URL}")"
        pong_exitstatus="$?"
        status="$(echo "$pong" | jq -r '.status')"
        if [[ "$pong_exitstatus" -eq "0" && "$status" == "ok" ]]; then
            NUM_SUCCESSES_IN_A_ROW=$((NUM_SUCCESSES_IN_A_ROW + 1))
            if [[ "${NUM_SUCCESSES_IN_A_ROW}" == "${SUCCESSES_NEEDED_IN_A_ROW}" ]]; then
                break
            fi
            info "Status is now: ${status}"
            sleep 2
            continue
        fi
        NUM_SUCCESSES_IN_A_ROW=0
        echo -n .
        sleep 5
    done
    echo
    if [[ "${NUM_SUCCESSES_IN_A_ROW}" != "${SUCCESSES_NEEDED_IN_A_ROW}" ]]; then
        info "Failed to connect to Central in namespace ${central_namespace}. Failed with ${NUM_SUCCESSES_IN_A_ROW} successes in a row"
        info "port-forwards:"
        pgrep port-forward
        info "pods:"
        kubectl -n "${central_namespace}" get pod
        exit 1
    fi
    set -e

    ci_export API_HOSTNAME "${API_HOSTNAME}"
    ci_export API_PORT "${API_PORT}"
    ci_export API_ENDPOINT "${API_ENDPOINT}"
}

record_build_info() {
    local central_namespace=${1:-stackrox}
    _record_build_info "${central_namespace}" || {
        # Failure to gather metrics is not a test failure
        info "WARNING: Job build info record failed"
    }
}

_record_build_info() {
    if ! is_CI; then
        return
    fi

    local central_namespace=${1:-stackrox}

    require_environment "ROX_PASSWORD"

    local build_info

    local metadata_url="https://${API_ENDPOINT}/v1/metadata"
    releaseBuild="$(curl -skS -u "admin:${ROX_PASSWORD}" "${metadata_url}" | jq -r '.releaseBuild')"

    if [[ "$releaseBuild" == "true" ]]; then
        build_info="release"
    else
        build_info="dev"
    fi

    # -race debug builds - use the image tag as the most reliable way to
    # determine the build under test.
    local central_image
    central_image="$(kubectl -n "${central_namespace}" get deploy central -o json | jq -r '.spec.template.spec.containers[0].image')"
    if [[ "${central_image}" =~ -rcd$ ]]; then
        build_info="${build_info},-race"
    fi

    update_job_record "build" "${build_info}"
}

restore_56_1_backup() {
    info "Restoring a 56.1 backup"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    gsutil cp gs://stackrox-ci-upgrade-test-fixtures/upgrade-test-dbs/stackrox_56_1_fixed_upgrade.zip .
    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        central db restore --timeout 2m stackrox_56_1_fixed_upgrade.zip
}

update_public_config() {
    info "Updating public config to ensure that it is overridden by restore"

    roxcurl /v1/config | jq . > ORIGINAL_CONFIG
    new_config=$(jq '. + { publicConfig: { header: { enabled: true, text: "hello" } } }' < ORIGINAL_CONFIG)
    roxcurl /v1/config -X PUT -d "{ \"config\": $new_config }" > /dev/null || touch DB_TEST_FAIL
}

db_backup_and_restore_test() {
    local output_dir="$1"
    local central_namespace=${2:-stackrox}

    info "Running a central database backup and restore test (central is expected in namespace ${central_namespace})"

    if [[ "$#" -ne 1 ]]; then
        die "Missing args. Usage: db_backup_and_restore_test <output dir> [ <namespace> ]"
    fi

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    # Ensure central is ready for requests after any previous tests
    wait_for_api "${central_namespace}"

    info "Backing up to ${output_dir}"
    mkdir -p "$output_dir"
    roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central backup --output "$output_dir" || touch DB_TEST_FAIL

    info "Updating public config"
    update_public_config

    if [[ ! -e DB_TEST_FAIL ]]; then
        if [ "${ROX_POSTGRES_DATASTORE:-}" == "true" ]; then
            info "Restoring from ${output_dir}/postgres_db_*"
            roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central db restore "$output_dir"/postgres_db_* || touch DB_TEST_FAIL
        else
            info "Restoring from ${output_dir}/stackrox_db_*"
            roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central db restore "$output_dir"/stackrox_db_* || touch DB_TEST_FAIL
        fi
    fi

    wait_for_api "${central_namespace}"

    info "Checking to see if restore overwrote previous config"

    roxcurl /v1/config | jq . > POST_RESTORE_CONFIG
    if [[ "$(cat ORIGINAL_CONFIG)" != "$(cat POST_RESTORE_CONFIG)" ]]; then
        info "config prior to backup is different from config after restore"
        diff ORIGINAL_CONFIG POST_RESTORE_CONFIG
        touch DB_TEST_FAIL
    fi

    [[ ! -f DB_TEST_FAIL ]] || die "The DB test failed"
}

handle_e2e_progress_failures() {
    info "Checking for deployment failure"

    local images_available=("Image_Availability" "Are the required images are available?")
    local stackrox_deployed=("Stackrox_Deployment" "Was Stackrox was deployed to the cluster?")

    local check_images=false
    local check_deployment=false

    if $check_images; then
        if [[ -f "${STATE_IMAGES_AVAILABLE}" ]]; then
            save_junit_success "${images_available[@]}" || true
            check_deployment=true
        else
            save_junit_failure "${images_available[@]}" \
                "Did the images build OK? If yes then the poll_for_system_test_images() timeout might need to be increased."
        fi
    fi

    if $check_deployment; then
        if [[ -f "${STATE_DEPLOYED}" ]]; then
            save_junit_success "${stackrox_deployed[@]}" || true
        else
            save_junit_failure "${stackrox_deployed[@]}" "Check the build log" || true
        fi
    fi
}

setup_automation_flavor_e2e_cluster() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: setup_automation_flavor_e2e_cluster <job_name>"
    fi

    local ci_job="$1"

    echo "SHARED_DIR: ${SHARED_DIR}"
    ls -l "${SHARED_DIR}"
    export KUBECONFIG="${SHARED_DIR}/kubeconfig"

    if [[ "$ci_job" =~ ^osd ]]; then
        info "Logging in to an OSD cluster"
        source "${SHARED_DIR}/dotenv"
        oc login "$CLUSTER_API_ENDPOINT" \
                --username "$CLUSTER_USERNAME" \
                --password "$CLUSTER_PASSWORD" \
                --insecure-skip-tls-verify=true
    fi
}

# When working as expected it takes less than one minute for the API server to
# reach ready. Often times out on OSD. If this call fails in CI we need to
# identify the source of pull/scheduling latency, request throttling, etc.
# I tried increasing the timeout from 5m to 20m for OSD but it did not help.
# shellcheck disable=SC2120
wait_for_central_db() {
    local central_namespace=${1:-stackrox}
    info "Waiting for Central DB to start in namespace ${central_namespace}"

    start_time="$(date '+%s')"
    max_seconds=300

    while true; do
        central_db_json="$(kubectl -n "${central_namespace}" get deploy/central-db -o json)"
        replicas="$(jq '.status.replicas' <<<"$central_db_json")"
        ready_replicas="$(jq '.status.readyReplicas' <<<"$central_db_json")"
        curr_time="$(date '+%s')"
        elapsed_seconds=$(( curr_time - start_time ))

        # Ready case
        if [[ "$replicas" == 1 && "$ready_replicas" == 1 ]]; then
            sleep 30
            break
        fi

        # Timeout case
        if (( elapsed_seconds > max_seconds )); then
            kubectl -n "${central_namespace}" get pod -o wide
            kubectl -n "${central_namespace}" get deploy -o wide
            echo >&2 "wait_for_central_db() timeout after $max_seconds seconds."
            exit 1
        fi

        # Otherwise report and retry
        echo "waiting ($elapsed_seconds/$max_seconds)"
        sleep 5
    done

    info "Central DB deployment in namespace ${central_namespace} is ready."
}

wait_for_object_to_appear() {
    if [[ "$#" -lt 2 ]]; then
        die "missing args. usage: wait_for_object_to_appear <namespace> <object> [<delay>]"
    fi

    local namespace="$1"
    local object="$2"
    local delay="${3:-300}"
    local waitInterval=20
    local tries=$(( delay / waitInterval ))
    local count=0
    until kubectl -n "$namespace" get "$object" > /dev/null 2>&1; do
        count=$((count + 1))
        if [[ $count -ge "$tries" ]]; then
            info "$namespace $object did not appear after $count tries"
            kubectl -n "$namespace" get "$object"
            return 1
        fi
        info "Waiting for $namespace $object to appear"
        sleep "$waitInterval"
    done

    return 0
}

wait_for_profile_bundles_to_be_ready() {
    wait_for_object_to_appear openshift-compliance profilebundle/ocp4
    wait_for_object_to_appear openshift-compliance profilebundle/rhcos4
    for pb in $(oc get pb -n openshift-compliance -o jsonpath="{.items[*].metadata.name}"); do
        local delay="300"
        local waitInterval=10
        local tries=$(( delay / waitInterval ))
        local count=0
        until [ "$(oc get pb "$pb" -n openshift-compliance -o jsonpath="{.status.dataStreamStatus}")" = "VALID" ]; do
            count=$((count + 1))
            if [[ $count -ge "$tries" ]]; then
                info "Failed to validate $pb profilebundle after $count tries"
                oc get pb "$pb" -n openshift-compliance
                return 1
            fi
            info "Validating $pb profilebundle"
            sleep "$waitInterval"
        done
        info "Validated $pb profilebundle"
    done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        usage
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
