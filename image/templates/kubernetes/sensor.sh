#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl get namespace {{.Namespace}} > /dev/null || kubectl create namespace {{.Namespace}}

if ! kubectl get secret/stackrox -n {{.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  kubectl create --namespace "{{.Namespace}}" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: {{.Namespace}}
type: kubernetes.io/dockerconfigjson
EOF
fi

{{if .RuntimeSupport}}
if ! kubectl get secret/collector-stackrox -n {{.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.CollectorRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  kubectl create --namespace "{{.Namespace}}" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: collector-stackrox
  namespace: {{.Namespace}}
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
	echo "    kubectl create clusterrolebinding temporary-admin --clusterrole=cluster-admin --user you@example.com"
	echo
	echo "(Be sure to use the full username your cluster knows for you.)"
	echo
	echo "Then, rerun this script."
	echo
	echo "Finally, revoke your temporary privileges:"
	echo "    kubectl delete clusterrolebinding temporary-admin"
	echo
	echo "Contact your cluster administrator if you cannot obtain sufficient permission."
	exit 1
}

echo "Creating RBAC roles..."
kubectl apply -f "$DIR/sensor-rbac.yaml" || print_rbac_instructions


{{if .MonitoringEndpoint}}
echo "Creating secrets for monitoring..."
kubectl create secret -n "{{.Namespace}}" generic monitoring --from-file="$DIR/monitoring-password" --from-file="$DIR/monitoring-ca.pem"
kubectl create cm -n "{{.Namespace}}" telegraf --from-file="$DIR/telegraf.conf"
{{- end}}


echo "Creating secrets for sensor..."
kubectl create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/ca.pem"

{{if .RuntimeSupport}}
echo "Creating secrets for collector..."
kubectl create secret -n "{{.Namespace}}" generic collector-tls --from-file="$DIR/collector-cert.pem" --from-file="$DIR/collector-key.pem" --from-file="$DIR/ca.pem"
{{- end}}

echo "Creating deployment..."
kubectl create -f "$DIR/sensor.yaml"
