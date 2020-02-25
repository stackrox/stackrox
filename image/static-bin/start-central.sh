#!/bin/sh

rm -rf /tmp/scorch.bleve || /bin/true # wipeout the temporary index on start
restore-central-db
/stackrox/bin/migrator
RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
