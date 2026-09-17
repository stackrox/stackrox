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

    # Memory usage
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${container}_mem.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${container}_mem.txt" "With Policy" \
        "${container} Memory Usage" "Memory (bytes)" \
        "${without_policy_dir}" "${with_policy_dir}" "${container}" \
        "${output_dir}/${container}_mem_usage.png"

    # CPU usage
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${container}_cpu.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${container}_cpu.txt" "With Policy" \
        "${container} CPU Usage" "CPU Cores" \
        "${without_policy_dir}" "${with_policy_dir}" "${container}" \
        "${output_dir}/${container}_cpu_usage.png"
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
