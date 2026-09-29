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
    ALL_MODES,
    mode_for_dir,
    series_style,
)

# Human-readable label per mode key, and the canonical order to draw modes in
# (baseline/alerting/idlepolicy for berserker, without/with for fake). Defined in
# plot_utils so every analysis script agrees on names and suffixes.
MODE_LABEL = {key: label for key, _suffix, label in ALL_MODES}
MODE_CANONICAL_ORDER = [key for key, _suffix, _label in ALL_MODES]

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
    plt.ylim(bottom=0)  # Anchor the y-axis at 0 so magnitudes are comparable across plots
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, filename), dpi=150)
    print(f"Saved: {filename}{print_suffix}")
    plt.close()


def _plot_series_vs_rate(output_dir, event_rates, filename, title, ylabel,
                         series_by_mode, mode_order, record):
    """Render a vs-event-rate comparison plot with one line (and trend line) per
    test mode. Works for any number of modes (2 for fake, 3 for berserker)."""
    plt.figure(figsize=(12, 7))
    equations = []
    for i, mode in enumerate(mode_order):
        color, marker = series_style(i)
        label = MODE_LABEL[mode]
        y = series_by_mode.get(mode, [])
        plt.plot(event_rates, y, marker=marker, linestyle='-', label=label,
                 linewidth=2, markersize=8, color=color)
        eq = add_trendline(event_rates, y, f'Trend ({label})', color)
        if eq:
            equations.append((label, eq[0], eq[1], color))
    _finalize_plot(output_dir, filename, title, ylabel, equations, record)


# How to reduce each metric's time series to one number per run, and how to plot it.
#   reader: 'avg'   average over the window (gauges: CPU, memory)
#           'max'   maximum over the window (monotone table sizes / row counts)
#           'rate'  average per-second rate of a cumulative counter
#           'events' like 'rate', with a fallback to the diagnostic bundles
#   conv:   post-scale ('gb', 'mb', or None)
#   component: which component's t=0 baseline to use
# (key, metrics filename, component, reader, conv)
METRIC_SPECS = [
    ('central_cpu',   'metrics_central_cpu.txt',               'central',    'avg',  None),
    ('central_mem',   'metrics_central_mem.txt',               'central',    'avg',  'gb'),
    ('central_ws',    'metrics_central_mem_workingset.txt',    'central',    'avg',  'gb'),
    ('central_heap',  'metrics_central_heap_inuse.txt',        'central',    'avg',  'gb'),
    ('centraldb_cpu', 'metrics_central-db_cpu.txt',            'central-db', 'avg',  None),
    ('centraldb_mem', 'metrics_central-db_mem.txt',            'central-db', 'avg',  'gb'),
    ('centraldb_ws',  'metrics_central-db_mem_workingset.txt', 'central-db', 'avg',  'gb'),
    ('sensor_cpu',    'metrics_sensor_cpu.txt',                'sensor',     'avg',  None),
    ('sensor_mem',    'metrics_sensor_mem.txt',                'sensor',     'avg',  'gb'),
    ('sensor_ws',     'metrics_sensor_mem_workingset.txt',     'sensor',     'avg',  'gb'),
    ('sensor_heap',   'metrics_sensor_heap_inuse.txt',         'sensor',     'avg',  'gb'),
    ('collector_cpu', 'metrics_collector_cpu.txt',             'collector',  'avg',  None),
    ('collector_mem', 'metrics_collector_mem.txt',             'collector',  'avg',  'gb'),
    ('fact_cpu',      'metrics_fact_cpu.txt',                  'collector',  'avg',  None),
    ('fact_mem',      'metrics_fact_mem.txt',                  'collector',  'avg',  'gb'),
    ('alerts_count',  'metrics_alerts.txt',                    'central-db', 'max',  None),
    ('alerts_size',   'metrics_alerts_bytes.txt',              'central-db', 'max',  'mb'),
    ('process_indicators_count', 'metrics_process_indicators.txt',       'central-db', 'max', None),
    ('process_indicators_size',  'metrics_process_indicators_bytes.txt', 'central-db', 'max', 'mb'),
    ('events_received',    'metrics_rox_sensor_file_access_events_received_total.txt', 'sensor', 'events', None),
    ('network_flows_received',      'metrics_network_flows_received.txt',      'sensor', 'rate', None),
    ('process_indicators_received', 'metrics_process_indicators_received.txt', 'sensor', 'rate', None),
]

