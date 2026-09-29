#!/usr/bin/env python3
"""
Print file-activity event drops per pipeline stage for each workload in a results
directory, across the three berserker test modes (baseline / alerting / idlepolicy).

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
    python3 drops-table.py <results_base_dir> [--deployments N] [rate1 rate2 ...]

    results_base_dir: e.g. perf_berserker/20m
    --deployments N:  berserker deployment count to report (default: 1). Selects
                      runs tagged _<N>dep; N=1 also matches legacy untagged runs.
    rates:            configured event rates to report (default: 100 500 1000 2500 5000)
"""
import importlib.util
import os
import sys

DEFAULT_RATES = [100, 500, 1000, 2500, 5000]
# Berserker test modes (see plot_utils.BERSERKER_MODES): (dir suffix, short label).
# Each table cell reports one value per mode in this order.
MODES = [('_baseline', 'base'), ('_alerting', 'alert'), ('_idlepolicy', 'idle')]
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

    mod = load_plot_helpers()
    argv, num_deployments = mod.parse_deployments(sys.argv[1:])
    base = argv[0].rstrip('/')
    rates = [int(a) for a in argv[1:]] or DEFAULT_RATES

    def find_run_dir(rate, mode_suffix):
        return mod.find_berserker_run_dir(base, rate, mode_suffix, num_deployments)

    def cell(values):
        def fmt(v):
            return 'n/a' if v is None else str(int(v))
        return "/".join(fmt(v) for v in values)

    order = "/".join(label for _, label in MODES)
    print(f"Results dir: {base}")
    print(f"Event drops per stage (total over the run). Each cell is {order}.\n")

    col_w = 22
    print(f"{'rate':>6} " + " ".join(f"{h:>{col_w}}" for h, _ in STAGES))
    for r in rates:
        run_dirs = [find_run_dir(r, suffix) for suffix, _ in MODES]
        cells = []
        for _, suffix in STAGES:
            values = [count(mod, rd, suffix) for rd in run_dirs]
            cells.append(f"{cell(values):>{col_w}}")
        print(f"{r:>6} " + " ".join(cells))


if __name__ == "__main__":
    main()
