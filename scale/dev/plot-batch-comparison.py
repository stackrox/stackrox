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

def read_metric_max(file_path):
    """Read a metric file and return the maximum value."""
    if not os.path.exists(file_path):
        return None

    with open(file_path, 'r') as f:
        lines = f.readlines()

    values = []
    for line in lines:
        parts = line.strip().split()
        if len(parts) != 2:
            continue
        try:
            val = float(parts[1])
            values.append(val)
        except ValueError:
            continue

    if not values:
        return None

    return max(values)

def plot_scaling_comparison(base_dir, output_dir):
    """
    Generate comparison plots across all batch sizes.

    Args:
        base_dir: Base directory containing test results
        output_dir: Output directory for plots
    """
    os.makedirs(output_dir, exist_ok=True)

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
        'alerts_count_without': [],
        'alerts_count_with': [],
        'alerts_size_without': [],
        'alerts_size_with': [],
    }

    for batch_size in batch_sizes:
        # Without policy
        if 'without' in data[batch_size]:
            without_dir = os.path.join(data[batch_size]['without'], 'metrics')
            metrics['central_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_cpu.txt')))
            metrics['central_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_mem.txt')))
            metrics['centraldb_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_cpu.txt')))
            metrics['alerts_count_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts.txt')))
            metrics['alerts_size_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts_bytes.txt')))
        else:
            for key in ['central_cpu_without', 'central_mem_without', 'centraldb_cpu_without',
                       'alerts_count_without', 'alerts_size_without']:
                metrics[key].append(None)

        # With policy
        if 'with' in data[batch_size]:
            with_dir = os.path.join(data[batch_size]['with'], 'metrics')
            metrics['central_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_cpu.txt')))
            metrics['central_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_mem.txt')))
            metrics['centraldb_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_cpu.txt')))
            metrics['alerts_count_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts.txt')))
            metrics['alerts_size_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts_bytes.txt')))
        else:
            for key in ['central_cpu_with', 'central_mem_with', 'centraldb_cpu_with',
                       'alerts_count_with', 'alerts_size_with']:
                metrics[key].append(None)

    # Plot 1: Central CPU vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['central_cpu_without'], 'o-', label='Without Policy', linewidth=2, markersize=8)
    plt.plot(event_rates, metrics['central_cpu_with'], 's-', label='With Policy', linewidth=2, markersize=8)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average CPU Usage (cores)', fontsize=12)
    plt.title('Central CPU Usage vs File Activity Event Rate', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'central_cpu_vs_rate.png'), dpi=150)
    print(f"Saved: central_cpu_vs_rate.png")
    plt.close()

    # Plot 2: Central Memory vs Event Rate
    plt.figure(figsize=(12, 7))
    mem_without_gb = [m / (1024**3) if m else None for m in metrics['central_mem_without']]
    mem_with_gb = [m / (1024**3) if m else None for m in metrics['central_mem_with']]
    plt.plot(event_rates, mem_without_gb, 'o-', label='Without Policy', linewidth=2, markersize=8)
    plt.plot(event_rates, mem_with_gb, 's-', label='With Policy', linewidth=2, markersize=8)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average Memory Usage (GB)', fontsize=12)
    plt.title('Central Memory Usage vs File Activity Event Rate', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'central_mem_vs_rate.png'), dpi=150)
    print(f"Saved: central_mem_vs_rate.png")
    plt.close()

    # Plot 3: Central-DB CPU vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['centraldb_cpu_without'], 'o-', label='Without Policy', linewidth=2, markersize=8)
    plt.plot(event_rates, metrics['centraldb_cpu_with'], 's-', label='With Policy', linewidth=2, markersize=8)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Average CPU Usage (cores)', fontsize=12)
    plt.title('Central-DB CPU Usage vs File Activity Event Rate', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'centraldb_cpu_vs_rate.png'), dpi=150)
    print(f"Saved: centraldb_cpu_vs_rate.png")
    plt.close()

    # Plot 4: Alert Count vs Event Rate
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['alerts_count_without'], 'o-', label='Without Policy', linewidth=2, markersize=8)
    plt.plot(event_rates, metrics['alerts_count_with'], 's-', label='With Policy', linewidth=2, markersize=8)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Maximum Alert Count', fontsize=12)
    plt.title('Alerts Table Row Count vs File Activity Event Rate', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'alerts_count_vs_rate.png'), dpi=150)
    print(f"Saved: alerts_count_vs_rate.png")
    plt.close()

    # Plot 5: Alert Table Size vs Event Rate
    plt.figure(figsize=(12, 7))
    size_without_mb = [s / (1024**2) if s else None for s in metrics['alerts_size_without']]
    size_with_mb = [s / (1024**2) if s else None for s in metrics['alerts_size_with']]
    plt.plot(event_rates, size_without_mb, 'o-', label='Without Policy', linewidth=2, markersize=8)
    plt.plot(event_rates, size_with_mb, 's-', label='With Policy', linewidth=2, markersize=8)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('Maximum Table Size (MB)', fontsize=12)
    plt.title('Alerts Table Size vs File Activity Event Rate', fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'alerts_size_vs_rate.png'), dpi=150)
    print(f"Saved: alerts_size_vs_rate.png")
    plt.close()

    # Plot 6: Policy Overhead (CPU)
    plt.figure(figsize=(12, 7))
    cpu_overhead = []
    for without, with_pol in zip(metrics['central_cpu_without'], metrics['central_cpu_with']):
        if without and with_pol and without > 0:
            overhead = ((with_pol - without) / without) * 100
            cpu_overhead.append(overhead)
        else:
            cpu_overhead.append(None)

    plt.plot(event_rates, cpu_overhead, 'o-', linewidth=2, markersize=8, color='red')
    plt.axhline(y=0, color='gray', linestyle='--', alpha=0.5)
    plt.xlabel('File Activity Event Rate (events/sec)', fontsize=12)
    plt.ylabel('CPU Overhead (%)', fontsize=12)
    plt.title('Policy Enforcement CPU Overhead', fontsize=14, fontweight='bold')
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, 'policy_cpu_overhead.png'), dpi=150)
    print(f"Saved: policy_cpu_overhead.png")
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
