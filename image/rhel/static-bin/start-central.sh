#!/usr/bin/env bash

set -euo pipefail

. "$(dirname "$0")/debug"

/stackrox/roxctl log-convert --module=start-central
/stackrox/bin/migrator || dump_cpu_info

RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
