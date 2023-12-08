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

scannerV4_test() {
    info "Starting ScannerV4 test"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "ROX_SCANNER_V4_ENABLED"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_older_central_without_scanner

    run_scannerV4_test
}

run_scannerV4_test() {
    info "Running scannerV4 test"
    info "Nothing yet..."
}

deploy_older_central_without_scanner() {
    export EARLIER_TAG="4.3.0-0-g23e64bf342"
    local host_os
    if is_darwin; then
        host_os="darwin"
    elif is_linux; then
        host_os="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    case "$(uname -m)" in
        x86_64) TEST_HOST_PLATFORM="${host_os}_amd64" ;;
        aarch64) TEST_HOST_PLATFORM="${host_os}_arm64" ;;
        arm64) TEST_HOST_PLATFORM="${host_os}_arm64" ;;
        ppc64le) TEST_HOST_PLATFORM="${host_os}_ppc64le" ;;
        s390x) TEST_HOST_PLATFORM="${host_os}_s390x" ;;
        *) die "Unknown architecture" ;;
    esac

    gsutil cp "gs://stackrox-ci/roxctl-$EARLIER_TAG" "bin/$TEST_HOST_PLATFORM/roxctl"

    chmod +x "bin/$TEST_HOST_PLATFORM/roxctl"
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" command -v roxctl
    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl version

    PATH="bin/$TEST_HOST_PLATFORM:$PATH" \
    MAIN_IMAGE_TAG="$EARLIER_TAG" \
    ./deploy/k8s/central.sh

    export_central_basic_auth_creds
}

scannerV4_test "$@"
