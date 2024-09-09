#!/bin/bash

set -eu -o pipefail

# shellcheck source=../hack/common.sh
source "$(dirname "$0")/../hack/common.sh"

die() {
  log "$@"
  exit 1
}

function main() {
    if [ $# -lt 2 ]; then
        die "Usage: $0 <number of attempts> <delay between attempts in seconds> <command> [ <command arg> ... ]"
    fi

    local -r n_attempts="${1:-}"; shift
    if ! [ "$n_attempts" -gt 0 ] 2>/dev/null; then
        die "Error: '$n_attempts' is not a valid number of attempts, please provide a positive natural number."
    fi

    local -r delay="${1:-}"; shift
    if ! { [ "$delay" -gt 0 ] 2>/dev/null || [ "$delay" -eq 0 ] 2>/dev/null; }; then
        die "Error: '$delay' is not a number of seconds."
    fi

    log "** Executing '$*' with $n_attempts attempts **"
    log
    retry "$n_attempts" "$delay" "$@"
}

main "$@"
