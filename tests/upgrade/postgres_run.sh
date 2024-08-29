#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests upgrade to Postgres.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

EARLIER_TAG="4.1.3"
EARLIER_SHA="8a4677e3d45ebc4f065ec052d1d66d6ead2084bb"
CURRENT_TAG="$(make --quiet --no-print-directory tag)"
PREVIOUS_RELEASES=("4.1.3" "4.2.5" "4.3.8" "4.4.5" "4.5.1")

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
    info "Starting upgrade test"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    # repo for old version with legacy database
    REPO_FOR_TIME_TRAVEL="/tmp/rox-postgres-upgrade-test"
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="stackrox-io"
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
    setup_podsecuritypolicies_config
    remove_existing_stackrox_resources

    touch "${UPGRADE_PROGRESS_POSTGRES_PREP}"

    test_upgrade_paths "$log_output_dir"
}

test_upgrade_paths() {
    info "Testing various upgrade paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    # To test we remain backwards compatible rollback to 4.1.x
    FORCE_ROLLBACK_VERSION="4.1.3"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    # There is an issue on gke v1.24 for these older releases where we may have a
    # timeout trying to get the metadata for the cloud provider.  Rather than extend
    # the general wait_for_api time period and potentially hide issues from other
    # tests we will extend the wait period for these tests.
    export MAX_WAIT_SECONDS=600

    ########################################################################################
    # Use roxctl to generate helm files and deploy older central                           #
    ########################################################################################
    deploy_earlier_postgres_central
    wait_for_api
    setup_client_TLS_certs

    restore_4_1_backup
    wait_for_api

    # Run with some scale to have data populated to migrate
    deploy_scaled_workload

    # Get the API_TOKEN for the upgrades
    export API_TOKEN="$(roxcurl /v1/apitokens/generate -d '{"name": "helm-upgrade-test", "role": "Admin"}' | jq -r '.token')"

    cd "$TEST_ROOT"

    # This test does a lot of upgrades and bounces.  If things take a little longer to bounce we can get entries in
    # logs indicating communication problems.  Those need to be allowed in the case of this test ONLY.
    cp scripts/ci/logcheck/allowlist-patterns /tmp/allowlist-patterns
    echo "# postgres was bounced, may see some connection errors" >> /tmp/allowlist-patterns
    echo "FATAL: terminating connection due to administrator command \(SQLSTATE 57P01\)" >> /tmp/allowlist-patterns
    echo "Unable to connect to Sensor at" >> /tmp/allowlist-patterns
    echo "No suitable kernel object downloaded for kernel" >> /tmp/allowlist-patterns
    echo "Unexpected HTTP request failure" >> /tmp/allowlist-patterns
    echo "UNEXPECTED:  Unknown message type" >> /tmp/allowlist-patterns
    # bouncing the database can result in this error
    echo "FATAL: the database system is shutting down" >> /tmp/allowlist-patterns
    # Using ci_export so the post tests have this as well
    ci_export ALLOWLIST_FILE "/tmp/allowlist-patterns"

    # Add some Postgres Access Scopes.  These should not survive a rollback.
    createPostgresScopes
    checkForPostgresAccessScopes

    touch "${UPGRADE_PROGRESS_POSTGRES_EARLIER_CENTRAL}"

    ########################################################################################
    # Bounce central to ensure everything starts back up.                                  #
    ########################################################################################
    info "Bouncing central"
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api
    sensor_wait
    # Bounce collectors to avoid restarts on initial module pull
    kubectl -n stackrox delete pod -l app=collector --grace-period=0

    # Verify data is still there
    checkForPostgresAccessScopes

    validate_upgrade "01-bounce-after-upgrade" "bounce after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "01_post_bounce"

    touch "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_BOUNCE}"

    ########################################################################################
    # Bounce central-db to ensure central recovers from the database outage.               #
    ########################################################################################
    info "Bouncing central-db"
    # Extend the MUTEX timeout just for this case as a restart of the db will cause locks to be held longer as it should
    kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=600
    wait_for_api
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central-db -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api
    wait_for_central_db

    # Verify data is still there
    checkForPostgresAccessScopes

    validate_upgrade "02-bounce-db-after-upgrade" "bounce central db after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "02_post_bounce-db"

    # Ensure central is ready for requests after any previous tests
    wait_for_api

    touch "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_DB_BOUNCE}"

    ########################################################################################
    # Upgrade back to latest to run the smoke tests by first walking previous releases     #
    ########################################################################################
    for str in ${PREVIOUS_RELEASES[@]}; do
      info "Walking upgrades -- release => $str"
      kubectl -n stackrox set image deploy/central "*=$REGISTRY/main:$str"
      kubectl -n stackrox set image deploy/central-db "*=$REGISTRY/central-db:$str"

      wait_for_api
      wait_for_central_db
    done

    ########################################################################################
    # Upgrade to current in order to run any Postgres -> Postgres migrations               #
    ########################################################################################
    kubectl -n stackrox set image deploy/central "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image deploy/central-db "*=$REGISTRY/central-db:$CURRENT_TAG"
    wait_for_api

    # Verify data is still there
    checkForPostgresAccessScopes

    validate_upgrade "03_postgres_postgres_upgrade" "Upgrade Postgres backed central" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "03_postgres_postgres_upgrade"

    touch "${UPGRADE_PROGRESS_POSTGRES_MIGRATIONS}"

    ########################################################################################
    # Rollback to the previous Postgres                                                    #
    ########################################################################################
    info "Rolling back to previous version with Postgres still enabled"
    force_rollback_to_previous_postgres
    wait_for_api

    validate_upgrade "04_postgres_postgres_rollback" "Rollback Postgres backed central" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "04_postgres_postgres_rollback"

    # Ensure central is ready for requests after any previous tests
    wait_for_api

    touch "${UPGRADE_PROGRESS_POSTGRES_ROLLBACK}"

    # Now go back to the current release
    kubectl -n stackrox set image deploy/central "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image deploy/central-db "*=$REGISTRY/central-db:$CURRENT_TAG"

    wait_for_api

    # Cleanup the scaled sensor before smoke tests
    helm uninstall -n stackrox stackrox-secured-cluster-services

    # Remove scaled Sensor from Central
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" cluster delete --name scale-remote

    info "Fetching a sensor bundle for cluster 'remote'"
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" version
    rm -rf sensor-remote
    "$TEST_ROOT/bin/$TEST_HOST_PLATFORM/roxctl" -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    info "Installing sensor"
    ./sensor-remote/sensor.sh
    kubectl -n stackrox set image deploy/sensor "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image deploy/admission-control "*=$REGISTRY/main:$CURRENT_TAG"
    kubectl -n stackrox set image ds/collector "collector=$REGISTRY/collector:$(make collector-tag)" \
        "compliance=$REGISTRY/main:$CURRENT_TAG"
    if [[ "$(kubectl -n stackrox get ds/collector -o=jsonpath='{$.spec.template.spec.containers[*].name}')" == *"node-inventory"* ]]; then
        echo "Upgrading node-inventory container"
        kubectl -n stackrox set image ds/collector "node-inventory=$REGISTRY/scanner-slim:$(make scanner-tag)"
    else
        echo "Skipping node-inventory container as this is not Openshift 4"
    fi

    sensor_wait
    # Bounce collectors to avoid restarts on initial module pull
    kubectl -n stackrox delete pod -l app=collector --grace-period=0

    wait_for_central_reconciliation

    rm -f FAIL
    remove_qa_test_results

    info "Running smoke tests"
    CLUSTER="$CLUSTER_TYPE_FOR_TEST" make -C qa-tests-backend smoke-test || touch FAIL
    store_qa_test_results "upgrade-paths-smoke-tests"
    [[ ! -f FAIL ]] || die "Smoke tests failed"

    touch "${UPGRADE_PROGRESS_POSTGRES_SMOKE_TESTS}"

    collect_and_check_stackrox_logs "$log_output_dir" "04_final"
}

