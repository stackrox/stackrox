#!/usr/bin/env bash
set -eoux pipefail

# Generate comparison plots for file activity performance tests
# Compares results without policy vs with policy

without_policy_dir=$1
with_policy_dir=$2
output_dir=$3

# Find python (try common locations)
python_bin=""
if command -v python3 &> /dev/null; then
    python_bin="python3"
elif command -v python &> /dev/null; then
    python_bin="python"
elif [ -f /home/jvirtane/.pyenv/versions/3.11.4/bin/python3.11 ]; then
    python_bin=/home/jvirtane/.pyenv/versions/3.11.4/bin/python3.11
else
    echo "Error: Python not found"
    exit 1
fi

# Create output directory
mkdir -p "$output_dir"

echo "Generating plots with component-specific baselines..."
echo "  Without policy dir: ${without_policy_dir}"
echo "  With policy dir: ${with_policy_dir}"
echo "  Output dir: ${output_dir}"
echo ""
echo "Baseline logic:"
echo "  - sensor: t=0 = first sensor data point"
echo "  - central/central-db: t=0 = when deployments >= 100 (or 90% of max)"
echo "  - berserker runs: t=0 = recorded berserker-ready time (all components)"
echo ""

# CPU and Memory plots for each component. collector and fact are containers in
# the collector DaemonSet pod (fact monitors file activity); their metric files
# only exist when collector metrics were collected, otherwise the plot is skipped.
for container in central central-db sensor collector fact; do
    echo "Plotting ${container} metrics..."

    # Memory usage (container_memory_usage_bytes: cgroup total, incl. reclaimable)
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${container}_mem.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${container}_mem.txt" "With Policy" \
        "${container} Memory Usage" "Memory (bytes)" \
        "${without_policy_dir}" "${with_policy_dir}" "${container}" \
        "${output_dir}/${container}_mem_usage.png"

    # Working set memory (usage minus reclaimable cache; the OOM-relevant number).
    # Only produced for runs scraped after this series was added; skip otherwise.
    if [[ -f "${without_policy_dir}/metrics_${container}_mem_workingset.txt" || \
          -f "${with_policy_dir}/metrics_${container}_mem_workingset.txt" ]]; then
        $python_bin plot-file-activity.py \
            "${without_policy_dir}/metrics_${container}_mem_workingset.txt" "Without Policy" \
            "${with_policy_dir}/metrics_${container}_mem_workingset.txt" "With Policy" \
            "${container} Working Set Memory" "Memory (bytes)" \
            "${without_policy_dir}" "${with_policy_dir}" "${container}" \
            "${output_dir}/${container}_mem_workingset.png"
    fi

    # CPU usage
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${container}_cpu.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${container}_cpu.txt" "With Policy" \
        "${container} CPU Usage" "CPU Cores" \
        "${without_policy_dir}" "${with_policy_dir}" "${container}" \
        "${output_dir}/${container}_cpu_usage.png"
done

# Live Go heap (go_memstats_heap_inuse_bytes), the clearest "real, actively used"
# memory number. Only the Go services export it (central, sensor); skip when the
# series is absent (older runs / non-Go components).
for container in central sensor; do
    if [[ -f "${without_policy_dir}/metrics_${container}_heap_inuse.txt" || \
          -f "${with_policy_dir}/metrics_${container}_heap_inuse.txt" ]]; then
        echo "Plotting ${container} Go heap..."
        $python_bin plot-file-activity.py \
            "${without_policy_dir}/metrics_${container}_heap_inuse.txt" "Without Policy" \
            "${with_policy_dir}/metrics_${container}_heap_inuse.txt" "With Policy" \
            "${container} Go Heap In-Use" "Memory (bytes)" \
            "${without_policy_dir}" "${with_policy_dir}" "${container}" \
            "${output_dir}/${container}_heap_inuse.png"
    fi
done

# Restart and OOM counts per component (cumulative counters; the final value is
# the total). berserker is included so an OOM-killed workload -- which would drag
# down the observed event rate -- is visible. Series only exist for runs scraped
# after these were added, and only for components the metric source covers; skip
# otherwise.
for container in central central-db sensor collector fact berserker; do
    for kind in restarts:Restarts ooms:OOM-Kills; do
        metric="${kind%%:*}"
        label="${kind##*:}"
        if [[ -f "${without_policy_dir}/metrics_${container}_${metric}.txt" || \
              -f "${with_policy_dir}/metrics_${container}_${metric}.txt" ]]; then
            echo "Plotting ${container} ${label}..."
            $python_bin plot-file-activity.py \
                "${without_policy_dir}/metrics_${container}_${metric}.txt" "Without Policy" \
                "${with_policy_dir}/metrics_${container}_${metric}.txt" "With Policy" \
                "${container} ${label} (cumulative)" "Count" \
                "${without_policy_dir}" "${with_policy_dir}" "${container}" \
                "${output_dir}/${container}_${metric}.png"
        fi
    done
done

# Database table sizes (only alerts and deployments are relevant for file activity)
# Tables use central-db baseline since they're part of the database
for table in alerts deployments; do
    echo "Plotting ${table} table metrics..."

    # Row count
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${table}.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${table}.txt" "With Policy" \
        "${table} Table Row Count" "Number of Rows" \
        "${without_policy_dir}" "${with_policy_dir}" "central-db" \
        "${output_dir}/${table}_row_count.png"

    # Size in bytes
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${table}_bytes.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${table}_bytes.txt" "With Policy" \
        "${table} Table Size" "Size (bytes)" \
        "${without_policy_dir}" "${with_policy_dir}" "central-db" \
        "${output_dir}/${table}_size_bytes.png"
done

echo "All plots generated in ${output_dir}"
echo ""
echo "Generated plots:"
ls -lh "${output_dir}"/*.png
