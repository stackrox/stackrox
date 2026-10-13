#!/usr/bin/env bash

set -Eeo pipefail

# shellcheck source=backup-cleanup.sh
source "$(dirname "${BASH_SOURCE[0]}")/backup-cleanup.sh"

# Experimental comparison switch: both paths use the same image and conversion.
UPGRADE_BACKUP_MODE="${ROX_POSTGRES_UPGRADE_BACKUP_MODE:-tar}"
case "${UPGRADE_BACKUP_MODE}" in
    tar|directory) ;;
    *) echo "Invalid ROX_POSTGRES_UPGRADE_BACKUP_MODE: expected tar or directory" >&2; exit 1 ;;
esac

RECOVERY_REQUESTED=false
for flag in FORCE_CLEANUP FORCE_NEW_BACKUP RESTORE_BACKUP FORCE_OLD_BINARIES; do
    if [[ "${!flag:-}" == true ]]; then
        RECOVERY_REQUESTED=true
    fi
done
if [[ "${UPGRADE_BACKUP_MODE}" == directory && "${RECOVERY_REQUESTED}" == true ]]; then
    echo "Directory backup prototype does not support recovery flags; use a fresh trial or manual recovery" >&2
    exit 1
fi

# Coarse elapsed seconds are sufficient to compare data-sized transfers. Keep
# command arguments/passwords out of these logs and preserve failures under -e.
upgrade_phase() {
    local now=${SECONDS}
    if [[ -n "${UPGRADE_PHASE:-}" ]]; then
        printf 'PG_UPGRADE_TIMING mode=%s phase=%s event=end elapsed_s=%s exit_code=0\n' \
            "${UPGRADE_BACKUP_MODE}" "${UPGRADE_PHASE}" "$((now - UPGRADE_PHASE_STARTED))"
    fi
    UPGRADE_PHASE=$1
    UPGRADE_PHASE_STARTED=${now}
    printf 'PG_UPGRADE_TIMING mode=%s phase=%s event=start\n' "${UPGRADE_BACKUP_MODE}" "${UPGRADE_PHASE}"
}

upgrade_timing_exit() {
    local result=$1 now=${SECONDS}
    printf 'PG_UPGRADE_TIMING mode=%s phase=%s event=end elapsed_s=%s exit_code=%s\n' \
        "${UPGRADE_BACKUP_MODE}" "${UPGRADE_PHASE}" "$((now - UPGRADE_PHASE_STARTED))" "${result}"
    printf 'PG_UPGRADE_TIMING mode=%s phase=initializer_total event=end elapsed_s=%s exit_code=%s\n' \
        "${UPGRADE_BACKUP_MODE}" "$((now - UPGRADE_STARTED))" "${result}"
}

# Return the list of possible backup locations based on the mounted volumes.
# The result is either two locations, if the backup volume is mounted, or only
# one otherwise.
get_backup_locations () {
    # If we got a backup volume, it will be mounted alongside with the main
    # data volume.
    BACKUP_VOLUME_MOUNT="/var/lib/postgresql/backup"
    DATA_VOLUME_MOUNT="/var/lib/postgresql/data/backup"

    local -n locations=$1

    if df | grep -q "${BACKUP_VOLUME_MOUNT}"; then
        locations+=("${BACKUP_VOLUME_MOUNT}");
    fi

    mkdir -p "${DATA_VOLUME_MOUNT}"
    locations+=("${DATA_VOLUME_MOUNT}");
}

# Inspect the file system mounted at the specified point, and tell if the
# available disk space is more than the specified threshold. E.g. we have a
# file system mounted at "/some/path" with available disk space 100GB, out of
# which 20GB (2147483648) are taken. In this scenario the call:
#
#   check_volume_use "/some/path" 26843545600
#
# will fail, because the available space is less than required.
check_available_space () {
    MOUNT_POINT=$1
    THRESHOLD=$2

    AVAILABLE=$(df "${MOUNT_POINT}" | tail -n -1 | awk '{print $4}')

    if [ -z "${AVAILABLE}" ]; then
        echo "No volume at ${MOUNT_POINT}"
        return 1
    fi

    if [ "${AVAILABLE}" -lt "${THRESHOLD}" ]; then
        echo "Volume at ${MOUNT_POINT} does not have enough available disk space"
        echo "Current value is ${AVAILABLE}, required ${THRESHOLD}"
        return 1
    fi

    echo "Volume at ${MOUNT_POINT} has enough available disk space (${AVAILABLE})"
    return 0
}

# Will be used by both initdb and postgresql-upgrade
export PGSETUP_INITDB_OPTIONS="--auth-host=scram-sha-256 \
                               --auth-local=scram-sha-256 \
                               --pwfile /run/secrets/stackrox.io/secrets/password \
                               --data-checksums"

