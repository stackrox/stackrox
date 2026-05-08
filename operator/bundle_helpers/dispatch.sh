#!/usr/bin/env bash
#
# Wrapper script for bundle helper tools.
#
# Usage: dispatch.sh <command> [args...]

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <command> [args...]" >&2
    exit 1
fi

command="$1"
shift

script_dir="$(dirname "$0")"

case "$command" in
fix-spec-descriptor-order|patch-csv)
    exec go run "${script_dir}/main.go" "$command" "$@"
    ;;
*)
    echo "Unknown command: $command" >&2
    echo "Available commands: fix-spec-descriptor-order, patch-csv" >&2
    exit 1
    ;;
esac
