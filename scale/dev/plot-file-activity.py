#!/usr/bin/env python3
"""
Plot file activity performance test results.

This script compares metrics between two test runs (typically without policy vs with policy)
and generates comparison plots for CPU, memory, and database table sizes.
"""

import matplotlib.pyplot as plt
import sys
import os

def read_file(file_path, base_time=None):
    """
    Read metric file and return timestamps and values.

    Args:
        file_path: Path to metrics file
        base_time: Optional baseline timestamp (in milliseconds)

    Returns:
        Tuple of (relative_times, values)
    """
    if not os.path.exists(file_path):
        print(f"Warning: File {file_path} does not exist, returning empty data")
        return [], []

    with open(file_path, 'r') as f:
        lines = f.readlines()

    timestamps = []
    values = []

    for line in lines:
        parts = line.strip().split()
        if len(parts) != 2:
            continue  # skip malformed lines
        try:
            ts, val = int(parts[0]), float(parts[1])
            timestamps.append(ts)
            values.append(val)
        except ValueError:
            continue  # skip unparseable lines

    if not timestamps:
        return [], []

    # Normalize time (subtract base timestamp if provided, otherwise use first timestamp)
    if base_time is None:
        base_time = timestamps[0]
    else:
        base_time = int(base_time)

    # Convert to seconds from milliseconds
    rel_time = [(t - base_time) / 1000.0 for t in timestamps]

    return rel_time, values

def plot_data(file1, label1, file2, label2, title, ylabel, base_time1=None, base_time2=None, output_file=None):
    """
    Plot comparison of two metric files.

    Args:
        file1: Path to first metrics file
        label1: Label for first dataset
        file2: Path to second metrics file
        label2: Label for second dataset
        title: Plot title
        ylabel: Y-axis label
        base_time1: Optional baseline timestamp for first dataset
        base_time2: Optional baseline timestamp for second dataset
        output_file: Output PNG file path
    """
    x1, y1 = read_file(file1, base_time1)
    x2, y2 = read_file(file2, base_time2)

    plt.figure(figsize=(12, 7))

    if x1 and y1:
        plt.plot(x1, y1, label=label1, marker='o', markersize=3, linewidth=1.5)
    if x2 and y2:
        plt.plot(x2, y2, label=label2, marker='x', markersize=3, linewidth=1.5)

    plt.xlabel('Time (seconds)', fontsize=12)
    plt.ylabel(ylabel, fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()

    if output_file:
        plt.savefig(output_file, dpi=150)
        print(f"Plot saved to {output_file}")
    else:
        plt.show()

    plt.close()

if __name__ == "__main__":
    if len(sys.argv) < 7:
        print("Usage: python plot-file-activity.py <file1> <label1> <file2> <label2> <title> <y_label> [base_time1] [base_time2] [output_png]")
        print("\nExample:")
        print("  python plot-file-activity.py \\")
        print("    results1/metrics_central_cpu.txt 'Without Policy' \\")
        print("    results2/metrics_central_cpu.txt 'With Policy' \\")
        print("    'Central CPU Usage' 'CPU Cores' \\")
        print("    output.png")
        sys.exit(1)

    file1 = sys.argv[1]
    label1 = sys.argv[2]
    file2 = sys.argv[3]
    label2 = sys.argv[4]
    title = sys.argv[5]
    y_label = sys.argv[6]

    # Optional parameters
    base_time1 = sys.argv[7] if len(sys.argv) > 7 else None
    base_time2 = sys.argv[8] if len(sys.argv) > 8 else None
    output_png = sys.argv[9] if len(sys.argv) > 9 else None

    plot_data(file1, label1, file2, label2, title, y_label, base_time1, base_time2, output_png)
