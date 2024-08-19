#!/usr/bin/env bash

set -euo pipefail
set -x

cluster_name="${1:-remote}"

ROX_ADMIN_PASSWORD="${ROX_ADMIN_PASSWORD:-$(< deploy/k8s/central-deploy/password)}"
ROX_API_ENDPOINT="${ROX_API_ENDPOINT:-localhost:8000}"

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
    echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

ROX_CLUSTER_ID="$(curl -sk --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") "https://${ROX_API_ENDPOINT}/v1/clusters?query=cluster:${cluster_name}" | jq -r '.clusters[0].id')"
if [[ -z "$ROX_CLUSTER_ID" ]]; then
    echo >&2 "No such cluster: ${cluster_name}"
    exit 1
fi

curl -sk --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") "https://${ROX_API_ENDPOINT}/v1/sensorupgrades/cluster/${ROX_CLUSTER_ID}" -X POST
