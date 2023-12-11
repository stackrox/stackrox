#!/usr/bin/env bash

set -Eeo pipefail

# Initialize DB if it does not exist
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  initdb --auth-host="$POSTGRES_HOST_AUTH_METHOD" --auth-local="$POSTGRES_HOST_AUTH_METHOD" --pwfile "$POSTGRES_PASSWORD_FILE" --data-checksums
fi
