#!/usr/bin/env bash

cleanup_stale_upgrade_backups() {
    local current_upgrade_dir="$1"
    shift

    local location old_backup backup_name
    for location in "$@"; do
        for old_backup in "${location}"/*/; do
            [ -d "${old_backup}" ] || continue
            backup_name="${old_backup%/}"
            backup_name="${backup_name##*/}"
            if [ "${backup_name}" != "${current_upgrade_dir}" ]; then
                echo "Removing stale backup ${old_backup}"
                rm -rf "${old_backup}"
            fi
        done
    done
}

cleanup_expired_upgrade_backups() {
    local retention_days="$1"
    shift

    if ! [[ "${retention_days}" =~ ^[0-9]+$ ]] || [ "${retention_days}" -eq 0 ]; then
        return
    fi

    local location
    for location in "$@"; do
        [ -d "${location}" ] || continue
        find "${location}" -maxdepth 1 -mindepth 1 -type d \
            -mtime +"${retention_days}" -print \
            -exec rm -rf {} + 2>/dev/null || true
    done
}
