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
  local -r image_tag_base="${2:-}"
  local -r image_registry="${image_tag_base%%/*}"

  case $# in
  3)
    local -r index_version="${3:-}"
    local -r operator_version="${3:-}"
    ;;
  4)
    local -r index_version="${3:-}"
    local -r operator_version="${4:-}"
    ;;
  *)
    echo -e "Usage:\n\t$0 [--allow-dirty-tag | -d] <operator_ns> <image_tag_base> <index-version> [<install-version>]" >&2
    echo -e "Example:\n\t$0 -d index-test quay.io/rhacs-eng/stackrox-operator 3.70.1 3.70.1" >&2
    echo -e "Note that KUTTL environment variable must be defined and point to a kuttl executable." >&2
    exit 1
    ;;
  esac

  check_version_tag "${operator_version}" "${allow_dirty_tag}"
  create_namespace "${operator_ns}"
  create_pull_secret "${operator_ns}" "${image_registry}"
  apply_operator_manifests "${operator_ns}" "${image_tag_base}" "${index_version}" "${operator_version}"

  if [[ "${USE_MIDSTREAM_IMAGES}" == "true" ]]; then
    if ! retry 15 5 kubectl -n "${operator_ns}" wait subscription.operators.coreos.com stackrox-operator-test-subscription --for condition=InstallPlanPending --timeout=60s; then
      log "Install plan failed to materialize."
      log "Dumping pod descriptions..."
      kubectl -n "${operator_ns}" describe pods -l "olm.catalogSource=stackrox-operator-test-index" || true
      log "Dumping catalog sources and subscriptions..."
      kubectl -n "${operator_ns}" describe "subscription.operators.coreos.com,catalogsource.operators.coreos.com" || true
      exit 1
    fi
    local -r current_csv=$(kubectl get -n "${operator_ns}" subscription.operators.coreos.com stackrox-operator-test-subscription -o jsonpath="{.status.currentCSV}")
    local -r operator_version_midstream=$(oc get csv -n "${operator_ns}" "${current_csv}" -o jsonpath='{.spec.version}')
    nurse_deployment_until_available "${operator_ns}" "${operator_version_midstream}"
  else
    approve_install_plan "${operator_ns}" "${operator_version}"
    nurse_deployment_until_available "${operator_ns}" "${operator_version}"
  fi
}

main "$@"
