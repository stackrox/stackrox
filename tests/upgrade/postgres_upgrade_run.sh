#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Tests PostgreSQL major-version upgrades.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

EARLIER_TAG="4.10.0"
EARLIER_SHA="7b817fa511ac4533cdf2d79a1b8e04d4d7557ad2"

# shellcheck source=../../scripts/lib.sh
source "$TEST_ROOT/scripts/lib.sh"

postgres_major_version() {
    # ROX-29205: Before postgres base image, scanner-v4-db used ENV PG_MAJOR.
    local version
    version="$(sed -n \
        -e 's/^[[:space:]]*ARG[[:space:]]\+PG_VERSION=\([1-9][0-9]*\)[[:space:]]*$/\1/p' \
        -e 's/^[[:space:]]*ENV[[:space:]]\+PG_MAJOR=\([1-9][0-9]*\)[[:space:]]*$/\1/p' \
        -e 's/^[[:space:]]*FROM[[:space:]].*\/postgresql-\([1-9][0-9]*\)-.*$/\1/p')"
    # Missing or ambiguous versions must fail the check, not silently skip coverage.
    [[ "$version" =~ ^[1-9][0-9]*$ ]] || return 1
    echo "$version"
}

# bumps_postgres_major sets <prefix>_EARLIER_PG and <prefix>_CURRENT_PG from the
# given Dockerfile, and succeeds if the current checkout bumps the major version.
bumps_postgres_major() {
    local prefix=${1:?missing first required argument: prefix}
    local dockerfile=${2:?missing second required argument: dockerfile}
    local earlier_version current_version

    earlier_version="$(git -C "$TEST_ROOT" show "$EARLIER_SHA:$dockerfile" | postgres_major_version)" || \
        die "Cannot determine the PostgreSQL major version of $dockerfile for $EARLIER_TAG ($EARLIER_SHA)"
    current_version="$(postgres_major_version < "$TEST_ROOT/$dockerfile")" || \
        die "Cannot determine the PostgreSQL major version of $dockerfile for the current checkout"

    printf -v "${prefix}_EARLIER_PG" '%s' "$earlier_version"
    printf -v "${prefix}_CURRENT_PG" '%s' "$current_version"

    (( current_version > earlier_version ))
}

check_postgres_upgrade() {
    # Either database bumping its major version is reason enough to run the
    # suite, and each is asserted on independently: they are separate images
    # with separate upgrade mechanisms, and the two bumps need not land
    # together.
    CENTRAL_DB_UPGRADE_REQUIRED=false
    if bumps_postgres_major CENTRAL_DB "image/postgres/Dockerfile"; then
        CENTRAL_DB_UPGRADE_REQUIRED=true
    fi
    SCANNER_V4_DB_UPGRADE_REQUIRED=false
    if bumps_postgres_major SCANNER_V4_DB "scanner/image/db/Dockerfile"; then
        SCANNER_V4_DB_UPGRADE_REQUIRED=true
    fi

    local central_db="central-db PostgreSQL ${CENTRAL_DB_EARLIER_PG} -> ${CENTRAL_DB_CURRENT_PG}"
    local scanner_v4_db="scanner-v4-db PostgreSQL ${SCANNER_V4_DB_EARLIER_PG} -> ${SCANNER_V4_DB_CURRENT_PG}"

    # check_postgres_version queries central-db, so these track that image.
    EARLIER_POSTGRES_VERSION="$CENTRAL_DB_EARLIER_PG"
    CURRENT_POSTGRES_VERSION="$CENTRAL_DB_CURRENT_PG"

    POSTGRES_UPGRADE_REQUIRED=false
    if [[ "$CENTRAL_DB_UPGRADE_REQUIRED" == true || "$SCANNER_V4_DB_UPGRADE_REQUIRED" == true ]]; then
        POSTGRES_UPGRADE_REQUIRED=true
        POSTGRES_UPGRADE_REASON="Running PostgreSQL upgrade suite against $EARLIER_TAG: ${central_db}, ${scanner_v4_db}."
    else
        POSTGRES_UPGRADE_REASON="Skipping PostgreSQL upgrade suite: neither database bumps its major version against $EARLIER_TAG (${central_db}, ${scanner_v4_db})."
    fi
    info "$POSTGRES_UPGRADE_REASON"
}

