# Vulnerability Page Performance: Implementation Plan

This document describes the implementation steps required to productize the severity
tabs feature. It assumes familiarity with the [design doc](design.md).

## Prerequisites

- Feature flags `ROX_VULN_MGMT_UNIFIED_CVE_VIEW` and `ROX_VULN_MGMT_REST_API`
  registered in `pkg/features/list.go` (done in PR #21607)
- Existing PR stack reviewed and merged through `feature-flags`

## Phase 1: Backend Optimizations (no UI changes, no flag dependency)

These changes improve performance for ALL users, not just those with the flag enabled.

### 1.1 Eliminate `jsonb_agg` from Phase 1 pagination

**Files**:
- `central/views/imagecve/view_impl.go`
- `central/views/imagecveflat/view_impl.go`

**Changes**:
- `withSelectCVEIdentifiersQuery()`: Select `search.CVE` instead of
  `search.CVEID.Distinct()`. The CVE name is part of the GROUP BY, so the search
  framework emits it directly — no `jsonb_agg` wrapper.
- `getFilteredCVEs()`: Use a lightweight `cveNameResponse` struct (`db:"cve"`) instead
  of deserializing the full `imageCVECoreResponse`. Collect CVE name strings.
- `withSelectCVECoreResponseQuery()` / `withSelectCVEFlatResponseQuery()`: Filter
  Phase 2 with `AddExactMatches(search.CVE, cvesToFilter...)` instead of
  `AddDocIDs(cveIDsToFilter...)`. Uses the existing `imagecvesv2_cvebaseinfo_cve`
  btree index.

**Caveat**: The `DistroTuples` resolver uses `GetCVEIDs()` (row IDs from Phase 2's
`jsonb_agg`) to load full protos. When Phase 2 filters by CVE name instead of row ID,
it can surface row IDs from partially-written concurrent scans. The fix is to change
`DistroTuples` to query by CVE name (see Phase 1.2) or gate behind the feature flag
and lazy-load.

**Verification**:
- `go test -tags sql_integration ./central/views/imagecve/...`
- `go test -tags sql_integration ./central/views/imagecveflat/...`
- `go test -tags sql_integration ./central/graphql/resolvers/... -run TestImageCVE`
- EXPLAIN ANALYZE on production-scale cluster confirming parallel hash aggregate
  instead of disk-spilling GroupAggregate

### 1.2 Change DistroTuples to query by CVE name

**File**: `central/graphql/resolvers/image_cve_core.go`

**Change**: In `DistroTuples()`, replace:
```go
query := search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetCVEIDs()...).ProtoQuery()
```
with:
```go
query := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetCVE()).ProtoQuery()
```

The distro data (summary, OS, CVSS per distro) is about the vulnerability itself, not
which image it appears in. Querying by CVE name returns the same distro information.
SAC on the loader handles access control.

This unblocks the standalone `jsonb_agg` elimination (Phase 1.1) by removing the
dependency on row IDs from Phase 2.

**Verification**:
- Expand a CVE row and verify summary/distro data loads correctly
- `go test -tags sql_integration ./central/graphql/resolvers/... -run TestImageCVE`

### 1.3 Batch exception counts

**File**: `central/graphql/resolvers/image_cve_core.go`

**Change**: After `ImageCVEs()` resolver returns CVE results, call
`batchExceptionCounts()` with all CVE names. Preload results into each resolver via
`hasPreloadedExceptionCount` / `preloadedExceptionCount` fields. The per-CVE
`ExceptionCount()` resolver checks for preloaded data before querying.

`batchExceptionCounts()` implementation in
`central/graphql/resolvers/vulnerability_requests.go`: single query with
`AddExactMatches(search.CVE, cves...)` and `AddBools(search.ExpiredRequest, false)`,
deduplicates with `set.NewStringSet`.

**Verification**:
- `go test -tags sql_integration ./central/graphql/resolvers/... -run TestImageCVE`
- Verify exception count badges appear correctly on CVE rows

## Phase 2: Backend Changes Behind Feature Flag

### 2.1 TopSeverity and TopEPSS fields

**Files**:
- `central/views/imagecve/types.go`: Add `GetTopSeverity()` and
  `GetEPSSProbability()` to `CveCore` interface
- `central/views/imagecve/db_response.go`: Add `TopSeverity` and `EpssProbability`
  fields to `imageCVECoreResponse`
- `central/views/imagecve/view_impl.go`: Add `MAX(severity)` and
  `MAX(epss_probability)` selects (always included, cheap scalar aggregates)
- `central/graphql/resolvers/image_cve_core.go`: Add `topSeverity` and
  `topEpssProbability` to GraphQL schema. Set `SkipGetImagesBySeverity = true`
  when flag is on.

**Same changes** for `central/views/imagecveflat/`.

### 2.2 REST CVE list endpoints

**New files** in `central/vulnmgmt/rest/`:
- `handler.go`: HTTP handler with `ServeHTTP` routing
- `cve_list.go`: `listCVEs()` — calls `cveView.Get()` + `cveView.Count()` +
  `batchExceptionCounts()`
- `cve_detail.go`: `getCVEDetail()` — queries CVE datastore for distro details
- `types.go`: `CVEListItem`, `CVEListResponse`, `CVEDetailItem`, `CVEDetailResponse`

**Registration**: `central/main.go` — add `CustomRoute` with trailing slash for
prefix matching: `/api/v2/vulnmgmt/cves/`

**Important**: Strip pagination from the Count query:
```go
countQuery := query.CloneVT()
countQuery.Pagination = nil
```

## Phase 3: Frontend Changes Behind Feature Flag

### 3.1 Severity tabs component

**New file**: `ui/.../WorkloadCves/components/SeverityTabs.tsx`

PatternFly `Tabs` component:
- URL state via `useURLStringUnion('severityTab', severityTabValues)`
- Each tab shows severity icon + label + count badge
- `counts` prop: `Partial<Record<SeverityTab, number>>`
- `onChange` callback for pagination/sort reset

### 3.2 Severity tab counts hook

**New file**: `ui/.../WorkloadCves/Overview/useSeverityTabCounts.ts`

Single GraphQL query with 5 aliased `imageCVECount` calls:
```graphql
query getSeverityTabCounts($criticalQuery: String, ...) {
    critical: imageCVECount(query: $criticalQuery)
    important: imageCVECount(query: $importantQuery)
    ...
}
```

Each variable gets the base search filter with the severity injected.
Uses `getRegexScopedQueryString()` to build each query string.
`skip: !enabled` prevents queries when tabs are not active.

**Important**: `queryValue` must be typed as `VulnerabilitySeverity` (not `string`)
to satisfy `QuerySearchFilter.Severity`.

### 3.3 Wire severity tabs into overview page

**File**: `ui/.../WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`

- Add `useURLStringUnion('severityTab', severityTabValues)` state
- Compute `useSeverityTabs = useUnifiedView && isViewingWithCves`
- When `useSeverityTabs`, inject severity into scoped search filter:
  ```ts
  workloadCvesScopedSearchFilter.Severity = [severityLabelToSeverity(activeSeverityTab)]
  ```
- `onSeverityTabChange(newSeverity)`: must accept the new severity value directly
  (URL state hasn't updated when handler fires). Reset pagination and set default sort
  using `activeEntityTabKey` (not hardcoded `'CVE'`).
- Hide `imagesBySeverity` / `cvesBySeverity` column via column overrides when tabs active
- Show `matchingCveCount` column on Image/Deployment tables when tabs active
- Strip Severity from default filters when unified view is on

### 3.4 Render tabs in VulnerabilitiesOverview

**File**: `ui/.../WorkloadCves/Overview/VulnerabilitiesOverview.tsx`

- Accept `useSeverityTabs` and `onSeverityTabChange` props
- Render `SeverityTabs` above all entity tab containers when active
- Set `includeCveSeverityFilters={isViewingWithCves && !useSeverityTabs}` to suppress
  severity filter chips when tabs are active
- Call `useSeverityTabCounts()` with the base search filter (without severity) and pass
  counts to `SeverityTabs`

### 3.5 Sort field changes

**File**: `ui/.../Vulnerabilities/utils/sortUtils.tsx`

`getWorkloadCveOverviewSortFields(entityTab, useUnifiedView)`:
- CVE tab + unified view: `['CVE', 'CVSS', 'Image Sha', 'CVE Created Time']`
  (severity removed — all rows share same severity)
- Image tab + unified view: `['Image', severityCountFields, 'Image OS', ...]`
  (severity count fields included for sort, column is hidden)
- Deployment tab + unified view: `['Deployment', severityCountFields, 'Cluster', ...]`

`getWorkloadCveOverviewDefaultSortOption(entityTab, searchFilter, useUnifiedView, activeSeverityTab)`:
- CVE tab: `{ field: 'Image Sha', direction: 'desc', aggregateBy: { aggregateFunc: 'count', distinct: 'true' } }`
  (affected images descending)
- Image/Deployment tab: `{ field: severitySortMap[activeSeverityTab], direction: 'desc' }`
  (active severity's CVE count descending)

### 3.6 Matching CVEs column

**Files**:
- `ui/.../WorkloadCves/Tables/ImageOverviewTable.tsx`
- `ui/.../WorkloadCves/Tables/DeploymentOverviewTable.tsx`

Add `matchingCveCount` column definition to `defaultColumns`. Column shows the sum of
all `imageCVECountBySeverity` totals. Since the query already includes the severity
filter from the tab, only CVEs of the active severity are counted. Toggling the
Fixable chip further narrows the count.

Column header is sortable via `getSortParams('Matching CVEs', getSeveritySortOptions(filteredSeverities))`.

Hidden when severity tabs are not active via column overrides in
`WorkloadCvesOverviewPage`.

### 3.7 Simplified GraphQL query

**File**: `ui/.../WorkloadCves/Tables/WorkloadCVEOverviewTable.tsx`

When unified view is on, use `simplifiedCveListQuery`:
```graphql
query getSimplifiedImageCVEList($query: String, $pagination: Pagination, $statusesForExceptionCount: [String!]) {
    imageCVEs(query: $query, pagination: $pagination) {
        cve
        topSeverity
        topCVSS
        topNvdCVSS
        topEpssProbability
        affectedImageCount
        firstDiscoveredInSystem
        publishedOn
        pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
    }
}
```

Drops `affectedImageCountBySeverity` (10 filtered count sub-queries) and
`distroTuples` (proto deserialization). `distroTuples` lazy-loaded on row expand
via `CVESummaryContent` gated on `isExpanded`.

### 3.8 REST consumption (optional)

**File**: `ui/.../services/VulnMgmtService.ts`

`appendSortParams()`:
- Use `pagination.sortOption` (singular, the only form grpc-gateway supports)
- Proto field name is `aggrFunc` (not `aggregateFunc`)
- Enum values must be uppercase (`COUNT`, `MAX`)

**File**: `ui/.../WorkloadCves/Overview/useImageCvesREST.ts`

Hook matching `useImageCves` interface, calls `fetchCVEList()`.

**File**: `ui/.../WorkloadCves/Overview/CVEsTableContainer.tsx`

Conditionally use REST or GraphQL based on `ROX_VULN_MGMT_REST_API` flag.

### 3.9 Lazy-load CVE summary on row expand

**File**: `ui/.../WorkloadCves/components/CVESummaryContent.tsx`

Only render when `isExpanded` is true. In `WorkloadCVEOverviewTable`, wrap
`CVESummaryContent` in `{isExpanded && <CVESummaryContent ... />}`.

## Phase 4: CI and Testing

### 4.1 Enable flags in CI

**Files**:
- `tests/e2e/lib.sh`: `ci_export ROX_VULN_MGMT_UNIFIED_CVE_VIEW true`
- `tests/e2e/lib-compat.sh`: Same

### 4.2 Tests to add/update

**Backend**:
- View integration tests: verify Phase 1/Phase 2 return correct results with CVE-name
  filtering (`central/views/imagecve/view_test.go`)
- Resolver tests: verify `SkipGetImagesBySeverity` is set when flag on, batch exception
  counts preloaded (`central/graphql/resolvers/image_cve_v2_core_test.go`)
- REST handler tests: verify list/detail endpoints, pagination, sort, flag gating
- Benchmark: `central/views/imagecve/two_phase_bench_test.go` — paginated vs
  unpaginated with multiple sort options

**Frontend**:
- `sortUtils.test.ts`: verify sort fields and default sort options for all entity tabs
  in unified view mode, verify severity tab changes update sort correctly
- Component test for `SeverityTabs`: tab rendering, URL state, onChange callback
- E2E: severity tab navigation, table data matches tab, sort behavior, filter interaction

## Phase 5: Productization Checklist

- [ ] UX review of severity tab design with PatternFly patterns
- [ ] Accessibility audit (tab keyboard navigation, screen reader labels)
- [ ] Documentation update (user-facing docs for the new tab layout)
- [ ] Performance benchmarks on production-scale data (>1M rows)
- [ ] Decision on Unknown severity tab (hide when count=0?)
- [ ] Decision on default filter behavior with severity tabs
- [ ] Decision on REST vs GraphQL as primary data path
- [ ] Feature flag graduation plan (when to remove the flag and make tabs default)
- [ ] Telemetry: track tab switches, sort changes, filter usage for adoption metrics
