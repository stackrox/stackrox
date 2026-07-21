#!/usr/bin/env python3
"""Raw/processed (no narrative) metrics dump for the VSOCK pull-mode
VM-scraping load test (ROX_VM_VSOCK_LOADTEST_HAMMER_MODE=true).

Scope: the roxagent <-> Sensor link only (dial/read/pull latency, per-VM
drop/failure rate, Sensor-side pull throughput). Central is out of scope --
this test doesn't need Central reachable at all.

Reads Sensor's own Prometheus metrics (sensor/common/virtualmachine/metrics)
straight from its plain HTTP metrics endpoint (ROX_METRICS_PORT, default
:9090 -- unauthenticated, unlike the mTLS-protected :9091 "secure metrics"
port).

Two modes, one query each way:
- Default (no --window): a SINGLE snapshot, reporting on the whole run since
  Sensor last started (using the rox_sensor_uptime_seconds gauge as the
  elapsed-time denominator). Counters/histograms are cumulative since start
  and reset on restart, so one query is enough to summarize an entire
  multi-hour/overnight run. A pod-restart check (via kubectl) warns if the
  numbers don't actually cover as long a period as you might expect.
- `--window N`: two snapshots N seconds apart, reporting on just that recent
  delta -- use for a "what's happening right now" live reading instead of
  the whole-run summary.

Output is a flat key: value list, one metric per line -- rates/percentiles/
percentages are computed (that's the "processed" part), but there is no
prose interpretation, verdict, or narrative. Do that downstream of this
script's output, not here.

Two things every consumer of this output needs to know (kept here, not
repeated in the output itself):
1. `<Xms` means the true percentile landed in the histogram's first bucket
   -- it's somewhere below X, not precisely known at this resolution.
2. Without the known_epoch fix (see
   docs/superpowers/prompts/2026-07-08-single-roundtrip-epoch-check.md),
   every scrapeVM() call makes TWO real dial+read round trips to the same
   VM: a generation-based "unchanged?" check (always negative here, since
   the fake report's generation never advances) immediately followed by a
   mandatory full re-fetch (real, always-on ROX-35597 epoch-mismatch
   dedup, firing every time because the fake report's epoch is randomized
   on every response). With the fix applied, roxagent folds the epoch
   check into the first response and this second round trip disappears
   (roundtrip_2_uninstrumented_avg_ms drops to ~0). Sensor's dial/read
   histograms only instrument the FIRST round trip; a second one, when it
   happens, only shows up inside `total_duration`
   (`roundtrip_2_uninstrumented_avg_ms` below is that gap, back-computed
   as total_sum - dial_sum - read_sum; there is no Central call anywhere
   on this path).

Same input always produces the same numbers -- no LLM involved.

Usage:
    # Whole run since Sensor started, one query. Use this for an overnight run.
    python3 scripts/vsock-loadtest-report.py

    # Recent 30s window instead (two snapshots, 30s apart).
    python3 scripts/vsock-loadtest-report.py --window 30

    # Metrics endpoint already reachable at this URL (skip port-forward).
    python3 scripts/vsock-loadtest-report.py --url http://localhost:9090/metrics
"""

import argparse
import re
import socket
import subprocess
import sys
import time
import urllib.request
from collections import defaultdict

METRIC_LINE_RE = re.compile(r'^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{([^}]*)\})?\s+(\S+)$')
LABEL_RE = re.compile(r'(\w+)="((?:[^"\\]|\\.)*)"')


def parse_metrics_text(text):
    """Returns {metric_name: [(labels_dict, value), ...]}."""
    metrics = defaultdict(list)
    for line in text.splitlines():
        if not line or line.startswith("#"):
            continue
        m = METRIC_LINE_RE.match(line)
        if not m:
            continue
        name, _, labels_str, value_str = m.groups()
        labels = dict(LABEL_RE.findall(labels_str)) if labels_str else {}
        try:
            value = float(value_str)
        except ValueError:
            continue
        metrics[name].append((labels, value))
    return metrics


