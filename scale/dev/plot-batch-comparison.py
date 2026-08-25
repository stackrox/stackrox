#!/usr/bin/env python3
"""
Create comparison plots showing how metrics vary across different batch sizes.

This script generates plots that show:
- How CPU/memory usage scales with event rate
- Database growth rate vs event rate
- Performance impact of policy enforcement across different rates
"""

import matplotlib.pyplot as plt
import numpy as np
import sys
import os
import glob
import re

def extract_batch_size(dirname):
    """Extract batch size from directory name."""
    match = re.search(r'batch-(\d+)', dirname)
    if match:
        return int(match.group(1))
    return None

def read_metric_average(file_path, start_offset=60.0, end_offset=None):
    """
    Read a metric file and return the average value over a time window.

    Args:
        file_path: Path to metrics file
        start_offset: Start time in seconds (to skip initial ramp-up)
        end_offset: End time in seconds (None = until end)

    Returns:
        Average value over the time window, or None if file doesn't exist
    """
    if not os.path.exists(file_path):
        return None

    with open(file_path, 'r') as f:
        lines = f.readlines()

    timestamps = []
    values = []

    for line in lines:
        parts = line.strip().split()
        if len(parts) != 2:
            continue
        try:
            ts, val = int(parts[0]), float(parts[1])
            timestamps.append(ts)
            values.append(val)
        except ValueError:
            continue

    if not timestamps:
        return None

    # Convert to relative time in seconds
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

def read_metric_max(file_path, start_offset=60.0, end_offset=None):
    """
    Read a metric file and return the maximum value over a time window.

    Args:
        file_path: Path to metrics file
        start_offset: Start time in seconds (to skip initial ramp-up)
        end_offset: End time in seconds (None = until end)

    Returns:
        Maximum value over the time window, or None if file doesn't exist
    """
    if not os.path.exists(file_path):
        return None

    with open(file_path, 'r') as f:
        lines = f.readlines()

    timestamps = []
    values = []

    for line in lines:
        parts = line.strip().split()
        if len(parts) != 2:
            continue
        try:
            ts, val = int(parts[0]), float(parts[1])
            timestamps.append(ts)
            values.append(val)
        except ValueError:
            continue

    if not timestamps:
        return None

    # Convert to relative time in seconds
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

def add_trendline(x_data, y_data, label, color, linestyle='--'):
    """
    Add a linear trend line to the current plot and return the equation.

    Args:
        x_data: X-axis values (event rates)
        y_data: Y-axis values (metric values)
        label: Label for the trend line
        color: Color for the trend line
        linestyle: Line style for the trend line

    Returns:
        Tuple of (slope, intercept) or None if insufficient data
    """
    # Filter out None values
    valid_points = [(x, y) for x, y in zip(x_data, y_data) if y is not None]
    if len(valid_points) < 2:
        return None  # Need at least 2 points for a trend line

    x_valid = [p[0] for p in valid_points]
    y_valid = [p[1] for p in valid_points]

    # Fit linear trend line
    coeffs = np.polyfit(x_valid, y_valid, 1)
    trend_y = np.polyval(coeffs, x_valid)

    # Plot trend line
    plt.plot(x_valid, trend_y, linestyle=linestyle, linewidth=1.5, color=color,
             alpha=0.7, label=label)

    # Return slope and intercept
    return (coeffs[0], coeffs[1])

def add_equation_text(equations, y_position=0.95):
    """
    Add trend line equations as text on the plot.

    Args:
        equations: List of tuples (label, slope, intercept, color)
        y_position: Vertical position for the text box (0-1, in axes coordinates)
    """
    if not equations:
        return

    equation_text = []
    for label, slope, intercept, color in equations:
        if slope is not None and intercept is not None:
            # Format equation: y = mx + b
            sign = '+' if intercept >= 0 else '-'
            equation_text.append(f"{label}: y = {slope:.4e}x {sign} {abs(intercept):.4f}")

    if equation_text:
        text_str = '\n'.join(equation_text)
        plt.text(0.02, y_position, text_str, transform=plt.gca().transAxes,
                fontsize=9, verticalalignment='top',
                bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))

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

    # Find all result directories
    pattern = os.path.join(base_dir, "file_activity_results_*_policy_*")
    result_dirs = glob.glob(pattern)

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
            metrics['central_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_cpu.txt'), START_OFFSET, END_OFFSET))
            metrics['central_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_mem.txt'), START_OFFSET, END_OFFSET))
            metrics['centraldb_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_cpu.txt'), START_OFFSET, END_OFFSET))
            metrics['centraldb_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_mem.txt'), START_OFFSET, END_OFFSET))
            metrics['sensor_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_cpu.txt'), START_OFFSET, END_OFFSET))
            metrics['sensor_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_mem.txt'), START_OFFSET, END_OFFSET))
            metrics['alerts_count_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts.txt'), START_OFFSET, END_OFFSET))
            metrics['alerts_size_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts_bytes.txt'), START_OFFSET, END_OFFSET))
        else:
            for key in ['central_cpu_without', 'central_mem_without', 'centraldb_cpu_without',
                       'centraldb_mem_without', 'sensor_cpu_without', 'sensor_mem_without',
                       'alerts_count_without', 'alerts_size_without']:
                metrics[key].append(None)

        # With policy
        if 'with' in data[batch_size]:
            with_dir = data[batch_size]['with']
            metrics['central_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_cpu.txt'), START_OFFSET, END_OFFSET))
            metrics['central_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_mem.txt'), START_OFFSET, END_OFFSET))
            metrics['centraldb_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_cpu.txt'), START_OFFSET, END_OFFSET))
            metrics['centraldb_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_mem.txt'), START_OFFSET, END_OFFSET))
            metrics['sensor_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_cpu.txt'), START_OFFSET, END_OFFSET))
            metrics['sensor_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_mem.txt'), START_OFFSET, END_OFFSET))
            metrics['alerts_count_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts.txt'), START_OFFSET, END_OFFSET))
            metrics['alerts_size_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts_bytes.txt'), START_OFFSET, END_OFFSET))
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
