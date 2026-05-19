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
# shellcheck source=../../tests/e2e/lib-yaml.sh
source "$ROOT/tests/e2e/lib-yaml.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../qa-tests-backend/scripts/lib.sh
source "$ROOT/qa-tests-backend/scripts/lib.sh"
# shellcheck source=../../qa-tests-backend/scripts/workload-identities/workload-identities.sh
source "$ROOT/qa-tests-backend/scripts/workload-identities/workload-identities.sh"

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

    local use_roxie_deploy="${USE_ROXIE_DEPLOY:-false}"
    export_test_environment

    setup_gcp
    setup_deployment_env false false
    if [[ "$use_roxie_deploy" == "true" ]]; then
        info "Using roxie-based pre-test teardown"
        roxie teardown all --single-namespace
    else
        info "Using traditional pre-test teardown"
        remove_existing_stackrox_resources
    fi
    setup_default_TLS_certs "$ROOT/$DEPLOY_DIR/default_TLS_certs"

    image_prefetcher_system_await

    if [[ "$use_roxie_deploy" == "true" ]]; then
        info "Using roxie-based config_part_1 for qa-tests-backend"

        local config_file
        config_file="$(mktemp)"

        merge_yaml "$config_file" <<EOF
central:
  namespace: stackrox
  pauseReconciliation: true
  resourceProfile: ci
securedCluster:
  namespace: stackrox
  pauseReconciliation: true
  resourceProfile: ci
EOF

        if pr_has_label test-konflux-images; then
            info "PR label 'test-konflux-images' detected, will be using Konflux-built images for deploying StackRox"
            patch_yaml "$config_file" ".roxie.konfluxImages = true"
        fi
        if [[ "$(yq eval ".roxie.konfluxImages" "$config_file")" == "true" ]]; then
            # Due to https://access.redhat.com/solutions/6540591 we need to patch the global pull secrets
            # to be able to pull images after applying image-rewriting rules for downstream images.
            # See https://docs.redhat.com/en/documentation/openshift_container_platform/4.12/html/images/
            #     managing-images#images-update-global-pull-secret_using-image-pull-secrets.
            #
            # Can be removed once https://github.com/stackrox/roxie/pull/186 lands.
            patch_global_openshift_pull_secret "quay.io/rhacs-eng" "${QUAY_RHACS_ENG_RO_USERNAME}" "${QUAY_RHACS_ENG_RO_PASSWORD}"
        fi
        deploy_stackrox_with_roxie_compat "$config_file"
        setup_client_TLS_certs "$ROOT/$DEPLOY_DIR/client_TLS_certs"
    else
        info "Using traditional config_part_1 for qa-tests-backend"
        setup_podsecuritypolicies_config
        deploy_stackrox "$ROOT/$DEPLOY_DIR/client_TLS_certs"
        deploy_default_psp
    fi
    deploy_optional_e2e_components
    setup_workload_identities
    deploy_webhook_server "$ROOT/$DEPLOY_DIR/webhook_server_certs"
    get_ECR_docker_pull_password
    # TODO(ROX-14759): Re-enable once image pulling is fixed.
    #deploy_clair_v4

    image_prefetcher_prebuilt_await
}

patch_global_openshift_pull_secret() {
    local registry="$1"
    local username="$2"
    local password="$3"

    info "Patching global OpenShift pull-secret to include credentials for ${registry}"
    local tmp_pull_secret; tmp_pull_secret=$(mktemp)

    oc get secret/pull-secret \
        -n openshift-config \
        --template='{{index .data ".dockerconfigjson" | base64decode}}' \
        > "$tmp_pull_secret"
    oc registry login \
        --registry="$registry" \
        --auth-basic="${username}:${password}" \
        --to="$tmp_pull_secret"
    oc set data secret/pull-secret \
        -n openshift-config \
        --from-file=.dockerconfigjson="$tmp_pull_secret"

    rm -f "$tmp_pull_secret"
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
    remove_qa_test_results

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

    setup_gcp
    set_ci_shared_export "test_target" "${test_target}"

    make -C qa-tests-backend "${test_target}" || touch FAIL

    cleanup_workload_identities
    store_qa_test_results "part-1-tests"
    [[ ! -f FAIL ]] || die "Part 1 tests failed"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    run_part_1 "$*"
fi
