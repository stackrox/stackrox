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
    export TEST_REAL_CP
    TEST_REAL_CP=$(command -v gcp || command -v cp)
    unset RESTORE_BACKUP FORCE_OLD_BINARIES FORCE_NEW_BACKUP FORCE_CLEANUP ROX_POSTGRES_UPGRADE_BACKUP_MODE
    mkdir -p "${PGDATA}" "${BACKUP_VOLUME}" "${DATA_VOLUME}" \
        "${TEST_ROOT}/scripts" "${TEST_ROOT}/bin" "${TEST_ROOT}/old/postgresql-15/bin"
    echo 15 > "${PGDATA}/PG_VERSION"
    echo original > "${PGDATA}/payload"
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
        if [[ "$*" == *backup-restore-test* || "$*" == *directory-copy* ]]; then
            [[ "${TEST_FAIL_VERIFICATION:-}" != true ]]
        else
            [[ "${TEST_FAIL_SOURCE_START:-}" != true ]]
        fi
        ;;
    pg_isready) exit 0 ;;
    pg_controldata) echo "Database cluster state: ${TEST_CLUSTER_STATE:-shutdown}" ;;
    pg_upgrade)
        [[ "${TEST_FAIL_UPGRADE:-}" != true ]]
        if [[ "$*" != *--check* ]]; then
            [[ "${TEST_FAIL_CONVERSION:-}" != true ]]
            # Simulate link-mode writes to working data, not to its backup.
            ln "${PGDATA}/payload" "${PGDATA}-new/payload"
        fi
        ;;
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
    cp)
        if [[ "${TEST_FAIL_COPY:-}" == true ]]; then
            touch "${*: -1}/partial"
            exit 73
        fi
        exec "${TEST_REAL_CP}" "$@"
        ;;
    sync)
        if [[ "${TEST_FAIL_SYNC_ON:-}" == "$(grep -c '^sync ' "${TEST_EVENTS}")" ]]; then
            exit 74
        fi
        ;;
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
    cp "${TEST_ROOT}/mock" "${TEST_ROOT}/bin/cp"
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

@test "default tar mode emits successful phase and backup timings" {
    run_init

    [ "${status}" -eq 0 ]
    [[ "${output}" == *"mode=tar phase=tar_extract event=end"* ]]
    [[ "${output}" == *"mode=tar phase=backup_total event=end"* ]]
    [[ "${output}" == *"mode=tar phase=initializer_total event=end"*"exit_code=0"* ]]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
}

@test "invalid backup mode fails before modifying source or backups" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=typo

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"Invalid ROX_POSTGRES_UPGRADE_BACKUP_MODE"* ]]
    [ ! -s "${TEST_EVENTS}" ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 15 ]
    assert_older_backups_preserved
}

@test "directory mode retains an independent verified copy without tar and survives restart" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory

    run_init

    [ "${status}" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 16 ]
    [ "$(cat "${BACKUP_VOLUME}/15-16/directory-copy/PG_VERSION")" = 15 ]
    [ ! -e "${BACKUP_VOLUME}/15-16/backup.tar" ]
    [ "$(grep -c '^tar ' "${TEST_EVENTS}")" -eq 0 ]
    [ "$(grep -c '^sync -f ' "${TEST_EVENTS}")" -eq 2 ]
    [[ "${output}" == *"mode=directory phase=backup_total event=end"* ]]
    [[ ! "${PGDATA}/payload" -ef "${BACKUP_VOLUME}/15-16/directory-copy/payload" ]]
    echo changed > "${PGDATA}/payload"
    [ "$(cat "${BACKUP_VOLUME}/15-16/directory-copy/payload")" = original ]

    run_init

    [ "${status}" -eq 0 ]
    [[ "${output}" != *PG_UPGRADE_TIMING* ]]
    [ "$(cat "${BACKUP_VOLUME}/15-16/directory-copy/payload")" = original ]
}

@test "directory mode keeps the existing space policy and supports data-volume fallback" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory TEST_SPACE_MODE=first-full

    run_init

    [ "${status}" -eq 0 ]
    [ -f "${DATA_VOLUME}/15-16/directory-copy/PG_VERSION" ]
    [ ! -e "${BACKUP_VOLUME}/15-16" ]
}

