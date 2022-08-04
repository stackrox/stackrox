#!/usr/bin/env bash
set -eo pipefail

# Usage: ./roxctl.sh <args>
# Small development wrapper around roxctl which automatically tries to guess Central's credentials in development environments deployed
# by the operator or local deploy scripts based on the configured $KUBECONFIG.

is_operator_on_openshift() {
  local result=0
  kubectl get clusterversions.config.openshift.io version | grep -v "No resources found" > /dev/null
  if [[ "$?" -ne "0" ]]; then
      result=1
  fi

  kubectl get centrals.platform.stackrox.io -n stackrox | grep -v "No resources found" > /dev/null
  if [[ "$?" -ne "0" ]]; then
    result=1
  fi
  return "$result"
}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

roxctl_bin="$DIR/../bin/linux/roxctl"
if [[ "$(uname)" == "Darwin"* ]]; then
  roxctl_bin="$DIR/../bin/darwin/roxctl"
fi

cache=""
if [[ -n "${KUBECONFIG}" ]]; then
  cache="/tmp/$(md5 -q ${KUBECONFIG})"
fi
endpoint="localhost:8000"
password="$(cat "$DIR/../deploy/k8s/central-deploy/password")"

if [[ -n "$cache" ]]; then
  endpoint=$(cat "${cache}" | awk '{print $1}')
  password=$(cat "${cache}" | awk '{print $2}')
elif is_operator_on_openshift; then
  endpoint="$(oc get route -n stackrox central -o json | jq -r '.spec.host'):443"
  password=$(oc get secret -n stackrox central-htpasswd -o json | jq -r '.data.password' | base64 --decode)
  printf "$endpoint\t$password" > "${cache}"
fi

"$roxctl_bin" -e "https://${endpoint}" -p "$password" --insecure-skip-tls-verify "$@"