# Initialize DB if it does not exist
if [ ! -s "${PGDATA}/PG_VERSION" ]; then
  # shellcheck disable=SC2086
  initdb $PGSETUP_INITDB_OPTIONS
else
    # Verify if we need to perform major version upgrade
    PG_BINARY_VERSION=$(postgres -V |\
        sed 's/postgres (PostgreSQL) \([0-9]*\).\([0-9]*\).*/\1/')

    PG_DATA_VERSION=$(cat "${PGDATA}/PG_VERSION")

    backup_locations=()
    get_backup_locations backup_locations

    # A prototype directory is not a legacy archive. Protect it even if a later
    # attempt selects tar mode, especially before FORCE_NEW_BACKUP can delete it.
    for location in "${backup_locations[@]}"; do
        prototype_backup="${location}/${POSTGRESQL_PREV_VERSION}-${PG_BINARY_VERSION}/directory-copy"
        if [[ -e "${prototype_backup}" || -L "${prototype_backup}" ]]; then
            if [[ "${PG_DATA_VERSION}" -lt "${PG_BINARY_VERSION}" || "${RECOVERY_REQUESTED}" == true ]]; then
                echo "Existing directory backup prototype requires manual recovery or a fresh trial: ${prototype_backup}" >&2
                exit 1
            fi
        fi
    done

    if [[ -v FORCE_CLEANUP && "${FORCE_CLEANUP}" == "true" ]]; then
        echo "Remove leftovers from previous upgrade."

        PGDATA_NEW="${PGDATA}-new"
        rm -rf "${PGDATA_NEW}"
    fi

    if [[ -v FORCE_NEW_BACKUP && "${FORCE_NEW_BACKUP}" == "true" ]]; then
        echo "Remove old backup."

        # Note that we use $POSTGRESQL_PREV_VERSION instead of PG_DATA_VERSION,
        # since we could be asked to restore a backup after an upgrade.
        # $POSTGRESQL_PREV_VERSION is an env variable set by the
        # postgresql-container image itself.
        for location in "${backup_locations[@]}"
        do
            echo "Removing ${location}/$POSTGRESQL_PREV_VERSION-$PG_BINARY_VERSION/"
            rm -rf "${location}/$POSTGRESQL_PREV_VERSION-$PG_BINARY_VERSION/"
        done
    fi

    if [[ -v RESTORE_BACKUP && "${RESTORE_BACKUP}" == "true" ]]; then
        echo "Restoring from a backup."

        # Note that we use $POSTGRESQL_PREV_VERSION instead of PG_DATA_VERSION,
        # since we could be asked to restore a backup after an upgrade.
        # $POSTGRESQL_PREV_VERSION is an env variable set by the
        # postgresql-container image itself.
        for location in "${backup_locations[@]}"
        do
            BACKUP_DIR="${location}/$POSTGRESQL_PREV_VERSION-$PG_BINARY_VERSION/"

            # Do not care about symlinks yet
            if [ -d "${BACKUP_DIR}" ]; then
                echo "Found an upgrade backup directory ${BACKUP_DIR}."
                PG_BACKUP_DIR="${BACKUP_DIR}"
                break;
            else
                echo "An upgrade backup directory ${BACKUP_DIR} does not exist, skip."
            fi
        done

        if [ -z "${PG_BACKUP_DIR}" ]; then
            echo "Upgrade backup directory is not found, restore is cancelled."
            exit 1
        else
            echo "Restoring from ${PG_BACKUP_DIR}"
        fi

        PGDATA_NEW="${PGDATA}-new"
        mkdir -p "${PGDATA_NEW}"
        tar -xf "${PG_BACKUP_DIR}/backup.tar" -C "${PGDATA_NEW}" --checkpoint=10000
        rm -rf "${PGDATA}"
        mv "${PGDATA_NEW}" "${PGDATA}"
    fi

    if [[ -v FORCE_OLD_BINARIES && "${FORCE_OLD_BINARIES}" == "true" ]]; then
        echo "Using old binaries, no upgrade needed"
        exit 0
    fi

    if [ "$PG_DATA_VERSION" -lt "$PG_BINARY_VERSION" ]; then
        UPGRADE_STARTED=${SECONDS}
        upgrade_phase preflight
        trap 'upgrade_timing_exit "$?"' EXIT

        if [[ "${UPGRADE_BACKUP_MODE}" == directory ]]; then
            # No completion/retry protocol in this prototype. Never reuse an
            # existing version-pair directory, including a partial copy or tar.
            for location in "${backup_locations[@]}"; do
                backup_path="${location}/${PG_DATA_VERSION}-${PG_BINARY_VERSION}"
                if [[ -L "${location}" || -e "${backup_path}" || -L "${backup_path}" ]]; then
                    echo "Directory backup prototype requires a fresh backup location: ${backup_path}" >&2
                    exit 1
                fi
            done
            source_link=$(find "${PGDATA}" -type l -print -quit)
            if [[ -n "${source_link}" ]]; then
                echo "Directory backup prototype does not support source symlinks: ${source_link}" >&2
                exit 1
            fi
        fi

        # Binaries version is newer, upgrade the data
        PGDATA_NEW="${PGDATA}-new"

        # Verify that the upgrade data directory does not exist. If it is,
        # there was an upgrade attempt.
        if [ -d "$PGDATA_NEW" ]; then
            echo "Upgraded data directory already exists, stop."
            exit 1
        fi

        # Validate the supported source version and binaries before deleting backups.
        if [ "${PG_DATA_VERSION}" != "${POSTGRESQL_PREV_VERSION:-}" ]; then
            echo "Unsupported PostgreSQL upgrade from ${PG_DATA_VERSION} to ${PG_BINARY_VERSION}."
            exit 1
        fi
        OLD_BINARIES="/usr/lib64/pgsql/postgresql-${PG_DATA_VERSION}/bin"
        NEW_BINARIES="/usr/bin"
        for binary in "${OLD_BINARIES}/postgres" "${OLD_BINARIES}/pg_ctl" \
            "${OLD_BINARIES}/pg_isready" "${OLD_BINARIES}/pg_controldata" \
            "${NEW_BINARIES}/initdb" "${NEW_BINARIES}/pg_upgrade"; do
            if [ ! -x "${binary}" ]; then
                echo "Required PostgreSQL upgrade binary is missing: ${binary}"
                exit 1
            fi
        done

        # Recovery operations must not automatically discard other recovery backups.
        ALLOW_BACKUP_CLEANUP=true
        if [[ "${RESTORE_BACKUP:-}" == true || "${FORCE_NEW_BACKUP:-}" == true ]]; then
            ALLOW_BACKUP_CLEANUP=false
        fi

        # Upgrade requires restrictive permissions and a cleanly stopped source DB.
        # Check its health before reclaiming space from any older backups.
        upgrade_phase source_health
        chmod 0700 "${PGDATA}"
        echo "Make sure PostgreSQL is shutdown clearly."
        "${OLD_BINARIES}/pg_ctl" start -w --timeout 86400 -o "-h 127.0.0.1"
        "${OLD_BINARIES}/pg_isready" -h 127.0.0.1
        "${OLD_BINARIES}/pg_ctl" stop -w

        STATUS=$("${OLD_BINARIES}/pg_controldata" -D "${PGDATA}" |\
                    grep "Database cluster state" |\
                    awk -F ':' '{print $2}' |\
                    tr -d '[:space:]')
        if [ "$STATUS" != "shutdown" ]; then
            echo "Cluster was not shutdown cleanly."
            exit 1
        fi

        # This is the amount of disk space we currently consume. Normally we
        # could use df as well, since the data will be the only disk space
        # consumer, but in testing environment it might not be the case.
        upgrade_phase space_check
        PG_DATA_USED=$(du -s "${PGDATA}" | awk '{print $1}')

        PG_BACKUP_VOLUME=""
        echo "Verifying backup locations ${backup_locations[*]}"
        for location in "${backup_locations[@]}"
        do
            # The backup volume needs to accommodate two copies of data, one is the
            # actual backup, and one is a restored copy, which will be deleted later.
            echo "${location}: Checking available disk space..."
            if check_available_space "${location}" $((PG_DATA_USED * 2)); then
                echo "Location has enough space."
                PG_BACKUP_VOLUME="${location}"
                break
            else
                echo "Not enough space."
            fi
        done

        # Reclaim space only if no location can hold the backup and verification copy.
        # Otherwise preserve older backups until the new backup has been verified.
        if [[ -z "${PG_BACKUP_VOLUME}" && "${ALLOW_BACKUP_CLEANUP}" == true ]]; then
            for location in "${backup_locations[@]}"; do
                cleanup_stale_upgrade_backups \
                    "${PG_DATA_VERSION}-${PG_BINARY_VERSION}" "${location}"
                if check_available_space "${location}" $((PG_DATA_USED * 2)); then
                    PG_BACKUP_VOLUME="${location}"
                    break
                fi
            done
        fi

        if [ -z "${PG_BACKUP_VOLUME}" ]; then
            echo "Not enough disk space, upgrade is cancelled."
            exit 1
        else
            echo "Backup will be stored in ${PG_BACKUP_VOLUME}"
        fi

        BACKUP_STARTED=${SECONDS}
        BACKUP_DIR="${PG_BACKUP_VOLUME}/$PG_DATA_VERSION-$PG_BINARY_VERSION/"
        if [[ "${UPGRADE_BACKUP_MODE}" == directory ]]; then
            upgrade_phase directory_copy
            # mkdir (without -p) also refuses a directory created since preflight.
            mkdir "${BACKUP_DIR}"
            BACKUP_VERIFY_PGDATA="${BACKUP_DIR}/directory-copy"
            mkdir "${BACKUP_VERIFY_PGDATA}"
            cp -a --reflink=never "${PGDATA}/." "${BACKUP_VERIFY_PGDATA}/"
            upgrade_phase backup_flush
            sync -f "${BACKUP_VERIFY_PGDATA}"
        else
            upgrade_phase tar_create
            # Do not care about symlinks yet
            if [ -d "${BACKUP_DIR}" ]; then
              echo "An upgrade backup directory already exists, skip."
            else
                # Do a backup before upgrading. Since the database is stopped we
                # may as well simple take a filesystem backup. Alternatives would
                # be pg_dump or pg_basebackup, both require running database
                # cluster.
                echo "Backup..."
                mkdir -p "${BACKUP_DIR}"
                tar -cf "${BACKUP_DIR}/backup.tar" -C "${PGDATA}" --checkpoint=10000 .
                upgrade_phase backup_flush
                sync "${BACKUP_DIR}/backup.tar"
            fi

            upgrade_phase tar_extract
            BACKUP_VERIFY_PGDATA="${BACKUP_DIR}/backup-restore-test"
            mkdir -p "${BACKUP_VERIFY_PGDATA}"
            tar -xf "${BACKUP_DIR}/backup.tar" -C "${BACKUP_VERIFY_PGDATA}" --checkpoint=10000
        fi

        echo "Verify backup..."
        upgrade_phase backup_verify
        "${OLD_BINARIES}/pg_ctl" \
            -D "${BACKUP_VERIFY_PGDATA}" \
            -w start -o "-h 127.0.0.1"
        "${OLD_BINARIES}/pg_ctl" \
            -D "${BACKUP_VERIFY_PGDATA}" \
            -w stop

        if [[ "${UPGRADE_BACKUP_MODE}" == directory ]]; then
            # Verification starts/stops the copy, so flush its writes too. The
            # retained copy must remain independent of link-mode working data.
            upgrade_phase verification_flush
            sync -f "${BACKUP_VERIFY_PGDATA}"
            echo "Retained experimental directory backup: ${BACKUP_VERIFY_PGDATA}"
        else
            upgrade_phase verification_cleanup
            rm -rf "${BACKUP_VERIFY_PGDATA}"
        fi
        printf 'PG_UPGRADE_TIMING mode=%s phase=backup_total event=end elapsed_s=%s exit_code=0\n' \
            "${UPGRADE_BACKUP_MODE}" "$((SECONDS - BACKUP_STARTED))"

        upgrade_phase stale_backup_cleanup
        if [ "${ALLOW_BACKUP_CLEANUP}" == true ]; then
            cleanup_stale_upgrade_backups \
                "${PG_DATA_VERSION}-${PG_BINARY_VERSION}" "${backup_locations[@]}"
        fi

        echo "Upgrade..."
        upgrade_phase target_init
        # Good idea to --check first
        # shellcheck disable=SC2086
        "${NEW_BINARIES}/initdb" $PGSETUP_INITDB_OPTIONS "${PGDATA_NEW}"

        upgrade_phase conversion_check
        PGPASSWORD=$(cat /run/secrets/stackrox.io/secrets/password) \
            "${NEW_BINARIES}/pg_upgrade" \
                --old-bindir="${OLD_BINARIES}" \
                --new-bindir="${NEW_BINARIES}" \
                --old-datadir="${PGDATA}" \
                --new-datadir="${PGDATA_NEW}" \
                -j 4 -k --check

        RESULT=$?
        if [ $RESULT -ne 0 ]; then
            echo "Upgrade check failed."
            find "${PGDATA_NEW}" -name pg_upgrade_server.log -exec cat {} \;
            exit 1
        fi

        upgrade_phase conversion
        PGPASSWORD=$(cat /run/secrets/stackrox.io/secrets/password) \
            "${NEW_BINARIES}/pg_upgrade" \
                --old-bindir="${OLD_BINARIES}" \
                --new-bindir="${NEW_BINARIES}" \
                --old-datadir="${PGDATA}" \
                --new-datadir="${PGDATA_NEW}" \
                -j 4 -k

        RESULT=$?
        if [ $RESULT -ne 0 ]; then
            echo "Upgrade failed."
            find "${PGDATA_NEW}" -name pg_upgrade_server.log -exec cat {} \;
            exit 1
        fi

        upgrade_phase finalization
        mv "${PGDATA}"/*.conf "${PGDATA_NEW}"
        rm -rf "${PGDATA}"
        mv "${PGDATA_NEW}" "${PGDATA}"
    fi
fi
