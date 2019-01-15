#!/usr/bin/env bash
# Finds large files that have been checked in to Git.
set -euo pipefail

SCRIPT="$(python -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"

whitelist_file="$(dirname "${SCRIPT}")/whitelist"
[[ -f "${whitelist_file}" ]] || { echo >&2 "Couldn't find whitelist file. Exiting..."; exit 1; }

large_files=$(git ls-tree --full-tree -l -r HEAD $(git rev-parse --show-toplevel) | awk '$4 > 50*1024 {print$5}')
non_whitelisted_files=($({ echo "${large_files}"; cat "${whitelist_file}"; cat "${whitelist_file}"; } | sort | uniq -u))

[[ "${#non_whitelisted_files[@]}" == 0 ]] || {
  echo "Found large files in the working tree. Please remove them!"
  echo "If you must add them, you need to explicitly add them to the whitelist in tools/large-git-files/whitelist."
  echo "Files were: "
  printf "  %s\n" ${non_whitelisted_files[@]}
  exit 1
} >&2
