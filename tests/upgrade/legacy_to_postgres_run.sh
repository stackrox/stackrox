#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests upgrade to Postgres.
# NOTE:  Version 3.74 was final release major version 3.
# The switch to major version 4.0 brought with it a complete database
# change.  RocksDB was replaced with Postgres.  With release 4.5 ACS will
# no longer support upgrading from a 3.X release to 4.5 and beyond.  For,
# posterity we will use this test to verify that a 3.X release can be upgraded
# to 4.4.  This test will also verify that an attempt to upgraded from
# a 3.X release to 4.5 will result in an error.
# TODO(ROX-23154) will add a test to ensure an upgrade from RocksDB to 4.5 will return an error

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
LAST_POSTGRES_TAG="4.4.0"
CURRENT_TAG="$(make --quiet --no-print-directory tag)"

# shellcheck source=../../scripts/lib.sh
source "$TEST_ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/lib.sh
source "$TEST_ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$TEST_ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$TEST_ROOT/tests/scripts/setup-certs.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$TEST_ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/upgrade/lib.sh
source "$TEST_ROOT/tests/upgrade/lib.sh"
# shellcheck source=../../tests/upgrade/validation.sh
source "$TEST_ROOT/tests/upgrade/validation.sh"

test_upgrade() {
    info "Starting Rocks to 4.4 Postgres back to Rocks at 3.74 upgrade/rollback test"

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
    DEPLOY_DIR="deploy/k8s"
    QUAY_REPO="stackrox-io"
    REGISTRY="quay.io/$QUAY_REPO"

    export OUTPUT_FORMAT="helm"
    export STORAGE="pvc"
    export CLUSTER_TYPE_FOR_TEST=K8S

    preamble
    setup_deployment_env false false
    setup_podsecuritypolicies_config
    remove_existing_stackrox_resources
    touch "${UPGRADE_PROGRESS_LEGACY_PREP}"

    test_upgrade_paths "$log_output_dir"

    helm_uninstall_and_cleanup
}

