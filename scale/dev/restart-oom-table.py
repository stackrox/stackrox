#!/usr/bin/env python3
"""
Print the number of container restarts and OOM-kills per component for each rate in a
run, across the run's policies (one value per policy per cell).

Restarts (kube_pod_container_status_restarts_total) and OOM-kills
(container_oom_events_total) are cumulative counters, so the maximum over the run is
the total count. Values come from the same metric files the plots use (see
prometheus-query-file-activity.sh); components/cells missing the series (older runs,
or a metric source not scraped in that cluster) show as n/a.

Config-driven: reads <run_dir>/run-config.json for the rates and policies, and the
rate-<rate>__<policy> cell dirs (written by run-perf-tests.sh).

Usage:
    python3 restart-oom-table.py <run_dir> [rate1 rate2 ...]

    run_dir: a run's results dir holding run-config.json (e.g. perf/berserker-file-activity)
    rates:   event rates to report (default: all rates in the config)
"""
import os
import sys

from plot_utils import cell_dir, table_run_args, read_metric_max

# Every component we scrape restart/OOM counts for (see the scraper).
COMPONENTS = ['central', 'central-db', 'sensor', 'collector', 'fact', 'berserker']
# (suffix, header) for each counter we report.
KINDS = [('restarts', 'Restarts'), ('ooms', 'OOM-Kills')]


def count(run_dir, component, kind):
    """Total count (max of the cumulative counter) for a component, or None."""
    if run_dir is None:
        return None
    path = os.path.join(run_dir, f'metrics_{component}_{kind}.txt')
    # start_offset=0: these are lifetime counters, so include every point; the max
    # is the final total.
    return read_metric_max(path, 0.0, None, None)


def main():
    run_dir, _cfg, rates, policies = table_run_args(sys.argv[1:], __doc__)

    def cell_dir_for(rate, policy):
        d = cell_dir(run_dir, rate, policy)
        return d if os.path.isdir(d) else None

    def cell(values):
        def fmt(v):
            return 'n/a' if v is None else str(int(v))
        return "/".join(fmt(v) for v in values)

    order = "/".join(policies)
    print(f"Run: {run_dir}")
    print(f"Each cell is {order} (total over the run).\n")

    col_w = 18
    for kind, header in KINDS:
        print(f"=== {header} ===")
        print(f"{'rate':>6} " + " ".join(f"{c:>{col_w}}" for c in COMPONENTS))
        for r in rates:
            run_dirs = [cell_dir_for(r, p) for p in policies]
            cells = []
            for c in COMPONENTS:
                values = [count(rd, c, kind) for rd in run_dirs]
                cells.append(f"{cell(values):>{col_w}}")
            print(f"{r:>6} " + " ".join(cells))
        print()


if __name__ == "__main__":
    main()
