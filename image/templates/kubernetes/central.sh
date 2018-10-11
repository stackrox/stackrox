#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl get namespace {{.K8sConfig.Namespace}} > /dev/null || kubectl create namespace {{.K8sConfig.Namespace}}

if ! kubectl get secret/{{.K8sConfig.ImagePullSecret}} -n {{.K8sConfig.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.K8sConfig.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  kubectl create --namespace "{{.K8sConfig.Namespace}}" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: {{.K8sConfig.ImagePullSecret}}
  namespace: {{.K8sConfig.Namespace}}
type: kubernetes.io/dockerconfigjson
EOF
fi

{{if not .K8sConfig.MonitoringType.None}}
# Add monitoring client configmap
kubectl create cm -n "{{.K8sConfig.Namespace}}" telegraf --from-file="$DIR/telegraf.conf"
{{- end}}

# Add Central secrets
kubectl create secret -n "{{.K8sConfig.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
kubectl create secret -n "{{.K8sConfig.Namespace}}" generic central-jwt --from-file="$DIR/jwt-key.der"
kubectl create -f "${DIR}/central.yaml"

echo "Central has been deployed"