# Standard vs-rate plots (one per metric, one line per mode). events_received is
# excluded here and drawn separately (it has an "ideal" reference line and custom
# y-scaling). {win} is filled with the analysis window at runtime.
# (metric key, output png, title template, y-axis label)
STANDARD_PLOTS = [
    ('central_cpu',   'central_cpu_vs_rate.png',
     'Central CPU Usage vs File Activity Event Rate\n(averaged over {win})', 'Average CPU Usage (cores)'),
    ('central_mem',   'central_mem_vs_rate.png',
     'Central Memory Usage vs File Activity Event Rate\n(averaged over {win})', 'Average Memory Usage (GB)'),
    ('central_ws',    'central_ws_vs_rate.png',
     'Central Working Set Memory vs File Activity Event Rate\n(averaged over {win})', 'Average Working Set (GB)'),
    ('central_heap',  'central_heap_vs_rate.png',
     'Central Go Heap In-Use vs File Activity Event Rate\n(averaged over {win})', 'Average Go Heap In-Use (GB)'),
    ('centraldb_cpu', 'centraldb_cpu_vs_rate.png',
     'Central-DB CPU Usage vs File Activity Event Rate\n(averaged over {win})', 'Average CPU Usage (cores)'),
    ('centraldb_mem', 'centraldb_mem_vs_rate.png',
     'Central-DB Memory Usage vs File Activity Event Rate\n(averaged over {win})', 'Average Memory Usage (GB)'),
    ('centraldb_ws',  'centraldb_ws_vs_rate.png',
     'Central-DB Working Set Memory vs File Activity Event Rate\n(averaged over {win})', 'Average Working Set (GB)'),
    ('sensor_cpu',    'sensor_cpu_vs_rate.png',
     'Sensor CPU Usage vs File Activity Event Rate\n(averaged over {win})', 'Average CPU Usage (cores)'),
    ('sensor_mem',    'sensor_mem_vs_rate.png',
     'Sensor Memory Usage vs File Activity Event Rate\n(averaged over {win})', 'Average Memory Usage (GB)'),
    ('sensor_ws',     'sensor_ws_vs_rate.png',
     'Sensor Working Set Memory vs File Activity Event Rate\n(averaged over {win})', 'Average Working Set (GB)'),
    ('sensor_heap',   'sensor_heap_vs_rate.png',
     'Sensor Go Heap In-Use vs File Activity Event Rate\n(averaged over {win})', 'Average Go Heap In-Use (GB)'),
    ('collector_cpu', 'collector_cpu_vs_rate.png',
     'Collector CPU Usage vs File Activity Event Rate\n(summed across pods, averaged over {win})',
     'Average CPU Usage (cores, all pods)'),
    ('collector_mem', 'collector_mem_vs_rate.png',
     'Collector Memory Usage vs File Activity Event Rate\n(summed across pods, averaged over {win})',
     'Average Memory Usage (GB, all pods)'),
    ('fact_cpu',      'fact_cpu_vs_rate.png',
     'Fact (file-activity monitor) CPU Usage vs Event Rate\n(summed across pods, averaged over {win})',
     'Average CPU Usage (cores, all pods)'),
    ('fact_mem',      'fact_mem_vs_rate.png',
     'Fact (file-activity monitor) Memory Usage vs Event Rate\n(summed across pods, averaged over {win})',
     'Average Memory Usage (GB, all pods)'),
    ('alerts_count',  'alerts_count_vs_rate.png',
     'Alerts Table Row Count vs File Activity Event Rate\n(maximum within {win})', 'Maximum Alert Count'),
    ('alerts_size',   'alerts_size_vs_rate.png',
     'Alerts Table Size vs File Activity Event Rate\n(maximum within {win})', 'Maximum Table Size (MB)'),
    ('process_indicators_count', 'process_indicators_count_vs_rate.png',
     'Process Indicators Table Row Count vs File Activity Event Rate\n(maximum within {win})',
     'Maximum Row Count'),
    ('process_indicators_size',  'process_indicators_size_vs_rate.png',
     'Process Indicators Table Size vs File Activity Event Rate\n(maximum within {win})',
     'Maximum Table Size (MB)'),
    ('network_flows_received', 'network_flows_received_vs_rate.png',
     'Sensor Network Flows Received Rate vs File Activity Event Rate\n(averaged over {win})',
     'Network Flows Received (flows/sec)'),
    ('process_indicators_received', 'process_indicators_received_vs_rate.png',
     'Sensor Process Indicators Received Rate vs File Activity Event Rate\n(averaged over {win})',
     'Process Indicators Received (/sec)'),
]


