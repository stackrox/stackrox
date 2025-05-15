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
export QA_DEPLOY_WAIT_INFO="/tmp/wait-for-kubectl-object"

# If `envsubst` is contained in a non-standard directory `env -i` won't be able to
# execute it, even though it can be located via `$PATH`, hence we retrieve the absolute path of
# `envsubst`` before passing it to `env`.
envsubst=$(command -v envsubst)

CHECK_POD_RESTARTS_TEST_NAME="Check unexpected pod restarts"
# Define map of all stackrox pod->containers and their log dump files.
declare -A POD_CONTAINERS_MAP
POD_CONTAINERS_MAP["pod: central - container: central"]="central-[A-Za-z0-9]+-[A-Za-z0-9]+-central-previous.log"
POD_CONTAINERS_MAP["pod: central-db - container: init-db"]="central-db-[A-Za-z0-9]+-[A-Za-z0-9]+-init-db-previous.log"
POD_CONTAINERS_MAP["pod: central-db - container: central-db"]="central-db-[A-Za-z0-9]+-[A-Za-z0-9]+-central-db-previous.log"
POD_CONTAINERS_MAP["pod: config-controller - container: manager"]="config-controller-[A-Za-z0-9]+-[A-Za-z0-9]+-manager-previous.log"
POD_CONTAINERS_MAP["pod: scanner - container: scanner"]="scanner-[A-Za-z0-9]+-[A-Za-z0-9]+-scanner-previous.log"
POD_CONTAINERS_MAP["pod: scanner-db - container: init-db"]="scanner-db-[A-Za-z0-9]+-[A-Za-z0-9]+-init-db-previous.log"
POD_CONTAINERS_MAP["pod: scanner-db - container: db"]="scanner-db-[A-Za-z0-9]+-[A-Za-z0-9]+-db-previous.log"
POD_CONTAINERS_MAP["pod: scanner-v4 - container: matcher"]="scanner-v4-[A-Za-z0-9]+-[A-Za-z0-9]+-matcher-previous.log"
POD_CONTAINERS_MAP["pod: scanner-v4 - container: indexer"]="scanner-v4-[A-Za-z0-9]+-[A-Za-z0-9]+-indexer-previous.log"
POD_CONTAINERS_MAP["pod: scanner-v4-db - container: init-db"]="scanner-v4-db-[A-Za-z0-9]+-[A-Za-z0-9]+-init-db-previous.log"
POD_CONTAINERS_MAP["pod: scanner-v4-db - container: db"]="scanner-v4-db-[A-Za-z0-9]+-[A-Za-z0-9]+-db-previous.log"
POD_CONTAINERS_MAP["pod: sensor - container: sensor"]="sensor-[A-Za-z0-9]+-[A-Za-z0-9]+-sensor-previous.log"
POD_CONTAINERS_MAP["pod: admission-control - container: admission-control"]="admission-control-[A-Za-z0-9]+-[A-Za-z0-9]+-admission-control-previous.log"
POD_CONTAINERS_MAP["pod: collector - container: collector"]="collector-[A-Za-z0-9]+-collector-previous.log"
POD_CONTAINERS_MAP["pod: collector - container: compliance"]="collector-[A-Za-z0-9]+-compliance-previous.log"
POD_CONTAINERS_MAP["pod: collector - container: node-inventory"]="collector-[A-Za-z0-9]+-node-inventory-previous.log"

