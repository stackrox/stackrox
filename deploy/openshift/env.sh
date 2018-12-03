#!/usr/bin/env bash
set -e

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export NAMESPACE=${NAMESPACE:-stackrox}
echo "Kubernetes namespace set to $NAMESPACE"

export MONITORING_SUPPORT=${MONITORING_SUPPORT:-true}
export ROX_HTPASSWD_AUTH=${ROX_HTPASSWD_AUTH:-true}
