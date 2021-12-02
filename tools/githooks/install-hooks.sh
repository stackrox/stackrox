#!/usr/bin/env bash
set -eo pipefail

git_root=$(git rev-parse --show-toplevel)
hooks_path=$(git --git-dir "$git_root" config --get --default="${git_root}/.git/hooks" core.hooksPath)
hooks_installed_file="${hooks_path}/.hooks_installed"

function install_hook
{
	hook_script="$1"
	hook_type=$(basename "$hook_script")
	hook_link="${hooks_path}/${hook_type}"

	if [[ -f "${hook_link}" && ! -f "${hooks_installed_file}" ]]; then
	  echo -n "Do you want to remove your existing $hook_type hook (yes|no)? "
	  read answer
	  if [[ "${answer}" == "yes" || "${answer}" == "y" ]]; then
	    rm "${hook_link}"
	    echo "Removed ${hook_link}"
	  else
	    echo "Aborted by user, answer was not yes"
	    exit 1
	  fi
	fi

	touch "${hooks_installed_file}"
	ln -sf "$(realpath --relative-to ${hooks_path} ${hook_script})" "${hook_link}"
}

for hook in $@
do
	install_hook "$hook"
done
