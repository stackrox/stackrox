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
from plot_utils import (
    determine_baseline_timestamp,
    add_trendline,
    add_equation_text,
    write_equation_table,
    series_style,
    load_run_config,
    cell_dir,
    run_duration_from_config,
    read_metric_average,
    read_metric_max,
    read_counter_rate,
    read_events_rate_from_bundles,
)


def _to_gb(values):
    """Convert a list of byte values to GiB, preserving None entries."""
    return [v / (1024**3) if v else None for v in values]


def _to_mb(values):
    """Convert a list of byte values to MiB, preserving None entries."""
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
                         series_by_combo, combos, labels, record):
    """Render a vs-event-rate comparison plot with one line (and trend line) per
    series, where a series is a (mode, deployment-count) combination. Works for any
    number of modes (2 for fake, 3 for berserker) and deployment counts."""
    plt.figure(figsize=(12, 7))
    equations = []
    for i, combo in enumerate(combos):
        color, marker = series_style(i)
        label = labels[combo]
        y = series_by_combo.get(combo, [])
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
     'Central Memory Usage vs File Activity Event Rate\n(averaged over {win})', 'Average Memory Usage (GiB)'),
    ('central_ws',    'central_ws_vs_rate.png',
     'Central Working Set Memory vs File Activity Event Rate\n(averaged over {win})', 'Average Working Set (GiB)'),
    ('central_heap',  'central_heap_vs_rate.png',
     'Central Go Heap In-Use vs File Activity Event Rate\n(averaged over {win})', 'Average Go Heap In-Use (GiB)'),
    ('centraldb_cpu', 'centraldb_cpu_vs_rate.png',
     'Central-DB CPU Usage vs File Activity Event Rate\n(averaged over {win})', 'Average CPU Usage (cores)'),
    ('centraldb_mem', 'centraldb_mem_vs_rate.png',
     'Central-DB Memory Usage vs File Activity Event Rate\n(averaged over {win})', 'Average Memory Usage (GiB)'),
    ('centraldb_ws',  'centraldb_ws_vs_rate.png',
     'Central-DB Working Set Memory vs File Activity Event Rate\n(averaged over {win})', 'Average Working Set (GiB)'),
    ('sensor_cpu',    'sensor_cpu_vs_rate.png',
     'Sensor CPU Usage vs File Activity Event Rate\n(averaged over {win})', 'Average CPU Usage (cores)'),
    ('sensor_mem',    'sensor_mem_vs_rate.png',
     'Sensor Memory Usage vs File Activity Event Rate\n(averaged over {win})', 'Average Memory Usage (GiB)'),
    ('sensor_ws',     'sensor_ws_vs_rate.png',
     'Sensor Working Set Memory vs File Activity Event Rate\n(averaged over {win})', 'Average Working Set (GiB)'),
    ('sensor_heap',   'sensor_heap_vs_rate.png',
     'Sensor Go Heap In-Use vs File Activity Event Rate\n(averaged over {win})', 'Average Go Heap In-Use (GiB)'),
    ('collector_cpu', 'collector_cpu_vs_rate.png',
     'Collector CPU Usage vs File Activity Event Rate\n(summed across pods, averaged over {win})',
     'Average CPU Usage (cores, all pods)'),
    ('collector_mem', 'collector_mem_vs_rate.png',
     'Collector Memory Usage vs File Activity Event Rate\n(summed across pods, averaged over {win})',
     'Average Memory Usage (GiB, all pods)'),
    ('fact_cpu',      'fact_cpu_vs_rate.png',
     'Fact CPU Usage vs Event Rate\n(summed across pods, averaged over {win})',
     'Average CPU Usage (cores, all pods)'),
    ('fact_mem',      'fact_mem_vs_rate.png',
     'Fact Memory Usage vs Event Rate\n(summed across pods, averaged over {win})',
     'Average Memory Usage (GiB, all pods)'),
    ('alerts_count',  'alerts_count_vs_rate.png',
     'Alerts Table Row Count vs File Activity Event Rate\n(maximum within {win})', 'Maximum Alert Count'),
    ('alerts_size',   'alerts_size_vs_rate.png',
     'Alerts Table Size vs File Activity Event Rate\n(maximum within {win})', 'Maximum Table Size (MiB)'),
    ('process_indicators_count', 'process_indicators_count_vs_rate.png',
     'Process Indicators Table Row Count vs File Activity Event Rate\n(maximum within {win})',
     'Maximum Row Count'),
    ('process_indicators_size',  'process_indicators_size_vs_rate.png',
     'Process Indicators Table Size vs File Activity Event Rate\n(maximum within {win})',
     'Maximum Table Size (MiB)'),
    ('network_flows_received', 'network_flows_received_vs_rate.png',
     'Sensor Network Flows Received Rate vs File Activity Event Rate\n(averaged over {win})',
     'Network Flows Received (flows/sec)'),
    ('process_indicators_received', 'process_indicators_received_vs_rate.png',
     'Sensor Process Indicators Received Rate vs File Activity Event Rate\n(averaged over {win})',
     'Process Indicators Received (/sec)'),
]


