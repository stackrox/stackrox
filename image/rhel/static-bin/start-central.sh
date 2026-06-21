#!/usr/bin/env bash

set -euo pipefail

RESTART_EXE="$(readlink -f "$0")" exec /stackrox/central "$@"
