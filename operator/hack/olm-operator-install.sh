#!/bin/bash
# Installs the operator using OLM which is already running in the cluster.
set -eu -o pipefail

source "$(dirname "$0")/common.sh"

function main() {
  local -r operator_ns="${1:-}"
  case $# in
  2)
    local -r index_version="${2:-}"
    local -r operator_version="${2:-}"
    ;;
  3)
    local -r index_version="${2:-}"
    local -r operator_version="${3:-}"
    ;;
  *)
    echo "Usage: $0 <operator_ns> <index-version> [<install-version>]" >&2
    exit 1
    ;;
  esac

  check_version_tag "${operator_version}"
  create_namespace "${operator_ns}"
  create_pull_secret "${operator_ns}"
  apply_operator_manifests "${operator_ns}" "${index_version}" "${operator_version}"

  approve_install_plan "${operator_ns}" "${operator_version}"
  nurse_deployment_until_available "${operator_ns}" "${operator_version}"
}

main "$@"
