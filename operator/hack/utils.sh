#!/bin/bash

function get_github_release() {
  local from=""
  local to=""

  while [[ "${1}" ]]; do
      case "${1}" in
          "--from")
              from="${2}"
              shift
              ;;
          "--to")
              to="${2}"
              shift
              ;;
          *)
              echo "utils:get_github_release: Unknown parameter: ${1}" >&2
              return 1
      esac

      if ! shift; then
          echo 'utils:get_github_release: Missing parameter argument.' >&2
          return 1
      fi
  done

  # File is already downloaded
  if [[ -f "${to}" ]]; then
    echo "utils:get_github_release: File ${to} already exists." >&2

    return 0
  fi

  [[ "${from}" = "" ]] && return 1
  [[ "${to}" = "" ]] && return 1

  local -r bin_dir=$(dirname "${to}")
  mkdir -p "${bin_dir}"

  curl --silent --fail --location --output "${to}" "${from}"
  chmod +x "${to}"

  [[ "$(uname -s)" != "Darwin" ]] || xattr -c "${to}"

  return 0
}
