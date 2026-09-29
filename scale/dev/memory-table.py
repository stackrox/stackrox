#!/usr/bin/env python3
"""
Print average memory per component for each workload in a results directory,
across the three berserker test modes (baseline / alerting / idlepolicy), for two
memory metrics side by side:

  - usage:  container_memory_usage_bytes (cgroup total; includes reclaimable
            page cache and freed-but-not-returned pages)
  - wset:   container_memory_working_set_bytes (usage minus reclaimable inactive
            cache; the OOM/limit-relevant "real" memory)

When wset tracks usage, the memory is genuinely in use (not reclaimable cache).
Values are averaged over the same window the plots use (60s .. 60s + test
length). Components/runs missing a series show as n/a.

Usage:
    python3 memory-table.py <results_base_dir> [--deployments N] [rate1 rate2 ...]

    results_base_dir: e.g. perf_berserker/20m
    --deployments N:  berserker deployment count to report (default: 1). Selects
                      runs tagged _<N>dep; N=1 also matches legacy untagged runs.
    rates:            configured event rates to report (default: 100 500 1000 2500 5000)
"""
import importlib.util
import os
import sys

START_OFFSET = 60.0
DEFAULT_RATES = [100, 500, 1000, 2500, 5000]
COMPONENTS = ['central', 'central-db', 'sensor', 'collector', 'fact']
# Berserker test modes (see plot_utils.BERSERKER_MODES): (dir suffix, short label).
# Each table cell reports one value per mode in this order.
MODES = [('_baseline', 'base'), ('_alerting', 'alert'), ('_idlepolicy', 'idle')]


def load_plot_helpers():
    """Import helpers from plot-batch-comparison.py (hyphenated, so use importlib)."""
    here = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(here, 'plot-batch-comparison.py')
    spec = importlib.util.spec_from_file_location('plot_batch_comparison', path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def avg_gb(mod, run_dir, component, metric):
    """Average of a memory series (GB) over the test-length window, or None."""
    if run_dir is None:
        return None
    # Window matched to the test length parsed from the dir name (see trend_window).
    _, end = mod.trend_window(run_dir, start=START_OFFSET)
    baseline = mod.determine_baseline_timestamp(run_dir, component)
    v = mod.read_metric_average(
        os.path.join(run_dir, f'metrics_{component}_{metric}.txt'),
        START_OFFSET, end, baseline)
    return None if v is None else v / 1e9


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    mod = load_plot_helpers()
    argv, num_deployments = mod.parse_deployments(sys.argv[1:])
    base = argv[0].rstrip('/')
    rates = [int(a) for a in argv[1:]] or DEFAULT_RATES

    def find_run_dir(rate, mode_suffix):
        return mod.find_berserker_run_dir(base, rate, mode_suffix, num_deployments)

    def fmt(v):
        return 'n/a' if v is None else f"{v:.2f}"

    order = "/".join(label for _, label in MODES)
    print(f"Results dir: {base}")
    print(f"Average GB over the test-length window. Cells are {order}.")
    print("usage = container_memory_usage_bytes; wset = container_memory_working_set_bytes.\n")

    col_w = 22
    for metric, header in [('mem', 'Usage (GB)'), ('mem_workingset', 'Working Set (GB)')]:
        print(f"=== {header} ===")
        print(f"{'rate':>6} " + " ".join(f"{c:>{col_w}}" for c in COMPONENTS))
        for r in rates:
            run_dirs = [find_run_dir(r, suffix) for suffix, _ in MODES]
            cells = []
            for c in COMPONENTS:
                cell = "/".join(fmt(avg_gb(mod, rd, c, metric)) for rd in run_dirs)
                cells.append(f"{cell:>{col_w}}")
            print(f"{r:>6} " + " ".join(cells))
        print()


if __name__ == "__main__":
    main()
