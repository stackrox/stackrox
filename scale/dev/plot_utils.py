#!/usr/bin/env python3
"""
Shared utilities for performance test plotting scripts.
"""

import os
import re

import numpy as np
import matplotlib.pyplot as plt


# Result dirs are named ..._<num_sensors>_<run_time>_file-activity-... where
# run_time is like '10m', '20m', '1h', '600s'. This captures that run_time token.
_RUN_TIME_RE = re.compile(r'_(\d+)([smh])_file-activity')
_UNIT_SECONDS = {'s': 1, 'm': 60, 'h': 3600}

# Trend fits / averages start after this ramp-up (seconds); the window then runs
# to the end of the nominal test (see trend_window).
TREND_START = 60.0
# Fallback test duration (seconds) when the run_time can't be parsed from a path.
DEFAULT_RUN_DURATION = 600.0


def run_duration_seconds(path, default=DEFAULT_RUN_DURATION):
    """
    Parse the nominal test duration (seconds) from a results path.

    Returns `default` when no run_time token is present (e.g. legacy paths).
    """
    m = _RUN_TIME_RE.search(path)
    if not m:
        return default
    return int(m.group(1)) * _UNIT_SECONDS[m.group(2)]


def trend_window(path, start=TREND_START, default_duration=DEFAULT_RUN_DURATION):
    """
    Return the (start, end) seconds over which to fit trends / average metrics.

    The window skips a fixed ramp-up (`start`) and runs to the end of the test:
    end = start + nominal test duration parsed from `path`. So a 10m test gives
    60-660s and a 20m test gives 60-1260s, keeping the analysis window matched to
    however long the test actually ran.
    """
    return start, start + run_duration_seconds(path, default_duration)


def read_metric_series(file_path):
    """
    Read a metrics file and return (timestamps_ms, values) sorted by timestamp.

    Lines are "<timestamp_ms> <value>"; malformed lines are skipped. Duplicate
    timestamps are collapsed to their maximum value. Prometheus can return more
    than one series for a component (e.g. a pod restart leaves a stale, low
    container reading alongside the live one), which shows up as two datapoints
    at the same timestamp. Those spurious readings are low, so keeping the max
    per timestamp drops them and leaves the real series.

    Negative values are dropped as invalid. Every metric read here (CPU usage,
    memory, table row counts, table sizes, event counts) is a non-negative
    physical quantity, so a negative reading is a "no data yet" sentinel (e.g.
    the table row-count query returns -1 before stats are gathered). Left in, a
    single -1 at the start of a series both dips the plotted line and skews the
    linear trend fit (the fit rotates to split the residuals, so the sentinel is
    never flagged as an outlier).

    Returns:
        Tuple of (timestamps, values); empty lists if the file is missing/empty.
    """
    if not os.path.exists(file_path):
        return [], []

    by_ts = {}
    with open(file_path, 'r') as f:
        for line in f:
            parts = line.strip().split()
            if len(parts) != 2:
                continue
            try:
                ts, val = int(parts[0]), float(parts[1])
            except ValueError:
                continue
            if val < 0:
                continue
            if ts not in by_ts or val > by_ts[ts]:
                by_ts[ts] = val

    timestamps = sorted(by_ts)
    values = [by_ts[ts] for ts in timestamps]
    return timestamps, values


def determine_baseline_timestamp(results_dir, component):
    """
    Determine the baseline timestamp (t=0) for a given component.

    Args:
        results_dir: Directory containing metrics files
        component: Component name (e.g., 'sensor', 'central', 'central-db')

    Returns:
        Baseline timestamp in milliseconds, or None if cannot be determined

    Logic:
        - Berserker runs: use the recorded berserker-ready timestamp for every
          component. Berserker is deployed last (all upstream components are
          already up), so its ready time is t=0 for the whole pipeline.
        - For sensor: use earliest timestamp across all sensor metric files
        - For central/central-db: use timestamp when deployments >= 100
          (or 90% of max if never reaches 100)
    """
    # Berserker runs record a single pipeline-wide t=0 (see
    # perf-test-file-activity-berserker.sh). When present, it applies to all
    # components; fall through to the fake-workload logic otherwise.
    berserker_ts_file = os.path.join(results_dir, 'berserker_ready_timestamp.txt')
    if os.path.exists(berserker_ts_file):
        with open(berserker_ts_file, 'r') as f:
            content = f.read().strip()
        try:
            return int(content)
        except ValueError:
            pass

    if component == 'sensor':
        # For sensor, use earliest timestamp across all sensor metric files
        sensor_files = [
            os.path.join(results_dir, 'metrics_sensor_cpu.txt'),
            os.path.join(results_dir, 'metrics_sensor_mem.txt')
        ]

        min_timestamp = None
        for sensor_file in sensor_files:
            if not os.path.exists(sensor_file):
                continue

            with open(sensor_file, 'r') as f:
                for line in f:
                    parts = line.strip().split()
                    if len(parts) == 2:
                        try:
                            ts = int(parts[0])
                            if min_timestamp is None or ts < min_timestamp:
                                min_timestamp = ts
                            break  # Only need first timestamp from each file
                        except ValueError:
                            continue

        return min_timestamp

    elif component in ['central', 'central-db']:
        # For central/central-db, use timestamp when deployments >= 100
        deployments_file = os.path.join(results_dir, 'metrics_deployments.txt')
        if not os.path.exists(deployments_file):
            return None

        timestamps = []
        values = []

        with open(deployments_file, 'r') as f:
            for line in f:
                parts = line.strip().split()
                if len(parts) == 2:
                    try:
                        ts, val = int(parts[0]), float(parts[1])
                        timestamps.append(ts)
                        values.append(val)
                    except ValueError:
                        continue

        if not timestamps:
            return None

        # Find first timestamp where deployments >= 100
        for ts, val in zip(timestamps, values):
            if val >= 100:
                return ts

        # If never reaches 100, use 90% of maximum
        max_deployments = max(values)
        threshold = 0.9 * max_deployments
        for ts, val in zip(timestamps, values):
            if val >= threshold:
                return ts

        # Fallback: use first timestamp
        return timestamps[0]

    return None


