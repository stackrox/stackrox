#!/usr/bin/env python3
"""
Shared utilities for performance test plotting scripts.
"""

import glob
import json
import os
import re
import zipfile
from datetime import datetime

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

# Marker cycle for overlaying multiple test-mode series on one plot (colors use the
# matplotlib C0/C1/... cycle by series index).
SERIES_MARKERS = ['o', 's', '^', 'x', 'd']


def series_style(index):
    """Return (color, marker) for the Nth series overlaid on a plot."""
    return f'C{index}', SERIES_MARKERS[index % len(SERIES_MARKERS)]


def run_duration_seconds(path, default=DEFAULT_RUN_DURATION):
    """
    Parse the nominal test duration (seconds) from a results path.

    Returns `default` when no run_time token is present (e.g. legacy paths).
    """
    m = _RUN_TIME_RE.search(path)
    if not m:
        return default
    return int(m.group(1)) * _UNIT_SECONDS[m.group(2)]


# --- Config-driven run layout (run-perf-tests.sh / perf-test.sh) -------------
# A run writes its cells under <resultsBaseDir>/<name>/ and copies the run config
# to <run_dir>/run-config.json. Each cell dir is named rate-<rate>__<policy>. The
# plotters read the config (rates with actual event rate + substituted value,
# policy nicknames, runTime, numDeployments) instead of parsing dir names, so the
# harness naming and the analysis stay in sync via a single source of truth.
RUN_CONFIG_FILENAME = 'run-config.json'


def load_run_config(run_dir):
    """Load <run_dir>/run-config.json (written by run-perf-tests.sh)."""
    with open(os.path.join(run_dir, RUN_CONFIG_FILENAME)) as fh:
        return json.load(fh)


def cell_dir(run_dir, rate, policy):
    """Path to the results dir for one (rate, policy) cell."""
    return os.path.join(run_dir, f"rate-{rate}__{policy}")


def run_duration_from_config(cfg, default=DEFAULT_RUN_DURATION):
    """Nominal test duration (seconds) from a run config's runTime (e.g. '10m')."""
    rt = str(cfg.get('runTime', '')).strip()
    m = re.fullmatch(r'(\d+)([smh])', rt)
    return int(m.group(1)) * _UNIT_SECONDS[m.group(2)] if m else default


def analysis_window_from_config(cfg, start=TREND_START):
    """Return the (start, end) seconds analysis window for a run config: skip a
    fixed ramp-up and run to start + the config's test duration. The config-driven
    counterpart of trend_window (which parses the duration from a dir name)."""
    return start, start + run_duration_from_config(cfg)


def table_run_args(argv, usage_doc):
    """
    Shared argument handling for the config-driven table scripts.

    Each table script is invoked as `<script> <run_dir> [rate1 rate2 ...]`, where
    run_dir holds run-config.json (written by run-perf-tests.sh). Returns
    (run_dir, cfg, rates, policies):
      - rates:    the event rates to report, sorted ascending; the config's full set
                  unless an explicit subset is passed as positional args.
      - policies: policy nicknames in config order (the table's per-cell columns).
    Prints usage_doc and exits when run_dir is missing.
    """
    if not argv:
        print(usage_doc)
        raise SystemExit(1)
    run_dir = argv[0].rstrip('/')
    cfg = load_run_config(run_dir)
    config_rates = sorted(r['rate'] for r in cfg['rates'])
    rates = [int(a) for a in argv[1:]] or config_rates
    policies = [p['nickname'] for p in cfg['policies']]
    return run_dir, cfg, rates, policies


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


def _relative_window(timestamps, values, start_offset, end_offset, base_time):
    """Shared prep for the metric reducers: convert to relative seconds and filter
    to [start_offset, end_offset]. Returns a list of (rel_time, value) points."""
    if not timestamps:
        return []
    if base_time is None:
        base_time = timestamps[0]
    out = []
    for t, v in zip(timestamps, values):
        rel = (t - base_time) / 1000.0
        if rel >= start_offset and (end_offset is None or rel <= end_offset):
            out.append((rel, v))
    return out


