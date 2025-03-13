#!/usr/bin/env bash

set -Eeo pipefail

# Initialize DB if it does not exist
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  initdb --auth-host=scram-sha-256 --auth-local=scram-sha-256 --pwfile /run/secrets/stackrox.io/secrets/password --data-checksums
else
    # Verify if we need to perform major version upgrade
    PG_BINARY_VERSION=$(postgres -V |\
        sed 's/postgres (PostgreSQL) \([0-9]*\).\([0-9]*\).*/\1/')

    if [ $(cat "${PGDATA}/PG_VERSION") -lt "$PG_BINARY_VERSION" ]; then
        # Data version is less than binaries, upgrade

        # Do a backup first, ideally move it to a separate volume. Since the
        # database is stopped we may as well simple take a filesystem backup.
        # Alternatives would be pg_dump or pg_basebackup, both require running
        # database cluster.
        echo "Backup..."
        export BACKUP_DIR="$PGDATA/backups/$(date +%s)"
        mkdir -p $BACKUP_DIR
        tar -cf $BACKUP_DIR/backup.tar -C $PGDATA --checkpoint .
        sync $BACKUP_DIR/backup.tar

        echo "Verify backup..."
        export BACKUP_VERIFY_PGDATA="$PGDATA/backup-test"
        export OLD_BINARIES="/usr/lib64/pgsql/postgresql-13/bin/"
        mkdir -p $BACKUP_VERIFY_PGDATA
        tar -xvf $BACKUP_DIR/backup.tar -C $BACKUP_VERIFY_PGDATA
        $OLD_BINARIES/pg_ctl -D $BACKUP_VERIFY_PGDATA -w start
        $OLD_BINARIES/pg_ctl -D $BACKUP_VERIFY_PGDATA -w stop
        rm -rf $BACKUP_VERIFY_PGDATA

        # Not sure how it works now, but during the upgrade group permissions
        # are rejected.
        chmod 0700 $PGDATA

        # Allow pg_upgrade to use the password
        export PGPASSWORD=$(cat /run/secrets/stackrox.io/secrets/password)

        # Copies the same options as the original initdb
        export PGSETUP_INITDB_OPTIONS="--auth-host=scram-sha-256 \
                                       --auth-local=scram-sha-256 \
                                       --pwfile /run/secrets/stackrox.io/secrets/password \
                                       --data-checksums"
        echo "Upgrade..."
        # Good idea to --check first
        PGSETUP_PGUPGRADE_OPTIONS='-j 4 -k' postgresql-upgrade "${PGDATA}"

        # Need to update statistics afterwards
        # vacuumdb --all --analyze-in-stages
    fi
fi
