#!/usr/bin/env bash

set -eo pipefail
shopt -s extglob

# Collapse whitespace — declare -A fails with multi-line or trailing-space values.
input="$(echo "$1" | tr '\n' ' ' | sed 's/  */ /g; s/ *$//')"
declare -A filters="$input"
base=${2:-$(git rev-parse origin/master)}
sha=${3:-$(git rev-parse HEAD)}

if [[ -z "${filters[*]}" ]]; then
  echo 'No filters provided. Please re-try with filters set like:' >&2
  echo '  ( ["ui"]="ui/*" ["go"]="*.go go.mod go.sum" )' >&2
  exit 1
fi

echo "# Find changes..." >&2
mapfile -t changes < <(git diff --name-only "${base}...${sha}" \
  | tee >(cat >&2))

echo "# Match filters..." >&2
declare -A matches
for k in "${!filters[@]}"; do
  read -ra exps <<< "${filters[$k]}"
  for line in "${changes[@]}"; do
    for exp in "${exps[@]}"; do
      # shellcheck disable=SC2053
      if [[ $line == $exp ]]; then
        matches["$k"]=1
        break 2
      fi
    done
  done
done

echo "matches=${!matches[*]}"
