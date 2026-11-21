#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

KUBE_COMMAND=${KUBE_COMMAND:-kubectl}
NAMESPACE="${ROX_NAMESPACE:-stackrox}"

${KUBE_COMMAND} get namespace "$NAMESPACE" &>/dev/null || ${KUBE_COMMAND} create namespace "$NAMESPACE"
${KUBE_COMMAND} annotate "namespace/${NAMESPACE}" --overwrite openshift.io/node-selector=""

if ! ${KUBE_COMMAND} get secret/stackrox -n "$NAMESPACE" > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "https://quay.io")"
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
