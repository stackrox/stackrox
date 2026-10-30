#!/usr/bin/env bash
set -eou pipefail

# Generate per-rate comparison plots for a config-driven perf run. Replaces
# MakeAllPlots-file-activity.sh: the rates and policies come from the run config
# (<run_dir>/run-config.json, written by run-perf-tests.sh) instead of hardcoded
# batch-size lists and workload-type branches, so one script handles both the fake
# and berserker workloads.
#
# Usage: make-all-plots.sh <run_dir> [output_base]
#   run_dir:     a run's results dir (holds run-config.json and the rate-*__* cells),
#                e.g. perf/berserker-file-activity
#   output_base: where to write the per-rate plot dirs (default: <run_dir>)
#
# For each rate it overlays one line per policy (the per-rate comparison) via
# MakePlots-file-activity.sh. For the scaling comparison across rates use
# plot-batch-comparison.py <run_dir>.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

run_dir="${1:?usage: $0 <run_dir> [output_base]}"
output_base="${2:-$run_dir}"

config="${run_dir}/run-config.json"
if [[ ! -f "$config" ]]; then
  echo "Error: ${config} not found. Pass a run results dir that holds run-config.json."
  exit 1
fi

name="$(jq -r '.name' "$config")"
mapfile -t rates < <(jq -r '.rates[].rate' "$config")
mapfile -t policies < <(jq -r '.policies[].nickname' "$config")

echo "Generating plots for run '${name}' in ${run_dir}"
echo "Rates: ${rates[*]}"
echo "Policies: ${policies[*]}"

for rate in "${rates[@]}"; do
  output_dir="${output_base}/plots_rate-${rate}"

  # Build (dir, label) pairs for the policies that actually have a cell dir. A
  # missing policy cell is warned about but does not abort; the rate is skipped only
  # when none of its policies ran.
  pair_args=()
  for policy in "${policies[@]}"; do
    cell="${run_dir}/rate-${rate}__${policy}"
    if [[ -d "$cell" ]]; then
      pair_args+=("$cell" "$policy")
    else
      echo "WARNING: cell dir not found: ${cell}"
    fi
  done

  if [[ ${#pair_args[@]} -eq 0 ]]; then
    echo "No cells for rate ${rate}; skipping"
    continue
  fi

  echo ""
  echo "=========================================="
  echo "Processing rate: ${rate} events/sec"
  echo "=========================================="
  "${DIR}/MakePlots-file-activity.sh" "$output_dir" "${pair_args[@]}"
  echo "Plots for rate ${rate} saved to ${output_dir}"
done

echo ""
echo "All plots generated. Per-rate plot dirs under ${output_base}/plots_rate-*"
echo "For the scaling comparison across rates: plot-batch-comparison.py ${run_dir}"
