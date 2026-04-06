---
date: 2026-03-23
topic: file-activity-pattern-load-estimation
focus: Wildcard pattern load estimation for anticipated alert volume in file activity monitoring
---

# Ideation: File Activity Pattern Load Estimation

## Codebase Context

### How File Activity Monitoring Works

- **Policy Wizard**: Users define file path patterns (glob syntax) + optional file operations (OPEN, CREATE, RENAME, DELETE, PERMISSION_CHANGE, OWNERSHIP_CHANGE) as runtime policy criteria
- **Glob Engine**: Go's `bmatcuk/doublestar/v4` library handles `*`, `**`, `?` patterns in `pkg/search/predicate/basematchers/base_matchers.go`
- **Detection Pipeline**: Every file access event flows through sensor's `processFileAccess()` loop (`sensor/common/detector/detector.go`), which evaluates it against ALL enabled file activity policies via `policySet.ForEach` -- no pre-filtering
- **Event Buffer**: Sensor uses a 20,000-item queue (`DetectorFileAccessBufferSize`) that drops events when full, tracked by `DetectorFileAccessDroppedCount` Prometheus counter
- **Validation**: UI validates absolute path + no traversal (`validateFilePath` in `policyCriteriaDescriptors.tsx`); backend validates syntax via `globstar.ValidatePattern`. Neither assesses pattern breadth
- **Dry-Run Gap**: The existing dry-run system (`predicateBasedDryRunPolicy` at `service_impl.go:444`) explicitly short-circuits for runtime policies: "Runtime violations are not available in this preview because they are generated in response to future events"
- **Feature Flag**: `ROX_SENSITIVE_FILE_ACTIVITY` gates the entire feature

### Key Architectural Constraints

- Sensor runs on every node -- performance changes have cluster-wide impact
- No existing telemetry for per-policy match rates (only aggregate queue metrics)
- File path cardinality is orders of magnitude higher than process name cardinality (rules out naive baseline approaches)
- The feature is behind a flag and not yet GA -- heavyweight infrastructure investments are premature

### Past Learnings

No prior institutional knowledge exists for this feature area.

## Ranked Ideas

### 1. Live Per-Policy Match Rate Telemetry

**Description:** Add a `policy_id`-labeled Prometheus counter inside sensor's `processFileAccess()` loop that increments on each successful glob match. Expose per-policy match rates (events/minute) via a new Central API endpoint that aggregates sensor metrics. Surface this data in the UI: a match rate column/sparkline in the policy list view, and a time-series chart on the policy detail page.

**Rationale:** This gives users the one number they actually need -- real match rate from real data in their real environment. The Prometheus infrastructure is 80% built: `DetectorFileAccessQueueOperations` and `DetectorFileAccessDroppedCount` already exist in `sensor/common/detector/metrics/metrics.go`. Adding a per-policy counter inside the detection loop follows the same pattern used by `processedNodeScan` metrics. Every other idea in this ideation attempts to approximate or predict this number -- this idea just measures it.

**Downsides:** Only provides data after a policy has been running (no pre-deployment estimation). Per-policy label cardinality in Prometheus can be problematic if users create many policies. Surfacing the metric in the UI requires a new API endpoint from Central that aggregates across sensors.

**Confidence:** 85%

**Complexity:** Medium

**Status:** Unexplored

---

### 2. Shadow Mode Policies

