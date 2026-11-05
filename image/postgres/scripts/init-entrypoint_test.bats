#!/usr/bin/env bats

# Bats isolates each test in a subshell; exported fixture settings are per test.
# shellcheck disable=SC2030,SC2031

# Exercise the actual entrypoint's filesystem operations and control flow, using
# temporary paths and fake PostgreSQL/space checks. Real upgrades need a DB image.
setup() {
    export TEST_ROOT="${BATS_TEST_TMPDIR}/init-entrypoint"
    export PGDATA="${TEST_ROOT}/volumes/data/pgdata"
    export BACKUP_VOLUME="${TEST_ROOT}/volumes/backup"
    export DATA_VOLUME="${TEST_ROOT}/volumes/data/backup"
    export POSTGRESQL_PREV_VERSION=15
    export TEST_EVENTS="${TEST_ROOT}/events"
    export TEST_SPACE_MODE=available
    export TEST_REAL_TAR
    TEST_REAL_TAR=$(command -v tar)
    unset RESTORE_BACKUP FORCE_OLD_BINARIES FORCE_NEW_BACKUP FORCE_CLEANUP
    mkdir -p "${PGDATA}" "${BACKUP_VOLUME}" "${DATA_VOLUME}" \
        "${TEST_ROOT}/scripts" "${TEST_ROOT}/bin" "${TEST_ROOT}/old/postgresql-15/bin"
    echo 15 > "${PGDATA}/PG_VERSION"
    touch "${PGDATA}/postgresql.conf" "${TEST_EVENTS}" "${TEST_ROOT}/password"

    # Only replace fixed runtime paths; retain the production script's control flow.
    sed -e "s|/var/lib/postgresql|${TEST_ROOT}/volumes|g" \
        -e "s|/usr/lib64/pgsql|${TEST_ROOT}/old|g" \
        -e "s|NEW_BINARIES=\"/usr/bin\"|NEW_BINARIES=\"${TEST_ROOT}/bin\"|g" \
        -e "s|/run/secrets/stackrox.io/secrets/password|${TEST_ROOT}/password|g" \
        "${BATS_TEST_DIRNAME}/init-entrypoint.sh" > "${TEST_ROOT}/scripts/init-entrypoint.sh"
    cp "${BATS_TEST_DIRNAME}/backup-cleanup.sh" "${TEST_ROOT}/scripts/"

    cat > "${TEST_ROOT}/mock" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
echo "${0##*/} $*" >> "${TEST_EVENTS}"
case "${0##*/}" in
    postgres) echo 'postgres (PostgreSQL) 16.10' ;;
    pg_ctl)
        if [[ "$*" == *backup-restore-test* ]]; then
            [[ "${TEST_FAIL_VERIFICATION:-}" != true ]]
        else
            [[ "${TEST_FAIL_SOURCE_START:-}" != true ]]
        fi
        ;;
    pg_isready) exit 0 ;;
    pg_controldata) echo "Database cluster state: ${TEST_CLUSTER_STATE:-shutdown}" ;;
    pg_upgrade) [[ "${TEST_FAIL_UPGRADE:-}" != true ]] ;;
    initdb)
        target="${*: -1}"
        [[ "${target}" == */pgdata-new ]] || target="${PGDATA}"
        mkdir -p "${target}"
        echo 16 > "${target}/PG_VERSION"
        touch "${target}/postgresql.conf"
        ;;
    df)
        if [[ $# == 0 ]]; then
            echo "fixture 1000000 0 1000000 0% ${BACKUP_VOLUME}"
        else
            available=1000000
            case "${TEST_SPACE_MODE}" in
                full) available=0 ;;
                first-full) [[ "$1" != "${BACKUP_VOLUME}" ]] || available=0 ;;
                reclaimable) [[ ! -f "$1/13-15/backup.tar" ]] || available=0 ;;
            esac
            echo "fixture 1000000 0 ${available} 0% $1"
        fi
        ;;
    tar)
        args=()
        for arg in "$@"; do
            [[ "${arg}" == --checkpoint=* ]] || args+=("${arg}")
        done
        exec "${TEST_REAL_TAR}" "${args[@]}"
        ;;
    sync) exit 0 ;;
