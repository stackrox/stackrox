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

# Generic secrets and client config maps
kubectl create cm -n "{{.K8sConfig.Namespace}}" influxdb --from-file="$DIR/influxdb.conf"
kubectl create cm -n "{{.K8sConfig.Namespace}}" kapacitor --from-file="$DIR/kapacitor.conf"

# Monitoring CA
cp "$DIR/../ca.pem" "$DIR/monitoring-ca.pem"
kubectl create secret -n "{{.K8sConfig.Namespace}}" generic monitoring --from-file="$DIR/monitoring-ca.pem" --from-file="$DIR/monitoring-password" \
--from-file="$DIR/monitoring-db-cert.pem" --from-file="$DIR/monitoring-db-key.pem" \
--from-file="$DIR/monitoring-ui-cert.pem" --from-file="$DIR/monitoring-ui-key.pem"

kubectl create -f "${DIR}/monitoring.yaml"
echo "Monitoring has been deployed"
