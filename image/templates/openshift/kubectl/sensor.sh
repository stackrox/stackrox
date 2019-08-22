#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc get project "stackrox" || oc new-project "stackrox"
oc project "stackrox"

oc apply -f "$DIR/sensor-rbac.yaml"
oc apply -f "$DIR/sensor-netpol.yaml"
oc apply -f  "$DIR/sensor-pod-security.yaml"

# OpenShift roles can be delayed to be added
sleep 5

if ! oc get secret/stackrox -n stackrox > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.ImageRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  oc create --namespace "stackrox" -f - <<EOF
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

oc secrets add serviceaccount/sensor secrets/stackrox --for=pull
oc secrets add serviceaccount/benchmark secrets/stackrox --for=pull

# Create secrets for sensor
oc create secret -n "stackrox" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/ca.pem"
oc -n "stackrox" label secret/sensor-tls app.kubernetes.io/name=stackrox 'auto-upgrade.stackrox.io/component=sensor'
oc create secret -n "stackrox" generic benchmark-tls --from-file="$DIR/benchmark-cert.pem" --from-file="$DIR/benchmark-key.pem" --from-file="$DIR/ca.pem"
oc -n "stackrox" label secret/benchmark-tls app.kubernetes.io/name=stackrox 'auto-upgrade.stackrox.io/component=sensor'

{{if ne .CollectionMethod "NO_COLLECTION"}}
if ! oc get secret/collector-stackrox -n stackrox > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.CollectorRegistry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  oc create --namespace "stackrox" -f - <<EOF
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

echo "Creating secrets for collector..."
oc create secret -n "stackrox" generic collector-tls --from-file="$DIR/collector-cert.pem" --from-file="$DIR/collector-key.pem" --from-file="$DIR/ca.pem"
oc -n "stackrox" label secret/collector-tls 'auto-upgrade.stackrox.io/component=sensor'

{{- end}}

{{if .MonitoringEndpoint}}
echo "Creating secrets for monitoring..."
oc create secret -n "stackrox" generic monitoring-client --from-file="$DIR/monitoring-client-cert.pem" --from-file="$DIR/monitoring-client-key.pem" --from-file="$DIR/monitoring-ca.pem"
oc -n "stackrox" label secret/monitoring-client 'auto-upgrade.stackrox.io/component=sensor'
oc create cm -n "stackrox" telegraf --from-file="$DIR/telegraf.conf"
oc -n "stackrox" label cm/telegraf 'auto-upgrade.stackrox.io/component=sensor'
{{- end}}

if [[ -d "$DIR/additional-cas" ]]; then
	echo "Creating secret for additional CAs for sensor..."
	oc -n stackrox create secret generic additional-ca-sensor --from-file="$DIR/additional-cas/"
	oc -n stackrox label secret/additional-ca-sensor app.kubernetes.io/name=stackrox  # no auto upgrade
fi

oc apply -f "$DIR/sensor.yaml"
