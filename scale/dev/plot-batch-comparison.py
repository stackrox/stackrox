#!/usr/bin/env python3
"""
Create comparison plots showing how metrics vary across different batch sizes.

This script generates plots that show:
- How CPU/memory usage scales with event rate
- Database growth rate vs event rate
- Performance impact of policy enforcement across different rates
"""

import matplotlib.pyplot as plt
import sys
import os
import glob
import re
from plot_utils import (
    determine_baseline_timestamp,
    add_trendline,
    add_equation_text,
    read_metric_series,
)

def extract_batch_size(dirname):
    """
    Extract batch size from directory name.

    Supports two formats:
    - batch-10 (fake workload): batch size 10 → 100 events/sec
    - file-activity-100 (berserker): event rate 100 → batch size 10 (for compatibility)
    """
    # Try fake workload pattern first: batch-X
    match = re.search(r'batch-(\d+)', dirname)
    if match:
        return int(match.group(1))

    # Try berserker pattern: file-activity-X (event rate)
    # Convert event rate to "batch size" for compatibility: rate / 10
    match = re.search(r'file-activity-(\d+)', dirname)
    if match:
        event_rate = int(match.group(1))
        return event_rate // 10

    return None

def read_metric_average(file_path, start_offset=60.0, end_offset=None, base_time=None):
    """
    Read a metric file and return the average value over a time window.

    Args:
        file_path: Path to metrics file
        start_offset: Start time in seconds (to skip initial ramp-up)
        end_offset: End time in seconds (None = until end)
        base_time: Optional baseline timestamp in milliseconds (if None, uses first timestamp)

    Returns:
        Average value over the time window, or None if file doesn't exist
    """
    # Read sorted, de-duplicated series (drops duplicate-timestamp artifacts)
    timestamps, values = read_metric_series(file_path)

    if not timestamps:
        return None

    # Convert to relative time in seconds
    if base_time is None:
        base_time = timestamps[0]
    rel_times = [(t - base_time) / 1000.0 for t in timestamps]

    # Filter to time window
    filtered_values = []
    for t, v in zip(rel_times, values):
        if t >= start_offset:
            if end_offset is None or t <= end_offset:
                filtered_values.append(v)

    if not filtered_values:
        return None

    return sum(filtered_values) / len(filtered_values)

def read_metric_max(file_path, start_offset=60.0, end_offset=None, base_time=None):
    """
    Read a metric file and return the maximum value over a time window.

    Args:
        file_path: Path to metrics file
        start_offset: Start time in seconds (to skip initial ramp-up)
        end_offset: End time in seconds (None = until end)
        base_time: Optional baseline timestamp in milliseconds (if None, uses first timestamp)

    Returns:
        Maximum value over the time window, or None if file doesn't exist
    """
    # Read sorted, de-duplicated series (drops duplicate-timestamp artifacts)
    timestamps, values = read_metric_series(file_path)

    if not timestamps:
        return None

    # Convert to relative time in seconds
    if base_time is None:
        base_time = timestamps[0]
    rel_times = [(t - base_time) / 1000.0 for t in timestamps]

    # Filter to time window
    filtered_values = []
    for t, v in zip(rel_times, values):
        if t >= start_offset:
            if end_offset is None or t <= end_offset:
                filtered_values.append(v)

    if not filtered_values:
        return None

    return max(filtered_values)