def fetch_metrics(url, timeout=10):
    with urllib.request.urlopen(url, timeout=timeout) as resp:
        return parse_metrics_text(resp.read().decode())


def sum_metric(metrics, name, label_filter=None):
    total = 0.0
    for labels, value in metrics.get(name, []):
        if label_filter and not all(labels.get(k) == v for k, v in label_filter.items()):
            continue
        total += value
    return total


def by_label(metrics, name, label_key):
    """Returns {label_value: summed_value} for one label key on one metric."""
    out = defaultdict(float)
    for labels, value in metrics.get(name, []):
        out[labels.get(label_key, "")] += value
    return dict(out)


def histogram_buckets(metrics, base_name, label_filter=None):
    """Returns sorted [(le_float, cumulative_count), ...] for a histogram,
    summed across any labels not in label_filter (e.g. summing all
    'outcome' values together)."""
    buckets = defaultdict(float)
    for labels, value in metrics.get(base_name + "_bucket", []):
        if label_filter and not all(labels.get(k) == v for k, v in label_filter.items()):
            continue
        le = labels.get("le")
        if le is None:
            continue
        le_f = float("inf") if le == "+Inf" else float(le)
        buckets[le_f] += value
    return sorted(buckets.items())


def delta_buckets(before, after):
    before_map = dict(before)
    after_map = dict(after)
    keys = sorted(set(before_map) | set(after_map))
    return [(le, after_map.get(le, 0.0) - before_map.get(le, 0.0)) for le in keys]


def histogram_quantile(buckets, q):
    """Same linear-interpolation approach as Prometheus's histogram_quantile.
    Returns (estimate, lower_bound) where lower_bound is the bucket boundary
    below the one the estimate was interpolated into -- 0.0 means "landed in
    the first bucket", a sign the bucket boundaries are too coarse to say
    anything more precise than 'somewhere below the first boundary'."""
    if not buckets:
        return None, None
    total = buckets[-1][1]
    if total <= 0:
        return None, None
    target = q * total
    prev_le, prev_count = 0.0, 0.0
    for le, count in buckets:
        if count >= target:
            if le == float("inf"):
                return prev_le, prev_le
            if count == prev_count:
                continue
            frac = (target - prev_count) / (count - prev_count)
            return prev_le + frac * (le - prev_le), prev_le
        prev_le, prev_count = le, count
    return buckets[-1][0], prev_le


def fmt_ms(estimate_and_lower):
    seconds, lower_bound = estimate_and_lower
    if seconds is None:
        return "n/a"
    label = f"{seconds * 1000:.1f}ms"
    return f"<{label}" if lower_bound == 0.0 else label


def wait_for_port(host, port, timeout=15):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            with socket.create_connection((host, port), timeout=1):
                return True
        except OSError:
            time.sleep(0.3)
    return False