**Description:** Introduce a third policy state ("Shadow") alongside Enabled and Disabled. In shadow mode, the sensor runs pattern matching against file access events normally but suppresses alert creation -- only match-rate metrics are recorded and surfaced. Users would create a policy in shadow mode, observe the match rate via telemetry (idea #1), then promote to enabled when satisfied with the volume. The UI would show a "Shadow mode results" panel on the policy page with match rate over time, top matched paths, and top matching deployments.

**Rationale:** Directly fills the gap left by the dry-run system's explicit exclusion of runtime policies (`service_impl.go:444`). This is the most complete answer to "how do I safely onboard a new file activity policy." The concept is well-established in security tooling (ModSecurity detection-only mode, AWS GuardDuty findings vs. auto-remediation). The existing policy model has a `disabled` boolean that could be extended to an enum (enabled/disabled/shadow).

**Downsides:** Requires changes across the entire stack: policy storage proto (new state), sensor detection pipeline (detect but suppress alerts), central alert service (aggregate shadow metrics), and UI (new state, new metrics display). A multi-sprint epic. Shadow mode policies still consume sensor CPU and queue capacity -- they protect from alert noise but not from matching cost. Depends on idea #1 for the metrics to be meaningful.

**Confidence:** 70%

**Complexity:** High

**Status:** Unexplored

---

### 3. Simple Pattern Guardrails

**Description:** Extend the existing `validateFilePath` function in `policyCriteriaDescriptors.tsx` to produce warning messages (not blocking errors) for patterns that are structurally very broad. Specifically: patterns containing `/**` at the root level, patterns that are just `/**`, or patterns where `**` appears as the first path segment after `/`. Display warnings using PatternFly's HelperText warning variant. Example: the pattern `/**` would show "This pattern matches every file on the system and will generate extreme alert volume. Consider narrowing to a specific directory."

**Rationale:** Pure frontend change, zero backend risk, minimal implementation effort. Not an estimation tool -- a safety net against the most catastrophic mistakes. The current `validateFilePath` function accepts `/**` as perfectly valid. While a heuristic breadth "score" is disconnected from reality (the critic correctly identified this), a simple check for known-dangerous patterns is grounded and actionable. This is analogous to a compiler warning, not a code metric.

**Downsides:** Only catches the most extreme cases. Cannot distinguish between `/etc/**` (moderate, well-scoped) and `/tmp/**` (high volume in practice). Not a substitute for real telemetry. Requires a small extension to the HelperText rendering in `PolicyCriteriaFieldInput.tsx` to support warning state (currently only supports error state from the validate function).

**Confidence:** 75%

**Complexity:** Low

**Status:** Unexplored

---

### 4. Lightweight Recent-Match Analysis

**Description:** When a user is authoring or editing a file activity policy, query Central's existing alert/violation storage to show aggregate statistics from policies with similar file path patterns. Display contextual information like "Policies monitoring paths under /etc generated ~X alerts/day in the last 7 days across Y deployments." Uses data Central already stores (alerts contain violation messages with matched file paths) rather than requiring new sensor-side event replay infrastructure.

**Rationale:** Provides empirical guidance at authoring time without the architectural cost of a full simulation endpoint. Central already has alert data queryable by policy. The approach is conservative -- it uses existing data rather than building new collection infrastructure for a feature behind a flag. Even approximate guidance ("other /etc policies are noisy") is more useful than zero guidance.

**Downsides:** Only useful if similar policies have already been deployed (chicken-and-egg for first-time users). "Similar pattern" matching is imprecise -- literal prefix comparison is the only tractable approach. Alert storage may not retain enough granularity to reconstruct per-path match distributions. Lower confidence than direct telemetry (idea #1).

**Confidence:** 55%

**Complexity:** Medium

**Status:** Unexplored

---

### 5. Simple Cross-Policy Overlap Notice

**Description:** When creating or editing a file activity policy, display an informational notice showing how many other file activity policies are currently enabled. Example: "You have 4 other file activity policies enabled. Each file access event is evaluated against every policy independently -- overlapping patterns will generate duplicate alerts." Include a link to a filtered view of the policy list showing only file activity policies for easy review.

**Rationale:** Since every file access event is checked against every enabled policy with no deduplication, overlapping patterns create a multiplicative alert volume problem that is invisible in the current UI. This notice costs almost nothing to implement (query the policy list, filter by category, count) and raises awareness of a system-level concern that no single-policy analysis can reveal.

**Downsides:** Does not compute actual pattern overlap (computationally hard for arbitrary globs). Only provides awareness, not actionable specifics. Low impact for users with few file activity policies. May feel like noise to experienced operators.

**Confidence:** 50%

**Complexity:** Low

**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| 1 | Client-side breadth scoring with visual gauge | Heuristic score from pattern structure is disconnected from real match behavior; `/etc/**` and `/tmp/**` score identically but have wildly different volumes |
| 2 | Full backend pattern simulation/probe endpoint | Architecturally brutal -- requires sensor to buffer raw paths, expose new API, central to aggregate across sensors; disproportionate for a flagged feature |
| 3 | Interactive filesystem tree visualizer | Synthetic trees are fiction (containers vary wildly); real trees require heavy infra; PatternFly lacks a suitable tree component |
| 4 | Events-per-minute projection with frequency distribution | Multiplies two unknowns (pattern breadth x operation frequency) and presents the result with false precision |
| 5 | General cross-policy glob overlap detection | Glob overlap is undecidable in the general case; conservative approximation yields constant false positives; low real-world frequency |
| 6 | Adaptive rate limiting / per-policy throttling | Sampling under rate limits means silently missing real security events -- dangerous for a security product; adds lock contention in sensor hot path |
| 7 | Filesystem baseline with deviation alerting | File path cardinality is orders of magnitude higher than process names; baselines would explode sensor memory; high false positive rate from normal filesystem churn |
| 8 | Pattern expansion sandbox with synthetic filesystem | JS glob matching (micromatch) has subtle behavioral differences from Go doublestar; synthetic tree teaches nothing about actual environment |
| 9 | Community pattern library with noise ratings | Premature for a flagged feature with no established user base; noise ratings are environment-dependent; "community" patterns would be internally authored |
| 10 | Bloom filter pre-screening | Premature optimization; doublestar already short-circuits on prefix mismatch; the broadest patterns (where performance matters most) have the shortest prefixes, making Bloom filters ineffective |
| 11 | Side-by-side pattern comparison tool | Extremely niche workflow; comparing two heuristic scores side by side doubles the meaninglessness |
| 12 | Automatic pattern refinement suggestions | Requires AI-hard intent inference (is `/etc/**` intentionally broad or a mistake?); only helps after the alert flood has already occurred |
| 13 | Comparative pattern diff for edits | Marginal UX convenience built on unreliable heuristics; unnecessary if per-policy telemetry (idea #1) exists |

## Session Log

- 2026-03-23: Initial ideation -- 40 candidates generated across 5 sub-agents, deduplicated to 18 unique ideas, 3 cross-cutting combinations identified, adversarial filtering with second-pass orchestrator review yielded 5 survivors