def _convert(v, conv):
    """Post-scale a value to GiB/MiB (binary, 1024-based), preserving None."""
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


def plot_scaling_comparison(run_dir, output_dir):
    """
    Generate policy-comparison plots across all rates for one run.

    Config-driven: reads <run_dir>/run-config.json for the rates (each with its
    actual event rate) and policies (one series per policy nickname), then reads the
    rate-<rate>__<policy> cell dirs. One line per policy is drawn on each
    metric-vs-rate plot. The x-axis is the actual event rate declared in the config.
    Args:
        run_dir:    a run's results dir (<resultsBaseDir>/<name>) holding run-config.json
        output_dir: output directory for plots
    """
    os.makedirs(output_dir, exist_ok=True)

    cfg = load_run_config(run_dir)

    # Collect every plot's trend-line equations so they can be written to a
    # single table (trendline_equations.csv) alongside the plots. record() is
    # called just before each savefig with that plot's name and equations list.
    equation_rows = []
    def record(plot_name, equations):
        for (series_name, slope, intercept, _color) in equations:
            equation_rows.append((plot_name, series_name, slope, intercept))

    START_OFFSET = 60.0   # Skip initial 60s ramp-up

    # Match the analysis window to the test length: [60s, 60s + run duration] from
    # the config's runTime (10m -> 60-660s, 20m -> 60-1260s).
    END_OFFSET = START_OFFSET + run_duration_from_config(cfg)
    TIME_WINDOW_DESC = f"{int(START_OFFSET)}-{int(END_OFFSET)}s"

    # Series = policies (one line each), drawn in config order; the first policy is
    # the reference for the overhead plot (baseline / nopolicy). X-axis = actual
    # event rate, sorted ascending.
    policies = [p['nickname'] for p in cfg['policies']]
    labels = {p: p for p in policies}
    event_rates = sorted(r['rate'] for r in cfg['rates'])

    if not event_rates or not policies:
        print(f"No rates/policies in {os.path.join(run_dir, 'run-config.json')}")
        return

    print(f"Run: {cfg.get('name')} ({cfg.get('loadType')}), deployments={cfg.get('numDeployments', 1)}")
    print(f"Event rates: {event_rates} events/sec")
    print(f"Series (policies): {policies}")

    # series[metric_key][policy] = [one value per event rate]
    series = {key: {p: [] for p in policies} for key, *_ in METRIC_SPECS}
    for rate in event_rates:
        for policy in policies:
            cell = cell_dir(run_dir, rate, policy)
            if not os.path.isdir(cell):
                cell = None  # missing cell -> None metrics (plotted as a gap)
            baseline_cache = {}  # component -> baseline, per cell
            for key, filename, component, reader, conv in METRIC_SPECS:
                v = _extract(cell, filename, component, reader,
                             baseline_cache, START_OFFSET, END_OFFSET)
                series[key][policy].append(_convert(v, conv))

    # Standard vs-rate plots: one line per policy.
    for key, png, title_tmpl, ylabel in STANDARD_PLOTS:
        _plot_series_vs_rate(output_dir, event_rates, png,
                             title_tmpl.format(win=TIME_WINDOW_DESC), ylabel,
                             series[key], policies, labels, record)

    # CPU overhead of each non-reference policy relative to the reference (first)
    # policy -- baseline for berserker, nopolicy for fake -- to isolate the policy's
    # cost. One line per non-ref policy; needs >= 2 policies to be meaningful.
    if len(policies) >= 2:
        ref = policies[0]
        ref_label = labels[ref]
        ref_series = series['central_cpu'][ref]
        plt.figure(figsize=(12, 7))
        equations = []
        for line_idx, policy in enumerate(policies[1:]):
            color, marker = series_style(line_idx)
            label = labels[policy]
            overhead = []
            for base_v, v in zip(ref_series, series['central_cpu'][policy]):
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
                       print_suffix=' (Central CPU increase vs the reference policy)')

    # File-access events received at sensor vs configured event rate. The dashed
    # line is the ideal (observed throughput == configured rate); a curve that
    # flattens below it means events are generated/delivered slower than requested.
    plt.figure(figsize=(12, 7))
    equations = []
    observed = []
    for i, policy in enumerate(policies):
        color, marker = series_style(i)
        label = labels[policy]
        y = series['events_received'][policy]
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
        print("Usage: python plot-batch-comparison.py <run_dir> [output_dir]")
        print("\n  <run_dir> is a run's results dir (holds run-config.json), e.g.")
        print("  perf/berserker-file-activity")
        print("\nExample:")
        print("  python plot-batch-comparison.py perf/berserker-file-activity")
        sys.exit(1)

    run_dir = sys.argv[1]
    output_dir = sys.argv[2] if len(sys.argv) > 2 else os.path.join(run_dir, "comparison_plots")

    plot_scaling_comparison(run_dir, output_dir)
