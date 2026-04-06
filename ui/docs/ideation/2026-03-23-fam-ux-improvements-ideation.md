---
date: 2026-03-23
topic: fam-ux-improvements
focus: File Activity Monitoring UI/UX improvements
---

# Ideation: File Activity Monitoring UX/UI Improvements

## Codebase Context

The FAM feature monitors file system events (CREATE, UNLINK, RENAME, PERMISSION_CHANGE, OWNERSHIP_CHANGE, OPEN) in containers and on nodes. Gated by `ROX_SENSITIVE_FILE_ACTIVITY` feature flag.

**Current UI surfaces:**
- **Violation Details:** FAM violations render as individual expandable `FileAccessCard` components within `RuntimeMessages.tsx`. Each card shows file path, operation, timestamp, hostname, process context (name, executable, UID), and file metadata (owner, group, permissions). All cards expand by default.
- **Violations List:** No FAM-specific search attributes. The `CompoundSearchFilter` has entities for Cluster, Deployment, Namespace, Node, Policy, Policy violation, Resource -- nothing for file activity.
- **Policy Editor:** Two criteria under "File activity" category: File Path (text/glob) and File Operation (dropdown). Both have `canBooleanLogic: false`. Available for DEPLOYMENT_EVENT and NODE_EVENT sources, RUNTIME lifecycle only.

**Key findings:**
- `BaseAlert.fileAccessViolation` field exists in the type system but is never read or rendered by the UI
- Process violations use `TimestampedEventCard<T>` (generic, with grouping and first/last occurrence) while FAM violations get individual cards with no aggregation
- `ProcessIndicator` on each `FileAccess` event has 10+ fields but only 3 are rendered (name, execFilePath, uid)
- Violations capped at 40 per alert; all 40 render as expanded cards
- No past institutional learnings documented for FAM

## Ranked Ideas

### 1. Surface Full Process Context and Deep-Link to Process Discovery
**Description:** Render the full `ProcessIndicator` data already present on every `FileAccess` event -- lineage (parent process chain), args, containerId, podId -- and add a hyperlink to `/main/risk/{deploymentId}` for the originating process. Currently `FileAccessCardContent.tsx` renders only `process.signal.name`, `process.signal.execFilePath`, and `process.signal.uid`, dropping lineage, args, container context, and all navigational links.
**Rationale:** Process lineage is the most forensically valuable field for distinguishing legitimate file operations from attacks. The data is already on the component; it just needs to be rendered. Enables one-click navigation from "this file was modified" to "here is the full process tree that did it."
**Downsides:** Adds visual density to the card. Lineage data may be null for some events. Deep links only work for deployment-based alerts (not node alerts).
**Confidence:** 95%
**Complexity:** Low
**Status:** Unexplored

### 2. Group File Access Events Using TimestampedEventCard
**Description:** Refactor `RuntimeMessages.tsx` to group `FILE_ACCESS` violations by `fileAccess.file.actualPath` and render via the existing `TimestampedEventCard<T>` generic component (already used for process violations), showing first/last occurrence timestamps and event count per path instead of individual cards.
**Rationale:** The generic component already exists and accepts `events`, `getTimestamp`, `getEventKey`, and `ContentComponent`. Currently 40 individual expanded cards create a wall of noise. Grouping by path with counts matches how process violations are already displayed, creating consistency.
**Downsides:** Grouping by path may not be the right axis for all investigations (some may prefer grouping by operation or process). Need to handle edge cases where the same path has different operations.
**Confidence:** 85%
**Complexity:** Low-Medium
**Status:** Unexplored

### 3. Investigate and Render the Ignored fileAccessViolation Field
**Description:** `BaseAlert` defines `fileAccessViolation: FileAccessViolation | null` (alert.proto.ts line 93), mirroring how `processViolation` provides aggregated process data. The UI never reads this field -- `ViolationDetailsPage.tsx` passes `alert.processViolation` and `alert.violations` but never `alert.fileAccessViolation`. Investigate whether the backend populates it; if so, render it.
**Rationale:** This is potentially free aggregated FAM data being silently discarded. If populated, it could power grouping, counting, and summary views without additional API calls. The `processViolation` pattern shows exactly how to consume such a field.
**Downsides:** May be null/unpopulated in current backend. Investigation required before implementation.
**Confidence:** 70%
**Complexity:** Low (investigation) to Medium (rendering)
**Status:** Unexplored

### 4. Add FAM Search Filters to the Violations List
**Description:** Add File Path and File Operation as searchable attributes in `ViolationsTableSearchFilter.utils.ts` using the existing `CompoundSearchFilter` infrastructure. The UI pattern is well-established (see `CompoundSearchFilter/attributes/` directory), but backend search indexing for file access fields must be verified.
**Rationale:** The #1 triage question -- "show me all violations involving /etc/shadow" -- is currently impossible from the violations list. Analysts must click into each violation individually. This is the highest-friction gap for FAM usability at scale.
**Downsides:** Requires backend verification that the ALERTS search category indexes file access fields. Not purely a UI change.
**Confidence:** 75%
**Complexity:** Medium (UI + backend coordination)
**Status:** Unexplored

