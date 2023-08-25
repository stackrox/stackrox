#!/bin/bash
# Upgrades the operator to the most recent version.
# Assumes a previous version is already installed via OLM using olm-operator-install.sh.
set -eu -o pipefail

# shellcheck source=./common.sh
source "$(dirname "$0")/common.sh"

declare allow_dirty_tag=false

function main() {
  case "$1" in
  -d | --allow-dirty-tag)
    allow_dirty_tag=true
    shift
    ;;
  esac

  if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [-d | --allow-dirty-tag] <operator_ns> <operator-version>" >&2
    exit 1
  fi

  local -r operator_ns="${1:-}"
  local -r operator_version="${2:-}"

  # Unfortunately simply changing to automatic approval does not work:
  # https://github.com/operator-framework/operator-lifecycle-manager/issues/2341
  # kubectl -n "${operator_ns}" patch subscription.operators.coreos.com stackrox-operator-test-subscription \
  #  --type=json -p '[{"op": "remove", "path": "/spec/startingCSV"}, {"op": "replace", "path": "/spec/installPlanApproval", "value": "Automatic"}]'

  # However OLM happens to create an install plan for the latest version anyway, as soon as the old one is done, so we just have to approve it:
  check_version_tag "${operator_version}" "${allow_dirty_tag}"
  approve_install_plan "${operator_ns}" "${operator_version}"
  nurse_deployment_until_available "${operator_ns}" "${operator_version}"
}

main "$@"
