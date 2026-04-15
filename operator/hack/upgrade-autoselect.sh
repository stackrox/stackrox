#!/usr/bin/env bash
# Invoke the appropriate upgrade make target depending on the OPERATOR_CLUSTER_TYPE env variable.

ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/../.."
readonly ROOT_DIR

case $OPERATOR_CLUSTER_TYPE in
openshift*)
  target="upgrade-via-olm"
  ;;
*)
  target="upgrade-via-chart"
  ;;
esac
make -C "${ROOT_DIR}/operator" "${target}" "$@"
