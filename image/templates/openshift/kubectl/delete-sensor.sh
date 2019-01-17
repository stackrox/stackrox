#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc delete -f "$DIR/sensor.yaml"
oc delete -n {{.Namespace}} secret sensor-tls benchmark-tls
oc delete -f "$DIR/sensor-rbac.yaml"

{{if .RuntimeSupport}}
oc -n {{.Namespace}} delete secret collector-tls collector-stackrox
{{- end}}

if [[ ! oc get -n {{.Namespace}} deploy/central ]]; then
    oc delete -n {{.Namespace}} secret stackrox
{{if .MonitoringEndpoint}}
    oc -n {{.Namespace}} delete secret monitoring-client
    oc -n {{.Namespace}} delete cm telegraf
{{- end}}
fi
