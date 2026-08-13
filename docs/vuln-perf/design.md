# Vulnerability Page Performance: Design & Decision Log

## Problem Statement

The Workload CVE overview page (`/main/vulnerabilities/workload-cves`) is unusably
slow at scale. On a cluster with ~4000 images and ~16,000 unique CVEs (1.6M rows in
`image_cves_v2`, 2.8GB), the page takes 20-30+ seconds to load.

### Root Causes

1. **Full-table aggregation for sort**: The two-phase pagination query's Phase 1 must
   `GROUP BY cve` across all 1.6M rows and compute filtered `COUNT(DISTINCT image)`
   aggregates per severity level to determine sort order. The severity-count sort
   requires 10 `COUNT(DISTINCT image) FILTER (WHERE severity = X)` sub-selects.

2. **`jsonb_agg` in Phase 1**: The Phase 1 query collects all row IDs via
   `jsonb_agg(distinct(image_cves_v2.id))` for every CVE group (16k groups, ~94 rows
   each). This forces PostgreSQL into single-threaded GroupAggregate with a 160MB
   disk-spilling sort. Replacing `jsonb_agg` with a simple CVE-name select enables
   parallel hash aggregation (37s -> 1s in EXPLAIN ANALYZE).

3. **GraphQL sub-resolver cascade**: The `imageCVEs` resolver fetches aggregated view
   data, then per-CVE sub-resolvers fire for `distroTuples` (full proto deserialization),
   `exceptionCount` (per-CVE DB query), and `affectedImageCountBySeverity` (10 filtered
   counts per CVE).

4. **Unnecessary work**: `distroTuples` data (CVE summary, OS, CVSS per distro) is
   fetched for all 20 rows even though it's only displayed on row expansion.

## Approaches Explored

### 1. TopSeverity (MAX) instead of severity count columns

Replace the 5-column "Images by severity" breakdown (Critical count, Important count,
etc.) with a single `MAX(severity)` column. Eliminates the 10 filtered COUNT aggregates
from the view query.

**Result**: Reduced the view query from ~27s to ~19s (single-phase). Meaningful
improvement but not sufficient alone because the full-table GROUP BY + sort is still
the dominant cost.

**Decision**: Keep as part of the solution. `TopSeverity` and `TopEPSSProbability`
fields added to the `CveCore` interface. `SkipGetImagesBySeverity` ReadOption controls
whether severity counts are computed.

### 2. Eliminate `jsonb_agg` from Phase 1

Phase 1 of two-phase pagination previously selected `CVEID` (row IDs via
`jsonb_agg(distinct(id))`) to identify the page of CVEs. Changed to select CVE names
directly (part of GROUP BY, no aggregation needed). Phase 2 filters by
`WHERE cvebaseinfo_cve IN (...)` using the existing btree index.

**EXPLAIN ANALYZE on production cluster (1.6M rows, 16k unique CVEs)**:

| Phase | Before (jsonb_agg) | After (CVE name) |
|-------|-------------------|-------------------|
| Phase 1 | 37,026ms (disk sort 160MB, no parallelism) | 993ms (in-memory, 2 parallel workers) |
| Phase 2 | 218ms (bitmap index scan) | 218ms (unchanged) |

**Decision**: Standalone improvement is real (37x on Phase 1) but limited in isolation
because the baseline's bottleneck is the severity count aggregations in Phase 1's sort,
not `jsonb_agg`. When deployed standalone, it also caused a data integrity issue: Phase 2's
`AddExactMatches(CVE name)` could surface row IDs from partially-written concurrent scans
that the old `AddDocIDs` approach would not have seen. The `DistroTuples` resolver's
loader then failed to find these IDs.

