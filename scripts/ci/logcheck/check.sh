#!/usr/bin/env bash

# check.sh
# Checks if a file contains any pattern from a configurable blocklist.
# Usage: check.sh <files...>
# It returns a non-zero exit status if any offending patterns have been found.

# The first line of output from this script when an offending pattern is found
# is the first line that matches a block pattern. The rest of output is standard
# grep output with filename prefix and context.

DIR="$(cd "$(dirname "$0")" && pwd)"

LINES_OF_CONTEXT=5
BLOCKLIST_FILE="${DIR}/blocklist-patterns"
allow_file="${ALLOWLIST_FILE:-${DIR}/allowlist-patterns}"

join_by() { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n' read -d '' -r -a blocklist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${BLOCKLIST_FILE}")

blocklist_pattern="$(join_by '|' "${blocklist_subpatterns[@]}")"

IFS=$'\n' read -d '' -r -a allowlist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${allow_file}")

allowlist_pattern="$(join_by '|' "${allowlist_subpatterns[@]}")"

if check_out="$(grep -vHP "$allowlist_pattern" "$@" | grep -"${LINES_OF_CONTEXT}" -Pi "$blocklist_pattern")"; then
    first_occurence="$(grep -vhP "$allowlist_pattern" "$@" | grep -Pi "$blocklist_pattern" | head -1)"
    echo "${first_occurence}"
    echo "${check_out}"
    exit 1
fi

exit 0
