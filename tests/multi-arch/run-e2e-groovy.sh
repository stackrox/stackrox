#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$ROOT/scripts/lib.sh"
source "$ROOT/scripts/ci/sensor-wait.sh"
source "$ROOT/tests/scripts/setup-certs.sh"
source "$ROOT/tests/e2e/lib.sh"

# TODO(sbostick): where are the groovy tests run from?
# TODO(sbostick): This was just a copy of the nongroovy tests entrypoint.
# TODO(sbostick): Although I'm pruning it and adding groovy tests invocation.

test_e2e() {
    info "Starting e2e tests"

    require_environment "KUBECONFIG"

    export_test_environment

    export SENSOR_HELM_DEPLOY=true
    export ROX_ACTIVE_VULN_REFRESH_INTERVAL=1m
    export ROX_NETPOL_FIELDS=true

    test_preamble
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs
    "$ROOT/tests/complianceoperator/create.sh"

    deploy_stackrox
    prepare_for_endpoints_test

    ############################################################
    info "E2E API Groovy tests"

    cd "$ROOT"
    make proto-generated-srcs

    cd "$ROOT/qa-tests-backend"
    gradle build -x test
    gradle test --tests='GlobalSearch'
    gradle test --tests='NetworkFlowTest'
    ### gradle test
    ############################################################
}

test_preamble() {
    require_executable "roxctl"

    if ! is_CI; then
        require_environment "MAIN_IMAGE_TAG" "This is typically the output from 'make tag'"

        if [[ "$(roxctl version)" != "$MAIN_IMAGE_TAG" ]]; then
            die "There is a version mismatch between roxctl and MAIN_IMAGE_TAG. A version mismatch can cause the deployment script to use a container roxctl which can have issues in dev environments."
        fi
        pwds="$(pgrep -f 'port-forward' -c || true)"
        if [[ "$pwds" -gt 5 ]]; then
            die "There are many port-fowards probably left over from a previous run of this test."
        fi
        cleanup_proxy_tests
        export MAIN_TAG="$MAIN_IMAGE_TAG"
    else
        MAIN_TAG=$(make --quiet tag)
        export MAIN_TAG
    fi

    export ROX_PLAINTEXT_ENDPOINTS="8080,grpc@8081"
    export ROXDEPLOY_CONFIG_FILE_MAP="$ROOT/scripts/ci/endpoints/endpoints.yaml"

    QUAY_REPO="rhacs-eng"
    if is_CI; then
        REGISTRY="quay.io/$QUAY_REPO"
    else
        REGISTRY="stackrox"
    fi

    SCANNER_IMAGE="$REGISTRY/scanner:$(cat "$ROOT"/SCANNER_VERSION)"
    export SCANNER_IMAGE
    SCANNER_DB_IMAGE="$REGISTRY/scanner-db:$(cat "$ROOT"/SCANNER_VERSION)"
    export SCANNER_DB_IMAGE

    export TRUSTED_CA_FILE="$ROOT/tests/bad-ca/untrusted-root-badssl-com.pem"
}

prepare_for_endpoints_test() {
    info "Preparation for endpoints_test.go"

    local gencerts_dir
    gencerts_dir="$(mktemp -d)"
    setup_client_CA_auth_provider
    setup_generated_certs_for_test "$gencerts_dir"
    patch_resources_for_test
    export SERVICE_CA_FILE="$gencerts_dir/ca.pem"
    export SERVICE_CERT_FILE="$gencerts_dir/sensor-cert.pem"
    export SERVICE_KEY_FILE="$gencerts_dir/sensor-key.pem"
    start_port_forwards_for_test
}


if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_e2e "$*"
fi
