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

# Determine baseline times (if they exist)
without_policy_baseline_time=""
with_policy_baseline_time=""

if [ -f "${without_policy_dir}/baseline_time.txt" ]; then
    without_policy_baseline_time="$(cat "${without_policy_dir}/baseline_time.txt")"
fi

if [ -f "${with_policy_dir}/baseline_time.txt" ]; then
    with_policy_baseline_time="$(cat "${with_policy_dir}/baseline_time.txt")"
fi

echo "Generating plots..."
echo "  Without policy dir: ${without_policy_dir}"
echo "  With policy dir: ${with_policy_dir}"
echo "  Output dir: ${output_dir}"

# CPU and Memory plots for each component
for container in central central-db sensor; do
    echo "Plotting ${container} metrics..."

    # Memory usage
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${container}_mem.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${container}_mem.txt" "With Policy" \
        "${container} Memory Usage" "Memory (bytes)" \
        "$without_policy_baseline_time" "$with_policy_baseline_time" \
        "${output_dir}/${container}_mem_usage.png"

    # CPU usage
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${container}_cpu.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${container}_cpu.txt" "With Policy" \
        "${container} CPU Usage" "CPU Cores" \
        "$without_policy_baseline_time" "$with_policy_baseline_time" \
        "${output_dir}/${container}_cpu_usage.png"
done

# Database table sizes (only alerts and deployments are relevant for file activity)
for table in alerts deployments; do
    echo "Plotting ${table} table metrics..."

    # Row count
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${table}.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${table}.txt" "With Policy" \
        "${table} Table Row Count" "Number of Rows" \
        "$without_policy_baseline_time" "$with_policy_baseline_time" \
        "${output_dir}/${table}_row_count.png"

    # Size in bytes
    $python_bin plot-file-activity.py \
        "${without_policy_dir}/metrics_${table}_bytes.txt" "Without Policy" \
        "${with_policy_dir}/metrics_${table}_bytes.txt" "With Policy" \
        "${table} Table Size" "Size (bytes)" \
        "$without_policy_baseline_time" "$with_policy_baseline_time" \
        "${output_dir}/${table}_size_bytes.png"
done

echo "All plots generated in ${output_dir}"
echo ""
echo "Generated plots:"
ls -lh "${output_dir}"/*.png
