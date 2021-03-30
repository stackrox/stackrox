#!/bin/sh

set -e

rm -rf /tmp/scorch.bleve || /bin/true # wipeout the temporary index on start
move-to-current 2>&1 | /stackrox/roxctl log-convert --module=move-to-current
/stackrox/bin/migrator

RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
