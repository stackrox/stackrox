#!/usr/bin/env bash

set -e

usage() {
    echo "usage: ./setup.sh <namespace> <anchore app name>"
}

if [[ $# -lt 2 ]]; then
  usage
  exit 1
fi

namespace="$1"
app_name="$2"

reqd_vulnerability_groups="debian:8"

if [[ ! $(anchore-cli --version) ]]; then
  pip3 install anchorecli
fi

ANCHORE_CLI_USER=admin
ANCHORE_CLI_PASS=$(kubectl -n "${namespace}" get secret --namespace "${namespace}" "${app_name}-anchore-engine" -o jsonpath="{.data.ANCHORE_ADMIN_PASSWORD}" | base64 --decode; echo)

# kubectl -n "${namespace}" port-forward "svc/${app_name}-anchore-engine-api" 8228:8228 > /dev/null &
# PID=$!
# ANCHORE_CLI_URL=http://localhost:8228/v1

# sleep 5

anchore-cli system wait
anchore-cli system status

anchore-cli system feeds list

# disable all top level feeds except vulnerabilities
feeds=$(anchore-cli system feeds list | tr -s ' ' | cut -d ' ' -f 1 | grep -v Feed | uniq | sed s/\(disabled\)// | grep -v vulnerabilities)
for feed in ${feeds}; do
  anchore-cli system feeds config --disable ${feed}
done

# delete all top level feeds except vulnerabilities
for feed in ${feeds}; do
  anchore-cli system feeds delete ${feed}
done

# enable vulnerabilities and the individual feed groups required by test.
anchore-cli system feeds config --enable vulnerabilities

groups=$(anchore-cli system feeds list | grep vulnerabilities | tr -s ' ' | cut -d ' ' -f 2 | grep -v disabled || true)
for group in ${groups}; do
  if [[ ! $(echo ${reqd_vulnerability_groups} | grep "${group}") ]]; then
    anchore-cli system feeds config --disable vulnerabilities --group ${group}
    anchore-cli system feeds delete vulnerabilities --group ${group}
  fi
done

anchore-cli system feeds list

# kill $PID
