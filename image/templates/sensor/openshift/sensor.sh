#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

KUBE_COMMAND=${KUBE_COMMAND:-oc}

${KUBE_COMMAND} get namespace stackrox &>/dev/null || ${KUBE_COMMAND} create namespace stackrox
${KUBE_COMMAND} -n stackrox annotate namespace/stackrox --overwrite openshift.io/node-selector=""

${KUBE_COMMAND} project "stackrox"

echo "Creating sensor secrets..."
${KUBE_COMMAND} apply -f "$DIR/sensor-secret.yaml"
echo "Creating sensor RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/sensor-rbac.yaml"
echo "Creating sensor security context constraints..."
${KUBE_COMMAND} apply -f "$DIR/sensor-scc.yaml"
echo "Creating sensor network policies..."
${KUBE_COMMAND} apply -f "$DIR/sensor-netpol.yaml"
echo "Creating sensor pod security policues..."
${KUBE_COMMAND} apply -f "$DIR/sensor-pod-security.yaml"

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

{{- if .AdmissionController }}
echo "Creating admission controller security context constraints..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-scc.yaml"
echo "Creating admission controller secrets..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-secret.yaml"
echo "Creating admission controller RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-rbac.yaml"
echo "Creating admission controller network policies..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-netpol.yaml"
echo "Creating admission controller pod security policies..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-pod-security.yaml"
echo "Creating admission controller deployment..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller.yaml"
{{- else }}
echo "Deleting admission controller webhook, if it exists"
${KUBE_COMMAND} delete validatingwebhookconfiguration stackrox --ignore-not-found
{{- end }}

echo "Creating collector security context constraints..."
${KUBE_COMMAND} apply -f "$DIR/collector-scc.yaml"
echo "Creating collector secrets..."
${KUBE_COMMAND} apply -f "$DIR/collector-secret.yaml"
echo "Creating collector RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/collector-rbac.yaml"
echo "Creating collector network policies..."
${KUBE_COMMAND} apply -f "$DIR/collector-netpol.yaml"
echo "Creating collector pod security policies..."
${KUBE_COMMAND} apply -f "$DIR/collector-pod-security.yaml"
echo "Creating collector daemon set..."
${KUBE_COMMAND} apply -f "$DIR/collector.yaml"

if [[ -f "$DIR/additional-ca-sensor.yaml" ]]; then
  echo "Creating secret for additional CAs for sensor..."
  ${KUBE_COMMAND} apply -f "$DIR/additional-ca-sensor.yaml"
fi

echo "Creating sensor deployment..."
${KUBE_COMMAND} apply -f "$DIR/sensor.yaml"

{{- if .CreateUpgraderSA }}
echo "Creating upgrader service account"
${KUBE_COMMAND} apply -f "${DIR}/upgrader-serviceaccount.yaml"
{{- else }}
if [[ -f "${DIR}/upgrader-serviceaccount.yaml" ]]; then
    printf "%s\n\n%s\n" "Did not create the upgrader service account. To create it later, please run" "${KUBE_COMMAND} apply -f \"${DIR}/upgrader-serviceaccount.yaml\""
fi
{{- end }}
