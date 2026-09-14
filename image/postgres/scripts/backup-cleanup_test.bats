#!/usr/bin/env bats

setup() {
    TEST_ROOT="${BATS_TEST_TMPDIR}/backup-cleanup"
    BACKUP_VOLUME="${TEST_ROOT}/backup-volume"
    DATA_VOLUME="${TEST_ROOT}/data-volume"
    mkdir -p "${BACKUP_VOLUME}" "${DATA_VOLUME}"

    # shellcheck source=backup-cleanup.sh
    source "${BATS_TEST_DIRNAME}/backup-cleanup.sh"
}

@test "stale cleanup removes backups from previous major upgrades" {
    mkdir -p \
        "${BACKUP_VOLUME}/13-15" \
        "${BACKUP_VOLUME}/14-15" \
        "${DATA_VOLUME}/14-15"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}" "${DATA_VOLUME}"

    [ ! -d "${BACKUP_VOLUME}/13-15" ]
    [ ! -d "${BACKUP_VOLUME}/14-15" ]
    [ ! -d "${DATA_VOLUME}/14-15" ]
}

@test "stale cleanup preserves the current upgrade backup and unrelated files" {
    mkdir -p "${BACKUP_VOLUME}/14-15" "${BACKUP_VOLUME}/15-16"
    touch "${BACKUP_VOLUME}/README"

    cleanup_stale_upgrade_backups "15-16" "${BACKUP_VOLUME}" "${DATA_VOLUME}"

    [ ! -d "${BACKUP_VOLUME}/14-15" ]
    [ -d "${BACKUP_VOLUME}/15-16" ]
    [ -f "${BACKUP_VOLUME}/README" ]
}

@test "stale cleanup accepts empty or missing backup locations" {
    run cleanup_stale_upgrade_backups \
        "15-16" \
        "${BACKUP_VOLUME}" \
        "${TEST_ROOT}/missing"

    [ "${status}" -eq 0 ]
}

@test "retention cleanup removes only expired backup directories" {
    mkdir -p \
        "${BACKUP_VOLUME}/expired" \
        "${BACKUP_VOLUME}/recent" \
        "${DATA_VOLUME}/expired"
    touch -t 202001010000 \
        "${BACKUP_VOLUME}/expired" \
        "${DATA_VOLUME}/expired"

    cleanup_expired_upgrade_backups 30 "${BACKUP_VOLUME}" "${DATA_VOLUME}"

    [ ! -d "${BACKUP_VOLUME}/expired" ]
    [ ! -d "${DATA_VOLUME}/expired" ]
    [ -d "${BACKUP_VOLUME}/recent" ]
}

@test "zero or invalid retention disables cleanup" {
    mkdir -p "${BACKUP_VOLUME}/expired-zero" "${BACKUP_VOLUME}/expired-invalid"
    touch -t 202001010000 \
        "${BACKUP_VOLUME}/expired-zero" \
        "${BACKUP_VOLUME}/expired-invalid"

    cleanup_expired_upgrade_backups 0 "${BACKUP_VOLUME}"
    cleanup_expired_upgrade_backups invalid "${BACKUP_VOLUME}"

    [ -d "${BACKUP_VOLUME}/expired-zero" ]
    [ -d "${BACKUP_VOLUME}/expired-invalid" ]
}
