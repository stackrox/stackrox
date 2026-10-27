#!/usr/bin/env bash
set -eou pipefail

# Run a full file-activity perf-test matrix described by a JSON run config:
# every policy (outer) x every rate (inner). Replaces run-file-activity-tests.sh;
# the batch sizes, test modes and workload type that used to be hardcoded now come
# from the config, so both the fake and berserker workloads use this one runner.
#
# Usage: run-perf-tests.sh <config.json>
# Example: ./run-perf-tests.sh configs/berserker-file-activity.json

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

config="${1:?usage: $0 <config.json>}"
if [[ ! -f "$config" ]]; then
  echo "Error: config not found: ${config}"; exit 1
fi

name="$(jq -r '.name' "$config")"
results_base_dir="$(jq -r '.resultsBaseDir' "$config")"

# Anchor a relative results base under scale/dev (matches perf-test.sh).
if [[ "$results_base_dir" = /* ]]; then
  run_dir="${results_base_dir}/${name}"
else
  run_dir="${DIR}/${results_base_dir}/${name}"
fi

# Stash the run config alongside the results so the plotters are config-driven
# (rate values, policy nicknames/labels, runTime, numDeployments) rather than
# parsing directory names.
mkdir -p "$run_dir"
cp "$config" "${run_dir}/run-config.json"

mapfile -t rates < <(jq -r '.rates[].rate' "$config")
mapfile -t policies < <(jq -r '.policies[].nickname' "$config")

# Run every rate for each policy. Iterate policy-outer so all rates for one scenario
# complete before moving to the next.
for policy in "${policies[@]}"; do
  echo "=== Running ${name} tests: policy=${policy} ==="
  for rate in "${rates[@]}"; do
    results_dir="${run_dir}/rate-${rate}__${policy}"

    # Resume support: skip a cell whose results dir already exists, so an
    # interrupted run can be restarted by re-invoking with the same config (each
    # cell takes many minutes). Delete a dir to force its cell to re-run; delete a
    # partial dir before resuming.
    if [[ -d "$results_dir" ]]; then
      echo "Results dir already exists, skipping: ${results_dir}"
      continue
    fi

    echo "Testing rate ${rate} (policy=${policy})..."
    "${DIR}/perf-test-cell.sh" "$config" "$rate" "$policy"

    # Clean up between cells.
    echo "Cleaning up before next cell..."
    "${DIR}/TeardownTest.sh" || true
    sleep 30
  done
done

echo "All ${name} tests completed!"
echo "Results are in ${run_dir}"
