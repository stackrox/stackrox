#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc get namespace {{.K8sConfig.Namespace}} > /dev/null || oc create namespace {{.K8sConfig.Namespace}}

echo "Creating Monitoring RBAC..."
oc apply -f "${DIR}/monitoring-rbac.yaml"

if ! oc get secret/stackrox -n {{.K8sConfig.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/../docker-auth.sh" -m k8s "{{.K8sConfig.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  oc create --namespace "{{.K8sConfig.Namespace}}" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: {{.K8sConfig.Namespace}}
type: kubernetes.io/dockerconfigjson
EOF
fi

# Generic secrets and client config maps
oc create cm -n "{{.K8sConfig.Namespace}}" influxdb --from-file="$DIR/influxdb.conf"

# Monitoring CA
cp "$DIR/../ca.pem" "$DIR/monitoring-ca.pem"
oc create secret -n "{{.K8sConfig.Namespace}}" generic monitoring --from-file="$DIR/monitoring-ca.pem" --from-file="$DIR/monitoring-password" \
--from-file="$DIR/monitoring-db-cert.pem" --from-file="$DIR/monitoring-db-key.pem" \
--from-file="$DIR/monitoring-ui-cert.pem" --from-file="$DIR/monitoring-ui-key.pem"

oc apply -f "${DIR}/monitoring.yaml"
echo "Monitoring has been deployed"
