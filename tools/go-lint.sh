#!/usr/bin/env bash
# golint wrapper that ignores rules we don't care about

IGNORED_RULES=(
  "error strings should not be capitalized or end with punctuation or a newline"
  "should not use dot imports"
  "should have a package comment, unless it's in another file for this package"
)

ignored=0
total=0

function run_golint_on_dir() {
  local dir=$1
  while read -r line; do
    local matched=0
    for rule in "${IGNORED_RULES[@]}"; do
      if [[ ${line} =~ .*${rule} ]]; then
        matched=1
        break
      fi
    done
    if (( matched )); then
      ignored=$((ignored + 1))
    else
      echo "${line}"
    fi
    total=$((total+1))
  done < <(golint -min_confidence 0 "${dir}"/*.go)
}

# golint behaves weirdly unless you run it on all the files in one directory at a time.
# Note that we don't pass in the package name here because we want to lint all files,
# irrespective of tags.
while IFS='' read -r dir || [[ -n "${dir}" ]]; do
  run_golint_on_dir "${dir}"
done < <(echo "$@" | xargs -n 1 dirname | sort | uniq)

echo "Found ${total} errors, ignored ${ignored}"
if (( total == ignored )); then
    exit 0
else
    exit 1
fi
