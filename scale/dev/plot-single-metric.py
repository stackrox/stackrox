#!/usr/bin/env python3
"""
Plot a single metric file as a time-series.
X-axis: Time (seconds from start)
Y-axis: Metric value
"""

import sys
import os
import matplotlib.pyplot as plt
from plot_utils import determine_baseline_timestamp, read_metric_series

def read_metric_file(file_path, base_time=None):
    """
    Read metric file and return time and value arrays.

    Args:
        file_path: Path to metric file
        base_time: Optional baseline timestamp in milliseconds (if None, uses first timestamp)
    """
    if not os.path.exists(file_path):
        return [], []

    # Read sorted, de-duplicated series (drops duplicate-timestamp artifacts)
    timestamps, values = read_metric_series(file_path)

    if not timestamps:
        return [], []

    # Convert to relative time in seconds
    if base_time is None:
        base_time = timestamps[0]
    rel_time = [(t - base_time) / 1000.0 for t in timestamps]

    return rel_time, values

def plot_metric(metric_file, title, ylabel, output_file, results_dir=None, component=None):
    """
    Create a time-series plot for a single metric.

    Args:
        metric_file: Path to metric file
        title: Plot title
        ylabel: Y-axis label
        output_file: Output file path
        results_dir: Optional directory containing metrics (for component-specific baseline)
        component: Optional component name (e.g., 'sensor', 'central', 'central-db')
    """
    base_time = None
    if results_dir and component:
        base_time = determine_baseline_timestamp(results_dir, component)
        if base_time:
            print(f"  Using component-specific baseline for {component}: t=0 at {base_time}ms")

    times, values = read_metric_file(metric_file, base_time)

    if not times:
        print(f"Warning: No data in {metric_file}, skipping plot")
        return

    plt.figure(figsize=(14, 7))
    plt.plot(times, values, linewidth=2, color='#1f77b4', marker='o', markersize=4)
    plt.xlabel('Time (seconds from start)', fontsize=12)
    plt.ylabel(ylabel, fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(output_file, dpi=150)
    plt.close()
    print(f"  Saved: {output_file}")

if __name__ == "__main__":
    if len(sys.argv) < 5:
        print("Usage: plot-single-metric.py <metric_file> <title> <ylabel> <output_file> [results_dir] [component]")
        print("\nExample (with component-specific baseline):")
        print("  plot-single-metric.py results/metrics_central_cpu.txt 'Central CPU' 'CPU Cores' output.png results central")
        sys.exit(1)

    metric_file = sys.argv[1]
    title = sys.argv[2]
    ylabel = sys.argv[3]
    output_file = sys.argv[4]
    results_dir = sys.argv[5] if len(sys.argv) > 5 else None
    component = sys.argv[6] if len(sys.argv) > 6 else None

    plot_metric(metric_file, title, ylabel, output_file, results_dir, component)
