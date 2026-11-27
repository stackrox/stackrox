#!/usr/bin/env python3
"""
Print average memory per component for each rate in a run, across the run's policies
(one value per policy per cell), for two memory metrics side by side:

  - usage:  container_memory_usage_bytes (cgroup total; includes reclaimable
            page cache and freed-but-not-returned pages)
  - wset:   container_memory_working_set_bytes (usage minus reclaimable inactive
            cache; the OOM/limit-relevant "real" memory)

When wset tracks usage, the memory is genuinely in use (not reclaimable cache).
Values are averaged over the same window the plots use (60s .. 60s + test length,
from the run config). Components/cells missing a series show as n/a.

Config-driven: reads <run_dir>/run-config.json for the rates and policies, and the
rate-<rate>__<policy> cell dirs (written by run-perf-tests.sh).

Usage:
    python3 memory-table.py <run_dir> [rate1 rate2 ...]

    run_dir: a run's results dir holding run-config.json (e.g. perf/berserker-file-activity)
    rates:   event rates to report (default: all rates in the config)
"""
import os
import sys

from plot_utils import (
    cell_dir,
    determine_baseline_timestamp,
    analysis_window_from_config,
    table_run_args,
    read_metric_average,
)

START_OFFSET = 60.0
COMPONENTS = ['central', 'central-db', 'sensor', 'collector', 'fact']


def avg_gib(run_dir, component, metric, end):
    """Average of a memory series (GiB) over the analysis window, or None."""
    if run_dir is None:
        return None
    baseline = determine_baseline_timestamp(run_dir, component)
    v = read_metric_average(
        os.path.join(run_dir, f'metrics_{component}_{metric}.txt'),
        START_OFFSET, end, baseline)
    # Binary GiB (1024**3), matching plot-batch-comparison.py's memory axes.
    return None if v is None else v / (1024 ** 3)


def main():
    run_dir, cfg, rates, policies = table_run_args(sys.argv[1:], __doc__)
    end = analysis_window_from_config(cfg, START_OFFSET)[1]

    def cell_dir_for(rate, policy):
        d = cell_dir(run_dir, rate, policy)
        return d if os.path.isdir(d) else None

    def fmt(v):
        return 'n/a' if v is None else f"{v:.2f}"

    order = "/".join(policies)
    print(f"Run: {run_dir}")
    print(f"Average GiB over the test-length window. Cells are {order}.")
    print("usage = container_memory_usage_bytes; wset = container_memory_working_set_bytes.\n")

    col_w = 22
    for metric, header in [('mem', 'Usage (GiB)'), ('mem_workingset', 'Working Set (GiB)')]:
        print(f"=== {header} ===")
        print(f"{'rate':>6} " + " ".join(f"{c:>{col_w}}" for c in COMPONENTS))
        for r in rates:
            run_dirs = [cell_dir_for(r, p) for p in policies]
            cells = []
            for c in COMPONENTS:
                cell = "/".join(fmt(avg_gib(rd, c, metric, end)) for rd in run_dirs)
                cells.append(f"{cell:>{col_w}}")
            print(f"{r:>6} " + " ".join(cells))
        print()


if __name__ == "__main__":
    main()
