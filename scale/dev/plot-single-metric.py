#!/usr/bin/env python3
"""
Plot a single metric file as a time-series.
X-axis: Time (seconds from start)
Y-axis: Metric value
"""

import sys
import os
import matplotlib.pyplot as plt

def read_metric_file(file_path):
    """Read metric file and return time and value arrays."""
    if not os.path.exists(file_path):
        return [], []

    timestamps = []
    values = []

    with open(file_path, 'r') as f:
        for line in f:
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
        return [], []

    # Convert to relative time in seconds
    base_time = timestamps[0]
    rel_time = [(t - base_time) / 1000.0 for t in timestamps]

    return rel_time, values

def plot_metric(metric_file, title, ylabel, output_file):
    """Create a time-series plot for a single metric."""
    times, values = read_metric_file(metric_file)

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
    if len(sys.argv) != 5:
        print("Usage: plot-single-metric.py <metric_file> <title> <ylabel> <output_file>")
        sys.exit(1)

    metric_file = sys.argv[1]
    title = sys.argv[2]
    ylabel = sys.argv[3]
    output_file = sys.argv[4]

    plot_metric(metric_file, title, ylabel, output_file)
