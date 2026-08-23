#!/usr/bin/env bash
set -eoux pipefail

# Generate plots for all file activity test results
# Processes all batch size configurations (10, 50, 100, 250, 500)

base_results_dir=${1:-perf}
num_sensors=${2:-1}
run_time=${3:-10m}

echo "Generating plots for all file activity test results..."
echo "Base results dir: ${base_results_dir}"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Batch sizes to process
batch_sizes=(10 50 100 250 500)

for batch in "${batch_sizes[@]}"; do
    workload="file-activity-batch-${batch}"

    without_policy_dir="${base_results_dir}/file_activity_results_${num_sensors}_${run_time}_${workload}_policy_false"
    with_policy_dir="${base_results_dir}/file_activity_results_${num_sensors}_${run_time}_${workload}_policy_true"
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
    output_dir="${base_results_dir}/plots_file-activity-batch-${batch}"
    if [ -d "$output_dir" ]; then
        plot_count=$(find "$output_dir" -name "*.png" | wc -l)
        echo "  ${output_dir}: ${plot_count} plots"
    fi
done
