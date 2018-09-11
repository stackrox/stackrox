#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

OC_PROJECT="{{.K8sConfig.Namespace}}"

oc get project "${OC_PROJECT}" || oc new-project "${OC_PROJECT}"

echo "Adding cluster roles to the service account..."
oc create -f "${DIR}/central-rbac.yaml"

oc create secret -n "{{.K8sConfig.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
oc create secret -n "{{.K8sConfig.Namespace}}" generic central-jwt --from-file="$DIR/jwt-key.der"
oc create -f "$DIR/central.yaml"
