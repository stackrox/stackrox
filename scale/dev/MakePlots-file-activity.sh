#!/usr/bin/env bash
set -eoux pipefail

# Generate per-run comparison plots for file activity performance tests, overlaying
# one line per test mode on each metric plot. Works for any number of modes:
#   fake workload:      Without Policy / With Policy               (2 series)
#   berserker workload: Baseline / Alerting policy / Idle policy   (3 series)
#
# Usage:
#   MakePlots-file-activity.sh <output_dir> <dir1> <label1> [<dir2> <label2> ...]
#
# Each (dir, label) pair is one test-mode run directory and its legend label.

output_dir=$1
shift

# Collect the (dir, label) pairs into parallel arrays.
DIRS=()
LABELS=()
while [[ $# -gt 0 ]]; do
    DIRS+=("$1")
    LABELS+=("$2")
    shift 2
done

if [[ ${#DIRS[@]} -eq 0 ]]; then
    echo "Error: at least one <dir> <label> pair is required"
    exit 1
fi

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

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Create output directory
mkdir -p "$output_dir"

echo "Generating plots with component-specific baselines..."
for i in "${!DIRS[@]}"; do
    echo "  ${LABELS[$i]}: ${DIRS[$i]}"
done
echo "  Output dir: ${output_dir}"
echo ""
echo "Baseline logic:"
echo "  - sensor: t=0 = first sensor data point"
echo "  - central/central-db: t=0 = when deployments >= 100 (or 90% of max)"
echo "  - berserker runs: t=0 = recorded berserker-ready time (all components)"
echo ""

# Plot one metric across all modes. Builds a --series triple (file, label, dir) for
# every mode and only invokes the plotter when at least one mode has the metric file
# (series added later, older runs, or metric not scraped in a cluster otherwise
# yield an empty plot).
#   $1 output png   $2 title   $3 ylabel   $4 component (may be empty)   $5 file suffix
plot_metric() {
    local out=$1 title=$2 ylabel=$3 component=$4 suffix=$5
    local series_args=() any=0 i f
    for i in "${!DIRS[@]}"; do
        f="${DIRS[$i]}/metrics_${suffix}.txt"
        series_args+=(--series "$f" "${LABELS[$i]}" "${DIRS[$i]}")
        [[ -f "$f" ]] && any=1
    done
    if [[ "$any" -eq 1 ]]; then
        echo "Plotting ${title}..."
        local comp_args=()
        [[ -n "$component" ]] && comp_args=(--component "$component")
        # Invoke by absolute path so plot_utils imports (script dir is added to
        # sys.path) while relative metric-file paths stay relative to the caller CWD.
        "$python_bin" "${DIR}/plot-file-activity.py" \
            --title "$title" --ylabel "$ylabel" \
            "${comp_args[@]}" --output "$out" "${series_args[@]}"
    fi
}

# CPU and Memory plots for each component. collector and fact are containers in
# the collector DaemonSet pod (fact monitors file activity); their metric files
# only exist when collector metrics were collected, otherwise the plot is skipped.
for container in central central-db sensor collector fact; do
    # Memory usage (container_memory_usage_bytes: cgroup total, incl. reclaimable)
    plot_metric "${output_dir}/${container}_mem_usage.png" \
        "${container} Memory Usage" "Memory (bytes)" "${container}" "${container}_mem"

    # Working set memory (usage minus reclaimable cache; the OOM-relevant number).
    plot_metric "${output_dir}/${container}_mem_workingset.png" \
        "${container} Working Set Memory" "Memory (bytes)" "${container}" "${container}_mem_workingset"

    # CPU usage
    plot_metric "${output_dir}/${container}_cpu_usage.png" \
        "${container} CPU Usage" "CPU Cores" "${container}" "${container}_cpu"
done

# Live Go heap (go_memstats_heap_inuse_bytes), the clearest "real, actively used"
# memory number. Only the Go services export it (central, sensor).
for container in central sensor; do
    plot_metric "${output_dir}/${container}_heap_inuse.png" \
        "${container} Go Heap In-Use" "Memory (bytes)" "${container}" "${container}_heap_inuse"
done

# Restart and OOM counts per component (cumulative counters; the final value is
# the total). berserker is included so an OOM-killed workload -- which would drag
# down the observed event rate -- is visible.
for container in central central-db sensor collector fact berserker; do
    plot_metric "${output_dir}/${container}_restarts.png" \
        "${container} Restarts (cumulative)" "Count" "${container}" "${container}_restarts"
    plot_metric "${output_dir}/${container}_ooms.png" \
        "${container} OOM-Kills (cumulative)" "Count" "${container}" "${container}_ooms"
done

# Event drops along the pipeline (cumulative counters; the curve shows when drops
# start and the final value is the total). fact stages come from the fact agent's
# own /metrics (stackrox_fact_*), sensor stages from the sensor detector/buffer.
#   "<file-suffix>:<component>:<title>"
for drop in \
    "fact_bpf_dropped:fact:fact BPF Events Dropped" \
    "fact_rate_limiter_dropped:fact:fact Rate-Limiter Events Dropped" \
    "fact_grpc_dropped:fact:fact gRPC Events Dropped" \
    "rox_sensor_file_activity_buffer_drops:sensor:sensor File-Activity Buffer Drops" \
    "rox_sensor_detector_file_access_queue_dropped_total:sensor:sensor Detector File-Access Drops"; do
    suffix="${drop%%:*}"
    rest="${drop#*:}"
    comp="${rest%%:*}"
    title="${rest#*:}"
    plot_metric "${output_dir}/${suffix}.png" \
        "${title} (cumulative)" "Events" "${comp}" "${suffix}"
done

# Process and network event volume received by Sensor (cumulative counters). These
# gauge the non-file-activity load Sensor carries in each mode; compare across modes
# and against the file-activity rate.
plot_metric "${output_dir}/network_flows_received.png" \
    "Sensor Network Flows Received (cumulative)" "Network Flows" "sensor" "network_flows_received"
plot_metric "${output_dir}/process_indicators_received.png" \
    "Sensor Process Indicators Received (cumulative)" "Process Indicators" "sensor" "process_indicators_received"

# Database table sizes. alerts and deployments are the file-activity-relevant
# tables; process_indicators is the storage-side view of process-event volume.
# Tables use the central-db baseline since they're part of the database.
for table in alerts deployments process_indicators; do
    plot_metric "${output_dir}/${table}_row_count.png" \
        "${table} Table Row Count" "Number of Rows" "central-db" "${table}"
    plot_metric "${output_dir}/${table}_size_bytes.png" \
        "${table} Table Size" "Size (bytes)" "central-db" "${table}_bytes"
done

echo "All plots generated in ${output_dir}"
echo ""
echo "Generated plots:"
ls -lh "${output_dir}"/*.png
