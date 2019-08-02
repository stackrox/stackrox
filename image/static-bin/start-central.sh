#!/bin/sh

restore-central-db
/stackrox/bin/migrator
RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