check_postgres_version() {
    local expected_version="$1"
    local db_password actual_version

    kubectl -n stackrox rollout status deployment/central-db --timeout="${MAX_WAIT_SECONDS:-300}s" || \
        die "Central DB rollout did not complete"

    db_password="$(kubectl -n stackrox get secret central-db-password -o jsonpath='{.data.password}' | base64 -d)" || \
        die "Cannot read the Central DB password"

    # Pass the password on stdin so it does not appear in the exec command.
    # shellcheck disable=SC2016
    actual_version="$(kubectl -n stackrox exec -i deploy/central-db -c central-db -- \
        bash -c 'PGPASSWORD="$(cat)" \
            psql -X -w -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -Atc "SHOW server_version"' <<< "$db_password")" || \
        die "Cannot query the running PostgreSQL version"

    info "Central DB is running PostgreSQL $actual_version; expected major version $expected_version"
    test_equals_non_silent "${actual_version%%.*}" "$expected_version"
}

SCANNER_V4_PGDATA="/var/lib/postgresql/data/pgdata"

scanner_v4_db_exec() {
    kubectl -n stackrox exec deploy/scanner-v4-db -c db -- bash -ec "$1"
}

# scanner_v4_db_script runs the script on stdin inside the db container, which
# keeps SQL quoting readable -- nothing has to survive a round of shell
# escaping on the way in.
scanner_v4_db_script() {
    kubectl -n stackrox exec -i deploy/scanner-v4-db -c db -- bash -s
}

# scanner_v4_db_pg_version echoes the major version of the data directory on
# the scanner-v4-db volume. That, rather than the server's reported version,
# is what the upgrade actually has to change.
scanner_v4_db_pg_version() {
    scanner_v4_db_exec "cat '$SCANNER_V4_PGDATA/PG_VERSION'" | tr -d '[:space:]'
}

