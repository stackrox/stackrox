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

# Result-dir prefix differs between the two workloads.
if [[ "$workload_type" == "berserker" ]]; then
    dir_prefix="berserker_file_activity_results"
else
    dir_prefix="file_activity_results"
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

    without_policy_dir="${base_results_dir}/${dir_prefix}_${num_sensors}_${run_time}_${workload}_policy_false"
    with_policy_dir="${base_results_dir}/${dir_prefix}_${num_sensors}_${run_time}_${workload}_policy_true"
    output_dir="${base_results_dir}/plots_${workload}"

    # Check if directories exist
    if [ ! -d "$without_policy_dir" ]; then
        echo "WARNING: Directory not found: ${without_policy_dir}"
        echo "Skipping batch size ${batch}"
        continue
    fi

    if [ ! -d "$with_policy_dir" ]; then
        echo "WARNING: Directory not found: ${with_policy_dir}"
        echo "Skipping batch size ${batch}"
        continue
    fi

    echo ""
    echo "=========================================="
    echo "Processing batch size: ${batch}"
    echo "Event rate: $((batch * 10)) events/sec"
    echo "=========================================="

    "${DIR}/MakePlots-file-activity.sh" \
        "$without_policy_dir" \
        "$with_policy_dir" \
        "$output_dir"

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
