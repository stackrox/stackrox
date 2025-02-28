#!/usr/bin/env bash

set -Eeo pipefail

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
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  initdb $PGSETUP_INITDB_OPTIONS
else
    # Verify if we need to perform major version upgrade
    PG_BINARY_VERSION=$(postgres -V |\
        sed 's/postgres (PostgreSQL) \([0-9]*\).\([0-9]*\).*/\1/')

    PG_DATA_VERSION=$(cat "${PGDATA}/PG_VERSION")

    if [ "$PG_DATA_VERSION" -lt "$PG_BINARY_VERSION" ]; then
        # Binaries version is newer, upgrade the data

        # This is the amount of disk space we currently consume. Normally we
        # could use df as well, since the data will be the only disk space
        # consumer, but in testing environment it might not be the case.
        PG_DATA_USED=$(du -s "${PGDATA}" | awk '{print $1}')
        PG_BACKUP_VOLUME="/backups"
        echo "Checking backup volume space..."

        # The backup volume needs to accomodate two copies of data, one is the
        # actual backup, and one is a restored copy, which will be deleted later.
        if ! check_available_space "${PG_BACKUP_VOLUME}" $((PG_DATA_USED * 2)); then
            echo "Not enough space. Checking data volume space..."
            PG_BACKUP_VOLUME="${PGDATA}/../"

            # If no luck, check the main data volume. It has to accomodate two
            # extra copies of data as well, one is the backup and one is the
            # restored copy.
            if ! check_available_space "${PG_BACKUP_VOLUME}" $((PG_DATA_USED * 2)); then
                echo "Not enough disk space, upgrade is cancelled"
                exit 1
            fi
        fi

        # After this point we know there is enough available disk space.

        BACKUP_DIR="${PG_BACKUP_VOLUME}/backups/$PG_DATA_VERSION-$PG_BINARY_VERSION/"
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
            tar -cf "${BACKUP_DIR}/backup.tar" -C $PGDATA --checkpoint=1000 .
            sync $BACKUP_DIR/backup.tar
        fi

        echo "Verify backup..."
        BACKUP_VERIFY_PGDATA="${BACKUP_DIR}/backup-restore-test"
        OLD_BINARIES="/usr/lib64/pgsql/postgresql-13/bin/"
        mkdir -p $BACKUP_VERIFY_PGDATA
        tar -xvf $BACKUP_DIR/backup.tar -C $BACKUP_VERIFY_PGDATA
        $OLD_BINARIES/pg_ctl -D $BACKUP_VERIFY_PGDATA -w start
        $OLD_BINARIES/pg_ctl -D $BACKUP_VERIFY_PGDATA -w stop
        rm -rf $BACKUP_VERIFY_PGDATA

        # Not sure how it works now, but during the upgrade group permissions
        # are rejected.
        chmod 0700 $PGDATA

        echo "Upgrade..."
        # Good idea to --check first
        PGSETUP_PGUPGRADE_OPTIONS='-j 4 -k' \
            PGPASSWORD=$(cat /run/secrets/stackrox.io/secrets/password) \
            postgresql-upgrade "${PGDATA}"

        # Need to update statistics afterwards
        # vacuumdb --all --analyze-in-stages
    fi
fi
