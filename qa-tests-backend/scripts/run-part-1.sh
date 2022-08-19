#!/usr/bin/env bash

# Tests part I of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/gcp.sh"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/tests/e2e/lib.sh"
source "$ROOT/tests/scripts/setup-certs.sh"
source "$ROOT/qa-tests-backend/scripts/lib.sh"

set -euo pipefail

test_part_1() {
    info "Starting test (qa-tests-backend part I)"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox

    deploy_default_psp
    deploy_webhook_server
    get_ECR_docker_pull_password

    run_tests_part_1
}

run_tests_part_1() {
    info "QA Automation Platform Part 1"
    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        oc get scc qatest-anyuid || oc create -f "${ROOT}/qa-tests-backend/src/k8s/scc-qatest-anyuid.yaml"
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    info "Running SAC tests 100 times, exiting on the first failure found"

    make -C qa-tests-backend sac-test || touch FAIL

    store_qa_test_results "part-1-tests"
    [[ ! -f FAIL ]] || die "SAC test failed"
    store_qa_test_results "part-1-tests"
}

test_part_1
