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
    require_environment "LONGTERM_LICENSE"
    export ROX_LICENSE_KEY="${LONGTERM_LICENSE}"

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

    if is_darwin; then
        TEST_HOST_OS="darwin"
    elif is_linux; then
        TEST_HOST_OS="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    require_executable "$TEST_ROOT/bin/$TEST_HOST_OS/roxctl"
    require_executable "$TEST_ROOT/bin/$TEST_HOST_OS/upgrader"

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

validate_sensor_bundle_via_upgrader() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: validate_sensor_bundle_via_upgrader <deploy_dir>"
    fi

    local deploy_dir="$1"

    info "Validating the sensor bundle via upgrader"

    kubectl proxy --port 28001 &
    local proxy_pid=$!
    sleep 5

    KUBECONFIG="$TEST_ROOT/scripts/ci/kube-api-proxy/config.yml" \
        "$TEST_ROOT/bin/$TEST_HOST_OS/upgrader" \
        -kube-config kubectl \
        -local-bundle "$deploy_dir/sensor-deploy" \
        -workflow validate-bundle

    kill "$proxy_pid"
}

test_sensor_bundle() {
    info "Testing the sensor bundle"

    rm -rf sensor-remote
    "$TEST_ROOT/bin/$TEST_HOST_OS/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    ./sensor-remote/sensor.sh

    kubectl -n stackrox patch deploy/sensor --patch '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"limits":{"cpu":"500m","memory":"500Mi"},"requests":{"cpu":"500m","memory":"500Mi"}}}]}}}}'

    sensor_wait

    ./sensor-remote/delete-sensor.sh
    rm -rf sensor-remote
}

test_upgrader() {
    info "Starting bin/upgrader tests"

    install_metrics_server_and_deactivate

    info "Creating a 'sensor-remote-new' cluster"

    rm -rf sensor-remote-new
    "$TEST_ROOT/bin/$TEST_HOST_OS/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor generate k8s \
        --main-image-repository "${MAIN_IMAGE_REPO:-$REGISTRY/main}" \
        --collector-image-repository "${COLLECTOR_IMAGE_REPO:-$REGISTRY/collector}" \
        --name remote-new \
        --create-admission-controller

    deploy_sensor_via_upgrader "for the first time, to test rollback" 3b2cbf78-d35a-4c2c-b67b-e37f805c14da
    rollback_sensor_via_upgrader 3b2cbf78-d35a-4c2c-b67b-e37f805c14da

    deploy_sensor_via_upgrader "from scratch" 9b8f2cbb-72c6-4e00-b375-5b50ce7f988b

    deploy_sensor_via_upgrader "again, but with the same upgrade process ID" 9b8f2cbb-72c6-4e00-b375-5b50ce7f988b

    deploy_sensor_via_upgrader "yet again, but with a new upgrade process ID" 789c9262-5dd3-4d58-a824-c2a099892bd6

    webhook_timeout_before_patch="$(kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o json | jq '.webhooks | .[0] | .timeoutSeconds')"
    echo "Webhook timeout before patch: ${webhook_timeout_before_patch}"
    webhook_timeout_after_patch="$((webhook_timeout_before_patch + 1))"
    echo "Desired webhook timeout after patch: ${webhook_timeout_after_patch}"

    info "Patch admission webhook"
    kubectl -n stackrox patch validatingwebhookconfiguration stackrox --type 'json' -p "[{'op':'replace','path':'/webhooks/0/timeoutSeconds','value':${webhook_timeout_after_patch}}]"
    if [[ "$(kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o json | jq '.webhooks | .[0] | .timeoutSeconds')" -ne "${webhook_timeout_after_patch}" ]]; then
        echo "Webhook not patched"
        kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o yaml
        exit 1
    fi

    info "Patch resources"
    kubectl -n stackrox set resources deploy/sensor -c sensor --requests 'cpu=1.1,memory=1.1Gi'

    deploy_sensor_via_upgrader "after manually patching webhook" 060a9fa6-0ed6-49ac-b70c-9ca692614707

    info "Verify the webhook was patched back by the upgrader"
    if [[ "$(kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o json | jq '.webhooks | .[0] | .timeoutSeconds')" -ne "${webhook_timeout_before_patch}" ]]; then
        echo "Webhook not patched"
        kubectl -n stackrox get validatingwebhookconfiguration/stackrox -o yaml
        exit 1
    fi

    deploy_sensor_via_upgrader "with yet another new ID" 789c9262-5dd3-4d58-a824-c2a099892bd7

    info "Verify resources were patched back by the upgrader"
    resources="$(kubectl -n stackrox get deploy/sensor -o 'jsonpath=cpu={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.cpu},memory={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.memory}')"
    if [[ "$resources" != 'cpu=1,memory=1Gi' ]]; then
        echo "Resources ($resources) not patched back!"
        kubectl -n stackrox get deploy/sensor -o yaml
        exit 1
    fi

    info "Patch resources and add preserve resources annotation. Also, check toleration preservation"
    kubectl -n stackrox annotate deploy/sensor "auto-upgrade.stackrox.io/preserve-resources=true"
    kubectl -n stackrox set resources deploy/sensor -c sensor --requests 'cpu=1.1,memory=1.1Gi'
    kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"tolerations":[{"effect":"NoSchedule","key":"thekey","operator":"Equal","value":"thevalue"}]}}}}'

    deploy_sensor_via_upgrader "after patching resources with preserve annotation" 789c9262-5dd3-4d58-a824-c2a099892bd8

    info "Verify resources were not patched back by the upgrader"
    resources="$(kubectl -n stackrox get deploy/sensor -o 'jsonpath=cpu={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.cpu},memory={.spec.template.spec.containers[?(@.name=="sensor")].resources.requests.memory}')"
    if [[ "$resources" != 'cpu=1100m,memory=1181116006400m' ]]; then
        echo "Resources ($resources) appear patched back!"
        kubectl -n stackrox get deploy/sensor -o yaml
        exit 1
    fi
    toleration="$(kubectl -n stackrox get deploy/sensor -o json | jq -rc '.spec.template.spec.tolerations[0] | (.effect + "," + .key + "," + .operator + "," + .value)')"
    echo "Found toleration: $toleration"
    if [[ "$toleration" != 'NoSchedule,thekey,Equal,thevalue' ]]; then
        echo "Tolerations were not passed through to new Sensor"
        kubectl -n stackrox get deploy/sensor -o yaml
        exit 1
    fi
}