def read_metric_average(file_path, start_offset=60.0, end_offset=None, base_time=None):
    """Average of a metric series over a time window, or None.

    start_offset skips the initial ramp-up; base_time (ms) sets t=0 (defaults to the
    first sample). Used for gauges (CPU, memory)."""
    points = _relative_window(*read_metric_series(file_path), start_offset, end_offset, base_time)
    if not points:
        return None
    return sum(v for _, v in points) / len(points)


def read_metric_max(file_path, start_offset=60.0, end_offset=None, base_time=None):
    """Maximum of a metric series over a time window, or None.

    Used for monotone series (table sizes/row counts) and cumulative counters whose
    final (max) value is the total (restarts, OOM-kills, drops)."""
    points = _relative_window(*read_metric_series(file_path), start_offset, end_offset, base_time)
    if not points:
        return None
    return max(v for _, v in points)


def read_counter_rate(file_path, start_offset=60.0, end_offset=None, base_time=None):
    """Average per-second rate of a cumulative counter over a time window, or None.

    Counters are monotonically increasing totals, so the throughput is the counter's
    increase across the window divided by the elapsed time between the first and last
    in-window samples. Guards against a counter reset (negative delta)."""
    points = _relative_window(*read_metric_series(file_path), start_offset, end_offset, base_time)
    if len(points) < 2:
        return None
    (t_first, v_first), (t_last, v_last) = points[0], points[-1]
    if t_last <= t_first:
        return None
    delta = v_last - v_first
    if delta < 0:
        return None
    return delta / (t_last - t_first)


def read_events_rate_from_bundles(run_dir, metric_name='rox_sensor_file_access_events_received_total'):
    """
    Fallback for runs whose Prometheus time-series file is missing/empty, e.g.
    runs collected before prometheus-query-file-activity.sh was fixed to query
    the correct sensor metric name.

    Each run captures two diagnostic bundles (diagnostic_bundle_1 and
    diagnostic_bundle_2); each is a StackRox diagnostic zip holding a point-in-time
    sensor metrics snapshot at sensor-metrics/remote/metrics.prom. With only two
    snapshots there is no time series, but the counter's average per-second rate is
    still recoverable: the increase between the two snapshots divided by the
    wall-clock gap from the zip filenames (stackrox_diagnostic_YYYY_MM_DD_HH_MM_SS.zip).

    Read-only: reads the bundle zips in place and never writes into the run dir.
    Returns the average rate (events/sec), or None if it can't be computed.
    """
    samples = []  # (datetime, counter_value)
    for bundle in ('diagnostic_bundle_1', 'diagnostic_bundle_2'):
        zips = glob.glob(os.path.join(run_dir, bundle, 'stackrox_diagnostic_*.zip'))
        if not zips:
            continue
        zip_path = sorted(zips)[0]
        m = re.search(r'stackrox_diagnostic_(\d{4}_\d{2}_\d{2}_\d{2}_\d{2}_\d{2})\.zip',
                      os.path.basename(zip_path))
        if not m:
            continue
        ts = datetime.strptime(m.group(1), '%Y_%m_%d_%H_%M_%S')
        value = None
        try:
            with zipfile.ZipFile(zip_path) as zf:
                with zf.open('sensor-metrics/remote/metrics.prom') as fh:
                    for raw in fh:
                        line = raw.decode('utf-8', 'replace')
                        if line.startswith(metric_name + ' '):
                            value = float(line.split()[1])
                            break
        except (KeyError, zipfile.BadZipFile, OSError, ValueError):
            continue
        if value is not None:
            samples.append((ts, value))

    if len(samples) < 2:
        return None
    samples.sort(key=lambda s: s[0])
    (t_first, v_first), (t_last, v_last) = samples[0], samples[-1]
    elapsed = (t_last - t_first).total_seconds()
    if elapsed <= 0 or v_last < v_first:
        return None
    return (v_last - v_first) / elapsed


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
    # Berserker runs record a single pipeline-wide t=0 (written by the berserker
    # load driver, lib/load-berserker.sh). When present, it applies to all
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
