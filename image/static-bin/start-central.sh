#!/usr/bin/env bash

set -euo pipefail

. "$(dirname "$0")/db-functions"
. "$(dirname "$0")/debug"

rm -rf /tmp/scorch.bleve || /bin/true # wipeout the temporary index on start
move-to-current 2>&1 | PERSISTENT_LOG=true /stackrox/roxctl log-convert --module=move-to-current
trunc_log 3000 | PERSISTENT_LOG=true /stackrox/roxctl log-convert --module=start-central
/stackrox/bin/logwatcher &
PERSISTENT_LOG=true /stackrox/bin/migrator || dump_cpu_info

RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
