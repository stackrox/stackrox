#!/usr/bin/env bash

# Runs Scanner V4 tests.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

# Could also use
#   CHART_BASE="/rhacs"
#   export DEFAULT_IMAGE_REGISTRY="quay.io/rhacs-eng"
# for the opensource flavour.
CHART_BASE=""
export DEFAULT_IMAGE_REGISTRY="quay.io/stackrox-io"
CURRENT_MAIN_IMAGE_TAG=${CURRENT_MAIN_IMAGE_TAG:-} # Setting a tag can be useful for local testing.
EARLIER_CHART_VERSION="4.3.0"
EARLIER_MAIN_IMAGE_TAG=$EARLIER_CHART_VERSION

scannerV4_test() {
    info "Starting ScannerV4 test"

    require_environment "ORCHESTRATOR_FLAVOR"
    export OUTPUT_FORMAT=helm
    export ROX_SCANNER_V4_ENABLED=true
    export USE_LOCAL_ROXCTL=true

    export_test_environment

    if [[ "$CI" = "true" ]]; then
        setup_gcp
    fi
    setup_deployment_env false false
    remove_existing_stackrox_resources

    if [[ "$CI" = "true" ]]; then
        setup_default_TLS_certs
    fi

    # Prepare earlier version
    if [[ -z "${CHART_REPOSITORY:-}" ]]; then
        CHART_REPOSITORY=$(mktemp -d "helm-charts.XXXXXX" -p /tmp)
    fi
    if [[ ! -e "${CHART_REPOSITORY}/.git" ]]; then
        git clone --depth 1 -b main https://github.com/stackrox/helm-charts "${CHART_REPOSITORY}"
    fi

    # Deploy earlier version without Scanner V4.
    export CENTRAL_CHART_DIR_OVERRIDE="${CHART_REPOSITORY}${CHART_BASE}/${EARLIER_CHART_VERSION}/central-services"
    info "Deplying central-services using chart $CENTRAL_CHART_DIR_OVERRIDE"
    if [[ -n "${EARLIER_MAIN_IMAGE_TAG:-}" ]]; then
        export MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG"
    fi
    deploy_stackrox
    unset MAIN_IMAGE_TAG
    unset CENTRAL_CHART_DIR_OVERRIDE

    # Upgrade to HEAD chart.
    info "Upgrading central-services using HEAD Helm chart"
    if [[ -n "${CURRENT_MAIN_IMAGE_TAG:-}" ]]; then
        export MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG"
    fi
    deploy_stackrox # This is doing an `helm upgrade --install ...` under the hood.

    run_scannerV4_test
}

run_scannerV4_test() {
    info "Running scannerV4 test"
    info "Nothing yet..."
}

scannerV4_test "$@"
