#!/usr/bin/env bash

# check.sh
# Checks if a file contains any pattern from a configurable blocklist.
# Usage: check.sh <files...>
# Exit status: 0 for clean logs, 1 for offending patterns, 2 for scan errors.

# The first line of output from this script when an offending pattern is found
# is the first line that matches a block pattern. The rest of output is standard
# grep output with filename prefix and context.

DIR="$(cd "$(dirname "$0")" && pwd)"

# Block/allow lists use PCRE. BSD grep has no -P; Homebrew GNU grep is `ggrep`.
if grep -P 'a' <<<'a' >/dev/null 2>&1; then
    GREP_CMD="grep"
elif command -v ggrep >/dev/null 2>&1; then
    GREP_CMD="ggrep"
else
    echo "logcheck requires GNU grep (-P). On macOS: brew install grep" >&2
    exit 2
fi

LINES_OF_CONTEXT=5
BLOCKLIST_FILE="${DIR}/blocklist-patterns"
allow_file="${ALLOWLIST_FILE:-${DIR}/allowlist-patterns}"

join_by() { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n' read -d '' -r -a blocklist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${BLOCKLIST_FILE}")

blocklist_pattern="$(join_by '|' "${blocklist_subpatterns[@]}")"

IFS=$'\n' read -d '' -r -a allowlist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${allow_file}")

allowlist_pattern="$(join_by '|' "${allowlist_subpatterns[@]}")"

if check_out="$(
    "${GREP_CMD}" -vHP "$allowlist_pattern" "$@" | "${GREP_CMD}" -"${LINES_OF_CONTEXT}" -Pi "$blocklist_pattern"
    grep_status=("${PIPESTATUS[@]}")
    # No matches (1) is normal; errors in either stage must fail the check.
    if (( grep_status[0] > 1 || grep_status[1] > 1 )); then
        exit 2
    fi
    exit "${grep_status[1]}"
)"; then
    first_occurence="$("${GREP_CMD}" -vhP "$allowlist_pattern" "$@" | "${GREP_CMD}" -Pi "$blocklist_pattern" | head -1)"
    echo "${first_occurence}"
    echo "${check_out}"
    exit 1
else
    check_status=$?
    if [[ "${check_status}" -ne 1 ]]; then
        exit "${check_status}"
    fi
fi

exit 0
