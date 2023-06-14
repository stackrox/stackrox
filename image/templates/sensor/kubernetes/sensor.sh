#!/usr/bin/env bash

set -e

# Launch StackRox Sensor
#
# Deploys the StackRox Sensor into the cluster
#
# Usage:
#   ./sensor.sh
#
# Using a different command:
#     The KUBE_COMMAND environment variable will override the default of kubectl
#
# Examples:
# To use kubectl to create resources (the default):
#     $ ./sensor.sh
# To use another command instead:
#     $ export KUBE_COMMAND='kubectl --context prod-cluster'
#     $ ./sensor.sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

KUBE_COMMAND=${KUBE_COMMAND:-kubectl}
SKIP_ORCHESTRATOR_CHECK=${SKIP_ORCHESTRATOR_CHECK:-false}
NAMESPACE=${NAMESPACE:-stackrox}

if [[ "${SKIP_ORCHESTRATOR_CHECK}" == "true" ]] ; then
    echo >&2  "WARN: Skipping orchestrator check..."
else
    count="$(${KUBE_COMMAND} api-resources | grep "securitycontextconstraints" -c || true)"
    if [[ "${count}" -gt 0 ]]; then
        echo >&2 "Detected an attempt to deploy a cluster bundle designed to be deployed on vanilla Kubernetes, \\
        on an OpenShift cluster. Please regenerate the cluster bundle using cluster type `openshift` and redeploy. \\
        If you think this message is in error, and would like to continue deploying the cluster bundle, please rerun \\
        the script using 'SKIP_ORCHESTRATOR_CHECK=true ./sensor.sh'"
        exit 1
    fi
fi

SUPPORTS_PSP=$(${KUBE_COMMAND} api-resources | grep "podsecuritypolicies" -c || true)

${KUBE_COMMAND} get namespace "$NAMESPACE" &>/dev/null || ${KUBE_COMMAND} create namespace "$NAMESPACE"

if ! ${KUBE_COMMAND} get secret/stackrox -n "$NAMESPACE" &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{ required "" .MainRegistry }}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  ${KUBE_COMMAND} create --namespace "$NAMESPACE" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: "$NAMESPACE"
type: kubernetes.io/dockerconfigjson
EOF
fi

if ! ${KUBE_COMMAND} get secret/collector-stackrox -n "$NAMESPACE" &>/dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{ required "" .CollectorRegistry }}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  ${KUBE_COMMAND} create --namespace "$NAMESPACE" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: collector-stackrox
  namespace: "$NAMESPACE"
type: kubernetes.io/dockerconfigjson
EOF
fi

function print_rbac_instructions {
	echo
	echo "Error: Kubernetes RBAC configuration failed."
	echo "Specific errors are listed above."
	echo
	echo "You may need to elevate your privileges first:"
	echo "    ${KUBE_COMMAND} create clusterrolebinding temporary-admin --clusterrole=cluster-admin --user you@example.com"
	echo
	echo "(Be sure to use the full username your cluster knows for you.)"
	echo
	echo "Then, rerun this script."
	echo
	echo "Finally, revoke your temporary privileges:"
	echo "    ${KUBE_COMMAND} delete clusterrolebinding temporary-admin"
	echo
	echo "Contact your cluster administrator if you cannot obtain sufficient permission."
	exit 1
}

echo "Creating sensor RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/sensor-rbac.yaml" || print_rbac_instructions
echo "Creating sensor network policies..."
${KUBE_COMMAND} apply -f "$DIR/sensor-netpol.yaml" || exit 1

if [[ -f "$DIR/sensor-pod-security.yaml" ]]; then
  # Checking if the cluster supports pod security policies
  if [[ "${SUPPORTS_PSP}" -eq 0 ]]; then
    echo "Pod security policies are not supported on this cluster. Skipping..."
  else
    echo "Creating sensor pod security policies..."
    ${KUBE_COMMAND} apply -f "$DIR/sensor-pod-security.yaml"
  fi
fi

{{ if .CreateUpgraderSA }}
echo "Creating upgrader service account"
${KUBE_COMMAND} apply -f "${DIR}/upgrader-serviceaccount.yaml" || print_rbac_instructions
{{- end }}

echo "Creating admission controller secrets..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-secret.yaml"
echo "Creating admission controller RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/admission-controller-rbac.yaml" || print_rbac_instructions
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

echo "Creating secrets for sensor..."
${KUBE_COMMAND} apply -f "$DIR/sensor-secret.yaml"

if [[ -f "$DIR/additional-ca-sensor.yaml" ]]; then
  echo "Creating secret for additional CAs for sensor..."
  ${KUBE_COMMAND} apply -f "$DIR/additional-ca-sensor.yaml"
fi

echo "Creating collector secrets..."
${KUBE_COMMAND} apply -f "$DIR/collector-secret.yaml"
echo "Creating collector RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/collector-rbac.yaml" || print_rbac_instructions
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

echo "Creating sensor deployment..."
${KUBE_COMMAND} apply -f "$DIR/sensor.yaml"

{{ if not .CreateUpgraderSA }}
if [[ -f "${DIR}/upgrader-serviceaccount.yaml" ]]; then
    printf "%s\n\n%s\n" "Did not create the upgrader service account. To create it later, please run" "${KUBE_COMMAND} apply -f \"${DIR}/upgrader-serviceaccount.yaml\""
fi
{{- end }}
