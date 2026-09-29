#!/usr/bin/env python3
"""
Print the number of container restarts and OOM-kills per component for each
workload in a results directory, across the three berserker test modes
(baseline / alerting / idlepolicy).

Restarts (kube_pod_container_status_restarts_total) and OOM-kills
(container_oom_events_total) are cumulative counters, so the maximum over the
run is the total count. Values come from the same metric files the plots use
(see prometheus-query-file-activity.sh); components/runs missing the series
(older runs, or a metric source not scraped in that cluster) show as n/a.

Usage:
    python3 restart-oom-table.py <results_base_dir> [rate1 rate2 ...]

    results_base_dir: e.g. perf_berserker/20m
    rates:            configured event rates to report (default: 100 500 1000 2500 5000)
"""
import glob
import importlib.util
import os
import sys

DEFAULT_RATES = [100, 500, 1000, 2500, 5000]
# Berserker test modes (see plot_utils.BERSERKER_MODES): (dir suffix, short label).
# Each table cell reports one value per mode in this order.
MODES = [('_baseline', 'base'), ('_alerting', 'alert'), ('_idlepolicy', 'idle')]
# Every component we scrape restart/OOM counts for (see the scraper).
COMPONENTS = ['central', 'central-db', 'sensor', 'collector', 'fact', 'berserker']
# (suffix, header) for each counter we report.
KINDS = [('restarts', 'Restarts'), ('ooms', 'OOM-Kills')]


def load_plot_helpers():
    """Import helpers from plot-batch-comparison.py (hyphenated, so use importlib)."""
    here = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(here, 'plot-batch-comparison.py')
    spec = importlib.util.spec_from_file_location('plot_batch_comparison', path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def count(mod, run_dir, component, kind):
    """Total count (max of the cumulative counter) for a component, or None."""
    if run_dir is None:
        return None
    path = os.path.join(run_dir, f'metrics_{component}_{kind}.txt')
    # start_offset=0: these are lifetime counters, so include every point; the
    # max is the final total.
    return mod.read_metric_max(path, 0.0, None, None)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    base = sys.argv[1].rstrip('/')
    rates = [int(a) for a in sys.argv[2:]] or DEFAULT_RATES
    mod = load_plot_helpers()

    def find_run_dir(rate, mode_suffix):
        # Resolve the run dir for a rate/mode without hardcoding num_sensors or
        # run_time; the trailing mode suffix keeps the rate match exact.
        matches = sorted(glob.glob(
            f"{base}/berserker_file_activity_results_*_file-activity-{rate}{mode_suffix}"))
        return matches[0] if matches else None

    def cell(values):
        def fmt(v):
            return 'n/a' if v is None else str(int(v))
        return "/".join(fmt(v) for v in values)

    order = "/".join(label for _, label in MODES)
    print(f"Results dir: {base}")
    print(f"Each cell is {order} (total over the run).\n")

    col_w = 18
    for kind, header in KINDS:
        print(f"=== {header} ===")
        heading = f"{'rate':>6} " + " ".join(f"{c:>{col_w}}" for c in COMPONENTS)
        print(heading)
        for r in rates:
            run_dirs = [find_run_dir(r, suffix) for suffix, _ in MODES]
            cells = []
            for c in COMPONENTS:
                values = [count(mod, rd, c, kind) for rd in run_dirs]
                cells.append(f"{cell(values):>{col_w}}")
            print(f"{r:>6} " + " ".join(cells))
        print()


if __name__ == "__main__":
    main()
