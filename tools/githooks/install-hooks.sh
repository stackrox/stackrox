#!/usr/bin/env bash
set -eo pipefail

git_root=$(git rev-parse --show-toplevel)
hooks_path="${git_root}/.git/hooks"
hooks_installed_file="${hooks_path}/.rox_hooks_installed"
pre_commit_script="${hooks_path}/pre-commit"

if [[ -f "${pre_commit_script}" && ! -f "${hooks_installed_file}" ]]; then
  echo -n "Do you want to remove your existing pre-commit hook (yes|no)? "
  read answer
  if [[ "${answer}" == "yes" || "${answer}" == "y" ]]; then
    rm "${pre_commit_script}"
    echo "Removed ${pre_commit_script}"
  else
    echo "Aborted by user, answer was not yes"
    exit 1
  fi
fi

touch "${hooks_installed_file}"
ln -sf "${git_root}/tools/githooks/pre-commit" "${pre_commit_script}"
