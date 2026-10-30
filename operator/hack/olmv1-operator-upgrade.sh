#!/bin/bash
# Upgrades the operator to a newer version using OLMv1 (ClusterExtension API).
# Assumes the operator was previously installed via OLMv1 using olmv1-operator-install.sh.
set -eu -o pipefail

# shellcheck source=./common.sh
source "$(dirname "$0")/common.sh"

function main() {
  local allow_dirty_tag=false

  case "${1:-}" in
  -d | --allow-dirty-tag)
    allow_dirty_tag=true
    shift
    ;;
  esac

  if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [-d | --allow-dirty-tag] <operator_ns> <csv-version>" >&2
    echo "Example: $0 rhacs-operator-system v5.1.0" >&2
    exit 1
  fi

  local -r operator_ns="${1:-}"
  local -r csv_version="${2:-}"

  check_version_tag "${csv_version}" "${allow_dirty_tag}"

  log "Upgrading operator to version ${csv_version} using OLMv1..."
  upgrade_clusterextension "${operator_ns}" "${csv_version}"
  wait_for_clusterextension_installed "${operator_ns}" "rhacs-operator-${operator_ns}" "${csv_version}"
  # TODO(OCPSTRAT-2311): configure the env vars in the spec rather than injecting when it's possible.
  inject_test_env_vars "${operator_ns}"
  verify_operator_deployment "${operator_ns}"

  log "OLMv1 operator upgrade completed successfully"
}

function upgrade_clusterextension() {
  local -r operator_ns="$1"
  local -r new_version="$2"
  # Include namespace in ClusterExtension name to allow parallel test instances
  # (ClusterExtension is cluster-scoped, so namespace alone doesn't isolate)
  local -r extension_name="rhacs-operator-${operator_ns}"

  log "Updating ClusterExtension to version ${new_version}..."

  if ! "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null patch clusterextension "${extension_name}" --type=merge -p "{\"spec\":{\"source\":{\"catalog\":{\"version\":\"${new_version}\"}}}}"; then
    log "Failed to update ClusterExtension"
    gather_olmv1_resources "${operator_ns}" "${extension_name}"
    return 1
  fi
}

main "$@"
