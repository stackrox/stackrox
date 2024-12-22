#!/usr/bin/env bash

set -euo pipefail

. "$(dirname "$0")/debug"

/stackrox/roxctl log-convert --module=start-central
MIGRATOR_PASSWORD=$(cat /run/secrets/stackrox.io/db-password/password) \
    /go/bin/tern migrate \
    --migrations /stackrox/migrations \
    --config /stackrox/migrations/tern.conf \
    || dump_cpu_info

RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