# shellcheck disable=SC2120
deploy_stackrox() {
    local tls_client_certs=${1:-}
    local central_namespace=${2:-stackrox}
    local sensor_namespace=${3:-stackrox}

    info "About to deploy StackRox (Central + Sensor)."

    setup_podsecuritypolicies_config

    deploy_stackrox_operator

    deploy_central "${central_namespace}"

    export_central_basic_auth_creds
    wait_for_api "${central_namespace}"
    export_central_cert "${central_namespace}"

    setup_client_TLS_certs "${tls_client_certs}"
    record_build_info "${central_namespace}"

    deploy_sensor "${sensor_namespace}" "${central_namespace}"
    echo "Sensor deployed. Waiting for sensor to be up"
    sensor_wait "${sensor_namespace}"

    # Bounce collectors to avoid restarts on initial module pull
    kubectl -n "${sensor_namespace}" delete pod -l app=collector --grace-period=0

    sensor_wait "${sensor_namespace}"

    wait_for_collectors_to_be_operational "${sensor_namespace}"

    pause_stackrox_operator_reconcile "${central_namespace}" "${sensor_namespace}"

    if kubectl -n "${central_namespace}" get deployment scanner-v4-indexer >/dev/null 2>&1; then
        wait_for_scanner_V4 "${central_namespace}"
    fi

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
    ci_export SETUP_WORKLOAD_IDENTITIES "${SETUP_WORKLOAD_IDENTITIES:-false}"

    ci_export ROX_BASELINE_GENERATION_DURATION "${ROX_BASELINE_GENERATION_DURATION:-1m}"
    ci_export ROX_NETWORK_BASELINE_OBSERVATION_PERIOD "${ROX_NETWORK_BASELINE_OBSERVATION_PERIOD:-2m}"
    ci_export ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL "${ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL:-true}"
    ci_export ROX_VULN_MGMT_LEGACY_SNOOZE "${ROX_VULN_MGMT_LEGACY_SNOOZE:-true}"
    ci_export ROX_DECLARATIVE_CONFIGURATION "${ROX_DECLARATIVE_CONFIGURATION:-true}"
    ci_export ROX_COMPLIANCE_ENHANCEMENTS "${ROX_COMPLIANCE_ENHANCEMENTS:-true}"
    ci_export ROX_POLICY_CRITERIA_MODAL "${ROX_POLICY_CRITERIA_MODAL:-true}"
    ci_export ROX_TELEMETRY_STORAGE_KEY_V1 "DISABLED"
    ci_export ROX_AUTH_MACHINE_TO_MACHINE "${ROX_AUTH_MACHINE_TO_MACHINE:-true}"
    ci_export ROX_COMPLIANCE_REPORTING "${ROX_COMPLIANCE_REPORTING:-true}"
    ci_export ROX_REGISTRY_RESPONSE_TIMEOUT "${ROX_REGISTRY_RESPONSE_TIMEOUT:-90s}"
    ci_export ROX_REGISTRY_CLIENT_TIMEOUT "${ROX_REGISTRY_CLIENT_TIMEOUT:-120s}"
    ci_export ROX_SCAN_SCHEDULE_REPORT_JOBS "${ROX_SCAN_SCHEDULE_REPORT_JOBS:-true}"
    ci_export ROX_PLATFORM_COMPONENTS "${ROX_PLATFORM_COMPONENTS:-true}"
    ci_export ROX_CVE_ADVISORY_SEPARATION "${ROX_CVE_ADVISORY_SEPARATION:-true}"
    ci_export ROX_EPSS_SCORE "${ROX_EPSS_SCORE:-true}"
    ci_export ROX_SBOM_GENERATION "${ROX_SBOM_GENERATION:-true}"
    ci_export ROX_CLUSTERS_PAGE_MIGRATION_UI "${ROX_CLUSTERS_PAGE_MIGRATION_UI:-false}"
    ci_export ROX_EXTERNAL_IPS "${ROX_EXTERNAL_IPS:-true}"
    ci_export ROX_NETWORK_GRAPH_EXTERNAL_IPS "${ROX_NETWORK_GRAPH_EXTERNAL_IPS:-false}"
    ci_export ROX_FLATTEN_CVE_DATA "${ROX_FLATTEN_CVE_DATA:-false}"
    ci_export ROX_VULNERABILITY_ON_DEMAND_REPORTS "${ROX_VULNERABILITY_ON_DEMAND_REPORTS:-true}"
    ci_export ROX_CUSTOMIZABLE_PLATFORM_COMPONENTS "${ROX_CUSTOMIZABLE_PLATFORM_COMPONENTS:-true}"

    if is_in_PR_context && pr_has_label ci-fail-fast; then
        ci_export FAIL_FAST "true"
    fi

    if [[ "${CI_JOB_NAME:-}" =~ gke ]]; then
        # GKE uses this network for services. Consider it as a private subnet.
        ci_export ROX_NON_AGGREGATED_NETWORKS "${ROX_NON_AGGREGATED_NETWORKS:-34.118.224.0/20}"
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

        make -C operator kuttl deploy-via-olm \
          INDEX_IMG_BASE="brew.registry.redhat.io/rh-osbs/iib" \
          INDEX_IMG_TAG="$(< operator/midstream/iib.json jq -r --arg version "$ocp_version" '.iibs[$version]')" \
          INSTALL_CHANNEL="$(< operator/midstream/iib.json jq -r '.operator.channel')" \
          INSTALL_VERSION="v$(< operator/midstream/iib.json jq -r '.operator.version')"
    else
        info "Deploying ACS operator"
        make -C operator kuttl deploy-via-olm \
          ROX_PRODUCT_BRANDING=RHACS_BRANDING
    fi
}

