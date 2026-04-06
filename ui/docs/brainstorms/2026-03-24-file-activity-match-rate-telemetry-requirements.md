---
date: 2026-03-24
topic: file-activity-match-rate-telemetry
---

# File Activity Per-Policy Match Rate Telemetry

## Problem Frame

Users creating file activity monitoring policies have no way to assess how noisy a policy is after deployment. The existing dry-run system explicitly short-circuits for runtime policies ("Runtime violations are not available in this preview because they are generated in response to future events"). The only existing metrics are aggregate queue-level counters (`DetectorFileAccessQueueOperations`, `DetectorFileAccessDroppedCount`) -- nothing per-policy. Users discover noise problems only through alert floods, with no data to guide tuning.

## Requirements

- R1. **Per-policy match counter in sensor**: Add a `policy_id`-labeled Prometheus counter inside the sensor's `processFileAccess()` detection loop that increments on each successful glob match. Follow the existing pattern used by `DetectorFileAccessQueueOperations` in `sensor/common/detector/metrics/metrics.go`.

- R2. **Cardinality cap**: Track match rates for a maximum of 50 file activity policies per sensor. If more policies exist, track only the most recently enabled ones. Drop tracking for policies that exceed the cap.

- R3. **Central aggregation API**: New Central API endpoint that queries per-policy match rate metrics from sensors, aggregates across all sensors in the cluster, and returns events-per-minute rates per policy.

- R4. **Policy detail page display**: Show the per-policy match rate on the policy detail page as a summary stat (current events/minute) with a sparkline showing the trend over the last 24 hours.

- R5. **Feature flag gating**: Gate all telemetry behavior (sensor counter, Central API, UI display) behind the existing `ROX_SENSITIVE_FILE_ACTIVITY` feature flag. No separate flag.

- R6. **File activity policies only**: Telemetry applies only to policies with file activity criteria (file path patterns + file operations). Other runtime policy types are unaffected.

## Success Criteria

- A user can view any enabled file activity policy's detail page and see its current match rate and 24h trend without leaving the page
- Match rate data reflects real sensor observations, not heuristic estimates
- The feature imposes negligible overhead on the sensor detection loop (counter increment is O(1))
- Cardinality is bounded regardless of how many file activity policies a user creates

## Scope Boundaries

- **Not pre-deployment estimation**: This feature only provides data after a policy has been running. Pre-deployment guidance (pattern guardrails, shadow mode) is out of scope.
- **No alerting on match rates**: The UI displays match rate data for user inspection. Automated alerts or thresholds based on match rate are out of scope.
- **No cross-policy comparison view**: Match rate is shown per-policy on the detail page only. A list-level column, dedicated monitoring dashboard, or cross-policy comparison is out of scope.
- **No historical storage**: Match rate data comes from live Prometheus counters with standard retention. Long-term historical storage of match rates is out of scope.

## Key Decisions

- **Metric format**: Events/minute rolling average -- intuitive for assessing noisiness, directly comparable across policies
- **UI placement**: Policy detail page only -- the natural place to investigate a specific policy's behavior, avoids list-level complexity
- **Visualization**: Summary stat + sparkline -- compact, informative, low UI footprint. Full time-series chart is overkill for v1
- **Feature flag**: Reuse `ROX_SENSITIVE_FILE_ACTIVITY` -- telemetry only makes sense when file activity monitoring is enabled
- **Cardinality management**: Cap at 50 tracked policies per sensor -- pragmatic bound that covers realistic usage
- **Data path**: Central aggregates sensor Prometheus metrics via existing scrape infrastructure, serves to UI through new API endpoint

## Dependencies / Assumptions

- Sensors already expose a `/metrics` endpoint that Central can scrape -- this is the existing pattern for `DetectorFileAccessQueueOperations`
- The `processFileAccess()` loop already has access to the policy ID after a successful match (via `CompiledPolicy`)
- PatternFly has sparkline-capable chart components (or the project already uses a charting library)

## Outstanding Questions

### Deferred to Planning

- [Affects R1][Technical] What is the exact insertion point for the counter increment in the detection loop? Need to trace from `processFileAccess()` through `policySet.ForEach` to the match result.
- [Affects R2][Technical] How should the cardinality cap be implemented? Options include a fixed-size LRU of policy IDs, or checking the count before registering a new label value.
- [Affects R3][Needs research] What is the existing pattern for Central-to-sensor metric aggregation? Is there precedent for Central querying sensor Prometheus endpoints, or does this need a new gRPC method?
- [Affects R3][Technical] What time windows should the API support for computing the rolling average? The UI needs "current rate" and "last 24h for sparkline" -- determine whether to push windowing to Central or the UI.
- [Affects R4][Needs research] Which charting library is currently used in the UI for sparklines or inline charts? Confirm PatternFly charts or an existing alternative.
- [Affects R4][Technical] Where exactly on the policy detail page should the match rate stat be placed? Need to review the current page layout.

## Next Steps

-> `/ce:plan` for structured implementation planning
