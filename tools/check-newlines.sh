#!/usr/bin/env bash

# Checks if all files given as arguments are either empty or end with newline characters.

join_by() { local IFS="$1"; shift; echo "$*"; }

# File extrensions to apply the "file must end with a newline character" rule to.
EXTENSIONS=(go sh yml yaml proto ts tsx)

EXT_REGEX="\\.($(join_by '|' "${EXTENSIONS[@]}"))\$"

fix=0

check_or_fix_newlines() {
  file="$1"
  [[ -s "$file" ]] || return 0  # skip check for empty files
  if [[ -n "$(tail -c 1 "$file")" ]]; then
    if (( fix )); then
      echo >&2 "Added missing newline at end of file: $file"
      echo >>"$file"
      return 0
    fi
    echo >&2 "Missing newline at end of file: $file"
    return 1
  fi
}

# Do nothing if no arguments given
[[ "$#" -gt 0 ]] || exit 0

if [[ "$1" == "--fix" ]]; then
  fix=1
  shift
fi

all_files=("$@")

IFS=$'\n' read -d '' -r -a files < <(
  printf '%s\n' "${all_files[@]}" | grep -E "$EXT_REGEX"
)

status=0
for file in "${files[@]}"; do
  check_or_fix_newlines "$file" || status=1
done

exit "$status"