deploy_central() {
    local central_namespace=${1:-stackrox}
    info "Deploying central to namespace ${central_namespace}"

    # If we're running a nightly build or race condition check, then set CGO_CHECKS=true so that central is
    # deployed with strict checks
    if [[ "${CI:-}" == "true" ]]; then
        if is_nightly_run || pr_has_label ci-race-tests || [[ "${CI_JOB_NAME:-}" =~ race-condition ]]; then
            ci_export CGO_CHECKS "true"
        fi

        if pr_has_label ci-race-tests || [[ "${CI_JOB_NAME:-}" =~ race-condition ]]; then
            ci_export IS_RACE_BUILD "true"
        fi
    fi

    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR}" == "true" ]]; then
        deploy_central_via_operator "${central_namespace}"
    else
        if [[ -z "${OUTPUT_FORMAT:-}" ]]; then
            if [[ "${CI:-}" == "true" ]] && pr_has_label ci-helm-deploy; then
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
    if ! kubectl get ns "${central_namespace}" >/dev/null 2>&1; then
        kubectl create ns "${central_namespace}"
    fi

    NAMESPACE="${central_namespace}" make -C operator stackrox-image-pull-secret

    ROX_ADMIN_PASSWORD="$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c12 || true)"
    centralAdminPasswordBase64="$(echo "$ROX_ADMIN_PASSWORD" | base64)"

    centralAdditionalCAIndented="$(sed 's,^,        ,' "${TRUSTED_CA_FILE:-/dev/null}")"
    if [[ -z $centralAdditionalCAIndented ]]; then
        disableSpecTLS="#"
    fi

    centralDefaultTlsSecretKeyBase64="$(base64 -w0 < "${ROX_DEFAULT_TLS_KEY_FILE}")"
    centralDefaultTlsSecretCertBase64="$(base64 -w0 < "${ROX_DEFAULT_TLS_CERT_FILE}")"

    central_exposure_loadBalancer_enabled="false"
    central_exposure_route_enabled="false"
    case "${LOAD_BALANCER}" in
    "lb") central_exposure_loadBalancer_enabled="true" ;;
    "route") central_exposure_route_enabled="true" ;;
    esac

    customize_envVars=""
    if [[ "${CGO_CHECKS:-}" == "true" ]]; then
        customize_envVars+=$'\n      - name: GOEXPERIMENT'
        customize_envVars+=$'\n        value: "cgocheck2"'
        customize_envVars+=$'\n      - name: MUTEX_WATCHDOG_TIMEOUT_SECS'
        customize_envVars+=$'\n        value: "15"'
    fi
    customize_envVars+=$'\n      - name: ROX_BASELINE_GENERATION_DURATION'
    customize_envVars+=$'\n        value: '"${ROX_BASELINE_GENERATION_DURATION}"
    customize_envVars+=$'\n      - name: ROX_DEVELOPMENT_BUILD'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_NETWORK_BASELINE_OBSERVATION_PERIOD'
    customize_envVars+=$'\n        value: '"${ROX_NETWORK_BASELINE_OBSERVATION_PERIOD}"
    customize_envVars+=$'\n      - name: ROX_PROCESSES_LISTENING_ON_PORT'
    customize_envVars+=$'\n        value: "'"${ROX_PROCESSES_LISTENING_ON_PORT:-true}"'"'
    customize_envVars+=$'\n      - name: ROX_TELEMETRY_STORAGE_KEY_V1'
    customize_envVars+=$'\n        value: "'"${ROX_TELEMETRY_STORAGE_KEY_V1:-DISABLED}"'"'
    customize_envVars+=$'\n      - name: ROX_RISK_REPROCESSING_INTERVAL'
    customize_envVars+=$'\n        value: "15s"'
    customize_envVars+=$'\n      - name: ROX_COMPLIANCE_ENHANCEMENTS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_AUTH_MACHINE_TO_MACHINE'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_COMPLIANCE_REPORTING'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_REGISTRY_RESPONSE_TIMEOUT'
    customize_envVars+=$'\n        value: '"${ROX_REGISTRY_RESPONSE_TIMEOUT:-90s}"
    customize_envVars+=$'\n      - name: ROX_REGISTRY_CLIENT_TIMEOUT'
    customize_envVars+=$'\n        value: '"${ROX_REGISTRY_CLIENT_TIMEOUT:-120s}"
    customize_envVars+=$'\n      - name: ROX_VULN_MGMT_LEGACY_SNOOZE'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_SCAN_SCHEDULE_REPORT_JOBS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_PLATFORM_COMPONENTS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_CVE_ADVISORY_SEPARATION'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_EPSS_SCORE'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_CLUSTERS_PAGE_MIGRATION_UI'
    customize_envVars+=$'\n        value: "false"'
    customize_envVars+=$'\n      - name: ROX_EXTERNAL_IPS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_NETWORK_GRAPH_EXTERNAL_IPS'
    customize_envVars+=$'\n        value: "false"'
    customize_envVars+=$'\n      - name: ROX_SBOM_GENERATION'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_FLATTEN_CVE_DATA'
    customize_envVars+=$'\n        value: "false"'
    customize_envVars+=$'\n      - name: ROX_VULNERABILITY_ON_DEMAND_REPORTS'
    customize_envVars+=$'\n        value: "true"'
    customize_envVars+=$'\n      - name: ROX_CUSTOMIZABLE_PLATFORM_COMPONENTS'
    customize_envVars+=$'\n        value: "true"'

    local scannerV4ScannerComponent="Default"
    case "${ROX_SCANNER_V4:-}" in
        true)  scannerV4ScannerComponent="Enabled"  ;;
        false) scannerV4ScannerComponent="Disabled" ;;
    esac

    CENTRAL_YAML_PATH="tests/e2e/yaml/central-cr.envsubst.yaml"
    # Different yaml for midstream images
    if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
        CENTRAL_YAML_PATH="tests/e2e/yaml/central-cr-midstream.envsubst.yaml"
    fi
    env - \
      centralAdminPasswordBase64="$centralAdminPasswordBase64" \
      disableSpecTLS="${disableSpecTLS:-}" \
      centralAdditionalCAIndented="$centralAdditionalCAIndented" \
      centralDefaultTlsSecretKeyBase64="$centralDefaultTlsSecretKeyBase64" \
      centralDefaultTlsSecretCertBase64="$centralDefaultTlsSecretCertBase64" \
      central_exposure_loadBalancer_enabled="$central_exposure_loadBalancer_enabled" \
      central_exposure_route_enabled="$central_exposure_route_enabled" \
      customize_envVars="$customize_envVars" \
      scannerV4ScannerComponent="$scannerV4ScannerComponent" \
    "${envsubst}" \
      < "${CENTRAL_YAML_PATH}" | kubectl apply -n "${central_namespace}" -f -

    wait_for_object_to_appear "${central_namespace}" deploy/central 300
}

