#!/usr/bin/env bash
set -eo pipefail

git_root=$(git rev-parse --show-toplevel)
hooks_path=$(git -C "$git_root" config --get --default="$git_root/.git/hooks" core.hooksPath)

function install_hook
{
  hook_script="$1"
  hook_type=$(basename "$hook_script")
  hook_link="$hooks_path/$hook_type"

  if [[ "$hook_link" -ef "$hook_script" ]]
  then
    echo "$hook_type has already been installed"
    return
  elif [[ -f "$hook_link" ]]
  then
    echo -n "Do you want to remove your existing $hook_type hook (yes|no)? "
    read answer
    if [[ "$answer" == "yes" || "$answer" == "y" ]]; then
      rm "$hook_link"
      echo "Removed $hook_link"
    else
      echo "Hook $hook_type has not been installed, answer was not yes"
      return
    fi
  fi

  command -v realpath >/dev/null || { echo "realpath not found, please make sure it is installed (e.g. by installing GNU coreutils)"; exit 1; }
  ln -sf "$(realpath --relative-to "$hooks_path" "$hook_script")" "$hook_link"
}

for hook in "$@"
do
  install_hook "$hook"
done
