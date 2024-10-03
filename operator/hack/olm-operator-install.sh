#!/bin/bash
# Installs the operator using OLM which is already running in the cluster.
set -eu -o pipefail

# shellcheck source=./common.sh
source "$(dirname "$0")/common.sh"

declare allow_dirty_tag=false

function main() {
  case "${1:-}" in
  -d | --allow-dirty-tag)
    allow_dirty_tag=true
    shift
    ;;
  esac

  local -r operator_ns="${1:-}"
  local -r index_image_repo="${2:-}"

  case $# in
  3)
    local -r index_image_tag="${3:-}"
    local -r csv_version="${3:-}"
    local -r operator_channel="latest"
    ;;
  4)
    local -r index_image_tag="${3:-}"
    local -r csv_version="${4:-}"
    local -r operator_channel="latest"
    ;;
  5)
    local -r index_image_tag="${3:-}"
    local -r csv_version="${4:-}"
    local -r operator_channel="${5:-}"
    ;;
  *)
    echo -e "Usage:\n\t$0 [--allow-dirty-tag | -d] <operator_ns> <index-image-repo> <index-image-tag> [<csv-version> [<install-channel>]]" >&2
    echo -e "Example:\n\t$0 -d index-test quay.io/rhacs-eng/stackrox-operator-index v3.70.1 v3.70.0" >&2
    echo -e "Note that KUTTL environment variable must be defined and point to a kuttl executable." >&2
    exit 1
    ;;
  esac

  check_version_tag "${csv_version}" "${allow_dirty_tag}"
  create_namespace "${operator_ns}"
  apply_operator_manifests "${operator_ns}" "${index_image_repo}" "${index_image_tag}" "${csv_version}" "${operator_channel}"

  if ! [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
    approve_install_plan "${operator_ns}" "${csv_version}"
  fi

  nurse_deployment_until_available "${operator_ns}" "${csv_version}"

}

main "$@"
