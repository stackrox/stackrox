#!/usr/bin/env bash

set -Eeo pipefail

# Initialize DB if it does not exist
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  initdb --auth-host=scram-sha-256 --auth-local=scram-sha-256 --pwfile /run/secrets/stackrox.io/secrets/password --data-checksums
fi
