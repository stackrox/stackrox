#!/usr/bin/env bash

# Minimal structural check for a completed upgrade backup. Tangible artifacts
# only: a legacy non-empty archive, or a published directory artifact whose
# metadata asserts completion and whose data directory has a PG_VERSION.
# Staging directories, links, and unverified copies are never backups.
is_complete_upgrade_backup() {
    local upgrade_dir="$1" metadata data

    if [[ -f "${upgrade_dir}/backup.tar" && -s "${upgrade_dir}/backup.tar" &&
        ! -L "${upgrade_dir}/backup.tar" ]]; then
        return 0
    fi

    metadata="${upgrade_dir}/directory-v1/metadata.tsv"
    data="${upgrade_dir}/directory-v1/pgdata"
    [[ -f "${metadata}" && ! -L "${metadata}" ]] || return 1
    [[ -d "${data}" && ! -L "${data}" && -f "${data}/PG_VERSION" && ! -L "${data}/PG_VERSION" ]] || return 1
    grep -qx $'format_version\t1' "${metadata}" || return 1
    grep -qx $'verified\ttrue' "${metadata}" || return 1
    return 0
}

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

    local location old_backup backup_name backup_old_version backup_new_version
    for location in "$@"; do
        # Strip trailing slashes before testing for symlinks, including on roots.
        while [[ "${location}" == */ ]]; do location="${location%/}"; done
        [[ "${location}" == /* && -d "${location}" && ! -L "${location}" ]] || continue

        for old_backup in "${location}"/*; do
            [[ -d "${old_backup}" && ! -L "${old_backup}" ]] || continue
            backup_name="${old_backup##*/}"
            [[ "${backup_name}" =~ ${upgrade_pattern} ]] || continue
            backup_old_version="${BASH_REMATCH[1]}"
            backup_new_version="${BASH_REMATCH[2]}"
            # Preserve the current rollback version and any newer or ambiguous pair.
            (( backup_old_version < backup_new_version && backup_old_version < current_old_version &&
                backup_new_version <= current_old_version )) || continue
            is_complete_upgrade_backup "${old_backup}" || continue

            echo "Removing obsolete upgrade backup ${old_backup}"
            if ! rm -rf -- "${old_backup}"; then
                echo "Warning: could not completely remove upgrade backup ${old_backup}" >&2
            fi
        done
    done
    return 0
}
