#!/usr/bin/env bash

# Compatibility test installation of ACS using CENTRAL_CHART_VERSION_OVERRIDE for central and SENSOR_CHART_VERSION_OVERRIDE for secured cluster

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../scripts/ci/create-webhookserver.sh
source "$ROOT/scripts/ci/create-webhookserver.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../qa-tests-backend/scripts/lib.sh
source "$ROOT/qa-tests-backend/scripts/lib.sh"

set -euo pipefail

compatibility_test() {
    require_environment "CENTRAL_CHART_VERSION_OVERRIDE"
    require_environment "SENSOR_CHART_VERSION_OVERRIDE"
    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    info "Starting test (compatibility test Central version - ${CENTRAL_CHART_VERSION_OVERRIDE}, Sensor version - ${SENSOR_CHART_VERSION_OVERRIDE})"

    export_test_environment

    if [[ "${SKIP_DEPLOY:-false}" = "false" ]]; then
        if [[ "${CI:-false}" = "true" ]]; then
            setup_gcp
        else
            info "Not running on CI: skipping cluster setup make sure cluster is already available"
        fi

        setup_deployment_env false false
        setup_podsecuritypolicies_config
        remove_existing_stackrox_resources
        setup_default_TLS_certs

        deploy_stackrox_with_custom_central_and_sensor_versions "${CENTRAL_CHART_VERSION_OVERRIDE}" "${SENSOR_CHART_VERSION_OVERRIDE}"
        echo "Stackrox deployed"
        kubectl -n stackrox get deploy,ds -o wide

        deploy_default_psp
        deploy_webhook_server
        get_ECR_docker_pull_password
    fi

    info "Running compatibility tests"

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        oc get scc qatest-anyuid || oc create -f "${ROOT}/qa-tests-backend/src/k8s/scc-qatest-anyuid.yaml"
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    make -C qa-tests-backend compatibility-test || touch FAIL

    update_junit_prefix_with_central_and_sensor_version

    store_qa_test_results "compatibility-test-central-v${CENTRAL_CHART_VERSION_OVERRIDE}-sensor-v${SENSOR_CHART_VERSION_OVERRIDE}"
    [[ ! -f FAIL ]] || die "compatibility-test-central-v${CENTRAL_CHART_VERSION_OVERRIDE}-sensor-v${SENSOR_CHART_VERSION_OVERRIDE}"
}

update_junit_prefix_with_central_and_sensor_version() {
    result_folder="${ROOT}/qa-tests-backend/build/test-results/testCOMPATIBILITY"
    info "Updating all test in $result_folder to have \"Central-v${CENTRAL_CHART_VERSION_OVERRIDE}_Sensor-v${SENSOR_CHART_VERSION_OVERRIDE}_\" prefix"
    for f in "$result_folder"/*.xml; do
        sed -i "s/testcase name=\"/testcase name=\"[Central-v${CENTRAL_CHART_VERSION_OVERRIDE}_Sensor-v${SENSOR_CHART_VERSION_OVERRIDE}] /g" "$f"
    done
}


compatibility_test
