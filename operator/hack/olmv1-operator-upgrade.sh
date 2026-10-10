#!/bin/bash
# Upgrades the operator to a newer version using OLMv1 (ClusterExtension API).
# Assumes the operator was previously installed via OLMv1 using olmv1-operator-install.sh.
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

  if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [-d | --allow-dirty-tag] <operator_ns> <csv-version>" >&2
    echo "Example: $0 rhacs-operator-system v3.70.1" >&2
    exit 1
  fi

  local -r operator_ns="${1:-}"
  local -r csv_version="${2:-}"

  check_version_tag "${csv_version}" "${allow_dirty_tag}"

  log "Upgrading operator to version ${csv_version} using OLMv1..."
  upgrade_clusterextension "${operator_ns}" "${csv_version}"
  wait_for_clusterextension_installed "${operator_ns}" "rhacs-operator-${operator_ns}" "${csv_version}"
  verify_operator_deployment "${operator_ns}"

  log "OLMv1 operator upgrade completed successfully"
}

function upgrade_clusterextension() {
  local -r operator_ns="$1"
  local -r new_version="$2"
  local -r extension_name="rhacs-operator-${operator_ns}"

  log "Updating ClusterExtension to version ${new_version}..."

  "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null patch clusterextension "${extension_name}" --type=merge -p "{\"spec\":{\"source\":{\"catalog\":{\"version\":\"${new_version}\"}}}}"

  if [[ $? -ne 0 ]]; then
    log "Failed to update ClusterExtension"
    gather_olmv1_resources "${operator_ns}" "${extension_name}"
    return 1
  fi
}

function verify_operator_deployment() {
  local -r operator_ns="$1"
  local -r deployment_name="rhacs-operator-controller-manager"

  log "Verifying upgraded operator deployment is available..."
  if ! retry 3 5 "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" wait deployments.apps "${deployment_name}" --for condition=available --timeout 60s; then
    log "Upgraded operator deployment failed to become available"
    gather_olmv1_resources "${operator_ns}" "rhacs-operator-${operator_ns}"
    return 1
  fi

  log "Upgraded operator deployment is healthy"
}

main "$@"
