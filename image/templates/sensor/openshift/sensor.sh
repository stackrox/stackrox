#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

KUBE_COMMAND=${KUBE_COMMAND:-oc}

${KUBE_COMMAND} get namespace stackrox &>/dev/null || ${KUBE_COMMAND} create namespace stackrox
${KUBE_COMMAND} -n stackrox annotate namespace/stackrox --overwrite openshift.io/node-selector=""

${KUBE_COMMAND} project "stackrox"

${KUBE_COMMAND} apply -f "$DIR/sensor-rbac.yaml"
${KUBE_COMMAND} apply -f "$DIR/sensor-scc.yaml"
${KUBE_COMMAND} apply -f "$DIR/sensor-netpol.yaml"
${KUBE_COMMAND} apply -f  "$DIR/sensor-pod-security.yaml"

# OpenShift roles can be delayed to be added
sleep 5

if ! ${KUBE_COMMAND} get secret/stackrox -n stackrox &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.ImageRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  ${KUBE_COMMAND} create --namespace "stackrox" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: stackrox
  labels:
    app.kubernetes.io/name: stackrox
type: kubernetes.io/dockerconfigjson
EOF
fi

${KUBE_COMMAND} secrets add serviceaccount/sensor secrets/stackrox --for=pull

# Create secrets for sensor
${KUBE_COMMAND} apply -f "$DIR/sensor-secret.yaml"

{{if ne .CollectionMethod "NO_COLLECTION"}}
if ! ${KUBE_COMMAND} get secret/collector-stackrox -n stackrox &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.CollectorRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  ${KUBE_COMMAND} create --namespace "stackrox" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: collector-stackrox
  namespace: stackrox
type: kubernetes.io/dockerconfigjson
EOF
fi
{{- end}}

echo "Creating secrets for collector..."
${KUBE_COMMAND} apply -f "$DIR/collector-secret.yaml"

if [[ -f "$DIR/additional-ca-sensor.yaml" ]]; then
  echo "Creating secret for additional CAs for sensor..."
  ${KUBE_COMMAND} apply -f "$DIR/additional-ca-sensor.yaml"
fi

echo "Creating deployment..."
${KUBE_COMMAND} apply -f "$DIR/sensor.yaml"

{{  if .CreateUpgraderSA }}
echo "Creating upgrader service account"
${KUBE_COMMAND} apply -f "${DIR}/upgrader-serviceaccount.yaml"
{{ else }}
if [[ -f "${DIR}/upgrader-serviceaccount.yaml" ]]; then
    printf "%s\n\n%s\n" "Did not create the upgrader service account. To create it later, please run" "${KUBE_COMMAND} apply -f \"${DIR}/upgrader-serviceaccount.yaml\""
fi
{{- end}}
