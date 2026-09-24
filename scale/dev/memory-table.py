#!/usr/bin/env python3
"""
Print average memory per component for each workload in a results directory,
without vs with policy, for two memory metrics side by side:

  - usage:  container_memory_usage_bytes (cgroup total; includes reclaimable
            page cache and freed-but-not-returned pages)
  - wset:   container_memory_working_set_bytes (usage minus reclaimable inactive
            cache; the OOM/limit-relevant "real" memory)

When wset tracks usage, the memory is genuinely in use (not reclaimable cache).
Values are averaged over the same window the plots use (60s .. 60s + test
length). Components/runs missing a series show as n/a.

Usage:
    python3 memory-table.py <results_base_dir> [rate1 rate2 ...]

    results_base_dir: e.g. perf_berserker/20m
    rates:            configured event rates to report (default: 100 500 1000 2500 5000)
"""
import glob
import importlib.util
import os
import sys

START_OFFSET = 60.0
DEFAULT_RATES = [100, 500, 1000, 2500, 5000]
COMPONENTS = ['central', 'central-db', 'sensor', 'collector', 'fact']


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

    base = sys.argv[1].rstrip('/')
    rates = [int(a) for a in sys.argv[2:]] or DEFAULT_RATES
    mod = load_plot_helpers()

    def find_run_dir(rate, policy):
        matches = sorted(glob.glob(
            f"{base}/berserker_file_activity_results_*_file-activity-{rate}_policy_{policy}"))
        return matches[0] if matches else None

    def fmt(v):
        return 'n/a' if v is None else f"{v:.2f}"

    print(f"Results dir: {base}")
    print("Average GB over the test-length window. Cells are without/with policy.")
    print("usage = container_memory_usage_bytes; wset = container_memory_working_set_bytes.\n")

    col_w = 15
    for metric, header in [('mem', 'Usage (GB)'), ('mem_workingset', 'Working Set (GB)')]:
        print(f"=== {header} ===")
        print(f"{'rate':>6} " + " ".join(f"{c:>{col_w}}" for c in COMPONENTS))
        for r in rates:
            wd = find_run_dir(r, 'false')
            wpd = find_run_dir(r, 'true')
            cells = []
            for c in COMPONENTS:
                cell = f"{fmt(avg_gb(mod, wd, c, metric))}/{fmt(avg_gb(mod, wpd, c, metric))}"
                cells.append(f"{cell:>{col_w}}")
            print(f"{r:>6} " + " ".join(cells))
        print()


if __name__ == "__main__":
    main()
