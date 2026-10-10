#!/usr/bin/env bats

setup() {
    TEST_ROOT="${BATS_TEST_TMPDIR}/backup-cleanup"
    BACKUP_VOLUME="${TEST_ROOT}/backup-volume"
    DATA_VOLUME="${TEST_ROOT}/data-volume"
    mkdir -p "${BACKUP_VOLUME}" "${DATA_VOLUME}"

    # shellcheck source=backup-cleanup.sh
    source "${BATS_TEST_DIRNAME}/backup-cleanup.sh"
}

make_backup() {
    mkdir -p "$1"
    echo "backup fixture" > "$1/backup.tar"
}

@test "cleanup removes recognized backups from earlier upgrades in both locations" {
    make_backup "${BACKUP_VOLUME}/13-15"
    make_backup "${BACKUP_VOLUME}/14-15"
    make_backup "${DATA_VOLUME}/13-14"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}" "${DATA_VOLUME}"

    [ ! -d "${BACKUP_VOLUME}/13-15" ]
    [ ! -d "${BACKUP_VOLUME}/14-15" ]
    [ ! -d "${DATA_VOLUME}/13-14" ]
}

@test "cleanup preserves current, newer, and ambiguous upgrade pairs regardless of age" {
    local pair
    for pair in 15-16 16-17 15-17 13-16 15-15 15-13 0-15 013-15 13-015 999999999999999999999-15; do
        make_backup "${BACKUP_VOLUME}/${pair}"
        touch -t 202001010000 "${BACKUP_VOLUME}/${pair}"
    done

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}"

    for pair in 15-16 16-17 15-17 13-16 15-15 15-13 0-15 013-15 13-015 999999999999999999999-15; do
        [ -f "${BACKUP_VOLUME}/${pair}/backup.tar" ]
    done
}

@test "cleanup preserves unrelated directories and files including numeric names without backups" {
    local name
    for name in lost+found unrelated 1foo-2 13-15-extra .13-15; do
        make_backup "${BACKUP_VOLUME}/${name}"
    done
    mkdir -p "${BACKUP_VOLUME}/13-15" "${BACKUP_VOLUME}/14-15"
    touch "${BACKUP_VOLUME}/14-15/backup.tar" "${BACKUP_VOLUME}/README"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}"

    for name in lost+found unrelated 1foo-2 13-15-extra .13-15 13-15 14-15; do
        [ -d "${BACKUP_VOLUME}/${name}" ]
    done
    [ -f "${BACKUP_VOLUME}/README" ]
}

@test "cleanup preserves symlinked directories and their targets" {
    make_backup "${TEST_ROOT}/outside"
    ln -s "${TEST_ROOT}/outside" "${BACKUP_VOLUME}/13-15"
    ln -s "${TEST_ROOT}/missing" "${BACKUP_VOLUME}/14-15"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}"

    [ -L "${BACKUP_VOLUME}/13-15" ]
    [ -L "${BACKUP_VOLUME}/14-15" ]
    [ -f "${TEST_ROOT}/outside/backup.tar" ]
}

@test "cleanup rejects symlinked backup roots with or without trailing slashes" {
    make_backup "${TEST_ROOT}/outside/13-15"
    ln -s "${TEST_ROOT}/outside" "${TEST_ROOT}/linked-root"

    cleanup_stale_upgrade_backups "15-16" "${TEST_ROOT}/linked-root" "${TEST_ROOT}/linked-root///"

    [ -f "${TEST_ROOT}/outside/13-15/backup.tar" ]
}

@test "cleanup preserves directories with linked or nonregular backup archives" {
    make_backup "${TEST_ROOT}/outside"
    mkdir -p "${BACKUP_VOLUME}/13-15" "${BACKUP_VOLUME}/14-15/backup.tar"
    ln -s "${TEST_ROOT}/outside/backup.tar" "${BACKUP_VOLUME}/13-15/backup.tar"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}"

    [ -L "${BACKUP_VOLUME}/13-15/backup.tar" ]
    [ -d "${BACKUP_VOLUME}/14-15/backup.tar" ]
    [ -f "${TEST_ROOT}/outside/backup.tar" ]
}

@test "cleanup does not follow nested symlinks in recognized backups" {
    make_backup "${BACKUP_VOLUME}/13-15"
    make_backup "${TEST_ROOT}/outside"
    ln -s "${TEST_ROOT}/outside" "${BACKUP_VOLUME}/13-15/nested"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}"

    [ ! -d "${BACKUP_VOLUME}/13-15" ]
    [ -f "${TEST_ROOT}/outside/backup.tar" ]
}

@test "cleanup is idempotent and accepts empty or missing locations" {
    make_backup "${BACKUP_VOLUME}/13-15"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}/"
    run cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}" "${DATA_VOLUME}" "${TEST_ROOT}/missing" ""

    [ "${status}" -eq 0 ]
    [ ! -d "${BACKUP_VOLUME}/13-15" ]
}

@test "invalid or non-forward current upgrades never delete backups" {
    make_backup "${BACKUP_VOLUME}/13-15"
    local pair
    for pair in "" invalid 16-15 15-15 015-16 0-16 15-999999999999999999999; do
        cleanup_stale_upgrade_backups "${pair}" "${BACKUP_VOLUME}"
        [ -f "${BACKUP_VOLUME}/13-15/backup.tar" ]
    done
}

@test "cleanup reports deletion failures and continues under errexit" {
    make_backup "${BACKUP_VOLUME}/13-15"
    make_backup "${BACKUP_VOLUME}/14-15"
    rm() {
        if [[ "${*: -1}" == */13-15 ]]; then
            echo "Permission denied" >&2
            return 1
        fi
        command rm "$@"
    }
    export -f rm cleanup_stale_upgrade_backups

    run bash -ec 'cleanup_stale_upgrade_backups 15-16 "$1"; echo continued' _ "${BACKUP_VOLUME}"

    [ "${status}" -eq 0 ]
    [[ "${output}" == *"Warning: could not completely remove"* ]]
    [[ "${output}" == *continued* ]]
    [ -f "${BACKUP_VOLUME}/13-15/backup.tar" ]
    [ ! -d "${BACKUP_VOLUME}/14-15" ]
}
