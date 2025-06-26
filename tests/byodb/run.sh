#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests for an external Postgres.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

CURRENT_TAG="$(make --quiet --no-print-directory tag)"

# shellcheck source=../../scripts/lib.sh
source "$TEST_ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/lib.sh
source "$TEST_ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../scripts/setup-certs.sh
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$TEST_ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/byodb/lib.sh
source "$TEST_ROOT/tests/byodb/lib.sh"

test_byodb() {
    info "Starting upgrade test"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_byodb <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="stackrox-io"
    REGISTRY="quay.io/$QUAY_REPO"

    export OUTPUT_FORMAT="helm"
    export CLUSTER_TYPE_FOR_TEST=K8S

    if is_CI; then
        export ROXCTL_IMAGE_REPO="quay.io/$QUAY_REPO/roxctl"
        require_environment "LONGTERM_LICENSE"
        export ROX_LICENSE_KEY="${LONGTERM_LICENSE}"
    fi

    preamble
    setup_deployment_env false false
    setup_podsecuritypolicies_config
    remove_existing_stackrox_resources

    run_byodb_test "$log_output_dir"
}

run_byodb_test() {
    info "Testing byodb"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: run_byodb_test <log-output-dir>"
    fi

    local log_output_dir="$1"

    # There is an issue on gke v1.24 for these older releases where we may have a
    # timeout trying to get the metadata for the cloud provider.  Rather than extend
    # the general wait_for_api time period and potentially hide issues from other
    # tests we will extend the wait period for these tests.
    export MAX_WAIT_SECONDS=600

    ########################################################################################
    # Use roxctl to generate helm files and deploy central backed by external database     #
    ########################################################################################
    deploy_external_postgres_central
    wait_for_api
    setup_client_TLS_certs

    # Get the API_TOKEN for the upgrades
    export API_TOKEN="$(roxcurl /v1/apitokens/generate -d '{"name": "helm-upgrade-test", "role": "Admin"}' | jq -r '.token')"

    cd "$TEST_ROOT"

    info "Fetching a sensor bundle for cluster 'remote'"
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" version
    rm -rf sensor-remote
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" -e "$API_ENDPOINT" --ca "" --insecure-skip-tls-verify sensor get-bundle remote
    [[ -d sensor-remote ]]

    info "Installing sensor"
    ./sensor-remote/sensor.sh
    kubectl -n stackrox set image deploy/sensor "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image deploy/admission-control "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image ds/collector "collector=$REGISTRY/collector:$(make collector-tag)" \
        "compliance=$REGISTRY/main:$CURRENT_TAG"
    if [[ "$(kubectl -n stackrox get ds/collector -o=jsonpath='{$.spec.template.spec.containers[*].name}')" == *"node-inventory"* ]]; then
        echo "Upgrading node-inventory container"
        kubectl -n stackrox set image ds/collector "node-inventory=$REGISTRY/scanner-slim:$(make scanner-tag)"
    else
        echo "Skipping node-inventory container as this is not Openshift 4"
    fi

    sensor_wait
    # Bounce collectors to avoid restarts on initial module pull
    kubectl -n stackrox delete pod -l app=collector --grace-period=0

    wait_for_central_reconciliation

    rm -f FAIL
    remove_qa_test_results

    info "Running smoke tests"
    CLUSTER="$CLUSTER_TYPE_FOR_TEST" make -C qa-tests-backend smoke-test || touch FAIL
    store_qa_test_results "byodb-smoke-tests"
    [[ ! -f FAIL ]] || die "Smoke tests failed"

    collect_and_check_stackrox_logs "$log_output_dir"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_byodb "$*"
fi
