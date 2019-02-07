{{if eq .ClusterType.String "OPENSHIFT_CLUSTER"}}
#!/usr/bin/env bash
# This route will be accessible through monitoring-stackrox.your-cluster-subdomain.
oc create route passthrough monitoring -n stackrox --service monitoring
# This route allows other clusters to connect via mutual TLS with the
# Server Name Indication "monitoring.stackrox".
oc create route passthrough monitoring-mtls -n stackrox --hostname=monitoring.stackrox --service monitoring

route_hostname="$(oc -n stackrox get route monitoring -o jsonpath='{.spec.host}')"
if [[ $? != 0 ]]; then
	echo >&2 "It seems like there was an issue creating the monitoring route."
	exit 1
fi

echo >&2 "Use ${route_hostname}:443 as the monitoring endpoint when adding remote clusters."
{{- end}}