def _convert(v, conv):
    """Post-scale a value to GB/MB, preserving None."""
    if v is None:
        return None
    if conv == 'gb':
        return v / (1024 ** 3)
    if conv == 'mb':
        return v / (1024 ** 2)
    return v


def _extract(run_dir, filename, component, reader, baseline_cache, start, end):
    """Reduce one metric file in one run to a single number (or None)."""
    if run_dir is None:
        return None
    if component not in baseline_cache:
        baseline_cache[component] = determine_baseline_timestamp(run_dir, component)
    baseline = baseline_cache[component]
    path = os.path.join(run_dir, filename)
    if reader == 'avg':
        return read_metric_average(path, start, end, baseline)
    if reader == 'max':
        return read_metric_max(path, start, end, baseline)
    if reader == 'rate':
        return read_counter_rate(path, start, end, baseline)
    if reader == 'events':
        v = read_counter_rate(path, start, end, baseline)
        if v is None:
            # Runs whose Prometheus series is missing/empty (older collection):
            # recover the average rate from the two diagnostic bundles.
            v = read_events_rate_from_bundles(run_dir)
        return v
    return None


def plot_scaling_comparison(base_dir, output_dir):
    """
    Generate mode-comparison plots across all batch sizes / event rates.

    One line per test mode (2 for the fake workload, 3 for berserker) is drawn on
    each metric-vs-rate plot. Args:
        base_dir: Base directory containing test results
        output_dir: Output directory for plots
    """
    os.makedirs(output_dir, exist_ok=True)

    # Collect every plot's trend-line equations so they can be written to a
    # single table (trendline_equations.csv) alongside the plots. record() is
    # called just before each savefig with that plot's name and equations list.
    equation_rows = []
    def record(plot_name, equations):
        for (series_name, slope, intercept, _color) in equations:
            equation_rows.append((plot_name, series_name, slope, intercept))

    START_OFFSET = 60.0   # Skip initial 60s ramp-up

    # Find all result directories (both fake workload and berserker), then keep
    # only those whose name ends in a known test-mode suffix.
    result_dirs = (glob.glob(os.path.join(base_dir, "file_activity_results_*")) +
                   glob.glob(os.path.join(base_dir, "berserker_file_activity_results_*")))
    result_dirs = [d for d in result_dirs if os.path.isdir(d) and mode_for_dir(d)]

    if not result_dirs:
        print(f"No test results found in {base_dir}")
        return

    # Match the analysis window to the test length: [60s, 60s + run duration],
    # parsed from a result dir name (10m -> 60-660s, 20m -> 60-1260s). All runs
    # in one base dir share a run_time, so any dir works.
    END_OFFSET = trend_window(result_dirs[0], start=START_OFFSET)[1]
    TIME_WINDOW_DESC = f"{int(START_OFFSET)}-{int(END_OFFSET)}s"

    # Organize by batch size and mode: data[batch_size][mode_key] = dir.
    data = {}
    modes_present = set()
    for dir_path in result_dirs:
        batch_size = extract_batch_size(dir_path)
        if batch_size is None:
            continue
        mode_key, _label = mode_for_dir(dir_path)
        data.setdefault(batch_size, {})[mode_key] = dir_path
        modes_present.add(mode_key)

    batch_sizes = sorted(data.keys())
    event_rates = [b * 10 for b in batch_sizes]  # batch_size * 10 events/sec
    if not batch_sizes:
        print(f"No test results found in {base_dir}")
        return

    # Draw modes in the canonical order, restricted to those actually present.
    mode_order = [m for m in MODE_CANONICAL_ORDER if m in modes_present]

    print(f"Found results for batch sizes: {batch_sizes}")
    print(f"Event rates: {event_rates} events/sec")
    print(f"Modes: {[MODE_LABEL[m] for m in mode_order]}")

    # series[metric_key][mode] = [one value per event rate]
    series = {key: {m: [] for m in mode_order} for key, *_ in METRIC_SPECS}
    for batch in batch_sizes:
        for mode in mode_order:
            run_dir = data[batch].get(mode)
            baseline_cache = {}  # component -> baseline, per run
            for key, filename, component, reader, conv in METRIC_SPECS:
                v = _extract(run_dir, filename, component, reader,
                             baseline_cache, START_OFFSET, END_OFFSET)
                series[key][mode].append(_convert(v, conv))

    # Standard vs-rate plots: one line per mode.
    for key, png, title_tmpl, ylabel in STANDARD_PLOTS:
        _plot_series_vs_rate(output_dir, event_rates, png,
                             title_tmpl.format(win=TIME_WINDOW_DESC), ylabel,
                             series[key], mode_order, record)

    # CPU overhead of each non-reference mode relative to the reference (first)
    # mode -- baseline for berserker, Without Policy for fake. One line per other
    # mode; needs at least two modes to be meaningful.
    if len(mode_order) >= 2:
        ref = mode_order[0]
        ref_label = MODE_LABEL[ref]
        plt.figure(figsize=(12, 7))
        equations = []
        for i, mode in enumerate(mode_order[1:], start=1):
            color, marker = series_style(i)
            label = MODE_LABEL[mode]
            overhead = []
            for base_v, v in zip(series['central_cpu'][ref], series['central_cpu'][mode]):
                if base_v and v and base_v > 0:
                    overhead.append((v - base_v) / base_v * 100)
                else:
                    overhead.append(None)
            plt.plot(event_rates, overhead, marker=marker, linestyle='-',
                     label=f'{label} vs {ref_label}', linewidth=2, markersize=8, color=color)
            eq = add_trendline(event_rates, overhead, f'Trend ({label})', color)
            if eq:
                equations.append((label, eq[0], eq[1], color))
        plt.axhline(y=0, color='gray', linestyle='--', alpha=0.5)
        _finalize_plot(output_dir, 'policy_cpu_overhead.png',
                       f'Central CPU Overhead vs {ref_label}\n({TIME_WINDOW_DESC})',
                       'CPU Overhead (%)', equations, record,
                       print_suffix=' (Central CPU increase vs the reference mode)')

    # File-access events received at sensor vs configured event rate. The dashed
    # line is the ideal (observed throughput == configured rate); a curve that
    # flattens below it means events are generated/delivered slower than requested.
    plt.figure(figsize=(12, 7))
    equations = []
    observed = []
    for i, mode in enumerate(mode_order):
        color, marker = series_style(i)
        label = MODE_LABEL[mode]
        y = series['events_received'][mode]
        plt.plot(event_rates, y, marker=marker, linestyle='-', label=label,
                 linewidth=2, markersize=8, color=color)
        eq = add_trendline(event_rates, y, f'Trend ({label})', color)
        if eq:
            equations.append((label, eq[0], eq[1], color))
        observed += [v for v in y if v is not None]
    if event_rates:
        max_rate = max(event_rates)
        plt.plot([0, max_rate], [0, max_rate], '--', color='gray', alpha=0.7,
                 label='Ideal (observed = configured)')
    # Scale the y-axis to the observed data, not the ideal line (which is orders
    # of magnitude higher and would flatten the real curves onto the x-axis).
    if observed:
        y_min, y_max = min(observed), max(observed)
        pad = (y_max - y_min) * 0.1 or (y_max * 0.1 or 1.0)
        plt.ylim(max(0.0, y_min - pad), y_max + pad)
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