test_upgrade_paths() {
    info "Testing various upgrade paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    EARLIER_TAG="3.74.0"
    FORCE_ROLLBACK_VERSION="$EARLIER_TAG"

    ########################################################################################
    # Use roxctl to generate helm files and deploy older central backed by RocksDB         #
    ########################################################################################
    deploy_earlier_rocks_central
    wait_for_api
    setup_client_TLS_certs

    restore_3_56_1_backup
    wait_for_api

    # Add some access scopes and see that they survive the upgrade and rollback process
    createRocksDBScopes
    checkForRocksAccessScopes

    # Get the API_TOKEN for the upgrades
    API_TOKEN="$(roxcurl /v1/apitokens/generate -d '{"name": "helm-upgrade-test", "role": "Admin"}' | jq -r '.token')"
    export API_TOKEN

    cd "$TEST_ROOT"
    download_roxctl "${LAST_POSTGRES_TAG}"

    touch "${UPGRADE_PROGRESS_LEGACY_ROCKSDB_CENTRAL}"

    ########################################################################################
    # Use helm to upgrade to current Postgres release.                                     #
    ########################################################################################
    info "Upgrade to ${LAST_POSTGRES_TAG} via helm"
    helm_upgrade_to_last_postgres
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

    touch "${UPGRADE_PROGRESS_LEGACY_TO_RELEASE}"

    ########################################################################################
    # Flip the Postgres flag to go back to RocksDB                                         #
    ########################################################################################

    # Need to push the flag to ci so that the collect scripts pull from
    # Postgres and not Rocks
    ci_export ROX_POSTGRES_DATASTORE "false"
    LAST_ROCKS_TAG="3.74.0-1-gfe924fce30"
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:${LAST_ROCKS_TAG}"; kubectl -n stackrox set env deploy/central ROX_POSTGRES_DATASTORE=false
    wait_for_api
    wait_for_scanner_to_be_ready

    # Check the database status to ensure it is using RocksDB
    check_legacy_db_status

    # Ensure we still have the access scopes added to Rocks
    checkForRocksAccessScopes
    # The scopes added after the initial upgrade to Postgres should no longer exist.
    verifyNoPostgresAccessScopes

    # Returned to Rocks by flipping the flag.  Validate that RocksDB backed central functions.
    validate_upgrade "01_to_rocks" "central upgrade postgres down to rocks" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    info "Fetching a sensor bundle for cluster 'remote'"
    roxctl version
    rm -rf sensor-remote
    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor get-bundle remote
    [[ -d sensor-remote ]]

    info "Installing sensor"
    # This old software version doesn't remove PSP from the bundle so we have to do it.
    info "Removing pod-security files"
    rm ./sensor-remote/*pod-security.yaml
    ./sensor-remote/sensor.sh
    kubectl -n stackrox set image deploy/sensor "*=$REGISTRY/main:$LAST_ROCKS_TAG"
    kubectl -n stackrox set image deploy/admission-control "*=$REGISTRY/main:$LAST_ROCKS_TAG"
    kubectl -n stackrox set image ds/collector "collector=$REGISTRY/collector:$(make collector-tag)" \
        "compliance=$REGISTRY/main:$LAST_ROCKS_TAG"

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

    touch "${UPGRADE_PROGRESS_RELEASE_BACK_TO_LEGACY}"

    collect_and_check_stackrox_logs "$log_output_dir" "02_final_back_to_Rocks"
}

helm_upgrade_to_last_postgres() {
    info "Helm upgrade to Postgres build ${LAST_POSTGRES_TAG}"

    # Need to push the flag to ci so that the collect scripts pull from
    # Postgres and not Rocks
    ci_export ROX_POSTGRES_DATASTORE "true"
    export CLUSTER="remote"

    roxctl version
    roxctl helm output central-services --image-defaults opensource --output-dir /tmp/stackrox-central-services-chart --remove


    local root_certificate_path="$(mktemp -d)/root_certs_values.yaml"
    create_certificate_values_file "$root_certificate_path"

    ########################################################################################
    # Use helm to upgrade to current Postgres release.                                     #
    ########################################################################################
    helm upgrade -n stackrox stackrox-central-services /tmp/stackrox-central-services-chart \
     --set central.db.enabled=true \
     --set central.db.password.generate=true \
     --set central.db.serviceTLS.generate=true \
     --set central.db.persistence.persistentVolumeClaim.createClaim=true \
     --set central.exposure.loadBalancer.enabled=true \
     -f "$TEST_ROOT/tests/upgrade/scale-values-public.yaml" \
     -f "$root_certificate_path" \
     --force

    # Return back to test root
    cd "$TEST_ROOT"
}

helm_uninstall_and_cleanup() {
    helm uninstall -n stackrox stackrox-central-services
    rm -rf /tmp/stackrox-central-services-chart
    rm -rf /tmp/early-stackrox-central-services-chart
}

check_legacy_db_status() {
    info "Checking the database is RocksDB"

    local dbStatus
    dbStatus=$(curl -sSk -X GET -u "admin:${ROX_PASSWORD}" https://"${API_ENDPOINT}"/v1/database/status)
    echo "database status: ${dbStatus}"
    test_equals_non_silent "$(echo "$dbStatus" | jq '.databaseType' -r)" "RocksDB"
}

restore_3_56_1_backup() {
    info "Restoring a backup from ACS version 3.56.1"

    require_environment "API_ENDPOINT"
    require_environment "ROX_PASSWORD"

    gsutil cp gs://stackrox-ci-upgrade-test-fixtures/upgrade-test-dbs/stackrox_56_1_fixed_upgrade.zip .
    roxctl version
    roxctl -e "$API_ENDPOINT" -p "$ROX_PASSWORD" \
        central db restore --timeout 2m stackrox_56_1_fixed_upgrade.zip
}

deploy_earlier_rocks_central() {
    info "Deploying: $EARLIER_TAG..."

    download_roxctl "${EARLIER_TAG}"

    # Let's try helm
    ROX_PASSWORD="$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c12 || true)"
    roxctl helm output central-services --image-defaults opensource --output-dir /tmp/early-stackrox-central-services-chart

    helm install -n stackrox --create-namespace stackrox-central-services /tmp/early-stackrox-central-services-chart \
         --set central.adminPassword.value="${ROX_PASSWORD}" \
         --set central.db.enabled=false \
         --set central.exposure.loadBalancer.enabled=true \
         --set system.enablePodSecurityPolicies=false \
         --set central.image.tag="${EARLIER_TAG}" \
         --set central.db.image.tag="${EARLIER_TAG}" \
         --set scanner.image.tag="$(cat SCANNER_VERSION)" \
         --set scanner.dbImage.tag="$(cat SCANNER_VERSION)"

    # Installing this way returns faster than the scripts but everything isn't running when it finishes like with
    # the scripts.  So we will give it a minute for things to get started before we proceed
    sleep 60

    ROX_USERNAME="admin"
    ci_export "ROX_USERNAME" "$ROX_USERNAME"
    ci_export "ROX_PASSWORD" "$ROX_PASSWORD"
}

download_roxctl() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: download_roxctl <version>"
    fi

    local version="$1"
    local output_dir="$(mktemp -d)"

    local host_os
    if is_darwin; then
        host_os="darwin"
    elif is_linux; then
        host_os="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    local platform
    case "$(uname -m)" in
        x86_64) platform="" ;;
        ppc64le) platform="-ppc64le" ;;
        s390x) platform="-s390x" ;;
        *) die "Unknown architecture" ;;
    esac

    info "Download roxctl $version"
    curl --location --retry 3 -sS --fail -o "${output_dir}/roxctl" \
        "https://mirror.openshift.com/pub/rhacs/assets/${version}/bin/${host_os}/roxctl${platform}"
    chmod +x "${output_dir}/roxctl"
    export PATH="$output_dir:$PATH"
    roxctl version
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    test_upgrade "$*"
fi
