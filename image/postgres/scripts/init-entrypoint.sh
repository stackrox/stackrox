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
        export PGPASSWORD=$(cat /run/secrets/stackrox.io/secrets/password)

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