esac
MOCK
    chmod +x "${TEST_ROOT}/mock"
    local binary
    for binary in postgres pg_ctl pg_isready pg_controldata; do
        cp "${TEST_ROOT}/mock" "${TEST_ROOT}/old/postgresql-15/bin/${binary}"
    done
    for binary in postgres initdb pg_upgrade df tar sync; do
        cp "${TEST_ROOT}/mock" "${TEST_ROOT}/bin/${binary}"
    done
    export PATH="${TEST_ROOT}/bin:${PATH}"
    make_upgrade_backup "${BACKUP_VOLUME}/13-15"
    make_upgrade_backup "${DATA_VOLUME}/13-15"
}

make_upgrade_backup() {
    mkdir -p "$1"
    "${TEST_REAL_TAR}" -cf "$1/backup.tar" -C "${PGDATA}" .
}

run_init() {
    run bash "${TEST_ROOT}/scripts/init-entrypoint.sh"
}

assert_older_backups_preserved() {
    [ -s "${BACKUP_VOLUME}/13-15/backup.tar" ]
    [ -s "${DATA_VOLUME}/13-15/backup.tar" ]
}

@test "ordinary startup preserves aged backups including the latest rollback backup" {
    echo 16 > "${PGDATA}/PG_VERSION"
    make_upgrade_backup "${BACKUP_VOLUME}/15-16"
    touch -t 202001010000 "${BACKUP_VOLUME}/15-16" "${BACKUP_VOLUME}/13-15"

    run_init

    [ "${status}" -eq 0 ]
    assert_older_backups_preserved
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "fresh initialization does not clean up existing backups" {
    rm "${PGDATA}/PG_VERSION"

    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 16 ]
    assert_older_backups_preserved
}

@test "unsupported source versions fail before cleanup" {
    echo 13 > "${PGDATA}/PG_VERSION"

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"Unsupported PostgreSQL upgrade"* ]]
    assert_older_backups_preserved
}

@test "missing upgrade binaries fail before cleanup" {
    rm "${TEST_ROOT}/old/postgresql-15/bin/pg_ctl"

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"Required PostgreSQL upgrade binary is missing"* ]]
    assert_older_backups_preserved
}

@test "source startup failure prevents cleanup even when space is needed" {
    export TEST_SPACE_MODE=reclaimable TEST_FAIL_SOURCE_START=true

    run_init

    [ "${status}" -ne 0 ]
    assert_older_backups_preserved
}

@test "unclean source database prevents cleanup" {
    export TEST_CLUSTER_STATE='in production'

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"Cluster was not shutdown cleanly"* ]]
    assert_older_backups_preserved
}

@test "an unfinished upgrade prevents automatic cleanup" {
    mkdir -p "${PGDATA}-new"

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"Upgraded data directory already exists"* ]]
    assert_older_backups_preserved
}

