#!/usr/bin/env bash

# Custom tests of qa-tests-backend. Formerly CircleCI gke-api-e2e-tests.

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

run_custom() {
    info "Starting test (Custom tests for power/s390x)"

    config_custom
    test_custom
}

config_custom() {
    info "Configuring the cluster to custom tests for power and s390x"

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

test_custom() {
    info "Running custom tests"

    if [[ "${ORCHESTRATOR_FLAVOR}" == "openshift" ]]; then
        oc get scc qatest-anyuid || oc create -f "${ROOT}/qa-tests-backend/src/k8s/scc-qatest-anyuid.yaml"
    fi

    export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"

    STACKROX_TESTNAMES=("AdmissionControllerNoImageScanTest")
    STACKROX_TESTNAMES+=("AttemptedAlertsTest" "AuditLogAlertsTest" "AuthServiceTest" "AutocompleteTest")
    STACKROX_TESTNAMES+=("CertExpiryTest" "CertRotationTest" "ClusterInitBundleTest" "ClustersTest")
    STACKROX_TESTNAMES+=("DeploymentEventGraphQLTest" "DiagnosticBundleTest")
    #STACKROX_TESTNAMES+=("Enforcement")
    STACKROX_TESTNAMES+=("GlobalSearch" "GroupsTest")
    STACKROX_TESTNAMES+=("IntegrationHealthTest")
    STACKROX_TESTNAMES+=("K8sRbacTest")
    STACKROX_TESTNAMES+=("NetworkBaselineTest" "NetworkSimulator" "NodeInventoryTest")
    STACKROX_TESTNAMES+=("PaginationTest" "ProcessesListeningOnPortsTest")
    STACKROX_TESTNAMES+=("RbacAuthTest" "RuntimePolicyTest" "RuntimeViolationLifecycleTest")
    STACKROX_TESTNAMES+=("SecretsTest" "SummaryTest")
    STACKROX_TESTNAMES+=("TLSChallengeTest")
    STACKROX_TESTNAMES+=("VulnMgmtSACTest" "VulnMgmtTest" "VulnMgmtWorkflowTest")

    #Initialize variables
    interval_sec=20

    #Generate srcs
    make -C qa-tests-backend compile
    #Change directory into qa-tests-backend
    cd qa-tests-backend
    #fetch list of tests
   for testName in "${STACKROX_TESTNAMES[@]}";
   do
    #execute test
      ./gradlew test --tests "$testName" || true

    #allow previous test data to cleanup
      sleep $interval_sec
    done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    run_custom "$*"
fi
