#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

KUBE_COMMAND=${KUBE_COMMAND:-oc}
SKIP_ORCHESTRATOR_CHECK=${SKIP_ORCHESTRATOR_CHECK:-false}

if [[ "${SKIP_ORCHESTRATOR_CHECK}" == "true" ]] ; then
    echo >&2  "WARN: Skipping orchestrator check..."
else
    count="$(${KUBE_COMMAND} api-resources | grep "securitycontextconstraints" -c || true)"
    if [[ "${count}" -eq 0 ]]; then
       echo >&2 "Detected an attempt to deploy a cluster bundle designed to be deployed on OpenShift, \\
        on a vanilla Kubernetes cluster. Please regenerate the cluster bundle using cluster type 'k8s' and redeploy. \\
        If you think this message is in error, and would like to continue deploying the cluster bundle, please rerun \\
        the script using 'SKIP_ORCHESTRATOR_CHECK=true ./sensor.sh'"
        exit 1
    fi
fi

SUPPORTS_PSP=$(${KUBE_COMMAND} api-resources | grep "podsecuritypolicies" -c || true)

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

if [[ -f "$DIR/sensor-pod-security.yaml" ]]; then
  # Checking if the cluster supports pod security policies
  if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
    echo "Pod security policies are not supported on this cluster. Skipping..."
  else
    echo "Creating sensor pod security policies..."
    ${KUBE_COMMAND} apply -f "$DIR/sensor-pod-security.yaml"
  fi
fi

# OpenShift roles can be delayed to be added
sleep 5

if ! ${KUBE_COMMAND} get secret/stackrox -n stackrox &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{ required "" .MainRegistry }}")"
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

${KUBE_COMMAND} secrets link serviceaccount/sensor secrets/stackrox --for=pull

if ! ${KUBE_COMMAND} get secret/collector-stackrox -n stackrox &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{ required "" .CollectorRegistry }}")"
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

echo "Creating admission controller security context constraints..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-scc.yaml"
echo "Creating admission controller secrets..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-secret.yaml"
echo "Creating admission controller RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-rbac.yaml"
echo "Creating admission controller network policies..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-netpol.yaml"
if [[ -f "$DIR/admission-controller-pod-security.yaml" ]]; then
  if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
    echo "Pod security policies are not supported on this cluster. Skipping..."
  else
    echo "Creating admission controller pod security policies..."
    ${KUBE_COMMAND} apply -f "$DIR/admission-controller-pod-security.yaml"
  fi
fi
echo "Creating admission controller deployment..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller.yaml"

echo "Creating collector security context constraints..."
${KUBE_COMMAND} apply -f "$DIR/collector-scc.yaml"
echo "Creating collector secrets..."
${KUBE_COMMAND} apply -f "$DIR/collector-secret.yaml"
echo "Creating collector RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/collector-rbac.yaml"
echo "Creating collector network policies..."
${KUBE_COMMAND} apply -f "$DIR/collector-netpol.yaml"
if [[ -f "$DIR/collector-pod-security.yaml" ]]; then
  if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
    echo "Pod security policies are not supported on this cluster. Skipping..."
  else
    echo "Creating collector pod security policies..."
    ${KUBE_COMMAND} apply -f "$DIR/collector-pod-security.yaml"
  fi
fi
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
