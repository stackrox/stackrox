#!/usr/bin/env bash

# shellcheck source=docker-entrypoint.sh
source "/usr/local/bin/docker-entrypoint.sh"

set -Eeo pipefail

# Initialize DB if it does not exist
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  initdb --auth-host="$POSTGRES_HOST_AUTH_METHOD" --auth-local="$POSTGRES_HOST_AUTH_METHOD" --pwfile "$POSTGRES_PASSWORD_FILE" --data-checksums
  file_env 'POSTGRES_PASSWORD'
  # PGPASSWORD is required for psql when authentication is required for 'local' connections via pg_hba.conf and is otherwise harmless
  export PGPASSWORD="${PGPASSWORD:-$POSTGRES_PASSWORD}"
  docker_temp_server_start "$@"
  echo
  echo "Creating $ROX_POSTGRES_DB database"
  echo
  # docker_setup_db expects POSTGRES_DB to be set
  export POSTGRES_DB="$ROX_POSTGRES_DB"
  docker_setup_db
  docker_temp_server_stop
  unset POSTGRES_DB
  unset PGPASSWORD
  echo
  echo "Done initializing database"
else
  echo "Nothing to do; database already exists"
fi
echo
