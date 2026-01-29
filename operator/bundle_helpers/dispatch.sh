#!/usr/bin/env bash
#
# Wrapper script for bundle helper tools.
#
# Provides an abstraction layer that allows switching between Python and Go
# implementations of bundle helper scripts without changing Makefile or Dockerfiles.
# The implementation is selected via the USE_GO_BUNDLE_HELPER environment variable.
#
# Usage: dispatch.sh <script-base-name> [args...]

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <script-base-name> [args...]" >&2
    exit 1
fi

script_name="$1"
shift

script_dir="$(dirname "$0")"

if [[ "${USE_GO_BUNDLE_HELPER:-false}" == "true" ]]; then
    exec go run "${script_dir}/main.go" "$script_name" "$@"
else
    exec "${script_dir}/${script_name}.py" "$@"
fi
