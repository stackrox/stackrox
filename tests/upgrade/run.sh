#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests upgrade. Formerly CircleCI gke-api-upgrade-tests.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/lib.sh"
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
source "$TEST_ROOT/tests/e2e/lib.sh"
source "$TEST_ROOT/tests/upgrade/lib.sh"

test_upgrade() {
    info "Starting upgrade test"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    REPO_FOR_TIME_TRAVEL="/tmp/rox-upgrade-test"
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="rhacs-eng"
    if is_CI; then
        REGISTRY="quay.io/$QUAY_REPO"
    else
        REGISTRY="stackrox"
    fi

    export OUTPUT_FORMAT="helm"
    export STORAGE="pvc"
    export CLUSTER_TYPE_FOR_TEST=K8S

    if is_CI; then
        export ROXCTL_IMAGE_REPO="quay.io/$QUAY_REPO/roxctl"
    fi

    preamble
    setup_deployment_env false false
    remove_existing_stackrox_resources

    info "Deploying central"
    "$TEST_ROOT/$DEPLOY_DIR/central.sh"
    get_central_basic_auth_creds
    wait_for_api
    setup_client_TLS_certs

    info "Deploying sensor"
    "$TEST_ROOT/$DEPLOY_DIR/sensor.sh"
    validate_sensor_bundle_via_upgrader "$TEST_ROOT/$DEPLOY_DIR"
    sensor_wait

    touch "${STATE_DEPLOYED}"

    test_sensor_bundle
    test_upgrader
    remove_existing_stackrox_resources
    test_upgrade_paths "$log_output_dir"
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

# test_upgrade_paths implements tests to run upgrades with Helm, roxctl/kubectl and performs various rollbacks/restores.
test_upgrade_paths() {
    info "Testing various upgrade paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    EARLIER_SHA="870568de0830819aae85f255dbdb7e9c19bd74e7"
    EARLIER_TAG="3.69.x-1-g870568de08"
    FORCE_ROLLBACK_VERSION="$EARLIER_TAG"
    export FORCE_ROLLBACK_VERSION

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    # required to handle scanner protos
    go mod tidy

    deploy_earlier_central
    wait_for_api
    restore_backup_test
    wait_for_api

    ########################################################################################
    # Test roxctl/bundle/kubectl upgrade by setting the deployed Central image via kubectl #
    ########################################################################################
    cd "$TEST_ROOT"

    kubectl -n stackrox set env deploy/central ROX_NETPOL_FIELDS="true"
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$(make --quiet tag)"
    wait_for_api

    validate_upgrade "00-3-69-x-to-current" "central upgrade to 3.69.x -> current" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    #####################
    # Test rollback     #
    #####################

    force_rollback
    wait_for_api

    cd "$REPO_FOR_TIME_TRAVEL"

    validate_upgrade "01-current-back-to-3-69-x" "forced rollback to 3.69.x from current" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    ######################################
    # Test helm upgrade after a rollback #
    ######################################
    cd "$TEST_ROOT"

    helm_upgrade_to_current
    wait_for_api

    info "Waiting for scanner to be ready"
    wait_for_scanner_to_be_ready

    validate_upgrade "02-after_rollback" "upgrade after rollback" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "00_initial_check"

    ######################################
    # Test backup and restore            #
    ######################################

    validate_db_backup_and_restore
    wait_for_api

    validate_upgrade "03-after-DB-backup-restore-pre-bounce" "after DB backup and restore (pre bounce)" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "01_pre_bounce"

    info "Bouncing central"
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api

    validate_upgrade "04-after-DB-backup-restore-post-bounce" "after DB backup and restore (post bounce)" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "02_post_bounce"

    ######################################
    # Test upgrade some tests            #
    ######################################

    info "Fetching a sensor bundle for cluster 'remote'"
    rm -rf sensor-remote
    "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    info "Installing sensor"
    ./sensor-remote/sensor.sh
    kubectl -n stackrox set image deploy/sensor "*=$REGISTRY/main:$(make --quiet tag)"
    kubectl -n stackrox set image deploy/admission-control "*=$REGISTRY/main:$(make --quiet tag)"
    kubectl -n stackrox set image ds/collector "collector=$REGISTRY/collector:$(cat COLLECTOR_VERSION)" \
        "compliance=$REGISTRY/main:$(make --quiet tag)"

    sensor_wait

    wait_for_central_reconciliation

    info "Running smoke tests"
    CLUSTER="$CLUSTER_TYPE_FOR_TEST" make -C qa-tests-backend smoke-test || touch FAIL
    store_qa_test_results "upgrade-paths-smoke-tests"
    [[ ! -f FAIL ]] || die "Smoke tests failed"

    collect_and_check_stackrox_logs "$log_output_dir" "03_final"
}

helm_upgrade_to_current() {
    info "Helm upgrade to current"

    # Get opensource charts and convert to development_build to support release builds
    roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart
    sed -i 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml

    helm upgrade -n stackrox stackrox-central-services /tmp/stackrox-central-services-chart

    kubectl -n stackrox get deploy -o wide
}

validate_db_backup_and_restore() {
    info "Backing up and restoring the DB"

    local db_backup="db_backup.zip"

    rm -f "$db_backup"
    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" central db backup --output "$db_backup"
    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" central db restore "$db_backup"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
