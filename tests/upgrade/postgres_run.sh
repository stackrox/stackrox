#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail
set -x

# Tests upgrade to Postgres.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/lib.sh"
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
source "$TEST_ROOT/tests/e2e/lib.sh"
source "$TEST_ROOT/tests/scripts/setup-certs.sh"

test_upgrade() {
    info "Starting Postgres upgrade test"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    REPO_FOR_TIME_TRAVEL="/tmp/rox-postgres-upgrade-test"
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="rhacs-eng"
    REGISTRY="quay.io/$QUAY_REPO"

    export OUTPUT_FORMAT="helm"
    export STORAGE="pvc"
    export CLUSTER_TYPE_FOR_TEST=K8S

    export ROXCTL_IMAGE_REPO="quay.io/$QUAY_REPO/roxctl"
    if is_CI; then
        require_environment "LONGTERM_LICENSE"
        export ROX_LICENSE_KEY="${LONGTERM_LICENSE}"
    fi

    preamble
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs
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

test_upgrade_paths() {
    info "Testing various upgrade paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    #EARLIER_SHA="9f82d2713cfec4b5c876d8dc0149f6d9cd70d349"
    #EARLIER_TAG="3.63.x-163-g2c4fe1563c"
    EARLIER_SHA="270d517e9572a8301a62e93d9299bd66c15ef4f8"
    EARLIER_TAG="3.71.x-200-g270d517e95"
    FORCE_ROLLBACK_VERSION="$EARLIER_TAG"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    deploy_earlier_central
    wait_for_api

    # use postgres
    export ROX_POSTGRES_DATASTORE="true"

    cd "$TEST_ROOT"
    helm_upgrade_to_current
    wait_for_api

    info "Waiting for scanner to be ready"
    wait_for_scanner_to_be_ready

    validate_upgrade "upgrade" "upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "00_initial_check"

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

# Need to move some of the functions to a lib if no change of course
deploy_earlier_central() {
    info "Deploying: $EARLIER_TAG..."

    mkdir -p "bin/$TEST_HOST_OS"
    if is_CI; then
        gsutil cp "gs://stackrox-ci/roxctl-$EARLIER_TAG" "bin/$TEST_HOST_OS/roxctl"
    else
        make cli
    fi
    #cp newrox bin/darwin/roxctl
    #cp newrox71 bin/darwin/roxctl
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
    info "Restoring a 56.1 backup into a current central"

    restore_56_1_backup
}

validate_upgrade() {
    if [[ "$#" -ne 3 ]]; then
        die "missing args. usage: validate_upgrade <stage name> <stage description> <upgrade_cluster_id>"
    fi

    local stage_name="$1"
    local stage_description="$2"
    local upgrade_cluster_id="$3"
    local policies_dir="../pkg/defaults/policies/files"

    info "Validating the upgrade with upgrade tests: $stage_description"

    CLUSTER="$CLUSTER_TYPE_FOR_TEST" \
        UPGRADE_CLUSTER_ID="$upgrade_cluster_id" \
        POLICIES_JSON_RELATIVE_PATH="$policies_dir" \
        make -C qa-tests-backend upgrade-test || touch FAIL
    store_qa_test_results "validate-upgrade-tests-${stage_name}"
    [[ ! -f FAIL ]] || die "Upgrade tests failed"
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
    if is_CI; then
        roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart
        sed -i 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml
    else
        make cli
        roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart --remove
        sed -i "" 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml
    fi
    yq e '.defaults.central.enableCentralDB=true' -i /tmp/stackrox-central-services-chart/internal/defaults.yaml
    kubectl -n stackrox apply -f $TEST_ROOT/tmp/secret-central-db-password
    kubectl -n stackrox apply -f $TEST_ROOT/tmp/01-central-11-pvc.yaml
    create_db_tls_secret
    kubectl -n stackrox set env deploy/central ROX_POSTGRES_DATASTORE=true

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

wait_for_central_reconciliation() {
    info "Waiting for central reconciliation"

    # Reconciliation is rather slow in this case, since the central has a DB with a bunch of deployments,
    # none of which exist. So when sensor connects, the reconciliation deletion takes a while to flush.
    # This causes flakiness with the smoke tests.
    # To mitigate this, wait for the deployments to get deleted before running the tests.
    local success=0
    for i in $(seq 1 90); do
        local numDeployments
        numDeployments="$(curl -sSk -u "admin:$ROX_PASSWORD" "https://$API_ENDPOINT/v1/summary/counts" | jq '.numDeployments' -r)"
        echo "Try number ${i}. Number of deployments in Central: $numDeployments"
        [[ -n "$numDeployments" ]]
        if [[ "$numDeployments" -lt 100 ]]; then
            success=1
            break
        fi
        sleep 10
    done
    [[ "$success" == 1 ]]
}

wait_for_scanner_to_be_ready() {
    echo "Waiting for scanner to be ready"
    start_time="$(date '+%s')"
    while true; do
      scanner_json="$(kubectl -n stackrox get deploy/scanner -o json)"
      replicas="$(jq '.status.replicas' <<<"${scanner_json}")"
      readyReplicas="$(jq '.status.readyReplicas' <<<"${scanner_json}")"
      echo "scanner replicas: $replicas"
      echo "scanner readyReplicas: $readyReplicas"
      if [[  "$replicas" == "$readyReplicas" ]]; then
        break
      fi
      if (( $(date '+%s') - start_time > 300 )); then
        kubectl -n stackrox get pod -o wide
        kubectl -n stackrox get deploy -o wide
        echo >&2 "Timed out after 5m"
        exit 1
      fi
      sleep 5
    done
    echo "Scanner is ready"
}

create_db_tls_secret() {
    cert_dir="$(mktemp -d)"
    cd $cert_dir
    kubectl -n stackrox exec -it deployment/central -- cat /run/secrets/stackrox.io/certs/ca.pem > $cert_dir/ca.pem
    kubectl -n stackrox exec -it deployment/central -- cat /run/secrets/stackrox.io/certs/ca-key.pem > $cert_dir/ca.key
    cn="CENTRAL_DB_SERVICE: Central DB"
    openssl genrsa -out key.pem 4096
    openssl req -new -key key.pem -subj "/CN=$cn" > newreq
    echo subjectAltName = DNS:central-db.stackrox.svc > extfile.cnf
    openssl x509 -sha256 -req -CA ca.pem -CAkey ca.key -CAcreateserial -out cert.pem -in newreq -extfile extfile.cnf
    kubectl -n stackrox create secret generic central-db-tls --save-config --dry-run=client --from-file=ca.pem --from-file=cert.pem --from-file=key.pem -o yaml | kubectl apply -f -
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
