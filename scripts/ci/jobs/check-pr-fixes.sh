#!/usr/bin/env bash

# This script is intended to be run in OpenShiftCI, and tells you whether any references to tickets
# claimed to be fixed by this PR are still referenced by a TODO.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

#set -euo pipefail
set +eo pipefail

check-pr-fixes() {
    echo 'Ensure that all TODO references to fixed tickets are gone'

    echo "JOB_SPEC=[${JOB_SPEC:-MISSING JOB_SPEC}]"
    is_in_PR_context || { echo "Not on a PR, nothing to do!"; exit 0; }

    IFS=$'\n' read -d '' -r -a tickets < <(
        get_pr_details | jq -r '.title' | grep -Eio '\brox-[[:digit:]]+\b' | sort | uniq)

    if [[ "${#tickets[@]}" == 0 ]]; then
        echo "This PR does not claim to fix any tickets!"
        exit 0
    fi

    echo "Tickets this PR claims to fix:"
    printf " - %s\n" "${tickets[@]}"

    "$(dirname "$0")/../scripts/check-todos.sh" "${tickets[@]}"
}

check-pr-fixes
