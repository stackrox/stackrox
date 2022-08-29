#!/usr/bin/env bash

# Compatibility test installation of ACS using MAIN_IMAGE_TAG for central SENSOR_IMAGE_TAG for secured cluster

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/gcp.sh"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/tests/e2e/lib.sh"
source "$ROOT/tests/scripts/setup-certs.sh"
source "$ROOT/qa-tests-backend/scripts/lib.sh"

set -euo pipefail

compatibility_test() {
    info "Starting test (sensor compatibility test $SENSOR_IMAGE_TAG)"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export_test_environment

    if [[ "${SKIP_DEPLOY:-false}" = "false" ]]; then
        if [[ "${CI:-false}" = "true" ]]; then
            setup_gcp
        else
            info "Not running on CI: skipping cluster setup make sure cluster is already available"
        fi

        setup_deployment_env false false
        remove_existing_stackrox_resources
        setup_default_TLS_certs

        deploy_stackrox
        echo "Stackrox deployed"

        deploy_default_psp
        deploy_webhook_server
        get_ECR_docker_pull_password
    fi

    info "Running compatibility tests"

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        oc get scc qatest-anyuid || oc create -f "${ROOT}/qa-tests-backend/src/k8s/scc-qatest-anyuid.yaml"
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    # TODO(ROX-12320): Update the list of tests we want to run during "compatibility tests"
    make -C qa-tests-backend compatibility-test || touch FAIL

    store_qa_test_results "compatibility-test-sensor-$SENSOR_IMAGE_TAG"
    [[ ! -f FAIL ]] || die "compatibility-test-sensor-$SENSOR_IMAGE_TAG"

	run_compatibility_test
}


compatibility_test