# scanner_v4_db_row_counts echoes "<database>|<table>|<rows>" for every user
# table. Derived from the catalogs rather than from Scanner's schema, so it
# does not need updating when the schema changes.
scanner_v4_db_row_counts() {
    scanner_v4_db_script <<'EOF'
set -eo pipefail
q() { psql -U postgres -h 127.0.0.1 -XAt "$@"; }

for db in $(q -c "SELECT datname FROM pg_database WHERE NOT datistemplate ORDER BY datname"); do
    tables=$(q -d "$db" -c "
        SELECT quote_ident(n.nspname) || '.' || quote_ident(c.relname)
        FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relkind = 'r' AND n.nspname NOT IN ('pg_catalog', 'information_schema')
        ORDER BY 1")
    for tbl in $tables; do
        echo "$db|$tbl|$(q -d "$db" -c "SELECT count(*) FROM $tbl")"
    done
done
EOF
}

# check_scanner_v4_db_upgraded asserts the data directory moved to the major
# version this checkout builds, that no leftovers from the swap survived, and
# that nothing was lost. The row counts are compared against the file written
# before the image bump.
check_scanner_v4_db_upgraded() {
    local baseline=${1:?missing first required argument: baseline-row-counts}
    local version

    version="$(scanner_v4_db_pg_version)"
    test_equals_non_silent "$version" "$SCANNER_V4_DB_CURRENT_PG"

    # A surviving ${PGDATA}-new means the swap never happened; a surviving
    # ${PGDATA}-old means the cleanup after it did not run. Both would be
    # invisible in the version check above.
    scanner_v4_db_exec "test ! -e '${SCANNER_V4_PGDATA}-new'" || \
        die "${SCANNER_V4_PGDATA}-new survived the scanner-v4-db upgrade"
    scanner_v4_db_exec "test ! -e '${SCANNER_V4_PGDATA}-old'" || \
        die "${SCANNER_V4_PGDATA}-old survived the scanner-v4-db upgrade"
    scanner_v4_db_exec "test -e '$SCANNER_V4_PGDATA/.init-ready'" || \
        die "The scanner-v4-db .init-ready sentinel is missing after the upgrade"

    scanner_v4_db_row_counts >/tmp/scanner-v4-db-rows-after
    if ! diff -u "$baseline" /tmp/scanner-v4-db-rows-after; then
        die "scanner-v4-db row counts changed across the upgrade"
    fi
    info "scanner-v4-db is on PostgreSQL $version with all row counts preserved."
}

# check_scanner_v4_no_warmup is the acceptance criterion that matters: losing
# the vulnerability store would force a matcher warm-up, and warm-up is a hard
# scan outage, because the matcher reports FailedPrecondition and Central's
# retry wrapper treats that as permanent rather than retrying.
check_scanner_v4_no_warmup() {
    kubectl -n stackrox rollout status deploy/scanner-v4-matcher --timeout=300s

    if kubectl -n stackrox logs deploy/scanner-v4-matcher --since=20m | grep -q "FailedPrecondition"; then
        kubectl -n stackrox logs deploy/scanner-v4-matcher --since=20m | grep "FailedPrecondition"
        die "The matcher re-entered warm-up after the scanner-v4-db upgrade"
    fi
    info "The matcher did not re-enter warm-up."
}

test_upgrade() {
    info "Starting postgres upgrade test"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade <log-output-dir>"
    fi

    local log_output_dir="$1"

    require_environment "KUBECONFIG"

    export_test_environment

    # repo for old version with legacy database
    REPO_FOR_TIME_TRAVEL="/tmp/rox-postgres-upgrade-test"
    QUAY_REPO="rhacs-eng"
    REGISTRY="quay.io/$QUAY_REPO"

    export CLUSTER_TYPE_FOR_TEST=K8S

    if is_CI; then
        export ROXCTL_IMAGE_REPO="quay.io/$QUAY_REPO/roxctl"
    fi

    preamble
    setup_deployment_env false false
    setup_podsecuritypolicies_config
    remove_existing_stackrox_resources

    touch "${UPGRADE_PROGRESS_POSTGRES_PREP}"

    test_upgrade_path "$log_output_dir"

    roxie teardown both --verbose --single-namespace

    # TODO(cdu): review note, remove before the PR. Two things to weigh here.
    #
    # First, per your call, Central's test_not_enough_disk_space stays and
    # scanner gets its own test rather than replacing it. The T7 subtask text
    # literally says "replace", so if a reviewer quotes the ticket at you,
    # this is the deviation they mean.
    #
    # Second, once both databases bump in the same release this runs two full
    # deploy-and-upgrade cycles on top of the happy path, i.e. three. That is
    # a long job. Worth deciding whether the scanner deferral test should be
    # gated behind a pragma instead of running on every eligible PR.
    #
    # Each disk-space test asserts a different contract, so each only runs
    # when its own database is the one bumping its major version.
    if [[ "$CENTRAL_DB_UPGRADE_REQUIRED" == true ]]; then
        remove_existing_stackrox_resources
        test_not_enough_disk_space "$log_output_dir"
    fi

    if [[ "$SCANNER_V4_DB_UPGRADE_REQUIRED" == true ]]; then
        remove_existing_stackrox_resources
        test_scanner_v4_db_upgrade_deferral "$log_output_dir"
    fi
}

test_upgrade_path() {
    info "Testing upgrade happy paths"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    # There is an issue on gke v1.24 for these older releases where we may have a
    # timeout trying to get the metadata for the cloud provider.  Rather than extend
    # the general wait_for_api time period and potentially hide issues from other
    # tests we will extend the wait period for these tests.
    export MAX_WAIT_SECONDS=600

    ########################################################################################
    # Use roxie to deploy older central                                                    #
    ########################################################################################
    deploy_earlier_tag_central
    wait_for_api
    check_postgres_version "$EARLIER_POSTGRES_VERSION"
    setup_client_TLS_certs

    # Older checkouts force GOTOOLCHAIN=local during dependency checks.
    # Put the toolchain selected for this checkout first on PATH for the build.
    # Needed for restore_backup to find roxctl.
    local build_goroot
    build_goroot="$(go env GOROOT)"
    PATH="${build_goroot}/bin:${PATH}" make cli

    # It's damn fiddly, restore is needed because later test will search for a
    # default secured cluster, created by it :(
    restore_backup
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

    # Extend the MUTEX timeout for this case as a restart of the db will cause locks to be held longer as it should
    kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=600

    # Recorded before the image bump, while the cluster is still on the
    # earlier major version, so the post-upgrade comparison has something to
    # compare against.
    kubectl -n stackrox rollout status deploy/scanner-v4-db --timeout=600s
    scanner_v4_db_row_counts >/tmp/scanner-v4-db-rows-before
    info "scanner-v4-db before the upgrade: PostgreSQL $(scanner_v4_db_pg_version), \
$(wc -l </tmp/scanner-v4-db-rows-before | tr -d ' ') tables"

    # Upgrade the images
    info "Upgrade ${EARLIER_TAG} => ${CURRENT_TAG}"
    kubectl -n stackrox set image deploy/central "*=${REGISTRY}/main:${CURRENT_TAG}"
    kubectl -n stackrox set image deploy/central-db "*=${REGISTRY}/central-db:${CURRENT_TAG}"
    kubectl -n stackrox set image deploy/scanner-v4-db "*=${REGISTRY}/scanner-v4-db:${CURRENT_TAG}"
    kubectl -n stackrox set image deploy/scanner-v4-indexer "*=${REGISTRY}/scanner-v4:${CURRENT_TAG}"
    kubectl -n stackrox set image deploy/scanner-v4-matcher "*=${REGISTRY}/scanner-v4:${CURRENT_TAG}"
    wait_for_api
    check_postgres_version "$CURRENT_POSTGRES_VERSION"

    ########################################################################################
    # scanner-v4-db must have migrated in place, with the vulnerability store  #
    # intact and no matcher warm-up.                                           #
    ########################################################################################
    kubectl -n stackrox rollout status deploy/scanner-v4-db --timeout=600s
    check_scanner_v4_db_upgraded /tmp/scanner-v4-db-rows-before
    check_scanner_v4_no_warmup

    ########################################################################################
    # Bounce central to ensure everything starts back up.                                  #
    ########################################################################################
    info "Bouncing central"
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api
    sensor_wait

    # Verify data is still there
    checkForPostgresAccessScopes

    validate_upgrade "01-bounce-after-upgrade" "bounce after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"
    collect_and_check_stackrox_logs "$log_output_dir" "upgrade_01_post_bounce"

    touch "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_BOUNCE}"

    ########################################################################################
    # Bounce central-db to ensure central recovers from the database outage.               #
    ########################################################################################
    info "Bouncing central-db"
    wait_for_api
    kubectl -n stackrox delete po "$(kubectl -n stackrox get po -l app=central-db -o=jsonpath='{.items[0].metadata.name}')" --grace-period=0
    wait_for_api
    wait_for_central_db
    check_postgres_version "$CURRENT_POSTGRES_VERSION"

    # Verify data is still there
    checkForPostgresAccessScopes

    validate_upgrade "02-bounce-db-after-upgrade" "bounce central db after postgres upgrade" "268c98c6-e983-4f4e-95d2-9793cebddfd7"

    collect_and_check_stackrox_logs "$log_output_dir" "upgrade_02_post_bounce-db"

    # Ensure central is ready for requests after any previous tests
    wait_for_api

    touch "${UPGRADE_PROGRESS_POSTGRES_CENTRAL_DB_BOUNCE}"

    ########################################################################################
    # Bounce scanner-v4-db a second time. This is the regression test for the              #
    # .init-ready sentinel: were it not written before the swap, the init                  #
    # container would see no sentinel on this restart and wipe the database it             #
    # had just upgraded.                                                                   #
    ########################################################################################
    info "Bouncing scanner-v4-db"
    kubectl -n stackrox delete po -l app=scanner-v4-db --grace-period=0
    kubectl -n stackrox rollout status deploy/scanner-v4-db --timeout=600s
    check_scanner_v4_db_upgraded /tmp/scanner-v4-db-rows-before
    check_scanner_v4_no_warmup

    collect_and_check_stackrox_logs "$log_output_dir" "upgrade_03_final"
}

# Verify the upgrade will not proceed without having enough disk space
test_not_enough_disk_space() {
    info "Testing upgrade fail: not enough disk space"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_upgrade_paths <log-output-dir>"
    fi

    local log_output_dir="$1"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    # There is an issue on gke v1.24 for these older releases where we may have a
    # timeout trying to get the metadata for the cloud provider.  Rather than extend
    # the general wait_for_api time period and potentially hide issues from other
    # tests we will extend the wait period for these tests.
    export MAX_WAIT_SECONDS=600

    ########################################################################################
    # Deploy older central using small PVC to face disk space shortage                     #
    ########################################################################################
    deploy_earlier_tag_central \
        --set central.spec.central.db.persistence.persistentVolumeClaim.size=1Gi
    wait_for_api
    check_postgres_version "$EARLIER_POSTGRES_VERSION"
    setup_client_TLS_certs

    # It's damn fiddly, restore is needed because later test will search for a
    # default secured cluster, created by it :(
    restore_backup
    wait_for_api

    # The 4.10 deployment mounts the backup PVC only in init-db. Expose the same
    # volume to central-db so the running container can fill it before upgrading.
    kubectl -n stackrox patch deployment/central-db --type=strategic -p \
        '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","volumeMounts":[{"name":"backup","mountPath":"/var/lib/postgresql/backup"}]}]}}}}'
    kubectl -n stackrox rollout status deployment/central-db --timeout="${MAX_WAIT_SECONDS}s"
    wait_for_api

    # Fill both PVCs: the operator creates a separate backup PVC twice the data PVC size.
    # Neither backup location should have room for a backup and its verification copy.
    # Expand PGDATA inside the container, and use the same KiB units as the entrypoint.
    # shellcheck disable=SC2016
    kubectl -n stackrox exec deploy/central-db -c central-db -- bash -euo pipefail -c '
        for location in /var/lib/postgresql/data /var/lib/postgresql/backup; do
            mounted_at=$(df -Pk "$location" | awk "END {print \$6}")
            if [[ "$mounted_at" != "$location" ]]; then
                echo "Expected a volume mounted at $location, found $mounted_at" >&2
                exit 1
            fi
        done

        fallocate -l 700mb "${PGDATA}/fake_data"
        fallocate -l 1GiB /var/lib/postgresql/backup/fake_data
        df -H "$PGDATA" /var/lib/postgresql/backup
        du -sch "$PGDATA"/* /var/lib/postgresql/backup/fake_data

        required=$(du -sk "$PGDATA" | awk "{print \$1 * 2}")
        for location in /var/lib/postgresql/data /var/lib/postgresql/backup; do
            available=$(df -Pk "$location" | awk "END {print \$4}")
            echo "$location: $available KiB available; $required KiB required for backup and verification"
            if (( available >= required )); then
                echo "Low-space fixture still has enough space at $location" >&2
                exit 1
            fi
        done
    '

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

    # Extend the MUTEX timeout for this case as a restart of the db will cause locks to be held longer as it should
    kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=600

    # Upgrade the image to PG16
    info "Upgrade ${EARLIER_TAG} => ${CURRENT_TAG}"
    kubectl -n stackrox set image \
        deploy/central "*=${REGISTRY}/main:${CURRENT_TAG}"
    kubectl -n stackrox set image \
        deploy/central-db "*=${REGISTRY}/central-db:${CURRENT_TAG}"
    wait_for_log_line stackrox deployment/central-db init-db \
        "Not enough disk space, upgrade is cancelled"

    kubectl -n stackrox patch pvc/central-db -p \
        '{"spec": {"resources": {"requests": {"storage": "4Gi"}}}}'
    kubectl -n stackrox rollout restart deployment/central-db
    wait_for_api
    check_postgres_version "$CURRENT_POSTGRES_VERSION"

    # Make sure we can restore from a physical backup and run with old binaries
    # after upgrade if needed.
    kubectl -n stackrox set env deploy/central-db RESTORE_BACKUP=true FORCE_OLD_BINARIES=true
    wait_for_api
    check_postgres_version "$EARLIER_POSTGRES_VERSION"

    collect_and_check_stackrox_logs "$log_output_dir" "disk_space_01_final"
}

# Verify scanner-v4-db defers the upgrade rather than aborting when pg_upgrade
# cannot run, and completes it once the obstruction is removed.
#
# This is deliberately not a port of test_not_enough_disk_space above.
# central-db exits 1 and waits for an operator; scanner-v4-db must not, because
# a crashlooping init-db takes scanning offline, and the matcher reports
# FailedPrecondition during a vulnerability store reload, which Central treats
# as permanent. So the assertion is the opposite one: the pod comes up anyway,
# on the previous major version, with the data intact.
test_scanner_v4_db_upgrade_deferral() {
    info "Testing scanner-v4-db upgrade deferral"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: test_scanner_v4_db_upgrade_deferral <log-output-dir>"
    fi

    local log_output_dir="$1"

    cd "$REPO_FOR_TIME_TRAVEL"
    git checkout "$EARLIER_SHA"

    export MAX_WAIT_SECONDS=600

    deploy_earlier_tag_central
    wait_for_api
    setup_client_TLS_certs
    restore_backup
    wait_for_api

    kubectl -n stackrox rollout status deploy/scanner-v4-db --timeout=600s
    scanner_v4_db_row_counts >/tmp/scanner-v4-db-rows-before

    cd "$TEST_ROOT"

    # Point maybe-upgrade at the previous major's pg_upgrade, which the image
    # ships alongside the current one. It rejects the new data directory that
    # initdb creates at the current major, so every transfer mode probe fails
    # and the upgrade takes its give-up path. Filling the volume would not do:
    # "pg_upgrade --clone" needs no second copy, so on a reflink-capable file
    # system a full volume does not stop the upgrade at all.
    local old_bindir="/usr/lib64/pgsql/postgresql-${SCANNER_V4_DB_EARLIER_PG}/bin"
    kubectl -n stackrox set env deploy/scanner-v4-db "NEW_BINARIES=${old_bindir}"

    info "Upgrade ${EARLIER_TAG} => ${CURRENT_TAG}"
    kubectl -n stackrox set image deploy/scanner-v4-db "*=${REGISTRY}/scanner-v4-db:${CURRENT_TAG}"

    # The whole point: the pod comes up, rather than the init container
    # aborting and crashlooping.
    kubectl -n stackrox rollout status deploy/scanner-v4-db --timeout=600s
    kubectl -n stackrox logs deploy/scanner-v4-db -c init-db

    # docker-entrypoint.sh starts the binaries matching the data directory, so
    # a declined upgrade leaves a working server on the previous major.
    test_equals_non_silent "$(scanner_v4_db_pg_version)" "$SCANNER_V4_DB_EARLIER_PG"
    scanner_v4_db_row_counts >/tmp/scanner-v4-db-rows-deferred
    if ! diff -u /tmp/scanner-v4-db-rows-before /tmp/scanner-v4-db-rows-deferred; then
        die "scanner-v4-db lost data while deferring the upgrade"
    fi
    check_scanner_v4_no_warmup

    # Removing the override restores the current binaries and rolls the
    # deployment, which retries the upgrade.
    info "Restoring the current binaries and restarting scanner-v4-db"
    kubectl -n stackrox set env deploy/scanner-v4-db NEW_BINARIES-
    kubectl -n stackrox rollout status deploy/scanner-v4-db --timeout=600s
    check_scanner_v4_db_upgraded /tmp/scanner-v4-db-rows-before
    check_scanner_v4_no_warmup

    collect_and_check_stackrox_logs "$log_output_dir" "scanner_v4_db_deferral_final"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_postgres_upgrade
    if [[ "${1:-}" == "--check" ]]; then
        # CI checks eligibility before provisioning a cluster or running pre/post-test hooks.
        if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
            echo "required=$POSTGRES_UPGRADE_REQUIRED" >> "$GITHUB_OUTPUT"
        fi
        if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
            echo "$POSTGRES_UPGRADE_REASON" >> "$GITHUB_STEP_SUMMARY"
        fi
    elif [[ "$POSTGRES_UPGRADE_REQUIRED" == true ]]; then
        CURRENT_TAG="${MAIN_IMAGE_TAG:-"$(make --quiet --no-print-directory tag)"}"
        # Load deployment helpers only when the suite will actually run.
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
        test_upgrade "$@"
    fi
fi
