#!/usr/bin/env bats

# Runs Scanner V4 tests using the Bats testing framework.

setup_file() {
    ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")"/../.. && pwd)"
    export ROOT

    # Use
    #   export CHART_BASE="/rhacs"
    #   export DEFAULT_IMAGE_REGISTRY="quay.io/rhacs-eng"
    # for the RHACS flavor.
    export CHART_BASE=""
    export DEFAULT_IMAGE_REGISTRY="quay.io/stackrox-io"

    export CURRENT_MAIN_IMAGE_TAG=${CURRENT_MAIN_IMAGE_TAG:-} # Setting a tag can be useful for local testing.
    export EARLIER_CHART_VERSION="4.3.0"
    export EARLIER_MAIN_IMAGE_TAG=$EARLIER_CHART_VERSION
    export USE_LOCAL_ROXCTL=true
    export ROX_PRODUCT_BRANDING=RHACS_BRANDING
    export CI=${CI:-false}

    # Prepare earlier version
    if [[ -z "${CHART_REPOSITORY:-}" ]]; then
        CHART_REPOSITORY=$(mktemp -d "helm-charts.XXXXXX" -p /tmp)
    fi
    if [[ ! -e "${CHART_REPOSITORY}/.git" ]]; then
        git clone --depth 1 -b main https://github.com/stackrox/helm-charts "${CHART_REPOSITORY}"
    fi
    export CHART_REPOSITORY
}

test_case_no=0

setup() {
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
    load "$ROOT/scripts/test_helpers.bats"

    set -euo pipefail

    require_environment "ORCHESTRATOR_FLAVOR"
    export_test_environment
    if [[ "$CI" = "true" ]]; then
        setup_gcp
        setup_deployment_env false false
    fi

    if (( test_case_no == 0 )); then
        # executing initial teardown to begin test execution in a well-defined state
       run remove_existing_stackrox_resources
    fi
    test_case_no=$(( test_case_no + 1))
}

teardown() {
    run remove_existing_stackrox_resources
}

@test "Upgrade from old Helm chart to HEAD Helm chart with Scanner v4 enabled" {
    local MAIN_IMAGE_TAG=""

    if [[ "$CI" = "true" ]]; then
        setup_default_TLS_certs
    fi

    # Deploy earlier version without Scanner V4.
    local _CENTRAL_CHART_DIR_OVERRIDE="${CHART_REPOSITORY}${CHART_BASE}/${EARLIER_CHART_VERSION}/central-services"
    info "Deplying StackRox services using chart ${_CENTRAL_CHART_DIR_OVERRIDE}"
    if [[ -n "${EARLIER_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$EARLIER_MAIN_IMAGE_TAG"
    fi
    (
        # shellcheck disable=SC2030,SC2031
        export MAIN_IMAGE_TAG
        # shellcheck disable=SC2030,SC2031
        export ROX_SCANNER_V4=true
        # shellcheck disable=SC2030,SC2031
        export CENTRAL_CHART_DIR_OVERRIDE="${_CENTRAL_CHART_DIR_OVERRIDE}"
        # shellcheck disable=SC2030,SC2031
        export OUTPUT_FORMAT=helm
        deploy_stackrox >&3
    )

    # Upgrade to HEAD chart without explicit disabling of Scanner v4.
    info "Upgrading StackRox using HEAD Helm chart"
    MAIN_IMAGE_TAG=""
    if [[ -n "${CURRENT_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG"
    fi
    (
        # shellcheck disable=SC2030,SC2031
        export MAIN_IMAGE_TAG
        # shellcheck disable=SC2030,SC2031
        export ROX_SCANNER_V4=true
        # shellcheck disable=SC2030,SC2031
        export OUTPUT_FORMAT=helm
        deploy_stackrox >&3 # This is doing an `helm upgrade --install ...` under the hood.
    )

    # Verify that Scanner v2 and v4 are up.
    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
}

@test "Fresh installation of HEAD Helm chart with Scanner v4 disabled" {
    MAIN_IMAGE_TAG=""
    info "Installing StackRox using HEAD Helm chart with Scanner v4 disabled"
    if [[ -n "${CURRENT_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG"
    fi
    (
        # shellcheck disable=SC2030,SC2031
        export MAIN_IMAGE_TAG
        # shellcheck disable=SC2030,SC2031
        export ROX_SCANNER_V4=false
        # shellcheck disable=SC2030,SC2031
        export OUTPUT_FORMAT=helm
        deploy_stackrox >&3
    )
    verify_scannerV2_deployed "stackrox"
    verify_no_scannerV4_deployed "stackrox"
}

@test "Fresh installation of central deployment bundle with Scanner v4 enabled" {
    MAIN_IMAGE_TAG=""
    info "Installing StackRox using central deployment bundle with Scanner v4 enabled"
    if [[ -n "${CURRENT_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG"
    fi
    (
        # shellcheck disable=SC2030,SC2031
        export MAIN_IMAGE_TAG
        # shellcheck disable=SC2030,SC2031
        export OUTPUT_FORMAT=kubectl
        deploy_stackrox
    )
    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
}

@test "Fresh installation of central deployment bundle with Scanner v4 disabled" {
    MAIN_IMAGE_TAG=""
    info "Installing StackRox using central deployment bundle with Scanner v4 disabled"
    if [[ -n "${CURRENT_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG"
    fi
    (
        # shellcheck disable=SC2030,SC2031
        export MAIN_IMAGE_TAG
        # shellcheck disable=SC2030,SC2031
        export OUTPUT_FORMAT=kubectl
        # shellcheck disable=SC2030,SC2031
        export ROX_SCANNER_V4=false
        deploy_stackrox
    )
    verify_scannerV2_deployed "stackrox"
    verify_no_scannerV4_deployed "stackrox"
}

@test "Fresh installation of HEAD Helm chart with Scanner v4 enabled" {
    MAIN_IMAGE_TAG=""
    info "Installing StackRox using HEAD Helm chart with Scanner v4 enabled"
    if [[ -n "${CURRENT_MAIN_IMAGE_TAG:-}" ]]; then
        MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG
        info "Overriding MAIN_IMAGE_TAG=$CURRENT_MAIN_IMAGE_TAG"
    fi
    (
        # shellcheck disable=SC2030,SC2031
        export MAIN_IMAGE_TAG
        # shellcheck disable=SC2030,SC2031
        export ROX_SCANNER_V4=true
        # shellcheck disable=SC2030,SC2031
        export OUTPUT_FORMAT=helm
        deploy_stackrox >&3
    )
    verify_scannerV2_deployed "stackrox"
    verify_scannerV4_deployed "stackrox"
}

verify_no_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    run kubectl -n "$namespace" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    refute_output --regexp "scanner-v4"
}

# TODO: For now, Scanner v2 is expected to run in parallel.
# This must be removed when Scanner v2 will be phased out.
verify_scannerV2_deployed() {
    local namespace=${1:-stackrox}
    wait_for_object_to_appear "$namespace" deploy/scanner 300
    wait_for_object_to_appear "$namespace" deploy/scanner-db 300
}

verify_scannerV4_deployed() {
    local namespace=${1:-stackrox}
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-db 300
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-indexer 300
    wait_for_object_to_appear "$namespace" deploy/scanner-v4-matcher 300
}