install_metrics_server_and_deactivate() {
    info "Install the metrics server and deactivate it to reproduce ROX-4429"

    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.3.6/components.yaml

    echo "Waiting for metrics.k8s.io to be in kubectl API resources..."
    local success=0
    for i in $(seq 1 10); do
        if kubectl api-resources 2>&1 | sed -e 's/^/out: /' | grep metrics.k8s.io; then
            success=1
            break
        fi
        sleep 5
    done
    [[ "$success" -eq 1 ]]

    ## Patch the metrics server to be unreachable
    kubectl -n kube-system patch svc/metrics-server --type json -p '[{"op": "replace", "path": "/spec/selector", "value": {"k8s-app": "non-existent"}}]'

    echo "Waiting for metrics.k8s.io to NOT be in kubectl API resources..."
    local success=0
    # shellcheck disable=SC2034
    for i in $(seq 1 10); do
        kubectl api-resources >stdout.out 2>stderr.out || true
        if grep -q 'metrics.k8s.io.*the server is currently unable to handle the request' stderr.out; then
            success=1
            break
        fi
        echo "metrics.k8s.io still in API resources. Will try again..."
        cat stdout.out
        sed -e 's/^/out: /' < stderr.out # (prefix output to avoid triggering prow log focus)
        sleep 5
    done
    [[ "$success" -eq 1 ]]
    rm -f stdout.out stderr.out

    info "deactivated"
}

deploy_sensor_via_upgrader() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: deploy_sensor_via_upgrader <stage> <upgrade_process_id>"
    fi

    local stage="$1"
    local upgrade_process_id="$2"

    info "Deploying sensor via upgrader: $stage"

    kubectl proxy --port 28001 &
    local proxy_pid=$!
    sleep 5

    ROX_UPGRADE_PROCESS_ID="$upgrade_process_id" \
        ROX_CENTRAL_ENDPOINT="$API_ENDPOINT" \
        ROX_MTLS_CA_FILE="$TEST_ROOT/sensor-remote-new/ca.pem" \
        ROX_MTLS_CERT_FILE="$TEST_ROOT/sensor-remote-new/sensor-cert.pem" \
        ROX_MTLS_KEY_FILE="$TEST_ROOT/sensor-remote-new/sensor-key.pem" \
        KUBECONFIG="$TEST_ROOT/scripts/ci/kube-api-proxy/config.yml" \
        "$TEST_ROOT/bin/$TEST_HOST_OS/upgrader" -workflow roll-forward -local-bundle sensor-remote-new -kube-config kubectl

    kill "$proxy_pid"

    sensor_wait
}

