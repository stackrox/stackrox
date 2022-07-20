#!/usr/bin/env bash

# Tests part I of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/gcp.sh"
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

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    (deploy_stackrox) || {
        local exitstatus="$?"
        touch "$ROOT/SKIP_PART_II"
        exit "$exitstatus"
    }

    deploy_default_psp
    deploy_webhook_server
    get_ECR_docker_pull_password

    run_tests_part_1
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
    elif is_in_PR_context && pr_has_label ci-all-qa-tests; then
        info "ci-all-qa-tests label was specified, so running all QA tests..."
        make -C qa-tests-backend test || touch FAIL
    elif is_in_PR_context; then
        info "In a PR context without ci-all-qa-tests, running BAT tests only..."
        make -C qa-tests-backend bat-test || touch FAIL
    elif is_tagged; then
        info "Tagged, running all QA tests..."
        make -C qa-tests-backend test || touch FAIL
    elif [[ -n "${QA_TEST_TARGET:-}" ]]; then
        info "Directed to run the '""${QA_TEST_TARGET:-}""' target..."
        make -C qa-tests-backend "${QA_TEST_TARGET:-}" || touch FAIL
    else
        info "An unexpected context. Defaulting to BAT tests only..."
        make -C qa-tests-backend bat-test || touch FAIL
    fi

    store_qa_test_results "part-1-tests"
    [[ ! -f FAIL ]] || die "Part 1 tests failed"
}

test_part_1
