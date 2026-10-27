#!/usr/bin/env python3
"""
Print file-activity event drops per pipeline stage for each rate in a run, across the
run's policies (one value per policy per cell).

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

Config-driven: reads <run_dir>/run-config.json for the rates and policies, and the
rate-<rate>__<policy> cell dirs (written by run-perf-tests.sh).

Usage:
    python3 drops-table.py <run_dir> [rate1 rate2 ...]

    run_dir: a run's results dir holding run-config.json (e.g. perf/berserker-file-activity)
    rates:   event rates to report (default: all rates in the config)
"""
import os
import sys

from plot_utils import cell_dir, table_run_args, read_metric_max

# (column header, metric-file suffix). The suffix matches metrics_<suffix>.txt as
# written by prometheus-query-file-activity.sh.
STAGES = [
    ('fact_bpf', 'fact_bpf_dropped'),
    ('fact_ratelim', 'fact_rate_limiter_dropped'),
    ('fact_grpc', 'fact_grpc_dropped'),
    ('sensor_buffer', 'rox_sensor_file_activity_buffer_drops'),
    ('sensor_detector', 'rox_sensor_detector_file_access_queue_dropped_total'),
]


def count(run_dir, suffix):
    """Total drops (max of the cumulative counter) for a stage, or None."""
    if run_dir is None:
        return None
    path = os.path.join(run_dir, f'metrics_{suffix}.txt')
    # start_offset=0: lifetime counters, so include every point; max is the total.
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
    print(f"Event drops per stage (total over the run). Each cell is {order}.\n")

    col_w = 22
    print(f"{'rate':>6} " + " ".join(f"{h:>{col_w}}" for h, _ in STAGES))
    for r in rates:
        run_dirs = [cell_dir_for(r, p) for p in policies]
        cells = []
        for _, suffix in STAGES:
            values = [count(rd, suffix) for rd in run_dirs]
            cells.append(f"{cell(values):>{col_w}}")
        print(f"{r:>6} " + " ".join(cells))


if __name__ == "__main__":
    main()
