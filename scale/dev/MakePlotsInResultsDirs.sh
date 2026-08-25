#!/usr/bin/env bash
set -eoux pipefail

# Generate time-series plots directly in each results directory
# This creates plots where x-axis = time and y-axis = metric value
# Each batch size gets its own directory with plots

base_results_dir=${1:-perf}
num_sensors=${2:-1}
run_time=${3:-10m}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Find python
python_bin=""
if command -v python3 &> /dev/null; then
    python_bin=python3
elif command -v python &> /dev/null; then
    python_bin=python
else
    echo "Python not found"
    exit 1
fi

batch_sizes=(10 50 100 250 500)

for batch in "${batch_sizes[@]}"; do
    workload="file-activity-batch-${batch}"
    event_rate=$((batch * 10))
    
    without_policy_dir="${base_results_dir}/file_activity_results_${num_sensors}_${run_time}_${workload}_policy_false"
    with_policy_dir="${base_results_dir}/file_activity_results_${num_sensors}_${run_time}_${workload}_policy_true"
    
    if [[ ! -d "$without_policy_dir" ]]; then
        echo "Skipping batch ${batch}: without_policy dir not found"
        continue
    fi
    
    if [[ ! -d "$with_policy_dir" ]]; then
        echo "Skipping batch ${batch}: with_policy dir not found"
        continue
    fi
    
    echo ""
    echo "=========================================="
    echo "Processing batch size: ${batch}"
    echo "Event rate: ${event_rate} events/sec"
    echo "=========================================="
    
    # Generate plots in WITHOUT policy directory
    echo "Generating plots in ${without_policy_dir}..."
    
    for container in central central-db sensor; do
        echo "  Plotting ${container} metrics..."
        
        # CPU plot
        $python_bin "${DIR}/plot-single-metric.py" \
            "${without_policy_dir}/metrics_${container}_cpu.txt" \
            "${container} CPU Usage (batch-${batch}, ${event_rate} events/sec)" \
            "CPU Cores" \
            "${without_policy_dir}/${container}_cpu_usage.png"
        
        # Memory plot
        $python_bin "${DIR}/plot-single-metric.py" \
            "${without_policy_dir}/metrics_${container}_mem.txt" \
            "${container} Memory Usage (batch-${batch}, ${event_rate} events/sec)" \
            "Memory (bytes)" \
            "${without_policy_dir}/${container}_mem_usage.png"
    done
    
    # Alerts plots (only if policy was false - shouldn't have many alerts)
    for table in alerts deployments; do
        echo "  Plotting ${table} table metrics..."
        
        $python_bin "${DIR}/plot-single-metric.py" \
            "${without_policy_dir}/metrics_${table}.txt" \
            "${table} Table Row Count (batch-${batch}, ${event_rate} events/sec)" \
            "Number of Rows" \
            "${without_policy_dir}/${table}_row_count.png"
        
        $python_bin "${DIR}/plot-single-metric.py" \
            "${without_policy_dir}/metrics_${table}_bytes.txt" \
            "${table} Table Size (batch-${batch}, ${event_rate} events/sec)" \
            "Size (bytes)" \
            "${without_policy_dir}/${table}_size_bytes.png"
    done
    
    echo "Plots saved to ${without_policy_dir}"
    
    # Generate plots in WITH policy directory
    echo "Generating plots in ${with_policy_dir}..."
    
    for container in central central-db sensor; do
        echo "  Plotting ${container} metrics..."
        
        # CPU plot
        $python_bin "${DIR}/plot-single-metric.py" \
            "${with_policy_dir}/metrics_${container}_cpu.txt" \
            "${container} CPU Usage (batch-${batch}, ${event_rate} events/sec, WITH POLICY)" \
            "CPU Cores" \
            "${with_policy_dir}/${container}_cpu_usage.png"
        
        # Memory plot
        $python_bin "${DIR}/plot-single-metric.py" \
            "${with_policy_dir}/metrics_${container}_mem.txt" \
            "${container} Memory Usage (batch-${batch}, ${event_rate} events/sec, WITH POLICY)" \
            "Memory (bytes)" \
            "${with_policy_dir}/${container}_mem_usage.png"
    done
    
    for table in alerts deployments; do
        echo "  Plotting ${table} table metrics..."
        
        $python_bin "${DIR}/plot-single-metric.py" \
            "${with_policy_dir}/metrics_${table}.txt" \
            "${table} Table Row Count (batch-${batch}, ${event_rate} events/sec, WITH POLICY)" \
            "Number of Rows" \
            "${with_policy_dir}/${table}_row_count.png"
        
        $python_bin "${DIR}/plot-single-metric.py" \
            "${with_policy_dir}/metrics_${table}_bytes.txt" \
            "${table} Table Size (batch-${batch}, ${event_rate} events/sec, WITH POLICY)" \
            "Size (bytes)" \
            "${with_policy_dir}/${table}_size_bytes.png"
    done
    
    echo "Plots saved to ${with_policy_dir}"
    echo ""
done

echo "=========================================="
echo "All time-series plots generated!"
echo "=========================================="
echo ""
echo "Summary:"
for batch in "${batch_sizes[@]}"; do
    workload="file-activity-batch-${batch}"
    without_policy_dir="${base_results_dir}/file_activity_results_${num_sensors}_${run_time}_${workload}_policy_false"
    with_policy_dir="${base_results_dir}/file_activity_results_${num_sensors}_${run_time}_${workload}_policy_true"
    
    if [[ -d "$without_policy_dir" ]]; then
        plot_count=$(find "$without_policy_dir" -name "*.png" | wc -l)
        echo "  ${without_policy_dir}: ${plot_count} plots"
    fi
    
    if [[ -d "$with_policy_dir" ]]; then
        plot_count=$(find "$with_policy_dir" -name "*.png" | wc -l)
        echo "  ${with_policy_dir}: ${plot_count} plots"
    fi
done
