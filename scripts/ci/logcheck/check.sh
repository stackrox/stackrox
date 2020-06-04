#!/usr/bin/env bash

# check.sh
# Checks if a file contains any pattern from a configurable blacklist.
# Usage: check.sh <files...>
# It returns a non-zero exit status if any offending patterns have been found.

DIR="$(cd "$(dirname "$0")" && pwd)"

BLACKLIST_FILE="${DIR}/blacklist-patterns"
WHITELIST_FILE="${DIR}/whitelist-patterns"

join_by() { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n' read -d '' -r -a blacklist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${BLACKLIST_FILE}")

blacklist_pattern="$(join_by '|' "${blacklist_subpatterns[@]}")"

IFS=$'\n' read -d '' -r -a whitelist_subpatterns < <(egrep -v '^(#.*|\s*)$' "${WHITELIST_FILE}")

whitelist_pattern="$(join_by '|' "${whitelist_subpatterns[@]}")"

grep -vP "$whitelist_pattern" "$@" | grep >&2 -Pni "$blacklist_pattern" && exit 1

exit 0
