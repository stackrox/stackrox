#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl delete -f "$DIR/sensor.yaml"
kubectl delete -n stackrox secret sensor-tls benchmark-tls
kubectl delete -f "$DIR/sensor-rbac.yaml"

{{if .RuntimeSupport}}
kubectl -n stackrox delete secret collector-tls collector-stackrox
{{end}}

{{if .AdmissionController}}
kubectl -n stackrox delete validatingwebhookconfiguration stackrox
{{end}}

if ! kubectl get -n stackrox deploy/central; then
    kubectl delete -n stackrox secret stackrox
{{if .MonitoringEndpoint}}
    kubectl -n stackrox delete secret monitoring-client
    kubectl -n stackrox delete cm telegraf
{{- end}}
fi
