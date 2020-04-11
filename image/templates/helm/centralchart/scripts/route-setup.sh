{{if eq .ClusterType.String "OPENSHIFT_CLUSTER"}}
#!/usr/bin/env bash
# This route will be accessible through central-stackrox.your-cluster-subdomain.
oc create route passthrough central -n stackrox --service central
# This route allows other clusters to connect via mutual TLS with the
# Server Name Indication "central.stackrox".
oc create route passthrough central-mtls -n stackrox --hostname=central.stackrox --service central
{{- end}}
