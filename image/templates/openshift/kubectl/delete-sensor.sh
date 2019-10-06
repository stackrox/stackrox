#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc -n stackrox delete secret -l auto-upgrade.stackrox.io/component=sensor

if ! oc get -n stackrox deploy/central > /dev/null; then
    oc delete -n stackrox secret stackrox
{{if .MonitoringEndpoint}}
    oc -n stackrox delete secret monitoring-client
    oc -n stackrox delete cm telegraf
{{- end}}
fi
