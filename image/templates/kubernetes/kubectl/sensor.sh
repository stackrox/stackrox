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

{{if and (ne .ImageRemote "stackrox-launcher-project-1/stackrox") (ne .ImageRemote "cloud-marketplace/stackrox-launcher-project-1/stackrox-kubernetes-security")}}
if ! ${KUBE_COMMAND} get namespace stackrox > /dev/null; then
  ${KUBE_COMMAND} create -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  annotations:
    openshift.io/node-selector: ""
  name: stackrox
EOF
fi

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
{{- end}}

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
echo "Creating Pod Security Policies..."
${KUBE_COMMAND} apply -f "$DIR/sensor-pod-security.yaml"

{{ if .CreateUpgraderSA}}
echo "Creating upgrader service account"
${KUBE_COMMAND} apply -f "${DIR}/upgrader-serviceaccount.yaml" || print_rbac_instructions
{{- end}}

{{if .AdmissionController}}
${KUBE_COMMAND} apply -f "$DIR/admission-controller.yaml"
{{- else}}
echo "Deleting admission controller webhook, if it exists"
${KUBE_COMMAND} delete -f "$DIR/admission-controller.yaml" || true
{{- end}}

{{if .MonitoringEndpoint}}
echo "Creating secrets for monitoring..."
if ${KUBE_COMMAND} create secret -n "stackrox" generic monitoring-client --from-file="$DIR/monitoring-client-cert.pem" --from-file="$DIR/monitoring-client-key.pem" --from-file="$DIR/monitoring-ca.pem"; then
	${KUBE_COMMAND} -n "stackrox" label secret/monitoring-client 'auto-upgrade.stackrox.io/component=sensor'
fi
if ${KUBE_COMMAND} create cm -n "stackrox" telegraf --from-file="$DIR/telegraf.conf"; then
	${KUBE_COMMAND} -n "stackrox" label cm/telegraf 'auto-upgrade.stackrox.io/component=sensor'
fi
{{- end}}


echo "Creating secrets for sensor..."
${KUBE_COMMAND} create secret -n "stackrox" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/ca.pem"
${KUBE_COMMAND} -n "stackrox" label secret/sensor-tls 'auto-upgrade.stackrox.io/component=sensor'

echo "Creating secrets for collector..."
${KUBE_COMMAND} create secret -n "stackrox" generic collector-tls --from-file="$DIR/collector-cert.pem" --from-file="$DIR/collector-key.pem" --from-file="$DIR/ca.pem"
${KUBE_COMMAND} -n "stackrox" label secret/collector-tls 'auto-upgrade.stackrox.io/component=sensor'

if [[ -d "$DIR/additional-cas" ]]; then
	echo "Creating secret for additional CAs for sensor..."
	${KUBE_COMMAND} -n stackrox create secret generic additional-ca-sensor --from-file="$DIR/additional-cas/"
	${KUBE_COMMAND} -n stackrox label secret/additional-ca-sensor app.kubernetes.io/name=stackrox  # no auto upgrade
fi

echo "Creating deployment..."
${KUBE_COMMAND} apply -f "$DIR/sensor.yaml"

{{ if not .CreateUpgraderSA}}
if [[ -f "${DIR}/upgrader-serviceaccount.yaml" ]]; then
    printf "%s\n\n%s\n" "Did not create the upgrader service account. To create it later, please run" "${KUBE_COMMAND} apply -f \"${DIR}/upgrader-serviceaccount.yaml\""
fi
{{- end}}
