#!/usr/bin/env bash
set -eou pipefail

ROX_ENDPOINT=${1:-localhost:8000}

clusters_json="$(curl --location --silent --request GET "https://${ROX_ENDPOINT}/v1/clusters" -k --header "Authorization: Bearer $ROX_API_TOKEN")"

cluster_id="$(echo "$clusters_json" | jq -r '.clusters[0].id')"

query="Cluster%3Aremote%2BNamespace%3Aqa&includePorts=true"
network_policy_json="$(curl --location --silent --request GET "https://${ROX_ENDPOINT}/v1/networkpolicies/generate/${cluster_id}?deleteExisting=NONE&query=$query" -k --header "Authorization: Bearer $ROX_API_TOKEN")"

network_policy=$(echo "$network_policy_json" | jq -r '.modification.applyYaml')

printf "%b\n" "$network_policy"
