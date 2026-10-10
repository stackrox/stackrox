#!/usr/bin/env bash

# Remove recognized backups outside the current image's supported rollback path.
# Call only during a supported forward upgrade, after validating the source DB.
# Missing, ambiguous, or linked backups are preserved. Cleanup is best effort;
# callers must check available space again before attempting a new backup.
cleanup_stale_upgrade_backups() {
    local current_upgrade_dir="$1"
    shift

    # Accept canonical, bounded major versions so arithmetic cannot overflow.
    local upgrade_pattern='^([1-9][0-9]{0,2})-([1-9][0-9]{0,2})$'
    [[ "${current_upgrade_dir}" =~ ${upgrade_pattern} ]] || return 0
    local current_old_version="${BASH_REMATCH[1]}"
    local current_new_version="${BASH_REMATCH[2]}"
    (( current_old_version < current_new_version )) || return 0

    local location old_backup backup_name old_version new_version
    for location in "$@"; do
        # Strip trailing slashes before testing for symlinks, including on roots.
        while [[ "${location}" == */ ]]; do location="${location%/}"; done
        [[ "${location}" == /* && -d "${location}" && ! -L "${location}" ]] || continue

        for old_backup in "${location}"/*; do
            [[ -d "${old_backup}" && ! -L "${old_backup}" ]] || continue
            backup_name="${old_backup##*/}"
            [[ "${backup_name}" =~ ${upgrade_pattern} ]] || continue
            old_version="${BASH_REMATCH[1]}"
            new_version="${BASH_REMATCH[2]}"
            # Preserve the current rollback version and any newer or ambiguous pair.
            (( old_version < new_version && old_version < current_old_version &&
                new_version <= current_old_version )) || continue
            [[ -f "${old_backup}/backup.tar" && -s "${old_backup}/backup.tar" &&
                ! -L "${old_backup}/backup.tar" ]] || continue

            echo "Removing obsolete upgrade backup ${old_backup}"
            if ! rm -rf -- "${old_backup}"; then
                echo "Warning: could not completely remove upgrade backup ${old_backup}" >&2
            fi
        done
    done
    return 0
}
