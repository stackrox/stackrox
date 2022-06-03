#!/usr/bin/env bash

# Tests part I of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/tests/e2e/lib.sh"
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

test_part_1() {
    info "Starting test (qa-tests-backend part I)"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    export_test_environment

    if is_OPENSHIFT_CI; then
        # TODO(RS-494) may provide roxctl
        make cli-linux
        install_built_roxctl_in_gopath
    fi

    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_central

    get_central_basic_auth_creds
    wait_for_api
    setup_client_TLS_certs

    deploy_sensor
    sensor_wait

    # Bounce collectors to avoid restarts on initial module pull
    kubectl -n stackrox delete pod -l app=collector --grace-period=0

    sensor_wait

    deploy_default_psp
    deploy_webhook_server
    deploy_authz_plugin
    get_ECR_docker_pull_password

    run_tests_part_1
}

# export_test_environment() - Persist environment variables for the remainder of
# this context (context == whatever pod or VM this test is running in)
# Existing settings are maintained to allow override for different test flavors.
export_test_environment() {
    ci_export ADMISSION_CONTROLLER_UPDATES "${ADMISSION_CONTROLLER_UPDATES:-true}"
    ci_export ADMISSION_CONTROLLER "${ADMISSION_CONTROLLER:-true}"
    ci_export COLLECTION_METHOD "${COLLECTION_METHOD:-ebpf}"
    ci_export GCP_IMAGE_TYPE "${GCP_IMAGE_TYPE:-COS}"
    ci_export LOAD_BALANCER "${LOAD_BALANCER:-none}"
    ci_export LOCAL_PORT "${LOCAL_PORT:-443}"
    ci_export MONITORING_SUPPORT "${MONITORING_SUPPORT:-false}"
    ci_export ROX_BASELINE_GENERATION_DURATION "${ROX_BASELINE_GENERATION_DURATION:-1m}"
    ci_export ROX_NETWORK_BASELINE_OBSERVATION_PERIOD "${ROX_NETWORK_BASELINE_OBSERVATION_PERIOD:-2m}"
    ci_export ROX_NEW_POLICY_CATEGORIES "${ROX_NEW_POLICY_CATEGORIES:-true}"
    ci_export SCANNER_SUPPORT "${SCANNER_SUPPORT:-true}"
}

deploy_central() {
    info "Deploying central"

    # If we're running a nightly build or race condition check, then set CGO_CHECKS=true so that central is
    # deployed with strict checks
    if is_nightly_tag || pr_has_label ci-race-tests; then
        ci_export CGO_CHECKS "true"
    fi

    if pr_has_label ci-race-tests; then
        ci_export IS_RACE_BUILD "true"
    fi

    if [[ -z "${OUTPUT_FORMAT:-}" ]]; then
        if pr_has_label ci-helm-deploy; then
            ci_export OUTPUT_FORMAT helm
        fi
    fi

    DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"
    "$ROOT/${DEPLOY_DIR}/central.sh"
}

deploy_sensor() {
    info "Deploying sensor"

    ci_export ROX_AFTERGLOW_PERIOD "15"
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
    "$ROOT/${DEPLOY_DIR}/sensor.sh"

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        # Sensor is CPU starved under OpenShift causing all manner of test failures:
        # https://stack-rox.atlassian.net/browse/ROX-5334
        # https://stack-rox.atlassian.net/browse/ROX-6891
        # et al.
        kubectl -n stackrox set resources deploy/sensor -c sensor --requests 'cpu=2' --limits 'cpu=4'
    fi
}

deploy_default_psp() {
    info "Deploy Default PSP for stackrox namespace"
    "${ROOT}/scripts/ci/create-default-psp.sh"
}

deploy_webhook_server() {
    info "Deploy Webhook server"

    local certs_dir
    certs_dir="$(mktemp -d)"
    "${ROOT}/scripts/ci/create-webhookserver.sh" "${certs_dir}"
    ci_export GENERIC_WEBHOOK_SERVER_CA_CONTENTS "$(cat "${certs_dir}/ca.crt")"
}

deploy_authz_plugin() {
    info "Deploy Default Authorization Plugin"

    "${ROOT}/scripts/ci/create-scopedaccessserver.sh"
}

get_ECR_docker_pull_password() {
    info "Get AWS ECR Docker Pull Password"

    aws --version
    local pass
    pass="$(aws --region="${AWS_ECR_REGISTRY_REGION}" ecr get-login-password)"
    ci_export AWS_ECR_DOCKER_PULL_PASSWORD "${pass}"
}

run_tests_part_1() {
    info "QA Automation Platform Part 1"

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        oc get scc qatest-anyuid || oc create -f "${ROOT}/qa-tests-backend/src/k8s/scc-qatest-anyuid.yaml"
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    if is_openshift_CI_rehearse_PR; then
        info "On an openshift rehearse PR, running BAT tests only..."
        make -C qa-tests-backend bat-test || touch FAIL
    elif pr_has_label ci-all-qa-tests; then
        info "ci-all-qa-tests label was specified, so running all QA tests..."
        make -C qa-tests-backend test || touch FAIL
    elif is_in_PR_context; then
        info "In a PR context without ci-all-qa-tests, running BAT tests only..."
        make -C qa-tests-backend bat-test || touch FAIL
    elif is_tagged; then
        info "Tagged, running all QA tests..."
        make -C qa-tests-backend test || touch FAIL
    else
        info "An unexpected context. Defaulting to BAT tests only..."
        make -C qa-tests-backend bat-test || touch FAIL
    fi

    store_qa_test_results "part-1-tests"
    [[ ! -f FAIL ]] || die "Part 1 tests failed"
}

test_part_1
