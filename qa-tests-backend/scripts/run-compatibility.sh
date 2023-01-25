#!/usr/bin/env bash

# Compatibility test installation of ACS using MAIN_IMAGE_TAG for central SENSOR_CHART_VERSION for secured cluster

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/gcp.sh"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/scripts/ci/create-webhookserver.sh"
source "$ROOT/tests/e2e/lib.sh"
source "$ROOT/tests/scripts/setup-certs.sh"
source "$ROOT/qa-tests-backend/scripts/lib.sh"

set -euo pipefail

compatibility_test() {
    require_environment "SENSOR_CHART_VERSION"
    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    info "Starting test (sensor compatibility test ${SENSOR_CHART_VERSION})"

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

        deploy_stackrox_with_custom_sensor "${SENSOR_CHART_VERSION}"
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

    update_junit_prefix_with_sensor_version

    store_qa_test_results "compatibility-test-sensor-${SENSOR_CHART_VERSION}"
    [[ ! -f FAIL ]] || die "compatibility-test-sensor-${SENSOR_CHART_VERSION}"
}

update_junit_prefix_with_sensor_version() {
    result_folder="${ROOT}/qa-tests-backend/build/test-results/testCOMPATIBILITY"
    info "Updating all test in $result_folder to have $SENSOR_CHART_VERSION prefix"
    for f in "$result_folder"/*.xml; do
        sed -i "s/testcase name=\"/testcase name=\"[${SENSOR_CHART_VERSION}] /g" "$f"
    done
}


compatibility_test
