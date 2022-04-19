#!/bin/bash

set -eou pipefail

function usage() {
  echo "
Usage:
  get-github-release.sh [options]

Options:
  --from   GitHub URL to the executable file.
  --to     A local path where the downloaded file will be saved.
" >&2
}

function usage_exit() {
  usage
  exit 1
}

function get_github_release() {
  local from=""
  local to=""

  while [[ -n "${1:-}" ]]; do
      case "${1:-}" in
          "--from")
              from="${2:-}"
              shift
              ;;
          "--to")
              to="${2:-}"
              shift
              ;;
          *)
              echo "Error: Unknown parameter: ${1}" >&2
              usage_exit
      esac

      if ! shift; then
          echo 'Error: Missing parameter argument.' >&2
          usage_exit
      fi
  done

  [[ "${from}" = "" ]] && echo 'Error: Parameter "from" is empty.' >&2 && usage_exit
  [[ "${to}" = "" ]] && echo 'Error: Parameter "to" is empty.' >&2 && usage_exit

  # File is already downloaded
  if [[ -f "${to}" ]]; then
    exit 0
  fi

  local -r bin_dir=$(dirname "${to}")
  mkdir -p "${bin_dir}"

  curl --silent --fail --location --output "${to}" "${from}"
  chmod +x "${to}"

  local -r kernel_name=$(uname -s) || true
  [[ "${kernel_name}" != "Darwin" ]] || xattr -c "${to}"

  echo "Successfully downloaded ${from} to ${to}."
}

get_github_release "$@"
