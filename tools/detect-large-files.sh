#!/usr/bin/env bash
# Finds large files that have been checked in to Git.
set -euo pipefail

GIT_REPO_TOP="$(git rev-parse --show-toplevel)"

allowlist_path=""
if [[ $# -eq 1 ]]
then
  allowlist_path="$1"
fi

allowed_files=""
if [[ -n "$allowlist_path" ]]
then
  LC_ALL=C LC_COLLATE=C sort --check "${allowlist_path}"
  allowed_files=$(egrep -v '^\s*(#.*)$' "${allowlist_path}" | xargs git -C "$GIT_REPO_TOP" ls-files --)
fi

large_files=$(git ls-tree --full-tree -l -r HEAD "$GIT_REPO_TOP" | awk '$4 > 50*1024 {print$5}')


IFS=$'\n' read -d '' -r -a non_allowed_files < <(
  {
    echo "${large_files}"
    echo "${allowed_files}"
    echo "${allowed_files}"
  } | sort | uniq -u
) || true


[[ "${#non_allowed_files[@]}" == 0 ]] || {
  echo "Found large files in the working tree. Please remove them!"
  echo "If you must add them, you need to explicitly add them to the allow list file ${allowlist_path:-and provide it as an argument to this script.}"
  echo "Files were: "
  printf '  %s\n' "${non_allowed_files[@]}"
  exit 1
} >&2
