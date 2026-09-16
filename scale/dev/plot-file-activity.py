#!/usr/bin/env python3
"""
Plot file activity performance test results.

This script compares metrics between two test runs (typically without policy vs with policy)
and generates comparison plots for CPU, memory, and database table sizes.
"""

import matplotlib.pyplot as plt
import sys
import os
from plot_utils import determine_baseline_timestamp, add_trendline, add_equation_text

# Fit trend lines only over the stable window (skip ramp-up and tail), matching
# the window used by plot-batch-comparison.py.
TREND_START = 60.0   # seconds
TREND_END = 660.0    # seconds


def window_points(x, y, start=TREND_START, end=TREND_END):
    """Return the (x, y) points whose x (seconds) falls within [start, end]."""
    xw, yw = [], []
    for xi, yi in zip(x, y):
        if start <= xi <= end:
            xw.append(xi)
            yw.append(yi)
    return xw, yw

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
    if base_time is None or base_time == '':
        base_time = timestamps[0]
    else:
        base_time = int(base_time)

    # Convert to seconds from milliseconds
    rel_time = [(t - base_time) / 1000.0 for t in timestamps]

    return rel_time, values

def plot_data(file1, label1, file2, label2, title, ylabel, results_dir1=None, results_dir2=None,
              component=None, base_time1=None, base_time2=None, output_file=None):
    """
    Plot comparison of two metric files.

    Args:
        file1: Path to first metrics file
        label1: Label for first dataset
        file2: Path to second metrics file
        label2: Label for second dataset
        title: Plot title
        ylabel: Y-axis label
        results_dir1: Directory containing first result set (for component-specific baseline)
        results_dir2: Directory containing second result set (for component-specific baseline)
        component: Component name for determining baseline (e.g., 'sensor', 'central', 'central-db')
        base_time1: Optional baseline timestamp for first dataset (legacy, overrides component-based)
        base_time2: Optional baseline timestamp for second dataset (legacy, overrides component-based)
        output_file: Output PNG file path
    """
    # Determine baselines using component-specific logic if component is provided
    if component and results_dir1 and (base_time1 is None or base_time1 == ''):
        base_time1 = determine_baseline_timestamp(results_dir1, component)
        if base_time1:
            print(f"  Using component-specific baseline for {component} (dir1): t=0 at {base_time1}ms")

    if component and results_dir2 and (base_time2 is None or base_time2 == ''):
        base_time2 = determine_baseline_timestamp(results_dir2, component)
        if base_time2:
            print(f"  Using component-specific baseline for {component} (dir2): t=0 at {base_time2}ms")

    x1, y1 = read_file(file1, base_time1)
    x2, y2 = read_file(file2, base_time2)

    plt.figure(figsize=(12, 7))

    eq1 = eq2 = None
    if x1 and y1:
        plt.plot(x1, y1, label=label1, marker='o', markersize=3, linewidth=1.5, color='C0')
        xw1, yw1 = window_points(x1, y1)
        eq1 = add_trendline(xw1, yw1, f'Trend ({label1}, {int(TREND_START)}-{int(TREND_END)}s)', 'C0')
    if x2 and y2:
        plt.plot(x2, y2, label=label2, marker='x', markersize=3, linewidth=1.5, color='C1')
        xw2, yw2 = window_points(x2, y2)
        eq2 = add_trendline(xw2, yw2, f'Trend ({label2}, {int(TREND_START)}-{int(TREND_END)}s)', 'C1')

    plt.xlabel('Time (seconds)', fontsize=12)
    plt.ylabel(ylabel, fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)

    equations = []
    if eq1:
        equations.append((label1, eq1[0], eq1[1], 'C0'))
    if eq2:
        equations.append((label2, eq2[0], eq2[1], 'C1'))
    add_equation_text(equations)

    plt.tight_layout()

    if output_file:
        plt.savefig(output_file, dpi=150)
        print(f"Plot saved to {output_file}")
    else:
        plt.show()

    plt.close()

if __name__ == "__main__":
    if len(sys.argv) < 7:
        print("Usage: python plot-file-activity.py <file1> <label1> <file2> <label2> <title> <y_label> [results_dir1] [results_dir2] [component] [output_png]")
        print("\nExample (with component-specific baselines):")
        print("  python plot-file-activity.py \\")
        print("    results1/metrics_central_cpu.txt 'Without Policy' \\")
        print("    results2/metrics_central_cpu.txt 'With Policy' \\")
        print("    'Central CPU Usage' 'CPU Cores' \\")
        print("    results1 results2 central \\")
        print("    output.png")
        print("\nLegacy usage (with explicit base times):")
        print("  python plot-file-activity.py <file1> <label1> <file2> <label2> <title> <y_label> <base_time1> <base_time2> <output_png>")
        sys.exit(1)

    file1 = sys.argv[1]
    label1 = sys.argv[2]
    file2 = sys.argv[3]
    label2 = sys.argv[4]
    title = sys.argv[5]
    y_label = sys.argv[6]

    # Optional parameters - support both new (component-based) and legacy (explicit base_time) modes
    results_dir1 = None
    results_dir2 = None
    component = None
    base_time1 = None
    base_time2 = None
    output_png = None

    if len(sys.argv) > 7:
        arg7 = sys.argv[7]
        # Check if arg7 looks like a directory path or a base time (number)
        if arg7 and not arg7.isdigit():
            # New mode: results_dir1
            results_dir1 = arg7
            results_dir2 = sys.argv[8] if len(sys.argv) > 8 else None
            component = sys.argv[9] if len(sys.argv) > 9 else None
            output_png = sys.argv[10] if len(sys.argv) > 10 else None
        else:
            # Legacy mode: base_time1
            base_time1 = arg7
            base_time2 = sys.argv[8] if len(sys.argv) > 8 else None
            output_png = sys.argv[9] if len(sys.argv) > 9 else None

    plot_data(file1, label1, file2, label2, title, y_label,
              results_dir1, results_dir2, component,
              base_time1, base_time2, output_png)
