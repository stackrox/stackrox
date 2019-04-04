#!/usr/bin/env bash

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

${KUBE_COMMAND} get namespace stackrox > /dev/null || ${KUBE_COMMAND} create namespace stackrox

if ! ${KUBE_COMMAND} get secret/stackrox -n stackrox > /dev/null; then
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
type: kubernetes.io/dockerconfigjson
EOF
fi

{{if ne .CollectionMethod "NO_COLLECTION"}}
if ! ${KUBE_COMMAND} get secret/collector-stackrox -n stackrox > /dev/null; then
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

echo "Creating RBAC roles..."
${KUBE_COMMAND} apply -f "$DIR/sensor-rbac.yaml" || print_rbac_instructions
echo "Creating network policies..."
${KUBE_COMMAND} apply -f "$DIR/sensor-netpol.yaml" || exit 1

{{if .AdmissionController}}
${KUBE_COMMAND} apply -f "$DIR/admission-controller.yaml"
{{- end}}

{{if .MonitoringEndpoint}}
echo "Creating secrets for monitoring..."
${KUBE_COMMAND} create secret -n "stackrox" generic monitoring-client --from-file="$DIR/monitoring-client-cert.pem" --from-file="$DIR/monitoring-client-key.pem" --from-file="$DIR/monitoring-ca.pem"
${KUBE_COMMAND} create cm -n "stackrox" telegraf --from-file="$DIR/telegraf.conf"
{{- end}}


echo "Creating secrets for sensor..."
${KUBE_COMMAND} create secret -n "stackrox" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/ca.pem"
${KUBE_COMMAND} create secret -n "stackrox" generic benchmark-tls --from-file="$DIR/benchmark-cert.pem" --from-file="$DIR/benchmark-key.pem" --from-file="$DIR/ca.pem"

{{if ne .CollectionMethod "NO_COLLECTION"}}
echo "Creating secrets for collector..."
${KUBE_COMMAND} create secret -n "stackrox" generic collector-tls --from-file="$DIR/collector-cert.pem" --from-file="$DIR/collector-key.pem" --from-file="$DIR/ca.pem"
{{- end}}

echo "Creating deployment..."
${KUBE_COMMAND} apply -f "$DIR/sensor.yaml"
