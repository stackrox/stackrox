#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

{{.K8sConfig.Command}} get namespace stackrox > /dev/null || {{.K8sConfig.Command}} create namespace stackrox

if ! {{.K8sConfig.Command}} get secret/stackrox -n stackrox > /dev/null; then
  registry_auth="$("${DIR}/../../docker-auth.sh" -m k8s "{{.K8sConfig.Registry}}")"
  [[ -n "$registry_auth" ]] || { echo >&2 "Unable to get registry auth info." ; exit 1 ; }
  {{.K8sConfig.Command}} create --namespace "stackrox" -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: stackrox
  namespace: stackrox
type: kubernetes.io/dockerconfigjson
EOF
fi
