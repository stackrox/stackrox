#!/bin/sh

rm -rf /tmp/scorch.bleve || /bin/true # wipeout the temporary index on start
restore-central-db 2>&1 | /stackrox/roxctl log-convert --module=restore-central-db
rollback-rocksdb 2>&1 | /stackrox/roxctl log-convert --module=rollback-rocksdb
/stackrox/bin/migrator
rocksdb-migration 2>&1 | /stackrox/roxctl log-convert --module=rocksdb-migration

RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
