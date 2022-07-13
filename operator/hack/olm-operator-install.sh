#!/bin/bash
# Installs the operator using OLM which is already running in the cluster.
set -eu -o pipefail

source "$(dirname "$0")/common.sh"

declare allow_dirty_tag=false
declare -r IMAGE_TAG_BASE="${IMAGE_TAG_BASE:-quay.io/stackrox-io/stackrox-operator}"

function main() {
  case "$1" in
  -d | --allow-dirty-tag)
    allow_dirty_tag=true
    shift
    ;;
  esac

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
    echo "Usage: $0 [--allow-dirty-tag | -d] <operator_ns> <index-version> [<install-version>]" >&2
    exit 1
    ;;
  esac

  check_version_tag "${operator_version}" "${allow_dirty_tag}"
  create_namespace "${operator_ns}"
  create_pull_secret "${operator_ns}" "${IMAGE_TAG_BASE%%/*}"
  apply_operator_manifests "${operator_ns}" "${IMAGE_TAG_BASE}" "${index_version}" "${operator_version}"

  approve_install_plan "${operator_ns}" "${operator_version}"
  nurse_deployment_until_available "${operator_ns}" "${operator_version}"
}

main "$@"
