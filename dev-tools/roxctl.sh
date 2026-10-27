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

roxctl_bin="$DIR/../bin/linux_amd64/roxctl"
if [[ "$(uname)" == "Darwin"* ]]; then
  roxctl_bin="$DIR/../bin/darwin_amd64/roxctl"
fi

cache=""
if [[ -n "${KUBECONFIG}" ]]; then
  if command -v md5sum &> /dev/null; then
    cache="/tmp/$(md5sum ${KUBECONFIG} | awk '{print $1}')"
  elif command -v md5 &> /dev/null; then
    cache="/tmp/$(md5 -q ${KUBECONFIG})"
  else
    echo "Programs md5 or md5sum were not found. Exiting."
    exit 1
  fi
fi

# Try to get endpoint and password from various sources
endpoint="localhost:8000"
password=""

if [[ -f "$cache" ]]; then
  # Use cached credentials if available
  endpoint=$(cat "${cache}" | awk '{print $1}')
  password=$(cat "${cache}" | awk '{print $2}')
elif is_operator_on_openshift; then
  # Get credentials from operator deployment on OpenShift
  endpoint="$(oc get route -n stackrox central -o json | jq -r '.spec.host'):443"
  password=$(oc get secret -n stackrox central-htpasswd -o json | jq -r '.data.password' | base64 --decode)
  printf "$endpoint\t$password" > "${cache}"
elif [[ -f "$DIR/../deploy/central-deploy/password" ]]; then
  # ROX-37649: Read from deploy/central-deploy (legacy deploy location)
  password="$(cat "$DIR/../deploy/central-deploy/password")"
elif kubectl get secret -n stackrox central-htpasswd &>/dev/null; then
  # Get password from Kubernetes secret (operator or Helm deployment)
  password=$(kubectl get secret -n stackrox central-htpasswd -o json | jq -r '.data.password' | base64 --decode)
else
  echo "Error: Could not find Central password. Tried:"
  echo "  1. Cache file: $cache"
  echo "  2. OpenShift route/secret"
  echo "  3. deploy/central-deploy/password"
  echo "  4. stackrox/central-htpasswd secret"
  exit 1
fi

"$roxctl_bin" -e "https://${endpoint}" -p "$password" --insecure-skip-tls-verify "$@"