@test "FORCE_CLEANUP clears an unfinished upgrade and allows cleanup after verification" {
    export FORCE_CLEANUP=true
    mkdir -p "${PGDATA}-new"
    touch "${PGDATA}-new/leftover"

    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 16 ]
    [ ! -e "${PGDATA}/leftover" ]
    [ ! -e "${PGDATA}-new" ]
    [ ! -e "${BACKUP_VOLUME}/13-15" ]
    [ ! -e "${DATA_VOLUME}/13-15" ]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "FORCE_CLEANUP preserves older backups when verification fails with space available" {
    export FORCE_CLEANUP=true TEST_FAIL_VERIFICATION=true
    mkdir -p "${PGDATA}-new"

    run_init

    [ "${status}" -ne 0 ]
    [ ! -e "${PGDATA}-new" ]
    assert_older_backups_preserved
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "FORCE_CLEANUP allows reclaiming obsolete backups to retry an upgrade" {
    export FORCE_CLEANUP=true TEST_SPACE_MODE=reclaimable
    mkdir -p "${PGDATA}-new"

    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 16 ]
    [ ! -e "${PGDATA}-new" ]
    [ ! -e "${BACKUP_VOLUME}/13-15" ]
    [ ! -e "${DATA_VOLUME}/13-15" ]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "available space preserves older backups until verification succeeds" {
    export TEST_FAIL_VERIFICATION=true

    run_init

    [ "${status}" -ne 0 ]
    assert_older_backups_preserved
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "space in the second location avoids deleting backups before verification" {
    export TEST_SPACE_MODE=first-full TEST_FAIL_VERIFICATION=true

    run_init

    [ "${status}" -ne 0 ]
    assert_older_backups_preserved
    [ -s "${DATA_VOLUME}/15-16/backup.tar" ]
}

@test "verified upgrade prunes earlier backups and preserves the current backup on restart" {
    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 16 ]
    [ ! -e "${BACKUP_VOLUME}/13-15" ]
    [ ! -e "${DATA_VOLUME}/13-15" ]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]

    run_init

    [ "${status}" -eq 0 ]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "space shortage reclaims earlier backups while preserving the current and newer pairs" {
    export TEST_SPACE_MODE=reclaimable
    make_upgrade_backup "${BACKUP_VOLUME}/15-16"
    make_upgrade_backup "${BACKUP_VOLUME}/16-17"
    cp "${BACKUP_VOLUME}/15-16/backup.tar" "${TEST_ROOT}/original-backup.tar"

    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 16 ]
    [ ! -e "${BACKUP_VOLUME}/13-15" ]
    [ -s "${BACKUP_VOLUME}/16-17/backup.tar" ]
    [ "$(cksum < "${TEST_ROOT}/original-backup.tar")" = "$(cksum < "${BACKUP_VOLUME}/15-16/backup.tar")" ]
}

@test "insufficient space after cleanup fails without replacing the source database" {
    export TEST_SPACE_MODE=full

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"Not enough disk space, upgrade is cancelled"* ]]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 15 ]
    [ ! -e "${BACKUP_VOLUME}/15-16" ]
}

@test "recovery flags suppress automatic cleanup when the upgrade succeeds" {
    local flag
    for flag in FORCE_NEW_BACKUP RESTORE_BACKUP; do
        echo 15 > "${PGDATA}/PG_VERSION"
        make_upgrade_backup "${BACKUP_VOLUME}/15-16"
        export "${flag}=true"

        run_init

        [ "${status}" -eq 0 ]
        assert_older_backups_preserved
        unset "${flag}"
    done
}

@test "recovery flags suppress reclamation even with FORCE_CLEANUP set" {
    export FORCE_CLEANUP=true TEST_SPACE_MODE=reclaimable
    local flag
    for flag in FORCE_NEW_BACKUP RESTORE_BACKUP; do
        mkdir -p "${PGDATA}-new"
        make_upgrade_backup "${BACKUP_VOLUME}/15-16"
        export "${flag}=true"

        run_init

        [ "${status}" -ne 0 ]
        [[ "${output}" == *"Not enough disk space, upgrade is cancelled"* ]]
        [ ! -e "${PGDATA}-new" ]
        assert_older_backups_preserved
        unset "${flag}"
    done
}

@test "restoring with old binaries preserves every backup" {
    make_upgrade_backup "${BACKUP_VOLUME}/15-16"
    echo 16 > "${PGDATA}/PG_VERSION"
    export RESTORE_BACKUP=true FORCE_OLD_BINARIES=true

    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 15 ]
    assert_older_backups_preserved
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "failed upgrade preserves the verified current backup for recovery" {
    export TEST_FAIL_UPGRADE=true

    run_init

    [ "${status}" -ne 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 15 ]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}
