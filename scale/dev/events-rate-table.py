#!/usr/bin/env python3
"""
Print the observed file-access event rate at the sensor versus the configured event
rate, for each rate in a run, across the run's policies.

This is the tabular companion to the events_received_vs_rate.png plot produced by
plot-batch-comparison.py: it uses the exact same helpers (read_counter_rate, with a
read-only fallback to the diagnostic bundles when the Prometheus time series is
missing), so the numbers match the plot.

Config-driven: reads <run_dir>/run-config.json for the rates and policies, and the
rate-<rate>__<policy> cell dirs (written by run-perf-tests.sh).

Usage:
    python3 events-rate-table.py <run_dir> [rate1 rate2 ...]

    run_dir: a run's results dir holding run-config.json (e.g. perf/berserker-file-activity)
    rates:   event rates to report (default: all rates in the config)

The observed value is averaged over the same window used by the plots
(60s .. 60s + test length, from the run config).
"""
import os
import sys

from plot_utils import (
    cell_dir,
    determine_baseline_timestamp,
    analysis_window_from_config,
    table_run_args,
    read_counter_rate,
    read_events_rate_from_bundles,
)

START_OFFSET = 60.0
EVENTS_FILE = 'metrics_rox_sensor_file_access_events_received_total.txt'


def observed_rate(run_dir, end):
    """Events/sec at the sensor for one cell, with the same fallback as the plot."""
    if run_dir is None:
        return None
    rate = read_counter_rate(
        os.path.join(run_dir, EVENTS_FILE),
        START_OFFSET, end,
        determine_baseline_timestamp(run_dir, 'sensor'))
    if rate is None:
        rate = read_events_rate_from_bundles(run_dir)
    return rate


def main():
    run_dir, cfg, rates, policies = table_run_args(sys.argv[1:], __doc__)
    end = analysis_window_from_config(cfg, START_OFFSET)[1]

    def cell_dir_for(rate, policy):
        d = cell_dir(run_dir, rate, policy)
        return d if os.path.isdir(d) else None

    def fmt(v):
        return f"{v:9.1f}" if v is not None else f"{'n/a':>9}"

    def pct(v, r):
        return f"{100 * v / r:7.0f}%" if v is not None else f"{'n/a':>8}"

    print(f"Run: {run_dir}")
    # Observed events/sec at the sensor per policy, then that as a % of the ideal
    # (configured) rate per policy.
    obs_hdr = " ".join(f"{p:>9}" for p in policies)
    pct_hdr = " ".join(f"{p + '%':>8}" for p in policies)
    print(f"{'rate':>6} {obs_hdr} {pct_hdr}")
    for r in rates:
        observed = [observed_rate(cell_dir_for(r, p), end) for p in policies]
        obs_cells = " ".join(fmt(v) for v in observed)
        pct_cells = " ".join(pct(v, r) for v in observed)
        print(f"{r:>6} {obs_cells} {pct_cells}")


if __name__ == "__main__":
    main()
