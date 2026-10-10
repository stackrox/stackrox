#!/usr/bin/env bash
# Invoke the appropriate upgrade make target depending on the OPERATOR_CLUSTER_TYPE env variable.

ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/../.."
readonly ROOT_DIR

# shellcheck source=./common.sh
source "${ROOT_DIR}/operator/hack/common.sh"

case $OPERATOR_CLUSTER_TYPE in
openshift*)
  # On OpenShift, decide between OLMv0 and OLMv1 based on OCP version
  if should_use_olmv1; then
    target="upgrade-via-olmv1"
  else
    target="upgrade-via-olm"
  fi
  ;;
*)
  target="deploy-via-chart"
  ;;
esac
make -C "${ROOT_DIR}/operator" "${target}" "$@"
