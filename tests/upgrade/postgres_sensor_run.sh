#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests sensor upgrades with Postgres enabled.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

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
# shellcheck source=../../tests/upgrade/lib.sh
source "$TEST_ROOT/tests/upgrade/lib.sh"
# shellcheck source=../../tests/upgrade/validation.sh
source "$TEST_ROOT/tests/upgrade/validation.sh"

test_upgrade() {
    info "Starting Postgres upgrade test with sensor"

    cd "$TEST_ROOT"

    # Need to push the flag to ci so that is where it needs to be for the part
    # of the test.
    ci_export ROX_POSTGRES_DATASTORE "true"
    export ROX_POSTGRES_DATASTORE "true"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    require_environment "KUBECONFIG"

    export_test_environment

    REPO_FOR_TIME_TRAVEL="/tmp/rox-postgres-sensor-upgrade-test"
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="rhacs-eng"
    REGISTRY="quay.io/$QUAY_REPO"

    export STORAGE="pvc"
    export CLUSTER_TYPE_FOR_TEST=K8S

    if is_CI; then
        export ROXCTL_IMAGE_REPO="quay.io/$QUAY_REPO/roxctl"
        require_environment "LONGTERM_LICENSE"
        export ROX_LICENSE_KEY="${LONGTERM_LICENSE}"
    fi

    preamble
    setup_deployment_env false false

    remove_existing_stackrox_resources

    info "Deploying central"
    "$TEST_ROOT/$DEPLOY_DIR/central.sh"
    export_central_basic_auth_creds
    wait_for_api
    setup_client_TLS_certs

    info "Deploying sensor"
    "$TEST_ROOT/$DEPLOY_DIR/sensor.sh"
    validate_sensor_bundle_via_upgrader "$TEST_ROOT/$DEPLOY_DIR"
    sensor_wait

    wait_for_collectors_to_be_operational

    touch "${STATE_DEPLOYED}"

    test_sensor_bundle
    test_upgrader
    remove_existing_stackrox_resources
}

preamble() {
    info "Starting test preamble"

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

    require_executable "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/roxctl"
    require_executable "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/upgrader"

    info "Will clone or update a clean copy of the rox repo for test at $REPO_FOR_TIME_TRAVEL"
    if [[ -d "$REPO_FOR_TIME_TRAVEL" ]]; then
        if is_CI; then
          die "Repo for time travel already exists! This is unexpected in CI."
        fi
        (cd "$REPO_FOR_TIME_TRAVEL" && git checkout master && git reset --hard && git pull)
    else
        (cd "$(dirname "$REPO_FOR_TIME_TRAVEL")" && git clone https://github.com/stackrox/stackrox.git "$(basename "$REPO_FOR_TIME_TRAVEL")")
    fi

    if is_CI; then
        if ! command -v yq >/dev/null 2>&1; then
            sudo wget https://github.com/mikefarah/yq/releases/download/v4.4.1/yq_linux_amd64 -O /usr/bin/yq
            sudo chmod 0755 /usr/bin/yq
        fi
    else
        require_executable yq
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
