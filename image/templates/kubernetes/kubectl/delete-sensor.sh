#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl delete -f "$DIR/sensor.yaml"
kubectl delete -n {{.Namespace}} secret sensor-tls benchmark-tls
kubectl delete -f "$DIR/sensor-rbac.yaml"

{{if .RuntimeSupport}}
kubectl -n {{.Namespace}} delete secret collector-tls collector-stackrox
{{- end}}

if [[ ! kubectl get -n {{.Namespace}} deploy/central ]]; then
    kubectl delete -n {{.Namespace}} secret stackrox
{{if .MonitoringEndpoint}}
    kubectl -n {{.Namespace}} delete secret monitoring-client
    kubectl -n {{.Namespace}} delete cm telegraf
{{- end}}
fi