@test "directory mode refuses existing legacy artifacts before starting the source" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory
    make_upgrade_backup "${BACKUP_VOLUME}/15-16"

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"requires a fresh backup location"* ]]
    [ "$(grep -c '^pg_ctl ' "${TEST_EVENTS}")" -eq 0 ]
    [ -s "${BACKUP_VOLUME}/15-16/backup.tar" ]
    assert_older_backups_preserved
}

@test "partial directory copy fails before conversion and cannot be reused" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory TEST_FAIL_COPY=true

    run_init

    [ "${status}" -eq 73 ]
    [[ "${output}" == *"phase=directory_copy event=end"*"exit_code=73"* ]]
    [ -f "${BACKUP_VOLUME}/15-16/directory-copy/partial" ]
    [ "$(grep -c '^pg_upgrade ' "${TEST_EVENTS}")" -eq 0 ]
    [ "$(cat "${PGDATA}/PG_VERSION")" = 15 ]
    assert_older_backups_preserved

    unset TEST_FAIL_COPY
    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"requires manual recovery or a fresh trial"* ]]
    [ "$(grep -c '^pg_upgrade ' "${TEST_EVENTS}")" -eq 0 ]
}

@test "each directory flush failure prevents conversion and keeps failure timing" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory
    local flush
    for flush in 1 2; do
        export TEST_FAIL_SYNC_ON="${flush}"
        : > "${TEST_EVENTS}"
        rm -rf "${BACKUP_VOLUME}/15-16"

        run_init

        [ "${status}" -eq 74 ]
        [[ "${output}" == *"phase=initializer_total event=end"*"exit_code=74"* ]]
        [[ "${output}" != *"phase=backup_total event=end"* ]]
        [ "$(grep -c '^pg_upgrade ' "${TEST_EVENTS}")" -eq 0 ]
        [ "$(cat "${PGDATA}/PG_VERSION")" = 15 ]
        assert_older_backups_preserved
    done
}

@test "directory verification failure retains the copy but prevents conversion" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory TEST_FAIL_VERIFICATION=true

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"phase=backup_verify event=end"*"exit_code=1"* ]]
    [ -f "${BACKUP_VOLUME}/15-16/directory-copy/PG_VERSION" ]
    [ "$(grep -c '^pg_upgrade ' "${TEST_EVENTS}")" -eq 0 ]
    assert_older_backups_preserved
}

@test "conversion failure preserves the independent directory backup" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory TEST_FAIL_CONVERSION=true

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"phase=conversion event=end"*"exit_code=1"* ]]
    [ "$(cat "${BACKUP_VOLUME}/15-16/directory-copy/PG_VERSION")" = 15 ]
    [ "$(cat "${BACKUP_VOLUME}/15-16/directory-copy/payload")" = original ]
}

@test "directory mode rejects recovery flags before destructive operations" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory
    mkdir -p "${PGDATA}-new"
    local flag
    for flag in RESTORE_BACKUP FORCE_NEW_BACKUP FORCE_CLEANUP FORCE_OLD_BINARIES; do
        export "${flag}=true"

        run_init

        [ "${status}" -ne 0 ]
        [[ "${output}" == *"does not support recovery flags"* ]]
        [ -d "${PGDATA}-new" ]
        assert_older_backups_preserved
        unset "${flag}"
    done
}

@test "tar mode cannot overwrite or restore an existing prototype directory" {
    mkdir -p "${BACKUP_VOLUME}/15-16/directory-copy"
    echo keep > "${BACKUP_VOLUME}/15-16/directory-copy/payload"
    local flag
    for flag in RESTORE_BACKUP FORCE_NEW_BACKUP FORCE_CLEANUP FORCE_OLD_BINARIES; do
        export "${flag}=true"

        run_init

        [ "${status}" -ne 0 ]
        [[ "${output}" == *"requires manual recovery or a fresh trial"* ]]
        [ "$(cat "${BACKUP_VOLUME}/15-16/directory-copy/payload")" = keep ]
        unset "${flag}"
    done
}

@test "directory mode rejects external source symlinks before copying" {
    export ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory
    ln -s "${TEST_ROOT}/password" "${PGDATA}/external"

    run_init

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"does not support source symlinks"* ]]
    [ ! -e "${BACKUP_VOLUME}/15-16" ]
    [ "$(grep -c '^pg_ctl ' "${TEST_EVENTS}")" -eq 0 ]
}
