#!/usr/bin/env bash
# Finds large files that have been checked in to Git.
set -euo pipefail

SCRIPT="$(python -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"

allowlist_file="$(dirname "${SCRIPT}")/allowlist"
[[ -f "${allowlist_file}" ]] || { echo >&2 "Couldn't find allowlist file. Exiting..."; exit 1; }

large_files=$(git ls-tree --full-tree -l -r HEAD "$(git rev-parse --show-toplevel)" | awk '$4 > 50*1024 {print$5}')
IFS=$'\n' read -d '' -r -a non_allowed_files < <(
  {
    echo "${large_files}"
    cat "${allowlist_file}"
    cat "${allowlist_file}"
  } | sort | uniq -u
) || true

[[ "${#non_allowed_files[@]}" == 0 ]] || {
  echo "Found large files in the working tree. Please remove them!"
  echo "If you must add them, you need to explicitly add them to the allowlist in tools/large-git-files/allowlist."
  echo "Files were: "
  printf "  %s\n" "${non_allowed_files[@]}"
  exit 1
} >&2
