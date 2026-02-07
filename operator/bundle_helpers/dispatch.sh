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
    echo "Available scripts: fix-spec-descriptor-order, patch-csv" >&2
    exit 1
fi

script_name="$1"
shift

script_dir="$(dirname "$0")"

case "$script_name" in
fix-spec-descriptor-order|patch-csv)
    if [[ "${USE_GO_BUNDLE_HELPER:-false}" == "true" ]]; then
        echo "No Go implementation of $script_name available yet." >&2
        exit 1
    fi
    exec "${script_dir}/${script_name}.py" "$@"
  ;;
esac
