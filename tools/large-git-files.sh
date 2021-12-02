#!/usr/bin/env bash
# Finds large files that have been checked in to Git.
set -euo pipefail

allowlist_path=".largefiles"
[[ $# -eq 1 ]] && allowlist_path="$1"

allowed_files=$(egrep -v '^\s*(#.*)$' "${allowlist_path}" | xargs git ls-files --)
large_files=$(git ls-tree --full-tree -l -r HEAD "$(git rev-parse --show-toplevel)" | awk '$4 > 50*1024 {print$5}')

IFS=$'\n' read -d '' -r -a non_allowed_files < <(
  {
    echo "${large_files}"
    echo "${allowed_files}"
    echo "${allowed_files}"
  } | sort | uniq -u
) || true


[[ "${#non_allowed_files[@]}" == 0 ]] || {
  echo "Found large files in the working tree. Please remove them!"
  echo "If you must add them, you need to explicitly add them to the allowlist in tools/large-git-files/allowlist."
  echo "Files were: "
  printf '  %s\n' "${non_allowed_files[@]}"
  exit 1
} >&2