rollback_sensor_via_upgrader() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: rollback_sensor_via_upgrader <upgrade_process_id>"
    fi

    local upgrade_process_id="$1"

    info "Rolling back sensor via upgrader"

    kubectl proxy --port 28001 &
    local proxy_pid=$!
    sleep 5

    ROX_UPGRADE_PROCESS_ID="$upgrade_process_id" \
        ROX_CENTRAL_ENDPOINT="$API_ENDPOINT" \
        ROX_MTLS_CA_FILE="$TEST_ROOT/sensor-remote-new/ca.pem" \
        ROX_MTLS_CERT_FILE="$TEST_ROOT/sensor-remote-new/sensor-cert.pem" \
        ROX_MTLS_KEY_FILE="$TEST_ROOT/sensor-remote-new/sensor-key.pem" \
        KUBECONFIG="$TEST_ROOT/scripts/ci/kube-api-proxy/config.yml" \
        "$TEST_ROOT/bin/$TEST_HOST_OS/upgrader" -workflow roll-back -kube-config kubectl

    kill "$proxy_pid"
}

test_upgrade_paths() {
    info "Testing various upgrade paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    EARLIER_SHA="9f82d2713cfec4b5c876d8dc0149f6d9cd70d349"
    EARLIER_TAG="3.63.x-163-g2c4fe1563c"
    FORCE_ROLLBACK_VERSION="$EARLIER_TAG"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    deploy_earlier_central
    wait_for_api
    restore_backup_test
    wait_for_api

    cd "$TEST_ROOT"

    kubectl -n stackrox set env deploy/central ROX_NETPOL_FIELDS="true"
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$(make --quiet tag)"
    wait_for_api

    validate_upgrade "00-3-63-x-to-current" "central upgrade to 3.63.x -> current" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    force_rollback
    wait_for_api

    cd "$REPO_FOR_TIME_TRAVEL"

    validate_upgrade "01-current-back-to-3-63-x" "forced rollback to 3.63.x from current" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    cd "$TEST_ROOT"

    helm_upgrade_to_current
    wait_for_api

    info "Waiting for scanner to be ready"
    wait_for_scanner_to_be_ready

    validate_upgrade "02-after_rollback" "upgrade after rollback" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "00_initial_check"

    validate_db_backup_and_restore
    wait_for_api

    validate_upgrade "03-after-DB-backup-restore-pre-bounce" "after DB backup and restore (pre bounce)" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "01_pre_bounce"

    info "Bouncing central"
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api

    validate_upgrade "04-after-DB-backup-restore-post-bounce" "after DB backup and restore (post bounce)" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "02_post_bounce"

    info "Fetching a sensor bundle for cluster 'remote'"
    rm -rf sensor-remote
    "$TEST_ROOT/bin/$TEST_HOST_OS/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
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

deploy_earlier_central() {
    info "Deploying: $EARLIER_TAG..."

    mkdir -p "bin/$TEST_HOST_OS"
    gsutil cp "gs://stackrox-ci/roxctl-$EARLIER_TAG" "bin/$TEST_HOST_OS/roxctl"
    chmod +x "bin/$TEST_HOST_OS/roxctl"
    PATH="bin/$TEST_HOST_OS:$PATH" command -v roxctl
    PATH="bin/$TEST_HOST_OS:$PATH" roxctl version
    PATH="bin/$TEST_HOST_OS:$PATH" \
    MAIN_IMAGE_TAG="$EARLIER_TAG" \
    SCANNER_IMAGE="$REGISTRY/scanner:$(cat SCANNER_VERSION)" \
    SCANNER_DB_IMAGE="$REGISTRY/scanner-db:$(cat SCANNER_VERSION)" \
    ./deploy/k8s/central.sh

    get_central_basic_auth_creds
}

restore_backup_test() {
    info "Restoring a 56.1 backup into a 58.x central"

    restore_56_1_backup
}

force_rollback() {
    info "Forcing a rollback to $FORCE_ROLLBACK_VERSION"

    local upgradeStatus
    upgradeStatus="$(curl -sSk -X GET -u "admin:$ROX_PASSWORD" "https://$API_ENDPOINT/v1/centralhealth/upgradestatus")"
    echo "upgrade status: $upgradeStatus"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.version' -r)" "$(make --quiet tag)"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.forceRollbackTo' -r)" "$FORCE_ROLLBACK_VERSION"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.canRollbackAfterUpgrade' -r)" "true"
    test_gt_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.spaceAvailableForRollbackAfterUpgrade' -r)" "$(echo "$upgradeStatus" | jq '.upgradeStatus.spaceRequiredForRollbackAfterUpgrade' -r)"

    kubectl -n stackrox get configmap/central-config -o yaml | yq e '{"data": .data}' - >/tmp/force_rollback_patch
    local central_config
    central_config=$(yq e '.data["central-config.yaml"]' /tmp/force_rollback_patch | yq e ".maintenance.forceRollbackVersion = \"$FORCE_ROLLBACK_VERSION\"" -)
    local config_patch
    config_patch=$(yq e ".data[\"central-config.yaml\"] |= \"$central_config\"" /tmp/force_rollback_patch)
    echo "config patch: $config_patch"

    kubectl -n stackrox patch configmap/central-config -p "$config_patch"
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$FORCE_ROLLBACK_VERSION"
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
