#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc get project "{{.K8sConfig.Namespace}}" || oc new-project "{{.K8sConfig.Namespace}}"

echo "Creating Central RBAC..."
oc apply -f "${DIR}/central-rbac.yaml"

if ! oc get secret/stackrox -n {{.K8sConfig.Namespace}} > /dev/null; then
  registry_auth="$("${DIR}/docker-auth.sh" -m k8s "{{.K8sConfig.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  oc create --namespace "{{.K8sConfig.Namespace}}" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: {{.K8sConfig.Namespace}}
type: kubernetes.io/dockerconfigjson
EOF
fi

{{if not .K8sConfig.MonitoringType.None}}
# Add monitoring client configmap
kubectl create cm -n "{{.K8sConfig.Namespace}}" telegraf --from-file="$DIR/telegraf.conf"
{{- end}}

oc -n {{.K8sConfig.Namespace}} secrets add serviceaccount/central secrets/stackrox --for=pull

oc create secret -n "{{.K8sConfig.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
oc create secret -n "{{.K8sConfig.Namespace}}" generic central-jwt --from-file="$DIR/jwt-key.der"
oc apply -f "$DIR/central.yaml"
