#!/usr/bin/env bash
set -eoux pipefail

# Generate plots for all file activity test results.
# Handles both the fake workload (default) and the berserker workload, which use
# different result-dir prefixes and workload names but the same per-pair plotter.
# Processes all batch sizes (10, 50, 100, 250, 500).

base_results_dir=${1:-perf}
num_sensors=${2:-1}
run_time=${3:-10m}
workload_type=${4:-fake}   # fake or berserker

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Batch sizes to process
batch_sizes=(10 50 100 250 500)

# Result-dir prefix and the set of test modes differ between the two workloads.
# Modes are "<dir-suffix>:<legend label>" and must mirror plot_utils.py
# (BERSERKER_MODES / FAKE_MODES).
if [[ "$workload_type" == "berserker" ]]; then
    dir_prefix="berserker_file_activity_results"
    modes=(
        "_baseline:Baseline (no policy)"
        "_alerting:Alerting policy"
        "_idlepolicy:Idle policy"
    )
else
    dir_prefix="file_activity_results"
    modes=(
        "_policy_false:Without Policy"
        "_policy_true:With Policy"
    )
fi

# Map a batch size to its workload name. Berserker names by event rate
# (batch * 10 events/sec); the fake workload names by batch.
workload_name_for() {
    local batch=$1
    if [[ "$workload_type" == "berserker" ]]; then
        echo "file-activity-$((batch * 10))"
    else
        echo "file-activity-batch-${batch}"
    fi
}

echo "Generating plots for all file activity test results..."
echo "Base results dir: ${base_results_dir}"
echo "Workload type: ${workload_type}"

for batch in "${batch_sizes[@]}"; do
    workload="$(workload_name_for "$batch")"
    output_dir="${base_results_dir}/plots_${workload}"

    # Build the (dir, label) pairs for the modes that actually have a results dir.
    # Missing modes are warned about but do not abort the batch; the batch is only
    # skipped when none of its modes were run.
    pair_args=()
    for mode in "${modes[@]}"; do
        suffix="${mode%%:*}"
        label="${mode#*:}"
        mode_dir="${base_results_dir}/${dir_prefix}_${num_sensors}_${run_time}_${workload}${suffix}"
        if [ -d "$mode_dir" ]; then
            pair_args+=("$mode_dir" "$label")
        else
            echo "WARNING: Directory not found: ${mode_dir}"
        fi
    done

    if [ ${#pair_args[@]} -eq 0 ]; then
        echo "No result dirs for batch size ${batch}; skipping"
        continue
    fi

    echo ""
    echo "=========================================="
    echo "Processing batch size: ${batch}"
    echo "Event rate: $((batch * 10)) events/sec"
    echo "=========================================="

    "${DIR}/MakePlots-file-activity.sh" \
        "$output_dir" \
        "${pair_args[@]}"

    echo "Plots for batch ${batch} saved to ${output_dir}"
done

echo ""
echo "=========================================="
echo "All plots generated!"
echo "=========================================="
echo ""
echo "Summary of generated plot directories:"
for batch in "${batch_sizes[@]}"; do
    workload="$(workload_name_for "$batch")"
    output_dir="${base_results_dir}/plots_${workload}"
    if [ -d "$output_dir" ]; then
        plot_count=$(find "$output_dir" -name "*.png" | wc -l)
        echo "  ${output_dir}: ${plot_count} plots"
    fi
done
