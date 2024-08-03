#!/usr/bin/env bash
set -eou pipefail

json_tests_file=$1
cluster_name_prefix=$2
utilities_dir=$3

DIR="$(cd "$(dirname "$0")" && pwd)"

ntests="$(jq '.perfTests | length' "$json_tests_file")"
for ((i = 0; i < ntests; i = i + 1)); do
        version="$(jq .perfTests "$json_tests_file" | jq .["$i"])"
	num_namespaces="$(echo "$version" | jq .numNamespaces)"
	num_deployments="$(echo "$version" | jq .numDeployments)"
	num_pods="$(echo "$version" | jq .numPods)"

	cluster_name="${cluster_name_prefix}-${i}"
	"${DIR}"/run-test.sh "${num_namespaces}" "${num_deployments}" "${num_pods}" "$cluster_name" "$utilities_dir" &
done
