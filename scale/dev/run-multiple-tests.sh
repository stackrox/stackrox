#!/usr/bin/env bash
set -eou pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
STACKROX_DIR="${DIR}/../.."
WORKLOAD_DIR="${STACKROX_DIR}/scale/workloads"

for num_deployments in 250 500 1000 2500 5000 10000 25000 50000 100000; do
  input_workload="${WORKLOAD_DIR}/process-baselines.yaml"
  output_workload="${WORKLOAD_DIR}/process-baselines-${num_deployments}.yaml"
  sed "s|numDeployments: .*|numDeployments: $num_deployments|" "$input_workload" > "$output_workload" 
  for lock_baselines in "true" "false"; do
    "${DIR}/TeardownTest.sh"
    "${DIR}/process-baselines-perf-test.sh" 1 10m "$lock_baselines" "$num_deployments" &> delete-log-1-10m-"${lock_baselines}"-"${num_deployments}".txt
  done
done
