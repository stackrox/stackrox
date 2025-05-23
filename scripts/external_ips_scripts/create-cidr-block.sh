#!/usr/bin/env bash
set -eou pipefail

ROX_ENDPOINT=${1:-localhost:8000}
cidr_block=${2:-8.8.8.0/24}
cidr_name=${3:-"testCIDR"}

clusters_json="$(curl --location --silent --request GET "https://${ROX_ENDPOINT}/v1/clusters" -k --header "Authorization: Bearer $ROX_API_TOKEN")"

cluster_id="$(echo "$clusters_json" | jq -r '.clusters[0].id')"

cidr_json='{"entity": {"cidr": "'"$cidr_block"'", "name": "'"$cidr_name"'", "id": ""}}'


create_cidr_block_response_json="$(curl --location --silent --request POST --data "$cidr_json" "https://${ROX_ENDPOINT}/v1/networkgraph/cluster/$cluster_id/externalentities" -k --header "Authorization: Bearer $ROX_API_TOKEN")"

echo "$create_cidr_block_response_json" | jq
