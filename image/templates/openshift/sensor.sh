#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc get project "{{.Namespace}}" || oc new-project "{{.Namespace}}"

oc create -f "$DIR/sensor-rbac.yaml"

if ! oc get secret/stackrox -n {{.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.ImageRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  oc create --namespace "{{.Namespace}}" -f - <<EOF
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

oc secrets add serviceaccount/sensor secrets/stackrox --for=pull
oc secrets add serviceaccount/benchmark secrets/stackrox --for=pull

# Create secrets for sensor
oc create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/ca.pem"

{{if .RuntimeSupport}}
if ! oc get secret/collector-stackrox -n {{.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.CollectorRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  oc create --namespace "{{.Namespace}}" -f - <<EOF
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

oc secrets add serviceaccount/collector secrets/collector-stackrox --for=pull

echo "Creating secrets for collector..."
kubectl create secret -n "{{.Namespace}}" generic collector-tls --from-file="$DIR/collector-cert.pem" --from-file="$DIR/collector-key.pem" --from-file="$DIR/ca.pem"

{{- end}}

oc create -f "$DIR/sensor.yaml"