# shellcheck disable=SC2120
deploy_sensor() {
    local sensor_namespace=${1:-stackrox}
    local central_namespace=${2:-stackrox}

    info "Deploying sensor into namespace ${sensor_namespace} (central is expected in namespace ${central_namespace})"

    ci_export ROX_AFTERGLOW_PERIOD "15"
    ci_export ROX_COLLECTOR_INTROSPECTION_ENABLE "true"

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
        ROX_CA_CERT_FILE="" # force sensor.sh to fetch the actual cert.
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
    local scanner_component_setting="Disabled"
    local central_endpoint="central.${central_namespace}.svc:443"

    info "Deploying sensor via operator into namespace ${sensor_namespace} (central is expected in namespace ${central_namespace})"
    if ! kubectl get ns "${sensor_namespace}" >/dev/null 2>&1; then
        kubectl create ns "${sensor_namespace}"
    fi

    NAMESPACE="${sensor_namespace}" make -C operator stackrox-image-pull-secret

    # shellcheck disable=SC2016
    echo "${ROX_ADMIN_PASSWORD}" | \
    kubectl -n "${central_namespace}" exec -i deploy/central -- bash -c \
    'ROX_ADMIN_PASSWORD=$(cat) roxctl central init-bundles generate my-test-bundle \
        --insecure-skip-tls-verify \
        --output-secrets -' \
    | kubectl -n "${sensor_namespace}" apply -f -

    if [[ -n "${COLLECTION_METHOD:-}" ]]; then
       echo "Overriding the product default collection method due to COLLECTION_METHOD variable: ${COLLECTION_METHOD}"
    else
       die "COLLECTION_METHOD not set"
    fi

    if [[ "${SENSOR_SCANNER_SUPPORT:-}" == "true" ]]; then
        scanner_component_setting="AutoSense"
    fi

    local secured_cluster_yaml_path="tests/e2e/yaml/secured-cluster-cr.envsubst.yaml"
    if [[ "${ROX_SCANNER_V4:-false}" == "true" ]]; then
        secured_cluster_yaml_path="tests/e2e/yaml/secured-cluster-cr-with-scanner-v4.envsubst.yaml"
    fi

    upper_case_collection_method="$(echo "$COLLECTION_METHOD" | tr '[:lower:]' '[:upper:]')"

    # forceCollection only has an impact when the collection method is EBPF
    # but upgrade tests can fail if forceCollection is used for 4.3 or older.
    if [[ "${upper_case_collection_method}" == "CORE_BPF" ]]; then
      sed -i.bak '/forceCollection/d' "${secured_cluster_yaml_path}"
    fi

    env - \
      collection_method="$upper_case_collection_method" \
      scanner_component_setting="$scanner_component_setting" \
      central_endpoint="$central_endpoint" \
    "${envsubst}" \
      < "${secured_cluster_yaml_path}" | kubectl apply -n "${sensor_namespace}" -f -

    wait_for_object_to_appear "${sensor_namespace}" deploy/sensor 300
    wait_for_object_to_appear "${sensor_namespace}" ds/collector 300

    collector_envs=()

    if [[ -n "${ROX_AFTERGLOW_PERIOD:-}" ]]; then
       collector_envs+=("ROX_AFTERGLOW_PERIOD=${ROX_AFTERGLOW_PERIOD}")
    fi

    if [[ -n "${ROX_COLLECTOR_INTROSPECTION_ENABLE:-}" ]]; then
       collector_envs+=("ROX_COLLECTOR_INTROSPECTION_ENABLE=${ROX_COLLECTOR_INTROSPECTION_ENABLE}")
    fi

    if [[ -n "${ROX_PROCESSES_LISTENING_ON_PORT:-}" ]]; then
       kubectl -n "${sensor_namespace}" set env deployment/sensor ROX_PROCESSES_LISTENING_ON_PORT="${ROX_PROCESSES_LISTENING_ON_PORT}"
       collector_envs+=("ROX_PROCESSES_LISTENING_ON_PORT=${ROX_PROCESSES_LISTENING_ON_PORT}")
    fi

    if [[ ${#collector_envs[@]} -gt 0 ]]; then
        kubectl -n "${sensor_namespace}" set env ds/collector "${collector_envs[@]}"
    fi
}

pause_stackrox_operator_reconcile() {
    if [[ "${DEPLOY_STACKROX_VIA_OPERATOR}" == "false" ]]; then
        return
    fi
    local central_namespace=${1:-stackrox}
    local sensor_namespace=${2:-stackrox}

    kubectl annotate -n "${central_namespace}" \
        centrals.platform.stackrox.io \
        stackrox-central-services \
        stackrox.io/pause-reconcile=true

    kubectl annotate -n "${sensor_namespace}" \
        securedclusters.platform.stackrox.io \
        stackrox-secured-cluster-services \
        stackrox.io/pause-reconcile=true
}

export_central_basic_auth_creds() {
    if [[ -n ${DEPLOY_DIR:-} && -f "${DEPLOY_DIR}/central-deploy/password" ]]; then
        info "Getting central basic auth creds from central-deploy/password"
        ROX_ADMIN_PASSWORD="$(cat "${DEPLOY_DIR}"/central-deploy/password)"
    elif [[ -n "${ROX_ADMIN_PASSWORD:-}" ]]; then
        info "Using existing ROX_ADMIN_PASSWORD env"
    else
        echo "Expected to find file ${DEPLOY_DIR}/central-deploy/password or ROX_ADMIN_PASSWORD env"
        exit 1
    fi

    ROX_USERNAME="admin"
    ci_export "ROX_USERNAME" "$ROX_USERNAME"
    ci_export "ROX_ADMIN_PASSWORD" "$ROX_ADMIN_PASSWORD"
}

export_central_cert() {
    # Export the internal central TLS certificate for roxctl to access central
    # through TLS-passthrough router by specifying the TLS server name.
    ci_export ROX_SERVER_NAME "central.${CENTRAL_NAMESPACE:-stackrox}"

    require_environment "API_ENDPOINT"
    require_environment "ROX_ADMIN_PASSWORD"

    local central_cert
    central_cert="$(mktemp -d)/central_cert.pem"
    info "Storing central certificate in ${central_cert} for ${API_ENDPOINT}"

    roxctl -e "$API_ENDPOINT" \
        central cert --insecure-skip-tls-verify 1>"$central_cert"

    ci_export ROX_CA_CERT_FILE "$central_cert"
    openssl x509 -in "${ROX_CA_CERT_FILE}" -subject -issuer -ext subjectAltName -noout
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
    require_environment "ROX_ADMIN_PASSWORD"
    require_environment "CLIENT_CA_PATH"

    export_central_cert
    roxctl -e "$API_ENDPOINT" \
        central userpki create test-userpki -r Analyst -c "$CLIENT_CA_PATH"
}

setup_generated_certs_for_test() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: setup_generated_certs_for_test <dir>"
    fi

    info "Setting up generated certs for test"

    local dir="$1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_ADMIN_PASSWORD"

    export_central_cert
    roxctl -e "$API_ENDPOINT" \
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

    if [[ "$COLLECTION_METHOD" == "NO_COLLECTION" ]]; then
        # With NO_COLLECTION, no collector containers are deployed
        # so no need to check for readiness
        return
    fi

    # Ensure collector DaemonSet state is stable
    kubectl rollout status daemonset collector --namespace "${sensor_namespace}" --timeout=5m --watch=true

    # Check each collector pod readiness.
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
                kubectl -n "${sensor_namespace}" logs -c collector "$pod" || true
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

    info "Checking port availability..."
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
            info "Port ${target_port} on ${API_HOSTNAME} is reachable."
            return
        fi
        sleep 1
    done
    die "Port ${target_port} on ${API_HOSTNAME} did not become reachable in time"
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

junit_report_pod_restarts() {
    local check_output="${1:-}"

    local previous_logs=()
    while IFS='' read -r line; do previous_logs+=("$line"); done < <(echo "${check_output}" | grep "copied to Artifacts" | cut -d" " -f1 | sort -u)

    local previous_log_regex=""
    declare -A found_previous_logs
    declare -A found_pod_container_keys
    for previous_log in "${previous_logs[@]}"
    do
        found_previous_logs["${previous_log}"]=""
        for map_key in "${!POD_CONTAINERS_MAP[@]}"
        do
            previous_log_regex="${POD_CONTAINERS_MAP[${map_key}]}"
            if [[ "${previous_log}" =~ $previous_log_regex ]]; then
                found_previous_logs["${previous_log}"]="${map_key}"
                found_pod_container_keys["${map_key}"]="found"
                break
            fi
        done
    done

    # (FAILURES - Fallback) Report failed, but not found pods in defined pod->container map.
    local crop_pod_name=""
    for previous_log in "${!found_previous_logs[@]}"
    do
        if [[ "${found_previous_logs[${previous_log}]}" != "" ]]; then
            continue
        fi

        crop_pod_name="$(echo "${previous_log}" | cut -d- -f1)"
        save_junit_failure "${CHECK_POD_RESTARTS_TEST_NAME}" "${crop_pod_name}" "${check_output}"
    done

    # (FAILURES) Report failed and found pods. We use improved test name matching.
    for map_key in "${!found_pod_container_keys[@]}"
    do
        save_junit_failure "${CHECK_POD_RESTARTS_TEST_NAME}" "${map_key}" "${check_output}"
    done

    # Report pods without restarts (SUCCESSES).
    local found_map_key=""
    for map_key in "${!POD_CONTAINERS_MAP[@]}"
    do
        found_map_key="${found_pod_container_keys[${map_key}]:-}"
        if [[ "${found_map_key}" == "found" ]]; then
            continue
        fi

        save_junit_success "${CHECK_POD_RESTARTS_TEST_NAME}" "${map_key}"
    done
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
            junit_report_pod_restarts "${check_out}"
            die "ERROR: Found at least one unexplained pod restart. ${check_out}"
        fi
        info "Restarts were considered benign"
        echo "${check_out}"
    else
        info "No pod restarts were found"
    fi

    junit_report_pod_restarts
}

check_for_errors_in_stackrox_logs() {
    info "Checking stackrox pod logs for errors"

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
        info "Checking app: ${app}, logs: ${logs}"
        # shellcheck disable=SC2086
        if [[ -n "${logs}" ]] && ! check_out="$(${LOGCHECK_SCRIPT} ${logs})"; then
            summary="$(summarize_check_output "${check_out}")"
            save_junit_failure "SuspiciousLog-${app}" "${summary}" "$check_out"
            failure_found="true"
            info "Found suspicious log in $app: $check_out"
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
    local centrals_supported=false
    local securedclusters_supported=false

    if [[ ${#namespaces[@]} == 0 ]]; then
        namespaces+=( "stackrox" )
    fi

    info "Tearing down StackRox resources for namespaces ${namespaces[*]}..."

    # Check API Server Capabilities.
    local k8s_api_resources
    k8s_api_resources=$(kubectl api-resources -o name)
    if echo "${k8s_api_resources}" | grep -q "^securitycontextconstraints\.security\.openshift\.io$"; then
        resource_types="${resource_types},SecurityContextConstraints"
    fi
    if echo "${k8s_api_resources}" | grep -q "^podsecuritypolicies\.policy$"; then
        psps_supported=true
        global_resource_types="${global_resource_types},psp"
    fi
    if echo "${k8s_api_resources}" | grep -q "^centrals\.platform\.stackrox\.io$"; then
        centrals_supported=true
    fi
    if echo "${k8s_api_resources}" | grep -q "^securedclusters\.platform\.stackrox\.io$"; then
        securedclusters_supported=true
    fi

    (
        # Delete StackRox CRs first to give the operator a chance to properly finish the resource cleanup.
        if [[ "${securedclusters_supported}" == "true" ]]; then
            # Remove stackrox.io/pause-reconcile annotation since it prevents
            # deletion of secured cluster in static clusters
            kubectl annotate -n stackrox \
            securedclusters.platform.stackrox.io \
            stackrox-secured-cluster-services \
            stackrox.io/pause-reconcile-

            kubectl get securedclusters -o name | while read -r securedcluster; do
                kubectl -n "${namespace}" delete --ignore-not-found --wait "${securedcluster}"
                # Wait until resources are actually deleted.
                kubectl wait -n "${namespace}"  --for=delete deployment/sensor --timeout=60s
            done
        fi
        if [[ "${centrals_supported}" == "true" ]]; then
            # Remove stackrox.io/pause-reconcile annotation since it prevents
            # deletion of central in static clusters
               kubectl annotate -n stackrox \
                centrals.platform.stackrox.io \
                stackrox-central-services \
                stackrox.io/pause-reconcile-

            kubectl get centrals -o name | while read -r central; do
                kubectl -n "${namespace}" delete --ignore-not-found --wait "${central}"
                kubectl wait -n "${namespace}"  --for=delete deployment/central --timeout=60s
            done
        fi
        if [[ "$psps_supported" = "true" ]]; then
            kubectl delete -R -f scripts/ci/psp --wait
        fi

        for namespace in "${namespaces[@]}"; do
            if kubectl get ns "$namespace" >/dev/null 2>&1; then
                kubectl -n "$namespace" delete "$resource_types" -l "app.kubernetes.io/name=stackrox" --wait
            fi
            kubectl delete --ignore-not-found ns "$namespace" --wait
        done

        kubectl delete "${global_resource_types}" -l "app.kubernetes.io/name=stackrox" --wait
        kubectl delete crd securitypolicies.config.stackrox.io --wait

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

        # midstream ocp specific
        if kubectl get ns stackrox-operator >/dev/null 2>&1; then
            kubectl -n stackrox-operator delete "$resource_types" -l "app=rhacs-operator" --wait
        fi
        kubectl delete --ignore-not-found ns stackrox-operator --wait
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


wait_for_ready_deployment() {
    local namespace="$1"
    local deployment_name="$2"
    local max_seconds="$3"

    info "Waiting for deployment ${deployment_name} to be ready in namespace ${namespace}"

    start_time="$(date '+%s')"
    while true; do
        deployment_json="$(kubectl -n "${namespace}" get "deploy/${deployment_name}" -o json)"
        replicas="$(jq '.status.replicas' <<<"$deployment_json")"
        ready_replicas="$(jq '.status.readyReplicas' <<<"$deployment_json")"
        curr_time="$(date '+%s')"
        elapsed_seconds=$(( curr_time - start_time ))

        # Ready case. First we need to make sure that "$replicas" is an integer and not
        # something like "null", which would cause an execution error while
        # evaluating [[ "$replicas" -gt 0 ]].
        if [[ "$replicas" =~ ^[0-9]+$ && "$replicas" -gt 0 && "$replicas" == "$ready_replicas" ]]; then
            sleep 10
            break
        fi

        # Timeout case
        if (( elapsed_seconds > max_seconds )); then
            kubectl -n "${namespace}" get pod -o wide
            kubectl -n "${namespace}" get deploy -o wide
            die "wait_for_ready_deployment() timeout after $max_seconds seconds."
        fi

        # Otherwise report and retry
        info "Still waiting (${elapsed_seconds}s/${max_seconds}s)..."
        sleep 5
    done

    info "Deployment ${deployment_name} is ready in namespace ${namespace}."
}

# shellcheck disable=SC2120
wait_for_scanner_V4() {
    local namespace="$1"
    local max_seconds=${MAX_WAIT_SECONDS:-300}
    info "Waiting for Scanner V4 to become ready..."
    if [[ "${ORCHESTRATOR_FLAVOR:-}" == "openshift" ]]; then
        # OCP Interop tests are run on minimal instances and will take longer
        # Allow override with MAX_WAIT_SECONDS
        max_seconds=${MAX_WAIT_SECONDS:-600}
        info "Waiting ${max_seconds}s (increased for openshift-ci provisioned clusters) for central api and $(( max_seconds * 6 )) for ingress..."
    fi

    wait_for_ready_deployment "$namespace" "scanner-v4-indexer" "$max_seconds"
    wait_for_ready_deployment "$namespace" "scanner-v4-matcher" "$max_seconds"
}

# shellcheck disable=SC2120
wait_for_api() {
    local central_namespace=${1:-stackrox}
    info "Waiting for Central to be ready in namespace ${central_namespace}"

    start_time="$(date '+%s')"
    max_seconds=${MAX_WAIT_SECONDS:-300}
    if [[ "${ORCHESTRATOR_FLAVOR:-}" == "openshift" ]]; then
        # OCP Interop tests are run on minimal instances and will take longer
        # Allow override with MAX_WAIT_SECONDS
        max_seconds=${MAX_WAIT_SECONDS:-600}
        info "Waiting ${max_seconds}s (increased for openshift-ci provisioned clusters) for central api and $(( max_seconds * 6 )) for ingress..."
    fi
    max_ingress_seconds=$(( max_seconds * 6 ))

    wait_for_ready_deployment "$central_namespace" "central" "$max_seconds"
    info "Central deployment is ready in namespace ${central_namespace}."
    info "Waiting for Central API endpoint"

    LOAD_BALANCER="${LOAD_BALANCER:-}"
    case "${LOAD_BALANCER}" in
        lb)
            get_ingress_endpoint "${central_namespace}" svc/central-loadbalancer '.status.loadBalancer.ingress[0] | .ip // .hostname' "${max_ingress_seconds}"
            API_HOSTNAME="${ingress_endpoint}"
            API_PORT=443
            ;;
        route)
            get_ingress_endpoint "${central_namespace}" routes/central '.spec.host' "${max_ingress_seconds}"
            API_HOSTNAME="${ingress_endpoint}"
            API_PORT=443
            ;;
        *)
            API_HOSTNAME=localhost
            API_PORT=8000
            ;;
    esac

    API_ENDPOINT="${API_HOSTNAME}:${API_PORT}"
    PING_URL="https://${API_ENDPOINT}/v1/ping"
    NUM_SUCCESSES_IN_A_ROW=0
    SUCCESSES_NEEDED_IN_A_ROW=6

    info "Attempting to get ${SUCCESSES_NEEDED_IN_A_ROW} 'ok' responses in a row from ${PING_URL}"

    set +e
    # shellcheck disable=SC2034
    for i in $(seq 1 150); do
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
        info "Curl exited with status ${pong_exitstatus} and returned '${pong}'."
        sleep 5
    done
    echo
    if [[ "${NUM_SUCCESSES_IN_A_ROW}" != "${SUCCESSES_NEEDED_IN_A_ROW}" ]]; then
        info "Failed to connect to Central in namespace ${central_namespace}. Saw at most ${NUM_SUCCESSES_IN_A_ROW} successes in a row."
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

get_ingress_endpoint() {
    local namespace="$1"
    local object="$2"
    local field_accessor="$3"
    local timeout="${4:-1800}"

    local cli_cmd="kubectl"
    if [[ "$object" =~ route ]]; then
        cli_cmd="oc"
    fi

    local start_time curr_time elapsed_seconds endpoint
    start_time="$(date '+%s')"

    while true; do
        endpoint=$("${cli_cmd}" -n "${namespace}" get "${object}" -o json | jq -r "${field_accessor}")
        if [[ -n "${endpoint}" ]] && [[ "${endpoint}" != "null" ]]; then
            info "Found ingress endpoint: ${endpoint}"
            ingress_endpoint="${endpoint}"
            return
        fi

        curr_time="$(date '+%s')"
        elapsed_seconds=$(( curr_time - start_time ))

        if (( elapsed_seconds > timeout )); then
            "${cli_cmd}" -n "${namespace}" get "${object}" -o json
            echo >&2 "get_ingress_endpoint() timeout after $timeout seconds."
            exit 1
        fi

        sleep 5
    done
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

    require_environment "ROX_ADMIN_PASSWORD"

    local build_info

    local metadata_url="https://${API_ENDPOINT}/v1/metadata"
    releaseBuild="$(curl -skS --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") "${metadata_url}" | jq -r '.releaseBuild')"

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

restore_4_1_postgres_backup() {
    info "Restoring a 4.1 postgres backup"

    require_environment "API_ENDPOINT"
    require_environment "ROX_ADMIN_PASSWORD"

    gsutil cp gs://stackrox-ci-upgrade-test-fixtures/upgrade-test-dbs/postgres_db_4_1.sql.zip .
    export_central_cert
    roxctl -e "$API_ENDPOINT" \
        central db restore --timeout 5m postgres_db_4_1.sql.zip
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
    require_environment "ROX_ADMIN_PASSWORD"

    # Ensure central is ready for requests after any previous tests
    wait_for_api "${central_namespace}"
    export_central_cert "${central_namespace}"

    info "Backing up to ${output_dir}"
    mkdir -p "$output_dir"
    # TODO(PR#15173): Temporarily reset the server name to fix CI:
    roxctl -s "" -e "${API_ENDPOINT}" central backup --output "$output_dir" || touch DB_TEST_FAIL

    info "Updating public config"
    update_public_config

    if [[ ! -e DB_TEST_FAIL ]]; then
        info "Restoring from ${output_dir}/postgres_db_*"
        roxctl -e "${API_ENDPOINT}" central db restore "$output_dir"/postgres_db_* || touch DB_TEST_FAIL
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
    info "Checking for progress events"

    local images_available=("Image_Availability" "Were the required images built successfully by GitHub Actions?")
    local stackrox_deployed=("Stackrox_Deployment" "Was Stackrox deployed to the cluster?")

    local check_deployment=false

    if [[ -f "${STATE_IMAGES_AVAILABLE}" ]]; then
        save_junit_success "${images_available[@]}"
        check_deployment=true
    else
        local build_results="build results are unknown"
        if [[ -f "${STATE_BUILD_RESULTS}" ]]; then
            build_results="$(cat "${STATE_BUILD_RESULTS}")"
        fi
        read -r -d '' build_details <<- _EO_DETAILS_ || true
Check the build workflow runs on GitHub:
${build_results}
_EO_DETAILS_
        save_junit_failure "${images_available[@]}" "${build_details}"
    fi

    case "$CI_JOB_NAME" in
    *gke-upgrade-tests)
        record_upgrade_test_progess
        ;;
    *operator-e2e-tests)
        check_deployment=false
        ;;
    *)
        info "No job specific progress markers are saved for: ${CI_JOB_NAME}"
        ;;
    esac

    if $check_deployment; then
        if [[ -f "${STATE_DEPLOYED}" ]]; then
            save_junit_success "${stackrox_deployed[@]}"
        else
            if [[ -f "${QA_DEPLOY_WAIT_INFO}" ]]; then
                save_junit_failure "${stackrox_deployed[0]}" "$(cat "${QA_DEPLOY_WAIT_INFO}")" "Check the build log"
            else
                save_junit_failure "${stackrox_deployed[@]}" "Check the build log"
            fi
        fi
    else
        save_junit_skipped "${stackrox_deployed[@]}"
    fi
}

record_upgrade_test_progess() {
    # Record the progress of the upgrade test. This order is tightly coupled to
    # the order of execution in .openshift-ci/ci_tests.py UpgradeTest and the
    # files listed below. This is essentially a check for the existence of state
    # tracking files that the upgrade test leaves in its wake as it progresses.

    # tests/upgrade/postgres_sensor_run.sh
    record_progress_step "${UPGRADE_PROGRESS_SENSOR_BUNDLE}" "${STATE_DEPLOYED}" \
        "postgres_sensor_run" "roxctl sensor bundle test"
    record_progress_step "${UPGRADE_PROGRESS_UPGRADER}" "${UPGRADE_PROGRESS_SENSOR_BUNDLE}" \
        "postgres_sensor_run" "bin/upgrader tests"

    # tests/upgrade/postgres_run.sh
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_PREP}" "${UPGRADE_PROGRESS_UPGRADER}" \
        "postgres_run" "Preparation for postgres testing"
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_EARLIER_CENTRAL}" "${UPGRADE_PROGRESS_POSTGRES_PREP}" \
        "postgres_run" "Deployed earlier postgres central"
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_BOUNCE}" "${UPGRADE_PROGRESS_POSTGRES_EARLIER_CENTRAL}" \
        "postgres_run" "Bounced central"
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_DB_BOUNCE}" "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_BOUNCE}" \
        "postgres_run" "Bounced central-db"
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_MIGRATIONS}" "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_DB_BOUNCE}" \
        "postgres_run" "Test migrations with an upgrade to current"
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_ROLLBACK}" "${UPGRADE_PROGRESS_POSTGRES_MIGRATIONS}" \
        "postgres_run" "Test rollback to earlier postgres"
    record_progress_step "${UPGRADE_PROGRESS_POSTGRES_SMOKE_TESTS}" "${UPGRADE_PROGRESS_POSTGRES_ROLLBACK}" \
        "postgres_run" "Smoke tests"
}

