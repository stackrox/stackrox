# VM index-report rate limiter: completion-based vs time-based

E2E comparison of the time-based limiter (master) and the completion-based limiter
(this PR) under real VM enrichment load. Review-only; remove before merge.

## Setup

- OCP, 3×(7.5 CPU / 30 GiB) workers. Central + Scanner V4 only; real sensor scaled to 0.
- Controlled binary pair from the **same base commit**; only `pkg/rate` + the connection
  wiring differ. Deployed by overlaying `/stackrox/central`.
- Load: `tools/local-sensor` fake VM workload, 3000 unique VMs, **508 packages/report**
  (scale-test report size), ~150 reports/s offered (saturating).
- Metric: `rate(rox_central_rate_limiter_requests_total{outcome="accepted"})` (identical on both
  builds); `rate_limiter_in_flight_tokens` and `sensor_event_queue{Type="VirtualMachineIndexReport"}`
  (Add−Remove = backlog); `kubectl top` for CPU/RSS.

## Key fact

Time-based admits a **fixed rate (0.3/s default)** regardless of Scanner V4. Completion-based
admits **at Scanner V4's actual rate** (concurrency = `ROX_VM_INDEX_REPORT_BUCKET_CAPACITY`).
The scanner-v4-db CPU is the throughput bottleneck for VM reports.

## Upside — Scanner V4 has headroom (cap=30)

| Scanner V4 config | time-based | completion | gain |
|---|---|---|---|
| default (db 4 CPU; matcher 1 CPU, HPA 2–3) | 0.30 rep/s | 0.76 rep/s | **2.6×** |
| more resources (db 7 CPU; matcher 3×2 CPU) | 0.30 rep/s | 1.33 rep/s | **4.4×** |

Time-based leaves 60–77 % of Scanner V4 capacity unused. Completion tracks it with no tuning
(and was still cap-limited at boosted, so the real gain is larger). See `proof_upside.png`.

## Downside — Scanner V4 slower than the configured rate

Under-resourced Scanner (db 1 CPU, ceiling ~0.16/s), configured rate 1.0/s, central mem limit
1500 Mi, 15 min:

| | time-based (rate=1.0) | completion (cap=30) |
|---|---|---|
| admit rate | 1.0/s (Scanner-blind) | 0.16/s (= Scanner) |
| report backlog | 0 → 895, **unbounded** | flat ~18 |
| central RSS | 111 → 1391 Mi (pinned at limit, GC thrash) | 120 → 330 Mi (stable) |
| result | memory saturation → **OOM** | safe |

Time-based over-admits; the surplus (VM reports are not deduplicated) accumulates in central until
it OOMs — the failure the limiter is meant to prevent. Completion cannot over-admit. See
`proof_downside.png`.

## Conclusion

- Adaptive throughput: 2.6–4.4× more reports/s whenever Scanner V4 has headroom.
- Adaptive safety: time-based OOMs central when Scanner V4 is slow; completion degrades gracefully.
- Fewer knobs: the numeric rate no longer has to be hand-tuned to Scanner V4 capacity.
- `ROX_VM_INDEX_REPORT_BUCKET_CAPACITY` is now max in-flight concurrency; set it ≈ matcher
  parallelism (default), not 1 (cap=1 serializes and can be slower than time-based).
