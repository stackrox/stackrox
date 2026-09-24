#!/usr/bin/env python3
"""
Print file-activity event drops per pipeline stage for each workload in a results
directory, without vs with policy.

Events flow: kernel -> fact (bpf ring buffer -> rate limiter -> gRPC output) ->
sensor (detector queue / activity buffer) -> central. A drop anywhere means the
observed event rate is below the configured rate. This table surfaces where the
loss happens. Stages:

  fact_bpf        stackrox_fact_bpf_events_total{label=Dropped}         (kernel ring buffer -> fact userspace)
  fact_ratelim    stackrox_fact_rate_limiter_events_total{label=Dropped} (shed by FACT_RATE_LIMIT)
  fact_grpc       stackrox_fact_output_grpc_events_total{label=Dropped}  (couldn't send to sensor)
  sensor_buffer   rox_sensor_file_activity_buffer_drops                  (sensor file-activity buffer)
  sensor_detector rox_sensor_detector_file_access_queue_dropped_total    (sensor detector queue full)

All are cumulative counters, so the maximum over the run is the total. Values come
from the same metric files the plots use (see prometheus-query-file-activity.sh);
stages missing a series (older runs, fact monitoring not exposed, or a metric not
scraped in that cluster) show as n/a.

Usage:
    python3 drops-table.py <results_base_dir> [rate1 rate2 ...]

    results_base_dir: e.g. perf_berserker/20m
    rates:            configured event rates to report (default: 100 500 1000 2500 5000)
"""
import glob
import importlib.util
import os
import sys

DEFAULT_RATES = [100, 500, 1000, 2500, 5000]
# (column header, metric-file suffix). The suffix matches metrics_<suffix>.txt as
# written by prometheus-query-file-activity.sh.
STAGES = [
    ('fact_bpf', 'fact_bpf_dropped'),
    ('fact_ratelim', 'fact_rate_limiter_dropped'),
    ('fact_grpc', 'fact_grpc_dropped'),
    ('sensor_buffer', 'rox_sensor_file_activity_buffer_drops'),
    ('sensor_detector', 'rox_sensor_detector_file_access_queue_dropped_total'),
]


def load_plot_helpers():
    """Import helpers from plot-batch-comparison.py (hyphenated, so use importlib)."""
    here = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(here, 'plot-batch-comparison.py')
    spec = importlib.util.spec_from_file_location('plot_batch_comparison', path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def count(mod, run_dir, suffix):
    """Total drops (max of the cumulative counter) for a stage, or None."""
    if run_dir is None:
        return None
    path = os.path.join(run_dir, f'metrics_{suffix}.txt')
    # start_offset=0: lifetime counters, so include every point; max is the total.
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
    print("Event drops per stage (total over the run). Each cell is without/with policy.\n")

    col_w = 17
    print(f"{'rate':>6} " + " ".join(f"{h:>{col_w}}" for h, _ in STAGES))
    for r in rates:
        wd = find_run_dir(r, 'false')
        wpd = find_run_dir(r, 'true')
        cells = []
        for _, suffix in STAGES:
            cells.append(f"{cell(count(mod, wd, suffix), count(mod, wpd, suffix)):>{col_w}}")
        print(f"{r:>6} " + " ".join(cells))


if __name__ == "__main__":
    main()
