#!/usr/bin/env bash

# Tests part I of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

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

run_part_1() {
    info "Starting test (qa-tests-backend part I)"

    config_part_1
    test_part_1
}

config_part_1() {
    info "Configuring the cluster to run part 1 of e2e tests"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"

    DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    setup_podsecuritypolicies_config
    remove_existing_stackrox_resources
    setup_default_TLS_certs "$ROOT/$DEPLOY_DIR/default_TLS_certs"

    deploy_stackrox "$ROOT/$DEPLOY_DIR/client_TLS_certs"
    deploy_optional_e2e_components

    deploy_default_psp
    deploy_webhook_server "$ROOT/$DEPLOY_DIR/webhook_server_certs"
    get_ECR_docker_pull_password
    # TODO(ROX-14759): Re-enable once image pulling is fixed.
    #deploy_clair_v4
}

reuse_config_part_1() {
    info "Reusing config from a prior part 1 e2e test"

    DEPLOY_DIR="deploy/${ORCHESTRATOR_FLAVOR}"

    export_test_environment
    setup_deployment_env false false
    export_default_TLS_certs "$ROOT/$DEPLOY_DIR/default_TLS_certs"
    export_client_TLS_certs "$ROOT/$DEPLOY_DIR/client_TLS_certs"

    create_webhook_server_port_forward
    export_webhook_server_certs "$ROOT/$DEPLOY_DIR/webhook_server_certs"
    get_ECR_docker_pull_password

    wait_for_api
    export_central_basic_auth_creds

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"
}

test_part_1() {
    info "QA Automation Platform Part 1"

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        oc get scc qatest-anyuid || oc create -f "${ROOT}/qa-tests-backend/src/k8s/scc-qatest-anyuid.yaml"
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    rm -f FAIL
    local test_target

    if is_openshift_CI_rehearse_PR; then
        info "On an openshift rehearse PR, running BAT tests only..."
        test_target="bat-test"
    elif is_in_PR_context && pr_has_label ci-all-qa-tests; then
        info "ci-all-qa-tests label was specified, so running all QA tests..."
        test_target="test"
    elif is_in_PR_context; then
        info "In a PR context without ci-all-qa-tests, running BAT tests only..."
        test_target="bat-test"
    else
        info "Running all QA tests by default..."
        test_target="test"
    fi

    update_job_record "test_target" "${test_target}"

    test_target="pz-test-debug"
    make -C qa-tests-backend "${test_target}" || touch FAIL

    store_qa_test_results "part-1-tests"
    [[ ! -f FAIL ]] || die "Part 1 tests failed"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    run_part_1 "$*"
fi