def start_port_forward(namespace, deployment, local_port, remote_port):
    proc = subprocess.Popen(
        ["kubectl", "port-forward", "-n", namespace, f"deployment/{deployment}",
         f"{local_port}:{remote_port}"],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    if not wait_for_port("127.0.0.1", local_port):
        proc.terminate()
        raise RuntimeError(f"port-forward to {deployment}:{remote_port} did not come up in time")
    return proc


def stop_port_forward(proc):
    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()


def pod_restart_count(namespace, deployment):
    """Best-effort restart count for the deployment's pod(s), via kubectl. A
    restart resets Sensor's counters/histograms, so a whole-run report taken
    after one silently only covers time since the restart, not the full run.
    Returns None if it can't be determined (never fails the report over this)."""
    try:
        out = subprocess.run(
            ["kubectl", "get", "pods", "-n", namespace, "-l", f"app={deployment}",
             "-o", "jsonpath={range .items[*]}{.status.containerStatuses[0].restartCount}{\"\\n\"}{end}"],
            capture_output=True, text=True, timeout=10, check=True,
        )
        counts = [int(c) for c in out.stdout.split()]
        return max(counts) if counts else None
    except Exception:
        return None


def format_duration(seconds):
    h, rem = divmod(int(seconds), 3600)
    m, s = divmod(rem, 60)
    return f"{h}h{m:02d}m{s:02d}s"


# Prometheus-style "# HELP <name> <text>" line emitted right before each
# metric's value line. Written so nobody needs to know this test's design to
# read a report: each text names the two concrete events a duration measures
# the gap between, or exactly what a count/rate is counting, in plain words.
HELP = {
    "pod_restarts": (
        "How many times the Sensor process has restarted since this window began. "
        "A restart resets every counter and timer below to zero, so if this is more "
        "than 0, the numbers below may not cover the full period you think they do."
    ),
    "vms_in_last_cycle": (
        "How many VMs were running and got contacted the last time Sensor finished "
        "one full round of asking every VM for its report."
    ),
    "cycles": (
        "How many times Sensor has finished one full round of asking every running VM "
        "for its report (one 'cycle' = start asking VM 1, ..., finish asking the last "
        "VM). Shown as a total count for this window, and as cycles finished per second."
    ),
    "pull_attempts": (
        "How many times Sensor tried to fetch a report from one VM (each running VM is "
        "asked once per cycle, so this is roughly cycles x vms_in_last_cycle). Shown as "
        "a total count for this window, and as attempts per second."
    ),
    "pull_success": (
        "Of the attempts above, how many got back a usable report instead of an error. "
        "Shown as a count, the percentage of all attempts that succeeded, and successes "
        "per second."
    ),
    "report_payload_throughput_mb_s": (
        "How many megabytes of report data Sensor received from VMs per second, "
        "averaged over this window."
    ),
    "reports_sent_to_central": (
        "How many virtual-machine index reports Sensor has handed off toward Central "
        "(i.e. queued them onto its outgoing connection to Central) in this window, and "
        "how many per second. 'of which queued_ok' is the subset that were successfully "
        "queued (the rest failed before being queued, e.g. because Central wasn't "
        "reachable yet). Central itself is out of scope for this report -- this says "
        "nothing about whether Central actually received, processed, or acknowledged "
        "these reports, only that Sensor attempted to send them."
    ),
    "latency_dial_1st_roundtrip_ms": (
        "How long it takes from the moment Sensor starts opening a network connection "
        "to a VM until that connection is ready to use. Every successful pull actually "
        "opens two connections back-to-back to the same VM (see "
        "roundtrip_2_uninstrumented_avg_ms below for why) -- this line only measures the "
        "FIRST of the two. p50/p90/p99 = the time under which 50%/90%/99% of these "
        "connection-openings finished."
    ),
    "latency_read_1st_roundtrip_ms": (
        "How long it takes, after the connection above is open, until Sensor has fully "
        "received the VM's reply on it. Also only the FIRST of the two connections per "
        "pull (see roundtrip_2_uninstrumented_avg_ms below). p50/p90/p99 as above."
    ),
    "latency_total_both_roundtrips_ms": (
        "How long it takes from the moment Sensor starts trying to fetch a report from "
        "a VM until it has that report fully in hand -- covering BOTH connections it "
        "makes to that VM (see roundtrip_2_uninstrumented_avg_ms below). p50/p90/p99 as "
        "above."
    ),
    "latency_cycle_all_vms_ms": (
        "How long one full cycle takes: from the moment Sensor starts asking the first "
        "VM in that cycle until it has finished asking every VM in that cycle. "
        "p50/p90/p99 as above."
    ),
    "roundtrip_1_avg_ms": (
        "The average time for the FIRST connection Sensor makes per pull, start to "
        "finish (opening the connection plus receiving its reply). This first "
        "connection always asks the VM 'has your report changed since I last checked?' "
        "-- in this test the fake VM always answers 'yes', which is why there's a "
        "second connection (see roundtrip_2_uninstrumented_avg_ms below). Also shown as "
        "a percentage of latency_total_both_roundtrips_ms."
    ),
    "roundtrip_2_uninstrumented_avg_ms": (
        "The average time for the SECOND connection Sensor makes per pull. Because "
        "every 'has it changed?' answer in this test is 'yes', Sensor immediately opens "
        "a second connection to actually fetch the full report. There is no separate "
        "stopwatch for this second connection today, so this number is calculated "
        "indirectly, as (the total time for both connections) minus (the first "
        "connection's measured time). Also shown as a percentage of "
        "latency_total_both_roundtrips_ms."
    ),
    "latency_sensor_processing_ms": (
        "How long Sensor spends, after it has fully received a report from a VM, "
        "checking that report and handing it off internally to be sent onward -- this "
        "does NOT include any time spent talking to the VM or to Central. p50/p90/p99 "
        "as above."
    ),
    "enqueue_channel_full_events": (
        "How many times Sensor tried to hand a freshly-received report to its internal "
        "outgoing queue and found that queue already full -- a sign Sensor is receiving "
        "reports from VMs faster than it can pass them along internally. Shown as a "
        "count for this window, and as events per second."
    ),
    "blocking_enqueue_wait_ms": (
        "When the queue above was full, how long Sensor had to wait before there was "
        "room to add the report. p50/p99 as above. 'none observed' means the queue was "
        "never full in this window."
    ),
    "failure_status": (
        "For every pull attempt that did NOT succeed, why it failed (e.g. could not "
        "connect, timed out) and how many times each reason occurred, as a percentage "
        "of all attempts. 'none' means every attempt in this window succeeded."
    ),
}


def emit(out, key, value):
    """Appends a blank line, then a Prometheus-style '# HELP <key> <text>'
    line, then the '<key>: <value>' line -- every metric is self-explanatory
    on its own, without needing to know this test's design."""
    out.append("")
    out.append(f"# HELP {key} {HELP[key]}")
    out.append(f"{key}: {value}")


def build_report(before, after, window_s, restarts=None, whole_run=False):
    out = []
    label = f"whole_run={format_duration(window_s)}" if whole_run else f"window={window_s:.1f}s"
    out.append(f"# vsock-loadtest-report {label}")
    emit(out, "pod_restarts", restarts if restarts is not None else "unknown")

    d_cycles = sum_metric(after, "rox_sensor_vsock_pull_cycles_total") - sum_metric(before, "rox_sensor_vsock_pull_cycles_total")
    vms_in_cycle = sum_metric(after, "rox_sensor_vsock_pull_vms_in_cycle")

    req_status_before = by_label(before, "rox_sensor_vsock_pull_requests_total", "status")
    req_status_after = by_label(after, "rox_sensor_vsock_pull_requests_total", "status")
    all_statuses = set(req_status_before) | set(req_status_after)
    req_deltas = {s: req_status_after.get(s, 0.0) - req_status_before.get(s, 0.0) for s in all_statuses}
    total_requests = sum(req_deltas.values())
    success_requests = req_deltas.get("success", 0.0)
    success_pct = 100 * success_requests / total_requests if total_requests else 0.0

    d_bytes = sum_metric(after, "rox_sensor_vsock_pull_report_bytes_sum") - sum_metric(before, "rox_sensor_vsock_pull_report_bytes_sum")

    sent_status_before = by_label(before, "rox_sensor_virtual_machine_index_reports_sent_total", "status")
    sent_status_after = by_label(after, "rox_sensor_virtual_machine_index_reports_sent_total", "status")
    sent_statuses = set(sent_status_before) | set(sent_status_after)
    sent_deltas = {s: sent_status_after.get(s, 0.0) - sent_status_before.get(s, 0.0) for s in sent_statuses}
    sent_total = sum(sent_deltas.values())
    sent_ok = sent_deltas.get("success", 0.0)

    emit(out, "vms_in_last_cycle", f"{vms_in_cycle:.0f}")
    emit(out, "cycles", f"{d_cycles:.0f} ({d_cycles / window_s:.3f}/s)" if window_s > 0 else f"{d_cycles:.0f}")
    emit(out, "pull_attempts", f"{total_requests:.0f} ({total_requests / window_s:.1f}/s)" if window_s > 0 else f"{total_requests:.0f}")
    emit(out, "pull_success", f"{success_requests:.0f} ({success_pct:.2f}%, {success_requests / window_s:.1f}/s)" if window_s > 0 else f"{success_requests:.0f} ({success_pct:.2f}%)")
    if window_s > 0:
        emit(out, "report_payload_throughput_mb_s", f"{d_bytes / window_s / 1024 / 1024:.2f}")
    if sent_total > 0:
        if window_s > 0:
            emit(out, "reports_sent_to_central", f"{sent_total:.0f} ({sent_total / window_s:.1f}/s), of which queued_ok={sent_ok:.0f} ({sent_ok / window_s:.1f}/s)")
        else:
            emit(out, "reports_sent_to_central", f"{sent_total:.0f}, of which queued_ok={sent_ok:.0f}")
        sent_non_ok = {s: v for s, v in sent_deltas.items() if s != "success" and v > 0}
        for status, count in sorted(sent_non_ok.items(), key=lambda kv: -kv[1]):
            out.append(f"reports_sent_to_central[{status}]: {count:.0f} ({100 * count / sent_total:.3f}%)")

    for key, metric in [
        ("latency_dial_1st_roundtrip_ms", "rox_sensor_vsock_pull_dial_duration_seconds"),
        ("latency_read_1st_roundtrip_ms", "rox_sensor_vsock_pull_read_duration_seconds"),
        ("latency_total_both_roundtrips_ms", "rox_sensor_vsock_pull_total_duration_seconds"),
        ("latency_cycle_all_vms_ms", "rox_sensor_vsock_pull_cycle_duration_seconds"),
    ]:
        b = delta_buckets(histogram_buckets(before, metric), histogram_buckets(after, metric))
        p50, p90, p99 = (histogram_quantile(b, q) for q in (0.5, 0.9, 0.99))
        emit(out, key, f"p50={fmt_ms(p50)} p90={fmt_ms(p90)} p99={fmt_ms(p99)}")

    dial_sum = sum_metric(after, "rox_sensor_vsock_pull_dial_duration_seconds_sum") - sum_metric(before, "rox_sensor_vsock_pull_dial_duration_seconds_sum")
    read_sum = sum_metric(after, "rox_sensor_vsock_pull_read_duration_seconds_sum") - sum_metric(before, "rox_sensor_vsock_pull_read_duration_seconds_sum")
    dial_count = sum_metric(after, "rox_sensor_vsock_pull_dial_duration_seconds_count") - sum_metric(before, "rox_sensor_vsock_pull_dial_duration_seconds_count")
    total_sum = sum_metric(after, "rox_sensor_vsock_pull_total_duration_seconds_sum") - sum_metric(before, "rox_sensor_vsock_pull_total_duration_seconds_sum")
    second_trip_sum = max(total_sum - dial_sum - read_sum, 0.0)
    if dial_count > 0 and total_sum > 0:
        emit(out, "roundtrip_1_avg_ms", f"{1000 * (dial_sum + read_sum) / dial_count:.1f} ({100 * (dial_sum + read_sum) / total_sum:.1f}% of total)")
        emit(out, "roundtrip_2_uninstrumented_avg_ms", f"{1000 * second_trip_sum / dial_count:.1f} ({100 * second_trip_sum / total_sum:.1f}% of total)")

    proc_b = delta_buckets(
        histogram_buckets(before, "rox_sensor_virtual_machine_index_report_processing_duration_milliseconds"),
        histogram_buckets(after, "rox_sensor_virtual_machine_index_report_processing_duration_milliseconds"),
    )
    if proc_b and proc_b[-1][1] > 0:
        # Buckets are already in milliseconds for this one, not seconds.
        (p50, p50_lb), (p90, p90_lb), (p99, p99_lb) = (histogram_quantile(proc_b, q) for q in (0.5, 0.9, 0.99))
        fmt = lambda v, lb: f"<{v:.2f}ms" if lb == 0.0 else f"{v:.2f}ms"
        if p50 is not None:
            emit(out, "latency_sensor_processing_ms", f"p50={fmt(p50, p50_lb)} p90={fmt(p90, p90_lb)} p99={fmt(p99, p99_lb)}")

    blocked_before = sum_metric(before, "rox_sensor_virtual_machine_index_report_enqueue_blocked_total")
    blocked_after = sum_metric(after, "rox_sensor_virtual_machine_index_report_enqueue_blocked_total")
    d_blocked = blocked_after - blocked_before
    emit(out, "enqueue_channel_full_events", f"{d_blocked:.0f} ({d_blocked / window_s:.2f}/s)" if window_s > 0 else f"{d_blocked:.0f}")

    enqueue_b = delta_buckets(
        histogram_buckets(before, "rox_sensor_virtual_machine_index_report_blocking_enqueue_duration_milliseconds"),
        histogram_buckets(after, "rox_sensor_virtual_machine_index_report_blocking_enqueue_duration_milliseconds"),
    )
    if enqueue_b and enqueue_b[-1][1] > 0:
        (p50, _), (p99, _) = (histogram_quantile(enqueue_b, q) for q in (0.5, 0.99))
        emit(out, "blocking_enqueue_wait_ms", f"p50={p50:.1f} p99={p99:.1f} (n={enqueue_b[-1][1]:.0f})")
    else:
        emit(out, "blocking_enqueue_wait_ms", "none observed")

    non_success = {s: v for s, v in req_deltas.items() if s != "success" and v > 0}
    if non_success:
        out.append("")
        out.append(f"# HELP failure_status {HELP['failure_status']}")
        for status, count in sorted(non_success.items(), key=lambda kv: -kv[1]):
            out.append(f"failure_status[{status}]: {count:.0f} ({100 * count / total_requests:.3f}%)")
    else:
        emit(out, "failure_status", "none")

    return "\n".join(l for l in out if l is not None)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--namespace", default="stackrox")
    parser.add_argument("--deployment", default="sensor")
    parser.add_argument("--local-port", type=int, default=9090)
    parser.add_argument("--remote-port", type=int, default=9090)
    parser.add_argument("--window", type=float, default=None,
                        help="if set, take two snapshots this many seconds apart and report on that "
                             "recent window instead of the default whole-run-since-Sensor-start report")
    parser.add_argument("--url", default=None, help="metrics URL to use directly, skipping port-forward management")
    args = parser.parse_args()

    pf_proc = None
    url = args.url
    try:
        if not url:
            if not wait_for_port("127.0.0.1", args.local_port, timeout=0.5):
                print(f"Starting kubectl port-forward to {args.deployment}:{args.remote_port}...", file=sys.stderr)
                pf_proc = start_port_forward(args.namespace, args.deployment, args.local_port, args.remote_port)
            url = f"http://127.0.0.1:{args.local_port}/metrics"

        restarts = pod_restart_count(args.namespace, args.deployment)

        if args.window:
            print(f"Snapshot 1/2 from {url} ...", file=sys.stderr)
            before = fetch_metrics(url)
            t0 = time.monotonic()
            time.sleep(args.window)
            print("Snapshot 2/2 ...", file=sys.stderr)
            after = fetch_metrics(url)
            elapsed = time.monotonic() - t0
            print(build_report(before, after, elapsed, restarts=restarts, whole_run=False))
        else:
            print(f"Single snapshot from {url} (whole-run report, one query) ...", file=sys.stderr)
            after = fetch_metrics(url)
            uptime = sum_metric(after, "rox_sensor_uptime_seconds")
            if uptime <= 0:
                start_time = sum_metric(after, "process_start_time_seconds")
                uptime = time.time() - start_time if start_time else 0.0
            print(build_report({}, after, uptime, restarts=restarts, whole_run=True))
    finally:
        if pf_proc:
            stop_port_forward(pf_proc)


if __name__ == "__main__":
    main()
