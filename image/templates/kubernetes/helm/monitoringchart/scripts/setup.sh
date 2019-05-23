#!/usr/bin/env bash

# Setup secrets for StackRox Monitoring
#
# Usage:
#   ./setup.sh
#
# Using a different command:
#     The KUBE_COMMAND environment variable will override the default of kubectl
#
# Examples:
# To use the default command to create resources:
#     $ ./setup.sh
# To use another command instead:
#     $ export KUBE_COMMAND='kubectl --context prod-cluster'
#     $ ./setup.sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}


${KUBE_COMMAND} get namespace stackrox > /dev/null || ${KUBE_COMMAND} create namespace stackrox

if ! ${KUBE_COMMAND} get secret/stackrox -n stackrox > /dev/null; then
  registry_auth="$("${DIR}/../../docker-auth.sh" -m k8s "{{.K8sConfig.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  ${KUBE_COMMAND} create --namespace "stackrox" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: stackrox
type: kubernetes.io/dockerconfigjson
EOF
fi