def plot_scaling_comparison(base_dir, output_dir):
    """
    Generate comparison plots across all batch sizes.

    Args:
        base_dir: Base directory containing test results
        output_dir: Output directory for plots
    """
    os.makedirs(output_dir, exist_ok=True)

    # Use consistent time window for all metrics to ensure fair comparison
    # Tests may run for different durations, so we use the same window across all
    START_OFFSET = 60.0   # Skip initial 60s ramp-up
    END_OFFSET = 660.0    # Measure from 60s to 660s (10 minutes of stable data)
    TIME_WINDOW_DESC = "60-660s"

    # Find all result directories (both fake workload and berserker)
    pattern_fake = os.path.join(base_dir, "file_activity_results_*_policy_*")
    pattern_berserker = os.path.join(base_dir, "berserker_file_activity_results_*_policy_*")
    result_dirs = glob.glob(pattern_fake) + glob.glob(pattern_berserker)

    # Organize by batch size and policy
    data = {}  # {batch_size: {'without': dir, 'with': dir}}

    for dir_path in result_dirs:
        batch_size = extract_batch_size(dir_path)
        if batch_size is None:
            continue

        if batch_size not in data:
            data[batch_size] = {}

        if 'policy_false' in dir_path:
            data[batch_size]['without'] = dir_path
        elif 'policy_true' in dir_path:
            data[batch_size]['with'] = dir_path

    # Sort by batch size
    batch_sizes = sorted(data.keys())
    event_rates = [b * 10 for b in batch_sizes]  # batch_size * 10 events/sec

    if not batch_sizes:
        print(f"No test results found in {base_dir}")
        return

    print(f"Found results for batch sizes: {batch_sizes}")
    print(f"Event rates: {event_rates} events/sec")

    # Collect metrics
    metrics = {
        'central_cpu_without': [],
        'central_cpu_with': [],
        'central_mem_without': [],
        'central_mem_with': [],
        'centraldb_cpu_without': [],
        'centraldb_cpu_with': [],
        'centraldb_mem_without': [],
        'centraldb_mem_with': [],
        'sensor_cpu_without': [],
        'sensor_cpu_with': [],
        'sensor_mem_without': [],
        'sensor_mem_with': [],
        'alerts_count_without': [],
        'alerts_count_with': [],
        'alerts_size_without': [],
        'alerts_size_with': [],
    }

    for batch_size in batch_sizes:
        # Without policy
        if 'without' in data[batch_size]:
            without_dir = data[batch_size]['without']
            # Determine component-specific baselines
            central_baseline_without = determine_baseline_timestamp(without_dir, 'central')
            centraldb_baseline_without = determine_baseline_timestamp(without_dir, 'central-db')
            sensor_baseline_without = determine_baseline_timestamp(without_dir, 'sensor')

            metrics['central_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_cpu.txt'), START_OFFSET, END_OFFSET, central_baseline_without))
            metrics['central_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_mem.txt'), START_OFFSET, END_OFFSET, central_baseline_without))
            metrics['centraldb_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_cpu.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['centraldb_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_mem.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['sensor_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_cpu.txt'), START_OFFSET, END_OFFSET, sensor_baseline_without))
            metrics['sensor_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_mem.txt'), START_OFFSET, END_OFFSET, sensor_baseline_without))
            metrics['alerts_count_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['alerts_size_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts_bytes.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
        else:
            for key in ['central_cpu_without', 'central_mem_without', 'centraldb_cpu_without',
                       'centraldb_mem_without', 'sensor_cpu_without', 'sensor_mem_without',
                       'alerts_count_without', 'alerts_size_without']:
                metrics[key].append(None)

        # With policy
        if 'with' in data[batch_size]:
            with_dir = data[batch_size]['with']
            # Determine component-specific baselines
            central_baseline_with = determine_baseline_timestamp(with_dir, 'central')
            centraldb_baseline_with = determine_baseline_timestamp(with_dir, 'central-db')
            sensor_baseline_with = determine_baseline_timestamp(with_dir, 'sensor')

            metrics['central_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_cpu.txt'), START_OFFSET, END_OFFSET, central_baseline_with))
            metrics['central_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_mem.txt'), START_OFFSET, END_OFFSET, central_baseline_with))
            metrics['centraldb_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_cpu.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['centraldb_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_mem.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['sensor_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_cpu.txt'), START_OFFSET, END_OFFSET, sensor_baseline_with))
            metrics['sensor_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_mem.txt'), START_OFFSET, END_OFFSET, sensor_baseline_with))
            metrics['alerts_count_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['alerts_size_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts_bytes.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
        else:
            for key in ['central_cpu_with', 'central_mem_with', 'centraldb_cpu_with',
                       'centraldb_mem_with', 'sensor_cpu_with', 'sensor_mem_with',
                       'alerts_count_with', 'alerts_size_with']:
                metrics[key].append(None)

    # Plot 1: Central CPU vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['central_cpu_without'], 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, metrics['central_cpu_with'], 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, metrics['central_cpu_without'], 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, metrics['central_cpu_with'], 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average CPU Usage (cores)', fontsize=12)
    plt.title(f'Central CPU Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'central_cpu_vs_rate.png'), dpi=150)
    print(f"Saved: central_cpu_vs_rate.png")
    plt.close()

    # Plot 2: Central Memory vs Event Rate
    plt.figure(figsize=(12, 7))
    mem_without_gb = [m / (1024**3) if m else None for m in metrics['central_mem_without']]
    mem_with_gb = [m / (1024**3) if m else None for m in metrics['central_mem_with']]
    plt.plot(event_rates, mem_without_gb, 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, mem_with_gb, 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, mem_without_gb, 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, mem_with_gb, 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average Memory Usage (GB)', fontsize=12)
    plt.title(f'Central Memory Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'central_mem_vs_rate.png'), dpi=150)
    print(f"Saved: central_mem_vs_rate.png")
    plt.close()

    # Plot 3: Central-DB CPU vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['centraldb_cpu_without'], 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, metrics['centraldb_cpu_with'], 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, metrics['centraldb_cpu_without'], 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, metrics['centraldb_cpu_with'], 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average CPU Usage (cores)', fontsize=12)
    plt.title(f'Central-DB CPU Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'centraldb_cpu_vs_rate.png'), dpi=150)
    print(f"Saved: centraldb_cpu_vs_rate.png")
    plt.close()

    # Plot 4: Alert Count vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['alerts_count_without'], 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, metrics['alerts_count_with'], 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, metrics['alerts_count_without'], 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, metrics['alerts_count_with'], 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Maximum Alert Count', fontsize=12)
    plt.title(f'Alerts Table Row Count vs File Activity Event Rate\n(maximum within {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'alerts_count_vs_rate.png'), dpi=150)
    print(f"Saved: alerts_count_vs_rate.png")
    plt.close()

    # Plot 5: Alert Table Size vs Event Rate
    plt.figure(figsize=(12, 7))
    size_without_mb = [s / (1024**2) if s else None for s in metrics['alerts_size_without']]
    size_with_mb = [s / (1024**2) if s else None for s in metrics['alerts_size_with']]
    plt.plot(event_rates, size_without_mb, 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, size_with_mb, 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, size_without_mb, 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, size_with_mb, 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Maximum Table Size (MB)', fontsize=12)
    plt.title(f'Alerts Table Size vs File Activity Event Rate\n(maximum within {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'alerts_size_vs_rate.png'), dpi=150)
    print(f"Saved: alerts_size_vs_rate.png")
    plt.close()

    # Plot 6: Policy Enforcement CPU Overhead
    plt.figure(figsize=(12, 7))
    cpu_overhead = []
    for without, with_pol in zip(metrics['central_cpu_without'], metrics['central_cpu_with']):
        if without and with_pol and without > 0:
            overhead = ((with_pol - without) / without) * 100
            cpu_overhead.append(overhead)
        else:
            cpu_overhead.append(None)

    plt.plot(event_rates, cpu_overhead, 'o-', linewidth=2, markersize=8, color='red', label='CPU Overhead')
    eq1 = add_trendline(event_rates, cpu_overhead, 'Trend', 'red')
    plt.axhline(y=0, color='gray', linestyle='--', alpha=0.5)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('CPU Overhead (%)', fontsize=12)
    plt.title(f'Policy Enforcement CPU Overhead\n(Central CPU increase when policy enabled, {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Overhead', eq1[0], eq1[1], 'red'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'policy_cpu_overhead.png'), dpi=150)
    print(f"Saved: policy_cpu_overhead.png (Central CPU increase when policy is enabled)")
    plt.close()

    # Plot 7: Central-DB Memory vs Event Rate
    plt.figure(figsize=(12, 7))
    centraldb_mem_without_gb = [m / (1024**3) if m else None for m in metrics['centraldb_mem_without']]
    centraldb_mem_with_gb = [m / (1024**3) if m else None for m in metrics['centraldb_mem_with']]
    plt.plot(event_rates, centraldb_mem_without_gb, 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, centraldb_mem_with_gb, 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, centraldb_mem_without_gb, 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, centraldb_mem_with_gb, 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average Memory Usage (GB)', fontsize=12)
    plt.title(f'Central-DB Memory Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'centraldb_mem_vs_rate.png'), dpi=150)
    print(f"Saved: centraldb_mem_vs_rate.png")
    plt.close()

    # Plot 8: Sensor CPU vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['sensor_cpu_without'], 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, metrics['sensor_cpu_with'], 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, metrics['sensor_cpu_without'], 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, metrics['sensor_cpu_with'], 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average CPU Usage (cores)', fontsize=12)
    plt.title(f'Sensor CPU Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'sensor_cpu_vs_rate.png'), dpi=150)
    print(f"Saved: sensor_cpu_vs_rate.png")
    plt.close()

    # Plot 9: Sensor Memory vs Event Rate
    plt.figure(figsize=(12, 7))
    sensor_mem_without_gb = [m / (1024**3) if m else None for m in metrics['sensor_mem_without']]
    sensor_mem_with_gb = [m / (1024**3) if m else None for m in metrics['sensor_mem_with']]
    plt.plot(event_rates, sensor_mem_without_gb, 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, sensor_mem_with_gb, 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, sensor_mem_without_gb, 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, sensor_mem_with_gb, 'Trend (With Policy)', 'C1')
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average Memory Usage (GB)', fontsize=12)
    plt.title(f'Sensor Memory Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'sensor_mem_vs_rate.png'), dpi=150)
    print(f"Saved: sensor_mem_vs_rate.png")
    plt.close()

    print(f"\nAll comparison plots saved to {output_dir}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python plot-batch-comparison.py <base_results_dir> [output_dir]")
        print("\nExample:")
        print("  python plot-batch-comparison.py perf perf/comparison_plots")
        sys.exit(1)

    base_dir = sys.argv[1]
    output_dir = sys.argv[2] if len(sys.argv) > 2 else os.path.join(base_dir, "comparison_plots")

    plot_scaling_comparison(base_dir, output_dir)
