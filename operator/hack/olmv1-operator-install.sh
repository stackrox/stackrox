#!/bin/bash
# Installs the operator using OLMv1 (ClusterExtension API).
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

  case $# in
  5)
    local -r operator_ns="${1}"
    local -r index_image_repo="${2}"
    local -r index_image_tag="${3}"
    local -r starting_csv_version="${4}"
    local -r operator_channel="${5}"  # Unused in OLMv1, kept for signature compatibility
    ;;
  *)
    echo -e "Usage:\n\t$0 [--allow-dirty-tag | -d] <operator_ns> <index-image-repo> <index-image-tag> <starting-csv-version> <install-channel>" >&2
    echo -e "Example:\n\t$0 -d rhacs-operator-system quay.io/rhacs-eng/stackrox-operator-index v3.70.1 v3.70.0 latest" >&2
    echo -e "Note: This script uses OLMv1 (ClusterExtension API). The install-channel parameter is ignored." >&2
    exit 1
    ;;
  esac

  check_version_tag "${starting_csv_version}" "${allow_dirty_tag}"
  create_namespace "${operator_ns}"
  apply_olmv1_manifests "${operator_ns}" "${index_image_repo}" "${index_image_tag}" "${starting_csv_version}"
  wait_for_clusterextension_installed "${operator_ns}" "rhacs-operator-${operator_ns}" "${starting_csv_version}"
  inject_test_env_vars "${operator_ns}"
  verify_operator_deployment "${operator_ns}"

  log "OLMv1 operator installation completed successfully"
}

function apply_olmv1_manifests() {
  log "Applying OLMv1 operator manifests (ClusterExtension)..."
  local -r operator_ns="$1"
  local -r index_image_repo="$2"
  local -r index_image_tag="$3"
  local -r operator_version="$4"

  env -i PATH="${PATH}" \
    NAMESPACE="${operator_ns}" \
    INDEX_IMAGE_REPO="${index_image_repo}" \
    INDEX_IMAGE_TAG="${index_image_tag}" \
    OPERATOR_VERSION="${operator_version}" \
    envsubst < "${ROOT_DIR}/operator/hack/clusterextension.envsubst.yaml" \
    | "${ROOT_DIR}/operator/hack/retry-kubectl.sh" apply -f -
}

function verify_operator_deployment() {
  local -r operator_ns="$1"
  local -r deployment_name="rhacs-operator-controller-manager"

  log "Verifying operator deployment is available..."
  if ! retry 3 5 "${ROOT_DIR}/operator/hack/retry-kubectl.sh" < /dev/null -n "${operator_ns}" wait deployments.apps "${deployment_name}" --for condition=available --timeout 60s; then
    log "Operator deployment failed to become available"
    gather_olmv1_resources "${operator_ns}" "rhacs-operator-${operator_ns}"
    return 1
  fi

  log "Operator deployment is healthy"
}

main "$@"
