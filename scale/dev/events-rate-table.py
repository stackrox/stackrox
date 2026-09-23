#!/usr/bin/env python3
"""
Print the observed file-access event rate at the sensor versus the configured
berserker event rate, for each workload in a results directory.

This is the tabular companion to the events_received_vs_rate.png plot produced
by plot-batch-comparison.py: it uses the exact same helpers (read_counter_rate,
with a read-only fallback to the diagnostic bundles when the Prometheus time
series is missing), so the numbers match the plot.

Usage:
    python3 events-rate-table.py <results_base_dir> [rate1 rate2 ...]

    results_base_dir: e.g. perf_berserker_cpu_2
    rates:            configured event rates to report (default: 100 500 1000 2500 5000)

The observed value is averaged over the same window used by the plots, matched
to the test length (10m -> 60-660s, 20m -> 60-1260s).
"""
import glob
import importlib.util
import os
import sys

# Window start matches plot-batch-comparison.py (skip ramp-up); the end is
# derived per run from the test length (see observed_rate / trend_window).
START_OFFSET = 60.0
DEFAULT_RATES = [100, 500, 1000, 2500, 5000]
EVENTS_FILE = 'metrics_rox_sensor_file_access_events_received_total.txt'


def load_plot_helpers():
    """Import helpers from plot-batch-comparison.py (hyphenated, so use importlib)."""
    here = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(here, 'plot-batch-comparison.py')
    spec = importlib.util.spec_from_file_location('plot_batch_comparison', path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def observed_rate(mod, run_dir):
    """Events/sec at the sensor for one run, with the same fallback as the plot."""
    # Match the window to the test length parsed from run_dir (10m -> 60-660s,
    # 20m -> 60-1260s), same as the plots.
    start, end = mod.trend_window(run_dir, start=START_OFFSET)
    rate = mod.read_counter_rate(
        os.path.join(run_dir, EVENTS_FILE),
        start, end,
        mod.determine_baseline_timestamp(run_dir, 'sensor'))
    if rate is None:
        rate = mod.read_events_rate_from_bundles(run_dir)
    return rate


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    base = sys.argv[1].rstrip('/')
    rates = [int(a) for a in sys.argv[2:]] or DEFAULT_RATES
    mod = load_plot_helpers()

    def fmt(v):
        return f"{v:9.1f}" if v is not None else f"{'n/a':>9}"

    def pct(v, r):
        return f"{100 * v / r:7.0f}%" if v is not None else f"{'n/a':>8}"

    def find_run_dir(rate, policy):
        # Resolve the run dir for a rate/policy without hardcoding num_sensors or
        # run_time; the trailing _policy_<policy> keeps the rate match exact
        # (file-activity-100 vs file-activity-1000 don't collide).
        matches = sorted(glob.glob(
            f"{base}/berserker_file_activity_results_*_file-activity-{rate}_policy_{policy}"))
        return matches[0] if matches else None

    print(f"Results dir: {base}")
    print(f"{'rate':>6} {'without':>9} {'with':>9} {'w %ideal':>8} {'wp %ideal':>9}")
    for r in rates:
        wd = find_run_dir(r, 'false')
        wpd = find_run_dir(r, 'true')
        v = observed_rate(mod, wd) if wd else None
        vp = observed_rate(mod, wpd) if wpd else None
        print(f"{r:>6} {fmt(v)} {fmt(vp)} {pct(v, r)} {pct(vp, r)}")


if __name__ == "__main__":
    main()
