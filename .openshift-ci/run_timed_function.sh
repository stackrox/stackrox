#!/usr/bin/env bash

set -euo pipefail

if [[ "$#" -lt 2 ]]; then
    echo "usage: run_timed_function.sh <source-file> <function> [args...]" >&2
    exit 2
fi

source_file="$1"
function_name="$2"
shift 2

# Callers pass a repository-owned script so its functions are available here.
# shellcheck source=/dev/null
source "$source_file"
"$function_name" "$@"
