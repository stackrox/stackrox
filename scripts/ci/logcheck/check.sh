#!/usr/bin/env bash

# check.sh
# Checks if a file contains any pattern from a configurable blocklist.
# Usage: check.sh <files...>
# It returns a non-zero exit status if any offending patterns have been found.

DIR="$(cd "$(dirname "$0")" && pwd)"

BLOCKLIST_FILE="${DIR}/blocklist-patterns"
ALLOWLIST_FILE="${DIR}/allowlist-patterns"

join_by() { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n' read -d '' -r -a blocklist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${BLOCKLIST_FILE}")

blocklist_pattern="$(join_by '|' "${blocklist_subpatterns[@]}")"

IFS=$'\n' read -d '' -r -a allowlist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${ALLOWLIST_FILE}")

allowlist_pattern="$(join_by '|' "${allowlist_subpatterns[@]}")"

grep -vP "$allowlist_pattern" "$@" | grep -Pni "$blocklist_pattern" && exit 1

exit 0
