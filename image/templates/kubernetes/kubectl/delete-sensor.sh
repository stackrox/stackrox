#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl -n stackrox delete secret -l auto-upgrade.stackrox.io/component=sensor

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