**Standalone PR scrapped** (#22227). The optimization is folded into the feature-flagged
stack where `DistroTuples` is lazy-loaded and `SkipGetImagesBySeverity` is set, avoiding
the problematic code path.

### 3. REST endpoints to bypass GraphQL

REST endpoints (`/api/v2/vulnmgmt/cves/` and `/api/v2/vulnmgmt/cves/{cve}/detail`)
bypass the GraphQL resolver cascade. Single SQL round trip, flat JSON response, no
sub-resolver waterfall.

**Findings**:
- grpc-gateway URL parser does not support repeated message fields in query params
  (multi-sort via `sortOptions[0]` is silently dropped)
- Enum values in URL params must be uppercase (`COUNT` not `count`) and the proto
  field name must match exactly (`aggrFunc` not `aggregateFunc`)
- The singular `pagination.sortOption` form is the only working path for REST sort

**Decision**: Keep REST as an option behind `ROX_VULN_MGMT_REST_API` flag. The
grpc-gateway limitations make it less flexible than GraphQL for sort, but the
single-roundtrip benefit is real for simple queries.

### 4. Severity-based tabs (selected approach)

Replace the single CVE table with tabs per severity level (Critical, Important,
Moderate, Low, Unknown). Each tab narrows the query to `WHERE severity = X`.

**Why this works**:
- The `imagecvesv2_severity` btree index makes `WHERE severity = X` a fast index scan
  instead of a full table scan
- Within a severity tab, the severity-count sort columns are unnecessary (all rows share
  the same severity), eliminating the 10 filtered COUNT aggregates from Phase 1
- Sorting by `COUNT(DISTINCT image)` (affected images) within a tab is affordable on the
  narrowed dataset
- The old severity-count sort answered "which CVEs have the most critical-severity
  occurrences across images." Severity tabs + affected-image sort gives users the same
  triage signal with a clearer mental model

**Extended to all entity tabs**:
- CVE tab: "which CVEs of this severity affect the most images?"
- Image tab: "which images have the most CVEs of this severity?"
- Deployment tab: "which deployments are exposed to the most CVEs of this severity?"

Images and deployments appear in multiple severity tabs (an image with critical AND
moderate CVEs appears in both tabs). This is intentional: each tab answers a distinct
triage question.

### 5. Secondary sort by affected image count (explored and rejected)

Attempted to add `COUNT(DISTINCT image_id)` as a tiebreaker sort after
`MAX(severity)` to order CVEs within the same severity tier by impact.

**Result**: Added ~11s to the query. PostgreSQL must compute the COUNT aggregate
across all CVE groups for sort ordering, even though it's only a tiebreaker.
Not worth the cost.

**Decision**: Rejected for the all-CVEs view. Within severity tabs, this sort becomes
the PRIMARY sort (not a tiebreaker) and is affordable because the dataset is narrowed
by severity.

### 6. Lazy-loading DistroTuples / CVE summary

The `distroTuples` field (CVE summary, operating system, CVSS per distro) triggers
full proto deserialization via the `ImageCVEV2Loader`. In the original UI, this data
was fetched for all 20 rows on page load even though it's only visible on row expansion.

**Decision**: Render `CVESummaryContent` only when `isExpanded` is true. PatternFly
expandable rows are always in the DOM, so components inside them mount immediately.
Gating on `isExpanded` prevents 20 queries on page load.

### 7. Batch exception counts

The `exceptionCount` per-CVE resolver fired 20 individual `vulnReqStore.Count()` queries.
Replaced with a single `batchExceptionCounts()` call that queries all 20 CVE names at once,
preloading the results into each resolver.

**Decision**: Keep. Simple, measurable improvement.

## Architecture

### Feature Flags

| Flag | Purpose |
|------|---------|
| `ROX_VULN_MGMT_UNIFIED_CVE_VIEW` | Enables severity tabs, TopSeverity column, simplified GraphQL query, lazy DistroTuples, batch exception counts |
| `ROX_VULN_MGMT_REST_API` | Enables REST CVE list/detail endpoints as an alternative to GraphQL |

### Backend Changes

- `central/views/imagecve/view_impl.go`: `TopSeverity` (MAX) and `TopEPSSProbability`
  (MAX) always selected. `SkipGetImagesBySeverity` ReadOption skips severity count
  aggregates. Phase 1 uses CVE names instead of row IDs (eliminates `jsonb_agg`).
- `central/views/imagecveflat/view_impl.go`: Same changes for the flat view.
- `central/graphql/resolvers/image_cve_core.go`: Sets `SkipGetImagesBySeverity = true`
  when unified view flag is on. Batch exception count preloading.
- `central/vulnmgmt/rest/`: REST handler for CVE list and detail endpoints.
- `pkg/features/list.go`: Feature flag registration.

### Frontend Changes

- `SeverityTabs.tsx`: PatternFly Tabs with severity icons, URL state via
  `useURLStringUnion('severityTab', severityTabValues)`.
- `useSeverityTabCounts.ts`: Single GraphQL query with 5 aliased `imageCVECount`
  calls for tab badge counts.
- `WorkloadCvesOverviewPage.tsx`: Reads severity tab from URL, injects
  `Severity: [severityLabelToSeverity(activeSeverityTab)]` into the scoped search
  filter, resets pagination/sort on tab change, hides severity column, suppresses
  severity filter chips from toolbar.
- `VulnerabilitiesOverview.tsx`: Renders severity tabs above entity table containers,
  suppresses severity filter in toolbar when tabs active.
- `sortUtils.tsx`: CVE tab default sort = affected images descending. Image/Deployment
  tab default sort = active severity's CVE count descending.
- `ImageOverviewTable.tsx` / `DeploymentOverviewTable.tsx`: "Matching CVEs" column
  (sum of existing `imageCVECountBySeverity` totals) visible when tabs active, sortable.
- `VulnMgmtService.ts`: REST sort params use proto field name (`aggrFunc` not
  `aggregateFunc`) with uppercase enum values.

### Database

- Table: `image_cves_v2` — 1.6M rows, 2.8GB
- Key indexes: `imagecvesv2_severity` (btree on severity), `imagecvesv2_cvebaseinfo_cve`
  (btree on CVE name), `imagecvesv2_state` (btree on vulnerability state)
- No schema changes required. No migrations needed.

## PR Stack

| Branch | PR | Description |
|--------|-----|-------------|
| `feature-flags` | #21607 | Register feature flags |
| `top-severity-backend` | #21608 | TopSeverity/TopEPSS fields, ReadOptions, Phase 1 jsonb_agg elimination |
| `batch-exceptions` | #21609 | Batch exception count loading |
| `batch-scoped-counts` | #21610 | Batch scoped count loading |
| `top-severity-frontend` | #21611 | TopSeverityLabel, lazy CVESummaryContent, simplified GraphQL query |
| `rest-cve-list` | #21612 | REST CVE list/detail endpoints |
| `rest-cve-list-ui` | #21613 | Frontend REST consumption, sort param fixes |
| `severity-tabs` | — | Severity tab UI for all entity tabs, Matching CVEs column |
| `severity-rest-flags` | #21614 | CI flag enablement |

## Open Questions for Productization

1. **Severity filter interaction**: When severity tabs are active, the Severity filter
   chips are hidden from the toolbar. Should the default severity filters from localStorage
   be suppressed entirely, or should they map to the initial tab selection?

2. **Image/Deployment tab counts**: Tab counts show `imageCVECount` scoped by severity.
   For Image/Deployment tabs, images/deployments appear in multiple severity tabs. Should
   tab counts show entity count (images with at least one critical CVE) or CVE count
   (total critical CVEs across all images)?

3. **URL deep linking**: Current URL structure is
   `?entityTab=CVE&severityTab=Critical&vulnerabilityState=OBSERVED`. When sharing a link,
   the severity tab is preserved. Should default filters still be applied on first visit,
   or should the severity tab override them?

4. **Unknown severity tab**: Included for completeness in the prototype. In production,
   Unknown severity CVEs are extremely rare. Consider hiding the Unknown tab or showing
   it only when count > 0.

5. **REST multi-sort**: grpc-gateway URL params cannot encode repeated message fields.
   The REST path only supports single sort via `pagination.sortOption`. For the
   Image/Deployment severity-count sort this works (single field), but if multi-sort is
   ever needed, the REST handler would need custom parsing or a JSON body approach.

6. **jsonb_agg standalone PR**: The standalone optimization (#22227) was scrapped due to
   a data integrity issue with `AddExactMatches(CVE name)` in Phase 2 surfacing
   partially-written rows. Within the feature-flagged stack (where `DistroTuples` is
   lazy-loaded), this issue doesn't occur. For a standalone merge, the fix is to change
   `DistroTuples` to query by CVE name instead of row IDs.

7. **Performance testing**: The prototype was validated on clusters with 1.6M
   `image_cves_v2` rows. Production environments may have different data distributions.
   EXPLAIN ANALYZE benchmarks should be run on representative production-scale data
   before release.