def add_trendline(x_data, y_data, label, color, linestyle='--',
                  max_iterations=10, sigma=2.0):
    """
    Add a linear trend line to the current plot and return the equation.

    Outliers are removed iteratively: after each linear fit, points whose
    residual exceeds `sigma` standard deviations of the residuals are dropped
    and the line is refit. This repeats until no outliers remain or
    `max_iterations` is reached (always keeping at least 2 points).

    Args:
        x_data: X-axis values
        y_data: Y-axis values
        label: Label for the trend line
        color: Color for the trend line
        linestyle: Line style for the trend line
        max_iterations: Maximum number of outlier-removal iterations
        sigma: Residual threshold (in standard deviations) for an outlier

    Returns:
        Tuple of (slope, intercept) or None if insufficient data
    """
    # Filter out None values
    valid_points = [(x, y) for x, y in zip(x_data, y_data) if y is not None]
    if len(valid_points) < 2:
        return None  # Need at least 2 points for a trend line

    x_valid = np.array([p[0] for p in valid_points], dtype=float)
    y_valid = np.array([p[1] for p in valid_points], dtype=float)

    # Fit linear trend line, iteratively removing residual outliers
    coeffs = np.polyfit(x_valid, y_valid, 1)
    for _ in range(max_iterations):
        residuals = y_valid - np.polyval(coeffs, x_valid)
        std = residuals.std()
        if std == 0:
            break  # perfect fit, nothing to remove
        keep = np.abs(residuals) <= sigma * std
        if keep.all() or keep.sum() < 2:
            break  # no outliers, or removing more would leave < 2 points
        x_valid = x_valid[keep]
        y_valid = y_valid[keep]
        coeffs = np.polyfit(x_valid, y_valid, 1)

    trend_y = np.polyval(coeffs, x_valid)

    # Plot trend line
    plt.plot(x_valid, trend_y, linestyle=linestyle, linewidth=1.5, color=color,
             alpha=0.7, label=label)

    # Return slope and intercept
    return (coeffs[0], coeffs[1])


def format_equation(slope, intercept):
    """Format a linear trend as 'y = <slope>x + <intercept>' (or '' if undefined)."""
    if slope is None or intercept is None:
        return ''
    sign = '+' if intercept >= 0 else '-'
    return f"y = {slope:.4e}x {sign} {abs(intercept):.4f}"


def write_equation_table(csv_path, rows, upsert=False):
    """
    Write trend-line equations to a CSV table (columns:
    plot, series, slope, intercept, equation).

    Args:
        csv_path: Output CSV path.
        rows: Iterable of (plot, series, slope, intercept) tuples.
        upsert: When True, merge into an existing table keyed by (plot, series),
            so callers that emit one plot at a time (e.g. plot-file-activity.py,
            invoked once per metric) accumulate into a single table and re-runs
            update rows in place. When False, overwrite (for callers that collect
            every row before writing, e.g. plot-batch-comparison.py).
    """
    import csv
    fieldnames = ['plot', 'series', 'slope', 'intercept', 'equation']
    merged = {}
    if upsert and os.path.exists(csv_path):
        with open(csv_path, newline='') as f:
            for r in csv.DictReader(f):
                merged[(r['plot'], r['series'])] = r
    for plot, series, slope, intercept in rows:
        merged[(plot, series)] = {
            'plot': plot,
            'series': series,
            'slope': '' if slope is None else f"{slope:.6e}",
            'intercept': '' if intercept is None else f"{intercept:.6f}",
            'equation': format_equation(slope, intercept),
        }
    with open(csv_path, 'w', newline='') as f:
        w = csv.DictWriter(f, fieldnames=fieldnames)
        w.writeheader()
        for key in sorted(merged):
            w.writerow(merged[key])


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
            equation_text.append(f"{label}: {format_equation(slope, intercept)}")

    if equation_text:
        text_str = '\n'.join(equation_text)
        plt.text(0.02, y_position, text_str, transform=plt.gca().transAxes,
                fontsize=9, verticalalignment='top',
                bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))