record_progress_step() {
    if [[ "$#" -ne 4 ]]; then
        die "Missing args. Usage: record_upgrade_test_progess " \
            "<this_step_file> <previous_step_file> <JUNIT Class> <JUNIT Description>"
    fi

    local this_step_file="$1"
    local previous_step_file="$2"
    local junit_class="$3"
    local junit_step_description="$4"

    if [[ -f "${previous_step_file}" ]] && [[ -f "${this_step_file}" ]]; then
        save_junit_success "${junit_class}" "${junit_step_description}"
    elif [[ -f "${previous_step_file}" ]]; then
        save_junit_failure "${junit_class}" "${junit_step_description}" "See build.log for error details."
    elif [[ -f "${this_step_file}" ]]; then
        die "ERROR: This step file exists but the previous step does not. " \
            "This indicates a change in the order of test execution that needs to be resolved " \
            "against the record steps. " \
            "this: ${this_step_file}, previous: ${previous_step_file}"
    else
        save_junit_skipped "${junit_class}" "${junit_step_description}"
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
            echo "Waiting for $object in ns $namespace timed out." > "${QA_DEPLOY_WAIT_INFO}" || true
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

# update_junit_prefix_with_central_and_sensor_version appends the central and sensor tags to all test
# names in the result folder. This propagates into our artifacts and JIRA tasks created from failing tests
# Used in gke-version-compatibility-tests and gke-nongroovy-compatibility-tests
update_junit_prefix_with_central_and_sensor_version() {
    local short_central_tag="$1"
    local short_sensor_tag="$2"
    local result_folder="$3"

    info "Updating all test in $result_folder to have \"Central-v${short_central_tag}_Sensor-v${short_sensor_tag}_\" prefix"
    for f in "$result_folder"/*.xml; do
        [[ ! -e $f ]] && continue
        sed -i "s/testcase name=\"/testcase name=\"[Central-v${short_central_tag}_Sensor-v${short_sensor_tag}] /g" "$f"
    done
}
