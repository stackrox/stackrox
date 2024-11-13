#!/usr/bin/env bash

set -xEeo pipefail

# Initialize DB if it does not exist
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  # XXX: Why initdb has to be done here?
  initdb --auth-host=scram-sha-256 --auth-local=scram-sha-256 --pwfile /run/secrets/stackrox.io/secrets/password --data-checksums

  # It was much easier to do that in docker-entrypoint.sh for postgres image,
  # but since the initdb is done here, docker-entrypoint jumps over all the
  # initialization.
  PGPASSWORD=$(cat /run/secrets/stackrox.io/secrets/password) createdb -U postgres central_active
fi
