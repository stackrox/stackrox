#!/usr/bin/env bash

# Periodically check the cluster and SR deployment to debug test failures due to
# connectivity issues.

set +e

API_HOSTNAME=localhost
API_PORT=8000
if [[ "${LOAD_BALANCER}" == "lb" ]]; then
  API_HOSTNAME=$(./scripts/k8s/get-lb-ip.sh)
  API_PORT=443
fi
API_ENDPOINT="${API_HOSTNAME}:${API_PORT}"
METADATA_URL="https://${API_ENDPOINT}/v1/metadata"
echo "StackRox METADATA_URL is set to ${METADATA_URL}"

set -x
while true; do
  date
  kubectl get nodes
  ps axw | grep kube
  curl --silent --insecure --show-error "${METADATA_URL}" | jq
  sleep 60
done
