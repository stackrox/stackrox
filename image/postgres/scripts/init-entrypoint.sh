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
        # Good idea to --check first
        PGSETUP_PGUPGRADE_OPTIONS='-j 4 -k' postgresql-upgrade "${PGDATA}"

        # Need to update statistics afterwards
        # vacuumdb --all --analyze-in-stages
    fi
fi
