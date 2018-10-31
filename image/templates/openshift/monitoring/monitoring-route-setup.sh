#!/usr/bin/env bash
# This route will be accessible through monitoring-{{.K8sConfig.Namespace}}.your-cluster-subdomain.
oc create route passthrough monitoring -n {{.K8sConfig.Namespace}} --service monitoring
# This route allows other clusters to connect via mutual TLS with the
# Server Name Indication "monitoring.stackrox".
oc create route passthrough monitoring-mtls -n {{.K8sConfig.Namespace}} --hostname=monitoring.stackrox --service monitoring