### 5. One-Click Policy Creation from File Access Violation
**Description:** Add a "Create Policy" action button on `FileAccessCard` that navigates to the policy wizard pre-filled with the event's file path, operation, and event source. The `CreatePolicyFromSearch` pattern exists on the Risk page, though it uses Redux and a backend generation endpoint that may need extension for FAM fields.
**Rationale:** Eliminates context-switching between violation investigation and policy authoring. The data needed to pre-fill (actualPath/effectivePath, operation, event source) is already available on every `FileAccess` event.
**Downsides:** Existing `CreatePolicyFromSearch` uses Redux (codebase moving away from Redux) and a backend `generatePolicyFromSearch` endpoint that may not support FAM fields. A cleaner approach may need design. Medium implementation effort.
**Confidence:** 70%
**Complexity:** Medium
**Status:** Unexplored

### 6. Enable Boolean Logic for File Path and File Operation Criteria
**Description:** Flip `canBooleanLogic` from `false` to `true` on File Path and File Operation descriptors in `policyCriteriaDescriptors.tsx` (4 descriptors total: 2 for deployment events, 2 for node events). This enables OR logic for multiple paths/operations in a single policy.
**Rationale:** Currently monitoring `/etc/passwd OR /etc/shadow` requires two separate policies. The boolean logic UI infrastructure already works for other criteria (Image Registry, Image Tag, etc.). Two-line UI change.
**Downsides:** Backend must support boolean logic for file criteria. Must verify before flipping the flag. If backend does not support it, policies would be expressible in the UI but fail at evaluation time.
**Confidence:** 65%
**Complexity:** Low (UI: 2-line change; needs backend verification)
**Status:** Unexplored

### 7. Collapse-All / Expand-All with Smart Defaults
**Description:** Lift card expansion state from individual `FileAccessCard` components to the parent, add bulk toggle controls, and default to collapsed when event count exceeds 5. Currently `FileAccessCard.tsx` uses `useState(true)` per card with no shared state.
**Rationale:** When a violation has 30+ file access events, all 40 fully expanded cards create a massive scroll. No way to get an overview without reading every card. Trivial to implement; should be done alongside idea #2.
**Downsides:** Minor convenience, not transformative on its own.
**Confidence:** 90%
**Complexity:** Low
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| 1 | Visual operation-type indicators (icons/colors) | Cosmetic polish with near-zero triage value; do incidentally, not as own work item |
| 2 | Before/after diff for permission changes | Real value but 95% backend/collector work; not a UI task |
| 3 | File activity timeline visualization | No existing timeline component to reuse (premise was false); 40-event cap undermines value |
| 4 | Pre-built policy templates | Content/docs work, poor effort-to-value ratio as a UI feature |
| 5 | File access baseline (observe-then-alert) | Multi-quarter product epic across collector, backend, and UI |
| 6 | Cross-signal correlation (FAM + network + process) | Requires backend correlation engine that does not exist |
| 7 | Process-centric investigation view | Do grouping-by-path (#2) first; 40-event cap limits usefulness |
| 8 | FAM violation type in violations list columns | Data not available on ListAlert; requires backend API change |
| 9 | Container escape detection via path divergence | High false-positive rate without backend escape-confidence scoring |
| 10 | Cross-node file activity correlation | No file events API exists; backend-first project |
| 11 | File Activity Explorer (dedicated page) | Correct product vision but requires new backend API, routes, services |
| 12 | Contextual path intelligence (known path labels) | Maintenance burden outweighs marginal value |
| 13 | Table view with virtualization | Virtualization absurd for 40-item cap; subsumed by idea #2 |
| 14 | Directory-tree aggregation | Over-engineered for 40-event max dataset |
| 15 | Regex in policy criteria | Glob covers realistic cases; regex adds ReDoS risk |
| 16 | Compound FAM policy criteria (process+file) | Inter-field ANDing already works within policy sections |
| 17 | Investigation actions toolbar (combination) | Combines three nonexistent features; build constituents first |
| 18 | Smart grouping + timeline (combination) | Too much complexity bundled; execute #2 and #7 separately |
| 19 | Precision policy authoring (combination) | Unbundle it; execute #6 now, defer templates |

## Session Log
- 2026-03-23: Initial ideation -- 48 raw ideas generated across 6 divergent agents, merged/deduped to 28 unique candidates + 3 cross-cutting syntheses, 7 survived adversarial filtering by 2 independent critique agents
