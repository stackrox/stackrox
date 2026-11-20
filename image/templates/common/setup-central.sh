#!/usr/bin/env bash

# Setup secrets for StackRox Central
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
NAMESPACE="${ROX_NAMESPACE:-stackrox}"

${KUBE_COMMAND} get namespace "$NAMESPACE" &>/dev/null || ${KUBE_COMMAND} create namespace "$NAMESPACE"
# A failed namespace annotation must not decide this script's exit status (it was last, so its
# failure used to leak through for public images); warn and continue with exit status 0.
${KUBE_COMMAND} annotate "namespace/${NAMESPACE}" --overwrite openshift.io/node-selector="" || echo >&2 "Warning: failed to annotate namespace ${NAMESPACE}; continuing"

{{ if not .K8sConfig.PublicImages -}}
if ! ${KUBE_COMMAND} get secret/stackrox -n "$NAMESPACE" > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.K8sConfig.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  ${KUBE_COMMAND} create --namespace "$NAMESPACE" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: "$NAMESPACE"
  labels:
    app.kubernetes.io/name: stackrox
type: kubernetes.io/dockerconfigjson
EOF
fi
{{ end -}}
