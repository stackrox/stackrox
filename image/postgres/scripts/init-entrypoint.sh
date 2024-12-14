#!/usr/bin/env bash

set -xEeo pipefail

# Initialize DB if it does not exist
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  # XXX: Why initdb has to be done here?
  initdb --auth-host=scram-sha-256 --auth-local=scram-sha-256 --pwfile /run/secrets/stackrox.io/secrets/password --data-checksums
fi
