#!/usr/bin/env bash
# Invoke the appropriate upgrade make target depending on the OPERATOR_CLUSTER_TYPE env variable.

ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/../.."
readonly ROOT_DIR

# shellcheck source=./common.sh
source "${ROOT_DIR}/operator/hack/common.sh"

case $OPERATOR_CLUSTER_TYPE in
openshift*)
  # On OpenShift, decide between OLMv0 and OLMv1 based on OCP version
  olm_version=$(get_olm_version)
  target="upgrade-via-olm${olm_version}"
  ;;
*)
  target="deploy-via-chart"
  ;;
esac
make -C "${ROOT_DIR}/operator" "${target}" "$@"
