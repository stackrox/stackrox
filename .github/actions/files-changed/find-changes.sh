#!/usr/bin/env bash

set -x
declare -A filters="$1"
base=${2:-$(git rev-parse origin/master)}
sha=${3:-$(git rev-parse HEAD)}

if [[ -z "${filters[@]}" ]]; then
  echo 'No filters provided. Please re-try with filters set like "( ["ui"]="ui/**" ["gha"]=".github/**" )"'
  exit 1
fi

# compare with merge-base
echo "# Find changes..."
changes=$(git diff --name-only ${base}...${sha} \
  | tee >( cat >&2 ))

echo "# Match filters..."
# for each labeled ex
IFS=$'\n'
declare -A matches
for k in "${!filters[@]}"; do
  exp="${filters[$k]}"
  for line in ${changes}; do
    if [[ $line == $exp ]]; then
      matches["$k"]="${matches[$k]:-}${matches[$k]:+ }${line}"
    fi
  done
done

for k in "${!matches[@]}"; do
  echo "$k=\"${matches[$k]}\""
done \
  | tee ${GITHUB_OUTPUT:-/dev/null}
