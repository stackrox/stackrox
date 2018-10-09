#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl get namespace {{.K8sConfig.Namespace}} > /dev/null || kubectl create namespace {{.K8sConfig.Namespace}}

if ! kubectl get secret/{{.K8sConfig.ImagePullSecret}} -n {{.K8sConfig.Namespace}} > /dev/null; then
  if [ -z "${REGISTRY_USERNAME}" ]; then
    echo -n "Username for {{.K8sConfig.Registry}}: "
    read REGISTRY_USERNAME
    echo
  fi
  if [ -z "${REGISTRY_PASSWORD}" ]; then
    echo -n "Password for {{.K8sConfig.Registry}}: "
    read -s REGISTRY_PASSWORD
    echo
  fi

  kubectl create secret docker-registry \
    "{{.K8sConfig.ImagePullSecret}}" --namespace "{{.K8sConfig.Namespace}}" \
    --docker-server={{.K8sConfig.Registry}} \
    --docker-username="${REGISTRY_USERNAME}" \
    --docker-password="${REGISTRY_PASSWORD}" \
    --docker-email="support@stackrox.com"

	echo
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
