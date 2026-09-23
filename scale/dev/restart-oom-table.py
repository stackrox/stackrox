#!/usr/bin/env python3
"""
Print the number of container restarts and OOM-kills per component for each
workload in a results directory, without vs with policy.

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

    def find_run_dir(rate, policy):
        # Resolve the run dir for a rate/policy without hardcoding num_sensors or
        # run_time; the trailing _policy_<policy> keeps the rate match exact.
        matches = sorted(glob.glob(
            f"{base}/berserker_file_activity_results_*_file-activity-{rate}_policy_{policy}"))
        return matches[0] if matches else None

    def cell(v_without, v_with):
        def fmt(v):
            return 'n/a' if v is None else str(int(v))
        return f"{fmt(v_without)}/{fmt(v_with)}"

    print(f"Results dir: {base}")
    print("Each cell is without-policy/with-policy (total over the run).\n")

    col_w = 13
    for kind, header in KINDS:
        print(f"=== {header} ===")
        heading = f"{'rate':>6} " + " ".join(f"{c:>{col_w}}" for c in COMPONENTS)
        print(heading)
        for r in rates:
            wd = find_run_dir(r, 'false')
            wpd = find_run_dir(r, 'true')
            cells = []
            for c in COMPONENTS:
                cells.append(f"{cell(count(mod, wd, c, kind), count(mod, wpd, c, kind)):>{col_w}}")
            print(f"{r:>6} " + " ".join(cells))
        print()


if __name__ == "__main__":
    main()
