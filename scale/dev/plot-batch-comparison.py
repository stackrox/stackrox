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
import zipfile
from datetime import datetime
from plot_utils import (
    determine_baseline_timestamp,
    add_trendline,
    add_equation_text,
    read_metric_series,
    write_equation_table,
    trend_window,
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

def read_counter_rate(file_path, start_offset=60.0, end_offset=None, base_time=None):
    """
    Read a cumulative-counter metric file and return its average per-second rate
    over a time window (e.g. file-access events received per second).

    Counters are monotonically increasing totals, so averaging the raw value is
    meaningless; the throughput is the counter's increase across the window
    divided by the elapsed time between the first and last in-window samples.

    Args:
        file_path: Path to metrics file
        start_offset: Start time in seconds (to skip initial ramp-up)
        end_offset: End time in seconds (None = until end)
        base_time: Optional baseline timestamp in milliseconds (if None, uses first timestamp)

    Returns:
        Average rate (units/sec) over the window, or None if it can't be computed
    """
    # Read sorted, de-duplicated series (drops duplicate-timestamp artifacts)
    timestamps, values = read_metric_series(file_path)

    if not timestamps or len(timestamps) < 2:
        return None

    # Convert to relative time in seconds
    if base_time is None:
        base_time = timestamps[0]
    rel_times = [(t - base_time) / 1000.0 for t in timestamps]

    # Filter to time window
    points = [
        (t, v) for t, v in zip(rel_times, values)
        if t >= start_offset and (end_offset is None or t <= end_offset)
    ]
    if len(points) < 2:
        return None

    (t_first, v_first), (t_last, v_last) = points[0], points[-1]
    if t_last <= t_first:
        return None

    # Guard against a counter reset within the window (delta would go negative).
    delta = v_last - v_first
    if delta < 0:
        return None

    return delta / (t_last - t_first)

def read_events_rate_from_bundles(run_dir, metric_name='rox_sensor_file_access_events_received_total'):
    """
    Fallback for runs whose Prometheus time-series file is missing/empty, e.g.
    runs collected before prometheus-query-file-activity.sh was fixed to query
    the correct sensor metric name (older runs only have empty files for the
    never-existed names).

    Each run captures two diagnostic bundles (diagnostic_bundle_1 and
    diagnostic_bundle_2). Each bundle is a StackRox diagnostic zip that contains
    a point-in-time sensor metrics snapshot at sensor-metrics/remote/metrics.prom.
    With only two snapshots there is no time series, but the counter's average
    per-second rate is still recoverable: the increase between the two snapshots
    divided by the wall-clock gap between them, taken from the zip filenames
    (stackrox_diagnostic_YYYY_MM_DD_HH_MM_SS.zip).

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

def _to_gb(values):
    """Convert a list of byte values to GB, preserving None entries."""
    return [v / (1024**3) if v else None for v in values]


def _to_mb(values):
    """Convert a list of byte values to MB, preserving None entries."""
    return [v / (1024**2) if v else None for v in values]


def _finalize_plot(output_dir, filename, title, ylabel, equations, record,
                   xlabel='File Activity Event Rate (events/sec)', print_suffix=''):
    """Shared tail for every comparison plot: labels, legend, equation text,
    equation recording, save, and close."""
    plt.xlabel(xlabel, fontsize=12)
    plt.ylabel(ylabel, fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.legend(fontsize=11)
    plt.grid(True, alpha=0.3)
    add_equation_text(equations)
    record(os.path.splitext(filename)[0], equations)
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, filename), dpi=150)
    print(f"Saved: {filename}{print_suffix}")
    plt.close()


def _plot_two_series_vs_rate(output_dir, event_rates, filename, title, ylabel,
                             y_without, y_with, record):
    """Render the standard Without/With Policy vs event-rate comparison plot,
    with a trend line and equation per series."""
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, y_without, 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, y_with, 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    eq1 = add_trendline(event_rates, y_without, 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, y_with, 'Trend (With Policy)', 'C1')
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    _finalize_plot(output_dir, filename, title, ylabel, equations, record)


def plot_scaling_comparison(base_dir, output_dir):
    """
    Generate comparison plots across all batch sizes.

    Args:
        base_dir: Base directory containing test results
        output_dir: Output directory for plots
    """
    os.makedirs(output_dir, exist_ok=True)

    # Collect every plot's trend-line equations so they can be written to a
    # single table (trendline_equations.csv) alongside the plots. record() is
    # called just before each savefig with that plot's name and equations list.
    equation_rows = []
    def record(plot_name, equations):
        for (series, slope, intercept, _color) in equations:
            equation_rows.append((plot_name, series, slope, intercept))

    START_OFFSET = 60.0   # Skip initial 60s ramp-up

    # Find all result directories (both fake workload and berserker)
    pattern_fake = os.path.join(base_dir, "file_activity_results_*_policy_*")
    pattern_berserker = os.path.join(base_dir, "berserker_file_activity_results_*_policy_*")
    result_dirs = glob.glob(pattern_fake) + glob.glob(pattern_berserker)

    # Match the analysis window to the test length: [60s, 60s + run duration],
    # parsed from a result dir name (10m -> 60-660s, 20m -> 60-1260s). All runs
    # in one base dir share a run_time, so any dir works.
    END_OFFSET = trend_window(result_dirs[0], start=START_OFFSET)[1] if result_dirs else 660.0
    TIME_WINDOW_DESC = f"{int(START_OFFSET)}-{int(END_OFFSET)}s"

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
        # Working set (OOM-relevant) and live Go heap, for reasoning about how
        # much of the memory is real vs. reclaimable (see prometheus-query-file-activity.sh).
        'central_ws_without': [],
        'central_ws_with': [],
        'central_heap_without': [],
        'central_heap_with': [],
        'centraldb_cpu_without': [],
        'centraldb_cpu_with': [],
        'centraldb_mem_without': [],
        'centraldb_mem_with': [],
        'centraldb_ws_without': [],
        'centraldb_ws_with': [],
        'sensor_cpu_without': [],
        'sensor_cpu_with': [],
        'sensor_mem_without': [],
        'sensor_mem_with': [],
        'sensor_ws_without': [],
        'sensor_ws_with': [],
        'sensor_heap_without': [],
        'sensor_heap_with': [],
        'collector_cpu_without': [],
        'collector_cpu_with': [],
        'collector_mem_without': [],
        'collector_mem_with': [],
        'fact_cpu_without': [],
        'fact_cpu_with': [],
        'fact_mem_without': [],
        'fact_mem_with': [],
        'alerts_count_without': [],
        'alerts_count_with': [],
        'alerts_size_without': [],
        'alerts_size_with': [],
        # Observed file-access event throughput at the sensor (events/sec).
        'events_received_without': [],
        'events_received_with': [],
    }

    for batch_size in batch_sizes:
        # Without policy
        if 'without' in data[batch_size]:
            without_dir = data[batch_size]['without']
            # Determine component-specific baselines
            central_baseline_without = determine_baseline_timestamp(without_dir, 'central')
            centraldb_baseline_without = determine_baseline_timestamp(without_dir, 'central-db')
            sensor_baseline_without = determine_baseline_timestamp(without_dir, 'sensor')
            # collector and fact share the collector pod; one baseline covers both.
            collector_baseline_without = determine_baseline_timestamp(without_dir, 'collector')

            metrics['central_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_cpu.txt'), START_OFFSET, END_OFFSET, central_baseline_without))
            metrics['central_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_mem.txt'), START_OFFSET, END_OFFSET, central_baseline_without))
            metrics['central_ws_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_mem_workingset.txt'), START_OFFSET, END_OFFSET, central_baseline_without))
            metrics['central_heap_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central_heap_inuse.txt'), START_OFFSET, END_OFFSET, central_baseline_without))
            metrics['centraldb_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_cpu.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['centraldb_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_mem.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['centraldb_ws_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_central-db_mem_workingset.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['sensor_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_cpu.txt'), START_OFFSET, END_OFFSET, sensor_baseline_without))
            metrics['sensor_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_mem.txt'), START_OFFSET, END_OFFSET, sensor_baseline_without))
            metrics['sensor_ws_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_mem_workingset.txt'), START_OFFSET, END_OFFSET, sensor_baseline_without))
            metrics['sensor_heap_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_sensor_heap_inuse.txt'), START_OFFSET, END_OFFSET, sensor_baseline_without))
            metrics['collector_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_collector_cpu.txt'), START_OFFSET, END_OFFSET, collector_baseline_without))
            metrics['collector_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_collector_mem.txt'), START_OFFSET, END_OFFSET, collector_baseline_without))
            metrics['fact_cpu_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_fact_cpu.txt'), START_OFFSET, END_OFFSET, collector_baseline_without))
            metrics['fact_mem_without'].append(
                read_metric_average(os.path.join(without_dir, 'metrics_fact_mem.txt'), START_OFFSET, END_OFFSET, collector_baseline_without))
            metrics['alerts_count_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            metrics['alerts_size_without'].append(
                read_metric_max(os.path.join(without_dir, 'metrics_alerts_bytes.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_without))
            # File-access events are counted on the sensor; use the sensor baseline.
            # Fall back to the diagnostic bundles when the time series is absent
            # (runs collected before the query script used the correct metric name).
            events_rate_without = read_counter_rate(
                os.path.join(without_dir, 'metrics_rox_sensor_file_access_events_received_total.txt'),
                START_OFFSET, END_OFFSET, sensor_baseline_without)
            if events_rate_without is None:
                events_rate_without = read_events_rate_from_bundles(without_dir)
            metrics['events_received_without'].append(events_rate_without)
        else:
            for key in ['central_cpu_without', 'central_mem_without',
                       'central_ws_without', 'central_heap_without', 'centraldb_cpu_without',
                       'centraldb_mem_without', 'centraldb_ws_without',
                       'sensor_cpu_without', 'sensor_mem_without',
                       'sensor_ws_without', 'sensor_heap_without',
                       'collector_cpu_without', 'collector_mem_without',
                       'fact_cpu_without', 'fact_mem_without',
                       'alerts_count_without', 'alerts_size_without',
                       'events_received_without']:
                metrics[key].append(None)

        # With policy
        if 'with' in data[batch_size]:
            with_dir = data[batch_size]['with']
            # Determine component-specific baselines
            central_baseline_with = determine_baseline_timestamp(with_dir, 'central')
            centraldb_baseline_with = determine_baseline_timestamp(with_dir, 'central-db')
            sensor_baseline_with = determine_baseline_timestamp(with_dir, 'sensor')
            # collector and fact share the collector pod; one baseline covers both.
            collector_baseline_with = determine_baseline_timestamp(with_dir, 'collector')

            metrics['central_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_cpu.txt'), START_OFFSET, END_OFFSET, central_baseline_with))
            metrics['central_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_mem.txt'), START_OFFSET, END_OFFSET, central_baseline_with))
            metrics['central_ws_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_mem_workingset.txt'), START_OFFSET, END_OFFSET, central_baseline_with))
            metrics['central_heap_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central_heap_inuse.txt'), START_OFFSET, END_OFFSET, central_baseline_with))
            metrics['centraldb_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_cpu.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['centraldb_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_mem.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['centraldb_ws_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_central-db_mem_workingset.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['sensor_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_cpu.txt'), START_OFFSET, END_OFFSET, sensor_baseline_with))
            metrics['sensor_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_mem.txt'), START_OFFSET, END_OFFSET, sensor_baseline_with))
            metrics['sensor_ws_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_mem_workingset.txt'), START_OFFSET, END_OFFSET, sensor_baseline_with))
            metrics['sensor_heap_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_sensor_heap_inuse.txt'), START_OFFSET, END_OFFSET, sensor_baseline_with))
            metrics['collector_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_collector_cpu.txt'), START_OFFSET, END_OFFSET, collector_baseline_with))
            metrics['collector_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_collector_mem.txt'), START_OFFSET, END_OFFSET, collector_baseline_with))
            metrics['fact_cpu_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_fact_cpu.txt'), START_OFFSET, END_OFFSET, collector_baseline_with))
            metrics['fact_mem_with'].append(
                read_metric_average(os.path.join(with_dir, 'metrics_fact_mem.txt'), START_OFFSET, END_OFFSET, collector_baseline_with))
            metrics['alerts_count_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            metrics['alerts_size_with'].append(
                read_metric_max(os.path.join(with_dir, 'metrics_alerts_bytes.txt'), START_OFFSET, END_OFFSET, centraldb_baseline_with))
            # File-access events are counted on the sensor; use the sensor baseline.
            # Fall back to the diagnostic bundles when the time series is absent
            # (runs collected before the query script used the correct metric name).
            events_rate_with = read_counter_rate(
                os.path.join(with_dir, 'metrics_rox_sensor_file_access_events_received_total.txt'),
                START_OFFSET, END_OFFSET, sensor_baseline_with)
            if events_rate_with is None:
                events_rate_with = read_events_rate_from_bundles(with_dir)
            metrics['events_received_with'].append(events_rate_with)
        else:
            for key in ['central_cpu_with', 'central_mem_with',
                       'central_ws_with', 'central_heap_with', 'centraldb_cpu_with',
                       'centraldb_mem_with', 'centraldb_ws_with',
                       'sensor_cpu_with', 'sensor_mem_with',
                       'sensor_ws_with', 'sensor_heap_with',
                       'collector_cpu_with', 'collector_mem_with',
                       'fact_cpu_with', 'fact_mem_with',
                       'alerts_count_with', 'alerts_size_with',
                       'events_received_with']:
                metrics[key].append(None)

    # The bulk of the comparison plots share one shape: Without/With Policy
    # against event rate, with a trend line per series. Describe each as
    # (filename, title, ylabel, without_values, with_values) and render them in
    # a loop; the two plots that break the mold (policy CPU overhead and events
    # received) are handled separately below.
    standard_plots = [
        ('central_cpu_vs_rate.png',
         f'Central CPU Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average CPU Usage (cores)',
         metrics['central_cpu_without'], metrics['central_cpu_with']),
        ('central_mem_vs_rate.png',
         f'Central Memory Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Memory Usage (GB)',
         _to_gb(metrics['central_mem_without']), _to_gb(metrics['central_mem_with'])),
        ('central_ws_vs_rate.png',
         f'Central Working Set Memory vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Working Set (GB)',
         _to_gb(metrics['central_ws_without']), _to_gb(metrics['central_ws_with'])),
        ('central_heap_vs_rate.png',
         f'Central Go Heap In-Use vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Go Heap In-Use (GB)',
         _to_gb(metrics['central_heap_without']), _to_gb(metrics['central_heap_with'])),
        ('centraldb_cpu_vs_rate.png',
         f'Central-DB CPU Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average CPU Usage (cores)',
         metrics['centraldb_cpu_without'], metrics['centraldb_cpu_with']),
        ('alerts_count_vs_rate.png',
         f'Alerts Table Row Count vs File Activity Event Rate\n(maximum within {TIME_WINDOW_DESC})',
         'Maximum Alert Count',
         metrics['alerts_count_without'], metrics['alerts_count_with']),
        ('alerts_size_vs_rate.png',
         f'Alerts Table Size vs File Activity Event Rate\n(maximum within {TIME_WINDOW_DESC})',
         'Maximum Table Size (MB)',
         _to_mb(metrics['alerts_size_without']), _to_mb(metrics['alerts_size_with'])),
        ('centraldb_mem_vs_rate.png',
         f'Central-DB Memory Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Memory Usage (GB)',
         _to_gb(metrics['centraldb_mem_without']), _to_gb(metrics['centraldb_mem_with'])),
        ('centraldb_ws_vs_rate.png',
         f'Central-DB Working Set Memory vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Working Set (GB)',
         _to_gb(metrics['centraldb_ws_without']), _to_gb(metrics['centraldb_ws_with'])),
        ('sensor_cpu_vs_rate.png',
         f'Sensor CPU Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average CPU Usage (cores)',
         metrics['sensor_cpu_without'], metrics['sensor_cpu_with']),
        ('sensor_mem_vs_rate.png',
         f'Sensor Memory Usage vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Memory Usage (GB)',
         _to_gb(metrics['sensor_mem_without']), _to_gb(metrics['sensor_mem_with'])),
        ('sensor_ws_vs_rate.png',
         f'Sensor Working Set Memory vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Working Set (GB)',
         _to_gb(metrics['sensor_ws_without']), _to_gb(metrics['sensor_ws_with'])),
        ('sensor_heap_vs_rate.png',
         f'Sensor Go Heap In-Use vs File Activity Event Rate\n(averaged over {TIME_WINDOW_DESC})',
         'Average Go Heap In-Use (GB)',
         _to_gb(metrics['sensor_heap_without']), _to_gb(metrics['sensor_heap_with'])),
        ('collector_cpu_vs_rate.png',
         f'Collector CPU Usage vs File Activity Event Rate\n(summed across pods, averaged over {TIME_WINDOW_DESC})',
         'Average CPU Usage (cores, all pods)',
         metrics['collector_cpu_without'], metrics['collector_cpu_with']),
        ('collector_mem_vs_rate.png',
         f'Collector Memory Usage vs File Activity Event Rate\n(summed across pods, averaged over {TIME_WINDOW_DESC})',
         'Average Memory Usage (GB, all pods)',
         _to_gb(metrics['collector_mem_without']), _to_gb(metrics['collector_mem_with'])),
        ('fact_cpu_vs_rate.png',
         f'Fact (file-activity monitor) CPU Usage vs Event Rate\n(summed across pods, averaged over {TIME_WINDOW_DESC})',
         'Average CPU Usage (cores, all pods)',
         metrics['fact_cpu_without'], metrics['fact_cpu_with']),
        ('fact_mem_vs_rate.png',
         f'Fact (file-activity monitor) Memory Usage vs Event Rate\n(summed across pods, averaged over {TIME_WINDOW_DESC})',
         'Average Memory Usage (GB, all pods)',
         _to_gb(metrics['fact_mem_without']), _to_gb(metrics['fact_mem_with'])),
    ]
    for filename, title, ylabel, y_without, y_with in standard_plots:
        _plot_two_series_vs_rate(output_dir, event_rates, filename, title, ylabel,
                                 y_without, y_with, record)

    # Policy enforcement CPU overhead: a single derived series (percent increase
    # in Central CPU when policy is enabled), with a zero reference line.
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
    equations = []
    if eq1: equations.append(('Overhead', eq1[0], eq1[1], 'red'))
    _finalize_plot(output_dir, 'policy_cpu_overhead.png',
                   f'Policy Enforcement CPU Overhead\n(Central CPU increase when policy enabled, {TIME_WINDOW_DESC})',
                   'CPU Overhead (%)', equations, record,
                   print_suffix=' (Central CPU increase when policy is enabled)')

    # File-access events received at sensor vs configured event rate. The dashed
    # line is the ideal (observed throughput == configured rate); a curve that
    # flattens below it means events are being generated or delivered more slowly
    # than the workload nominally requests.
    plt.figure(figsize=(12, 7))
    plt.plot(event_rates, metrics['events_received_without'], 'o-', label='Without Policy', linewidth=2, markersize=8, color='C0')
    plt.plot(event_rates, metrics['events_received_with'], 's-', label='With Policy', linewidth=2, markersize=8, color='C1')
    if event_rates:
        max_rate = max(event_rates)
        plt.plot([0, max_rate], [0, max_rate], '--', color='gray', alpha=0.7, label='Ideal (observed = configured)')
    eq1 = add_trendline(event_rates, metrics['events_received_without'], 'Trend (Without Policy)', 'C0')
    eq2 = add_trendline(event_rates, metrics['events_received_with'], 'Trend (With Policy)', 'C1')
    # Scale the y-axis to the observed data (Without/With Policy), not the ideal
    # line. The observed rate is orders of magnitude below the configured rate,
    # so letting the ideal line (0..max_rate) drive the y-range flattens the
    # actual curves onto the x-axis. The ideal line stays drawn as a reference
    # but runs off the top of the frame.
    observed = [v for v in metrics['events_received_without'] + metrics['events_received_with'] if v is not None]
    if observed:
        y_min, y_max = min(observed), max(observed)
        pad = (y_max - y_min) * 0.1 or (y_max * 0.1 or 1.0)
        plt.ylim(max(0.0, y_min - pad), y_max + pad)
    equations = []
    if eq1: equations.append(('Without Policy', eq1[0], eq1[1], 'C0'))
    if eq2: equations.append(('With Policy', eq2[0], eq2[1], 'C1'))
    _finalize_plot(output_dir, 'events_received_vs_rate.png',
                   f'File-Access Events Received vs Configured Event Rate\n(averaged over {TIME_WINDOW_DESC})',
                   'Observed Events Received at Sensor (events/sec)', equations, record,
                   xlabel='Configured File Activity Event Rate (events/sec)')

    # Write every plot's trend-line equations to one table alongside the plots.
    write_equation_table(os.path.join(output_dir, 'trendline_equations.csv'), equation_rows)

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
