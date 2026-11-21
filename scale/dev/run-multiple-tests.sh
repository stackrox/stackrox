#!/usr/bin/env bash
set -eoux pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
STACKROX_DIR="${DIR}/../.."
WORKLOAD_DIR="${STACKROX_DIR}/scale/workloads"

export MAIN_IMAGE_TAG=4.9.x-377-g5493ce6cc3

for num_deployments in 250 500 1000 2500 5000 10000 25000; do
  input_workload="${WORKLOAD_DIR}/process-baselines.yaml"
  output_workload="${WORKLOAD_DIR}/process-baselines-${num_deployments}-deployments.yaml"
  sed "s|numDeployments: .*|numDeployments: $num_deployments|" "$input_workload" > "$output_workload" 
  for lock_baselines in "true" "false"; do
    error_code=1
    while [[ "$error_code" -gt 0 ]]; do
      error_code=0
      "${DIR}/TeardownTest.sh" || true
      "${DIR}/process-baselines-perf-test.sh" 1 10m "$lock_baselines" "$num_deployments" &> delete-log-1-10m-"${lock_baselines}"-"${num_deployments}".txt || error_code=$?
    done
  done
done
