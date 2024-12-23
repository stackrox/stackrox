#!/usr/bin/env bash
set -eou pipefail

output_dir=$1

info_dir="$output_dir"/info
log_dir="$output_dir"/log

mkdir -p "$info_dir"
mkdir -p "$log_dir"

mapfile -t deployments < <(kubectl -n stackrox get deployment | tail -n +2 | awk '{print $1}')

for deployment in "${deployments[@]}"; do
    kubectl -n stackrox describe deployment "$deployment" > "${info_dir}/${deployment}-describe.txt"
    kubectl -n stackrox get deployment "$deployment" -o yaml > "${info_dir}/${deployment}.yaml"
done

kubectl -n stackrox describe ds collector > "${info_dir}/collector-describe.txt"
kubectl -n stackrox get ds collector -o yaml > "${info_dir}/collector.yaml"

mapfile -t pods < <(kubectl -n stackrox get pods | tail -n +2 | awk '{print $1}')

for pod in "${pods[@]}"; do
    kubectl -n stackrox logs "$pod" > "${log_dir}/${pod}.txt"
done

kubectl -n stackrox get pods &> "${output_dir}/pods.txt"
