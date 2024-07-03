#!/usr/bin/env bash
set -eou pipefail

vm_name=$1
oc_bin=$2
env_version=$3
elastic_username=$4
elastic_password=$5
cluster_name_prefix=$6

DIR="$(cd "$(dirname "$0")" && pwd)"

"${DIR}"/create-vm.sh "$vm_name" "$oc_bin" "$env_version"
gcloud compute ssh "$vm_name" --project "$project" --command "~/install-and-run.sh $elastic_username $elastic_password $json_test_file $cluster_name_prefix"
