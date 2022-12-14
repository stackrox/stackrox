#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests upgrade to Postgres.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

INITIAL_POSTGRES_TAG="3.73.x-75-gbab217c487"
INITIAL_POSTGRES_SHA="bab217c48736c4dbe4757fbb4a61579b3051bd9d"
CURRENT_TAG="$(make --quiet tag)"

source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/lib.sh"
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
source "$TEST_ROOT/tests/e2e/lib.sh"
source "$TEST_ROOT/tests/upgrade/lib.sh"
source "$TEST_ROOT/tests/upgrade/validation.sh"

test_upgrade() {
    info "Starting Rocks to Postgres upgrade test"

    # Need to push the flag to ci so that is where it needs to be for the part
    # of the test.  We start this test with RocksDB
    ci_export ROX_POSTGRES_DATASTORE "false"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    # repo for old version with legacy database
    REPO_FOR_TIME_TRAVEL="/tmp/rox-postgres-upgrade-test"
    # repo for old version with postgres database so we can perform a subsequent
    # postgres->postgres upgrade
    REPO_FOR_POSTGRES_TIME_TRAVEL="/tmp/rox-postgres-postgres-upgrade-test"
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="rhacs-eng"
    REGISTRY="quay.io/$QUAY_REPO"

    export OUTPUT_FORMAT="helm"
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

    info "Will clone or update a clean copy of the rox repo for legacy DB test at $REPO_FOR_TIME_TRAVEL"
    if [[ -d "$REPO_FOR_TIME_TRAVEL" ]]; then
        if is_CI; then
          die "Repo for time travel already exists! This is unexpected in CI."
        fi
        (cd "$REPO_FOR_TIME_TRAVEL" && git checkout master && git reset --hard && git pull)
    else
        (cd "$(dirname "$REPO_FOR_TIME_TRAVEL")" && git clone https://github.com/stackrox/stackrox.git "$(basename "$REPO_FOR_TIME_TRAVEL")")
    fi

    info "Will clone or update a clean copy of the rox repo for Postgres DB test at $REPO_FOR_POSTGRES_TIME_TRAVEL"
        if [[ -d "$REPO_FOR_POSTGRES_TIME_TRAVEL" ]]; then
            if is_CI; then
              die "Repo for time travel already exists! This is unexpected in CI."
            fi
            (cd "$REPO_FOR_POSTGRES_TIME_TRAVEL" && git checkout master && git reset --hard && git pull)
        else
            (cd "$(dirname "$REPO_FOR_POSTGRES_TIME_TRAVEL")" && git clone https://github.com/stackrox/stackrox.git "$(basename "$REPO_FOR_POSTGRES_TIME_TRAVEL")")
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

    EARLIER_SHA="9f82d2713cfec4b5c876d8dc0149f6d9cd70d349"
    EARLIER_TAG="3.63.x-163-g2c4fe1563c"
    FORCE_ROLLBACK_VERSION="$EARLIER_TAG"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    deploy_earlier_central
    wait_for_api
    setup_client_TLS_certs
    restore_backup_test
    wait_for_api

    # Add some access scopes and see that they survive the upgrade and rollback process
    createRocksDBScopes
    checkForRocksAccessScopes

    # Grab a backup from rocks db to use later
    backup_dir="$(mktemp -d)"
    info "Backing up to ${backup_dir}"
    roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central backup --output "${backup_dir}" || touch DB_TEST_FAIL
    [[ ! -f DB_TEST_FAIL ]] || die "The DB test failed"

    export API_TOKEN="$(roxcurl /v1/apitokens/generate -d '{"name": "helm-upgrade-test", "role": "Admin"}' | jq -r '.token')"

    cd "$TEST_ROOT"

    helm_upgrade_to_postgres
    wait_for_api
    wait_for_scanner_to_be_ready

    # Upgraded to Postgres via helm.  Validate the upgrade.
    validate_upgrade "00_upgrade" "central upgrade to postgres" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    # Ensure the access scopes added to rocks still exist after the upgrade
    checkForRocksAccessScopes

    collect_and_check_stackrox_logs "$log_output_dir" "00_initial_check"

    # Add some Postgres Access Scopes.  These should not survive a rollback.
    createPostgresScopes
    checkForPostgresAccessScopes

    force_rollback_to_legacy
    wait_for_api
    wait_for_scanner_to_be_ready

    # We have rolled back, make sure access scopes added to Rocks still exist
    checkForRocksAccessScopes
    # The scopes added after the initial upgrade to Postgres should no longer exist.
    verifyNoPostgresAccessScopes

    # Now go back up to Postgres
    kubectl -n stackrox set env deploy/central ROX_POSTGRES_DATASTORE=true
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$INITIAL_POSTGRES_TAG"
    wait_for_api
    wait_for_scanner_to_be_ready

    # Ensure we still have the access scopes added to Rocks
    checkForRocksAccessScopes
    # The scopes added after the initial upgrade to Postgres should no longer exist.
    verifyNoPostgresAccessScopes

    # Add the Postgres access scopes back in
    createPostgresScopes

    info "Bouncing central"
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api

    checkForRocksAccessScopes
    checkForPostgresAccessScopes

    validate_upgrade "01-bounce-after-upgrade" "bounce after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "01_post_bounce"

    info "Bouncing central-db"
    # Extend the MUTEX timeout just for this case as a restart of the db will cause locks to be held longer as it should
    kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=600
    wait_for_api
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central-db -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api
    wait_for_central_db

    checkForRocksAccessScopes
    checkForPostgresAccessScopes

    validate_upgrade "02-bounce-db-after-upgrade" "bounce central db after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    # Since we bounced the DB we may see some errors.  Those need to be allowed in the case of this test ONLY.
    echo "# postgres was bounced, may see some connection errors" >> scripts/ci/logcheck/allowlist-patterns
    echo "FATAL: terminating connection due to administrator command \(SQLSTATE 57P01\)" >> scripts/ci/logcheck/allowlist-patterns
    echo >> scripts/ci/logcheck/allowlist-patterns

    collect_and_check_stackrox_logs "$log_output_dir" "02_post_bounce-db"

    # Ensure central is ready for requests after any previous tests
    wait_for_api

    # Now lets restore from a RocksDB based stackrox backup
    info "Restoring from ${backup_dir}/stackrox_db_*"
    roxctl -e "${API_ENDPOINT}" -p "${ROX_PASSWORD}" central db restore --timeout 2m "${backup_dir}"/stackrox_db_* || touch DB_TEST_FAIL
    [[ ! -f DB_TEST_FAIL ]] || die "The DB test failed"

    wait_for_api

    # Ensure we still have the access scopes added to Rocks
    checkForRocksAccessScopes
    # The scopes added after the initial upgrade to Postgres should no longer exist.
    verifyNoPostgresAccessScopes

    validate_upgrade "03_restore_rocks_to_postgres" "restore rocks db to Postgres" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "03_restore_rocks_to_postgres"

    # Now lets try a Postgres->Postgres upgrade
    kubectl -n stackrox set image deploy/central "*=$REGISTRY/main:$CURRENT_TAG"
    wait_for_api
    # Ensure we still have the access scopes added to Rocks
    checkForRocksAccessScopes

    validate_upgrade "04_postgres_postgres_upgrade" "Upgrade Postgres backed central" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "04_postgres_postgres_upgrade"

    info "Fetching a sensor bundle for cluster 'remote'"
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" version
    rm -rf sensor-remote
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    info "Installing sensor"
    ./sensor-remote/sensor.sh
    kubectl -n stackrox set image deploy/sensor "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image deploy/admission-control "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image ds/collector "collector=$REGISTRY/collector:$(cat COLLECTOR_VERSION)" \
        "compliance=$REGISTRY/main:$CURRENT_TAG"

    sensor_wait

    wait_for_central_reconciliation

    info "Running smoke tests"
    CLUSTER="$CLUSTER_TYPE_FOR_TEST" make -C qa-tests-backend smoke-test || touch FAIL
    store_qa_test_results "upgrade-paths-smoke-tests"
    [[ ! -f FAIL ]] || die "Smoke tests failed"

    collect_and_check_stackrox_logs "$log_output_dir" "05_final"
}

helm_upgrade_to_postgres() {
    info "Helm upgrade to Postgres build ${INITIAL_POSTGRES_TAG}"

    cd "$REPO_FOR_POSTGRES_TIME_TRAVEL"
    git checkout "$INITIAL_POSTGRES_SHA"

    # use postgres
    export ROX_POSTGRES_DATASTORE="true"
    # Need to push the flag to ci so that the collect scripts pull from
    # Postgres and not Rocks
    ci_export ROX_POSTGRES_DATASTORE "true"
    export CLUSTER="remote"

    # Get opensource charts and convert to development_build to support release builds
    if is_CI; then
        make cli
        PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl version
        PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart
        sed -i 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml
    else
        make cli
        roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart --remove
        sed -i "" 's#quay.io/stackrox-io#quay.io/rhacs-eng#' /tmp/stackrox-central-services-chart/internal/defaults.yaml
    fi

    # enable postgres
    password=`echo ${RANDOM}_$(date +%s-%d-%M) |base64|cut -c 1-20`
    kubectl -n stackrox create secret generic central-db-password --from-literal=password=$password
    kubectl -n stackrox apply -f $TEST_ROOT/tests/upgrade/pvc.yaml
    create_db_tls_secret

    helm upgrade -n stackrox stackrox-central-services /tmp/stackrox-central-services-chart --set central.db.enabled=true --set central.exposure.loadBalancer.enabled=true --force

    # return back to test root
    cd "$TEST_ROOT"
}

create_db_tls_secret() {
    echo "Create certificates for central db"

    cert_dir="$(mktemp -d)"
    # get root ca
    kubectl -n stackrox exec -i deployment/central -- cat /run/secrets/stackrox.io/certs/ca.pem > $cert_dir/ca.pem
    kubectl -n stackrox exec -i deployment/central -- cat /run/secrets/stackrox.io/certs/ca-key.pem > $cert_dir/ca.key
    # generate central-db certs
    openssl genrsa -out $cert_dir/key.pem 4096
    openssl req -new -key $cert_dir/key.pem -subj "/CN=CENTRAL_DB_SERVICE: Central DB" > $cert_dir/newreq
    echo subjectAltName = DNS:central-db.stackrox.svc > $cert_dir/extfile.cnf
    openssl x509 -sha256 -req -CA $cert_dir/ca.pem -CAkey $cert_dir/ca.key -CAcreateserial -out $cert_dir/cert.pem -in $cert_dir/newreq -extfile $cert_dir/extfile.cnf
    # create secret
    kubectl -n stackrox create secret generic central-db-tls --save-config --dry-run=client --from-file=$cert_dir/ca.pem --from-file=$cert_dir/cert.pem --from-file=$cert_dir/key.pem -o yaml | kubectl apply -f -
}

force_rollback_to_legacy() {
    info "Forcing a rollback to $FORCE_ROLLBACK_VERSION"

    local upgradeStatus
    upgradeStatus=$(curl -sSk -X GET -u "admin:${ROX_PASSWORD}" https://"${API_ENDPOINT}"/v1/centralhealth/upgradestatus)
    echo "upgrade status: ${upgradeStatus}"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.version' -r)" "${INITIAL_POSTGRES_TAG}"
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

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
