#!/usr/bin/env bash

# check.sh
# Checks if a file contains any pattern from a configurable blocklist.
# Usage: check.sh <files...>
# It returns a non-zero exit status if any offending patterns have been found.

DIR="$(cd "$(dirname "$0")" && pwd)"

BLOCKLIST_FILE="${DIR}/blocklist-patterns"
echo "INFO: ${ALLOWLIST_FILE}"
allow_file="${ALLOWLIST_FILE}"
# Allow for tests to set this as an environment variable
if [[ -z "${ALLOWLIST_FILE:-}" ]]; then
    echo "SHREWS -- in the if"
    allow_file="${DIR}/allowlist-patterns"
fi
echo "INFO: $(date): SHREWS -- check this"
echo "INFO: ${allow_file}"
cat "${allow_file}"
echo "INFO: $(date): SHREWS -- end"

join_by() { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n' read -d '' -r -a blocklist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${BLOCKLIST_FILE}")

blocklist_pattern="$(join_by '|' "${blocklist_subpatterns[@]}")"

IFS=$'\n' read -d '' -r -a allowlist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${allow_file}")

allowlist_pattern="$(join_by '|' "${allowlist_subpatterns[@]}")"

grep -vP "$allowlist_pattern" "$@" | grep -Pni "$blocklist_pattern" && exit 1

exit 0
