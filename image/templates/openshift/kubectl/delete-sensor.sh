#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc delete -f "$DIR/sensor.yaml"
oc delete -n stackrox secret sensor-tls benchmark-tls
oc delete -f "$DIR/sensor-rbac.yaml"

{{if ne .CollectionMethod "NO_COLLECTION"}}
oc -n stackrox delete secret collector-tls collector-stackrox
{{- end}}

if ! oc get -n stackrox deploy/central > /dev/null; then
    oc delete -n stackrox secret stackrox
{{if .MonitoringEndpoint}}
    oc -n stackrox delete secret monitoring-client
    oc -n stackrox delete cm telegraf
{{- end}}
fi
