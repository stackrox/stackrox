#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests for an external Postgres.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

CURRENT_TAG="$(make --quiet --no-print-directory tag)"

# shellcheck source=../../qa-tests-backend/scripts/run-part-1.sh
source "$TEST_ROOT/qa-tests-backend/scripts/run-part-1.sh"
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
    export ROX_ACTIVE_VULN_REFRESH_INTERVAL=1m
    export ROX_NETPOL_FIELDS=true

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

    cd "$TEST_ROOT"
    local log_output_dir="$1"

    # There is an issue on gke v1.24 for these older releases where we may have a
    # timeout trying to get the metadata for the cloud provider.  Rather than extend
    # the general wait_for_api time period and potentially hide issues from other
    # tests we will extend the wait period for these tests.
    export MAX_WAIT_SECONDS=600

    # Deploy a simple postgres to use as an external database
    deploy_external_postgres

    # Run the QA BAT Tests.  Part 1 only as Part 2 deals more with sensor
    run_part_1

    collect_and_check_stackrox_logs "$log_output_dir" "byodb_QA"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_byodb "$*"
fi
