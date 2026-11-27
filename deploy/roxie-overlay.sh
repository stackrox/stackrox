#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${SCRIPT_DIR}/roxie-overlay"
# On failure, write an ANSI reset to stdout: invisible in a terminal, but rejected by YAML parsers,
# so `roxie deploy --config <(roxie-overlay.sh ...)` aborts instead of deploying with an empty config.
go run . "$@" || { printf '\033[0m'; exit 1; }
