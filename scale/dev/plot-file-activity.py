#!/usr/bin/env python3
"""
Plot a single file-activity metric over time, overlaying one line per test mode.

Supports any number of series so it works for both the fake workload (two modes:
without/with policy) and the berserker workload (three modes: baseline / alerting
/ idlepolicy). Each series is a (metric file, label, results dir) triple; the
results dir is used only to resolve the component-specific t=0 baseline.
"""

import argparse
import os

import matplotlib.pyplot as plt

from plot_utils import (
    determine_baseline_timestamp,
    add_trendline,
    add_equation_text,
    read_metric_series,
    write_equation_table,
    trend_window,
    series_style,
)


def window_points(x, y, start, end):
    """Return the (x, y) points whose x (seconds) falls within [start, end]."""
    xw, yw = [], []
    for xi, yi in zip(x, y):
        if start <= xi <= end:
            xw.append(xi)
            yw.append(yi)
    return xw, yw


def read_file(file_path, base_time=None):
    """
    Read metric file and return (relative_times_seconds, values).

    base_time (ms) sets t=0; when None/empty the first timestamp is used.
    """
    if not os.path.exists(file_path):
        print(f"Warning: File {file_path} does not exist, returning empty data")
        return [], []

    # Read sorted, de-duplicated series (drops duplicate-timestamp artifacts)
    timestamps, values = read_metric_series(file_path)

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


def plot_data(series, title, ylabel, component=None, output_file=None):
    """
    Overlay one line (with a trend line) per series onto a single plot.

    Args:
        series: list of (file, label, results_dir) triples. results_dir may be
            None/'' to fall back to the metric file's own directory for the
            baseline and trend window.
        title: Plot title.
        ylabel: Y-axis label.
        component: Component name for the component-specific t=0 baseline
            (e.g. 'sensor', 'central', 'central-db'); when set, each series'
            results_dir resolves its own baseline.
        output_file: Output PNG path (shows interactively when None).
    """
    # The trend window (test length) comes from the first series' results dir,
    # falling back to the first metric file's directory for legacy layouts.
    first_file = series[0][0] if series else ''
    first_dir = series[0][2] if series else ''
    run_ref = first_dir or os.path.dirname(first_file) or '.'
    trend_start, trend_end = trend_window(run_ref)

    plt.figure(figsize=(12, 7))

    equations = []
    for i, (file_path, label, results_dir) in enumerate(series):
        base_time = None
        if component and results_dir:
            base_time = determine_baseline_timestamp(results_dir, component)
            if base_time:
                print(f"  Using component-specific baseline for {component} "
                      f"({label}): t=0 at {base_time}ms")

        x, y = read_file(file_path, base_time)
        if not (x and y):
            continue

        color, marker = series_style(i)
        plt.plot(x, y, label=label, marker=marker, markersize=3, linewidth=1.5, color=color)
        xw, yw = window_points(x, y, trend_start, trend_end)
        eq = add_trendline(
            xw, yw, f'Trend ({label}, {int(trend_start)}-{int(trend_end)}s)', color)
        if eq:
            equations.append((label, eq[0], eq[1], color))

    plt.xlabel('Time (seconds)', fontsize=12)
    plt.ylabel(ylabel, fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)

    add_equation_text(equations)

    plt.ylim(bottom=0)  # Anchor the y-axis at 0 so magnitudes are comparable across plots
    plt.tight_layout()

    if output_file:
        plt.savefig(output_file, dpi=150)
        print(f"Plot saved to {output_file}")
        # Save the trend-line equations alongside the plots. Keyed by plot name,
        # so each metric this script plots accumulates into one per-directory
        # table (trendline_equations.csv) and re-runs update rows in place.
        plot_name = os.path.splitext(os.path.basename(output_file))[0]
        rows = [(plot_name, label, slope, intercept)
                for (label, slope, intercept, _color) in equations]
        if rows:
            write_equation_table(
                os.path.join(os.path.dirname(output_file) or '.', 'trendline_equations.csv'),
                rows, upsert=True)
    else:
        plt.show()

    plt.close()


def main():
    parser = argparse.ArgumentParser(
        description="Overlay one metric over time across test modes.")
    parser.add_argument('--title', required=True)
    parser.add_argument('--ylabel', required=True)
    parser.add_argument('--component', default=None,
                        help="Component for the t=0 baseline (central, central-db, sensor, collector).")
    parser.add_argument('--output', default=None, help="Output PNG path.")
    parser.add_argument('--series', action='append', nargs=3, required=True,
                        metavar=('FILE', 'LABEL', 'DIR'),
                        help="A metric file, its legend label, and its results dir "
                             "(pass '' for DIR to use the file's own directory). Repeatable.")
    args = parser.parse_args()

    # Normalize empty DIR to None.
    series = [(f, label, (d or None)) for f, label, d in args.series]
    plot_data(series, args.title, args.ylabel, args.component, args.output)


if __name__ == "__main__":
    main()