force_rollback_to_previous_postgres() {
    info "Forcing a rollback to $FORCE_ROLLBACK_VERSION"

    local upgradeStatus
    upgradeStatus=$(curl -sSk -X GET --config <(curl_cfg user "admin:${ROX_PASSWORD}") https://"${API_ENDPOINT}"/v1/centralhealth/upgradestatus)
    echo "upgrade status: ${upgradeStatus}"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.version' -r)" "${CURRENT_TAG}"
    test_equals_non_silent "$(echo "$upgradeStatus" | jq '.upgradeStatus.canRollbackAfterUpgrade' -r)" "true"

    kubectl -n stackrox get configmap/central-config -o yaml | yq e '{"data": .data}' - >/tmp/force_rollback_patch
    local central_config
    central_config=$(yq e '.data["central-config.yaml"]' /tmp/force_rollback_patch | yq e ".maintenance.forceRollbackVersion = \"$FORCE_ROLLBACK_VERSION\"" -)
    local config_patch
    config_patch=$(yq e ".data[\"central-config.yaml\"] |= \"$central_config\"" /tmp/force_rollback_patch)
    echo "config patch: $config_patch"

    # downgrading to a version that does not understand process listening on ports
    # so turning that off in sensor and collector to prevent central crashes.
    # Sensor and Collector will be deleted a few steps after this so no need
    # to turn these back on.  Going forward unexpected messages will result in
    # an `UNEXPECTED` log instead of crashing central.  However that change is
    # not present in the initial 3.74 version.
    kubectl -n stackrox set env deploy/sensor ROX_PROCESSES_LISTENING_ON_PORT=false

    kubectl -n stackrox patch configmap/central-config -p "$config_patch"
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$FORCE_ROLLBACK_VERSION"
    kubectl -n stackrox set image deploy/central-db "*=$REGISTRY/central-db:$FORCE_ROLLBACK_VERSION"
}

deploy_scaled_workload() {
    info "Deploying a scaled workload"

    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl version

    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl helm output secured-cluster-services --image-defaults opensource --output-dir /tmp/early-stackrox-secured-services-chart --remove

    PATH="bin/$TEST_HOST_PLATFORM:$PATH" roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" central init-bundles generate scale-remote --output /tmp/cluster-init-bundle.yaml

    helm install -n stackrox --create-namespace \
        stackrox-secured-cluster-services /tmp/early-stackrox-secured-services-chart \
        -f /tmp/cluster-init-bundle.yaml \
        --set system.enablePodSecurityPolicies=false \
        --set clusterName=scale-remote \
        --set image.main.tag="${EARLIER_TAG}" \
        --set image.collector.tag="${EARLIER_TAG}" \
        --set centralEndpoint="$API_ENDPOINT"

    sensor_wait

    ./scale/launch_workload.sh scale-test
    wait_for_api

    info "Sleep for a bit to let the scale build"
    # shellcheck disable=SC2034
    for i in $(seq 1 150); do
        echo -n .
        sleep 5
    done

    info "Done with our nap for scaling"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
