#!/usr/bin/env bash

# check.sh
# Checks if a file contains any pattern from a configurable blacklist.
# Usage: check.sh <files...>
# It returns a non-zero exit status if any offending patterns have been found.

DIR="$(cd "$(dirname "$0")" && pwd)"

BLACKLIST_FILE="${DIR}/blacklist-patterns"

join_by() { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n' read -d '' -r -a subpatterns < <(egrep -v '^(#.*|\s*)$' "${BLACKLIST_FILE}")

pattern="$(join_by '|' "${subpatterns[@]}")"

grep >&2 -Pni "$pattern" "$@" && exit 1
exit 0
