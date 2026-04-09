# Deployment Status Default Filter — Design Spec

**Date:** 2026-03-31
**Branch:** deployment-tombstones
**Status:** Approved

---

## Problem

The deployment status filter (Deployed/Deleted) currently exists as a standalone `ToggleGroup` component (`DeploymentStatusFilter`) with a single-select URL parameter (`deploymentStatus`). This is inconsistent with how CVE severity and CVE status (fixable) filters work — those use multi-select checkboxes in the `DefaultFilterModal`, apply via the shared URL search filter, and automatically scope all entity tabs on the overview page.

The goal is to redesign deployment status to fully follow the CVE severity / CVE status pattern.

---

## Requirements

| # | Requirement |
|---|-------------|
| R1 | Deployment status filter supports multi-select: "Deployed", "Deleted", or both. |
| R2 | Filter appears in the "Default filters" modal alongside CVE severity and CVE status. |
| R3 | Default value: `['Deployed']` (only active deployments shown by default). |
| R4 | Filter is URL-persisted via the existing URL search filter (same `s[DEPLOYMENT_STATUS]` array format as SEVERITY). |
| R5 | Filter scopes all three entity tabs (CVE, Image, Deployment) on the overview page via `workloadCvesScopedQueryString`. |
| R6 | Filter scopes all queries (summary, images, deployments) on the CVE detail page (`ImageCvePage`). |
| R7 | Pagination count recomputes when filter changes (follows automatically from Apollo re-fetch on variable change). |
| R8 | Filter UI is gated on the `ROX_DEPLOYMENT_TOMBSTONES` feature flag. |
| R9 | The existing `DeploymentStatusFilter` ToggleGroup, `useDeploymentStatus` hook, and `getDeploymentStatusQueryString` are removed. |

---

## Query Semantics

| Selected values | Appended to base query | Backend effect |
|----------------|------------------------|----------------|
| `['Deployed']` (default) | nothing | `withTombstoneExclusion` adds `tombstone_deleted_at IS NULL` → only active. |
| `['Deleted']` | `+Tombstone Deleted At:*` | Field mentioned → exclusion bypassed; IS NOT NULL applied → only tombstoned. |
| `['Deployed', 'Deleted']` | `+Tombstone Deleted At:*,-*` | Field mentioned → exclusion bypassed. Comma-separated values for one field form a **disjunction**: `(IS NOT NULL OR IS NULL)` = all rows. |
| `[]` (empty; not reachable with default filter) | nothing | Treated as `['Deployed']` — exclusion adds null filter → only active. |

**Important:** Two separate `+`-joined entries for the same field (`+Tombstone Deleted At:*+Tombstone Deleted At:-*`) would overwrite in the parser. The correct encoding for a multi-value disjunction is **comma-separated values within a single field entry**: `Tombstone Deleted At:*,-*`. The `+` is the AND-conjunction separator between fields; commas within one field value produce a disjunction.

The `+` character is the AND-conjunction separator for the backend search string. No backend Go changes are required.

---

## Frontend Changes

### 1. `Containers/Vulnerabilities/types.ts`

Add deployment status label type alongside existing `VulnerabilitySeverityLabel` and `FixableStatus`:

```typescript
export const deploymentStatusLabels = ['Deployed', 'Deleted'] as const;
export type DeploymentStatusLabel = (typeof deploymentStatusLabels)[number];
export function isDeploymentStatusLabel(value: unknown): value is DeploymentStatusLabel {
    return deploymentStatusLabels.some((s) => s === value);
}
```

Extend the Yup schema for `VulnMgmtLocalStorage` to include `DEPLOYMENT_STATUS`:

```typescript
// In vulnMgmtLocalStorageSchema > preferences > defaultFilters:
DEPLOYMENT_STATUS: yup.array(yup.string().required().oneOf(deploymentStatusLabels)).required(),
```

`DefaultFilters` is inferred from this schema and gains `DEPLOYMENT_STATUS: DeploymentStatusLabel[]` automatically.

### 2. `Containers/Vulnerabilities/utils/searchUtils.tsx`

Add `getDeploymentStatusScopedQueryString` (replaces the deleted `getDeploymentStatusQueryString`).

**Expected outputs for all cases:**

| Input `selectedStatuses` | Output (given `baseQuery = 'CVE:X'`) |
|--------------------------|---------------------------------------|
| `['Deployed']` | `'CVE:X'` (unchanged) |
| `['Deleted']` | `'CVE:X+Tombstone Deleted At:*'` |
| `['Deployed', 'Deleted']` | `'CVE:X+Tombstone Deleted At:*,-*'` |
| `undefined` | `'CVE:X'` (treated as Deployed-only, same as default) |
| `[]` | `'CVE:X'` (treated as Deployed-only) |

```typescript
export function getDeploymentStatusScopedQueryString(
    baseQuery: string,
    selectedStatuses: DeploymentStatusLabel[] | undefined
): string {
    const showDeployed = !selectedStatuses || selectedStatuses.length === 0
        || selectedStatuses.includes('Deployed');
    const showDeleted = selectedStatuses?.includes('Deleted') ?? false;

    if (showDeployed && showDeleted) {
        // Comma-separated values for one field form a disjunction in the backend parser:
        // (IS NOT NULL OR IS NULL) = all rows; field mention bypasses withTombstoneExclusion.
        return [baseQuery, 'Tombstone Deleted At:*,-*'].filter(Boolean).join('+');
    }
    if (showDeleted) {
        return [baseQuery, 'Tombstone Deleted At:*'].filter(Boolean).join('+');
    }
    // 'Deployed' only, empty, or undefined: no addition.
    return baseQuery;
}
```

Remove `getDeploymentStatusQueryString` (the old single-value function).

**Important:** `DEPLOYMENT_STATUS` values (`'Deployed'`, `'Deleted'`) are human-readable labels consumed exclusively by `getDeploymentStatusScopedQueryString` and must **not** be passed to `getVulnStateScopedQueryString` or `getRequestQueryStringForSearchFilter`. When spreading `querySearchFilter` into the scoped query call, `DEPLOYMENT_STATUS` must be excluded. See the `WorkloadCvesOverviewPage` and `ImageCvePage` sections below for the exact pattern.

### 3. `WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`

**Add `isTombstonesEnabled`:** `isFeatureFlagEnabled` is already available from `useFeatureFlags()` (line 130) but `isTombstonesEnabled` is not yet declared as a local constant. Add:

```typescript
const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');
```

**Default storage:** Add `DEPLOYMENT_STATUS: ['Deployed']` to `defaultStorage.preferences.defaultFilters`.

**`mergeDefaultAndLocalFilters`:** Add `DEPLOYMENT_STATUS` merge logic following the same `difference/concat` pattern as `SEVERITY` and `FIXABLE`. Note that `filter` is typed as `SearchFilter` which stores `string | string[]`; the `difference()` call uses the same loose cast already accepted by `SEVERITY` and `FIXABLE`:

```typescript
let DEPLOYMENT_STATUS = (filter.DEPLOYMENT_STATUS as string[] | undefined) ?? [];
DEPLOYMENT_STATUS = difference(DEPLOYMENT_STATUS, oldDefaults.DEPLOYMENT_STATUS, newDefaults.DEPLOYMENT_STATUS);
DEPLOYMENT_STATUS = DEPLOYMENT_STATUS.concat(newDefaults.DEPLOYMENT_STATUS);
return { ...filter, SEVERITY, FIXABLE, DEPLOYMENT_STATUS };
```

**`workloadCvesScopedQueryString`:** Wrap the existing query string. `DEPLOYMENT_STATUS` is read directly from `searchFilter` (not from `querySearchFilter`, to avoid it leaking into the base query):

```typescript
// Strip DEPLOYMENT_STATUS from the filter passed to getVulnStateScopedQueryString:
const { DEPLOYMENT_STATUS: _deploymentStatus, ...querySearchFilterWithoutStatus } = querySearchFilter;

const rawScopedQuery = isViewingWithCves
    ? getVulnStateScopedQueryString(
          { ...baseSearchFilter, ...querySearchFilterWithoutStatus },
          currentVulnerabilityState
      )
    : getZeroCveScopedQueryString({ ...baseSearchFilter, ...querySearchFilterWithoutStatus });

const workloadCvesScopedQueryString = isTombstonesEnabled
    ? getDeploymentStatusScopedQueryString(
          rawScopedQuery,
          searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
      )
    : rawScopedQuery;
```

**Pass `isTombstonesEnabled` to `DefaultFilterModal`** at its call site (line ~431). Add the prop:

```tsx
<DefaultFilterModal
    defaultFilters={localStorageValue.preferences.defaultFilters}
    setLocalStorage={updateDefaultFilters}
    isTombstonesEnabled={isTombstonesEnabled}
/>
```

### 4. `WorkloadCves/components/DefaultFilterModal.tsx`

Accept a new `isTombstonesEnabled: boolean` prop. Add "Deployment status" `FormGroup` with two `Checkbox` items, gated on the flag. Add a handler `handleDeploymentStatusChange` following the same pattern as `handleFixableChange`. Extend the `totalFilters` count:

```typescript
const totalFilters =
    defaultFilters.SEVERITY.length +
    defaultFilters.FIXABLE.length +
    (isTombstonesEnabled ? defaultFilters.DEPLOYMENT_STATUS.length : 0);
```

The form group appears after "CVE status":

```tsx
{isTombstonesEnabled && (
    <FormGroup label="Deployment status" isInline>
        <Checkbox
            label="Deployed"
            id="deployed-status"
            isChecked={values.DEPLOYMENT_STATUS.includes('Deployed')}
            onChange={(_event, isChecked) => handleDeploymentStatusChange('Deployed', isChecked)}
        />
        <Checkbox
            label="Deleted"
            id="deleted-status"
            isChecked={values.DEPLOYMENT_STATUS.includes('Deleted')}
            onChange={(_event, isChecked) => handleDeploymentStatusChange('Deleted', isChecked)}
        />
    </FormGroup>
)}
```

### 5. `WorkloadCves/ImageCve/ImageCvePage.tsx`

Replace hook-based filter with URL search filter.

**Remove:**
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'`
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'`
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'`
- `const deploymentStatus = useDeploymentStatus()`
- The `DeploymentStatusFilter` `SplitItem` JSX block

**Add:**
- `import { getDeploymentStatusScopedQueryString } from '../../utils/searchUtils'`
- Add `DeploymentStatusLabel` to the existing `import type { ... } from '../../types'` line (do not create a second import from the same module).

**Exclude `DEPLOYMENT_STATUS` from `querySearchFilter`** before passing to query builders. Add this after `const querySearchFilter = parseQuerySearchFilter(searchFilter)`:

```typescript
const { DEPLOYMENT_STATUS: _deploymentStatus, ...querySearchFilterWithoutStatus } = querySearchFilter;
```

Use `querySearchFilterWithoutStatus` in place of `querySearchFilter` in all calls to `getVulnStateScopedQueryString`.

**Update `query` (top-level, used by summary + image requests):**

```typescript
const query = isTombstonesEnabled
    ? getDeploymentStatusScopedQueryString(
          getVulnStateScopedQueryString(
              { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilterWithoutStatus },
              vulnerabilityState
          ),
          searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
      )
    : getVulnStateScopedQueryString(
          { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilterWithoutStatus },
          vulnerabilityState
      );
```

**Update `getDeploymentSearchQuery`:**

```typescript
function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
    const filters = { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilterWithoutStatus };
    if (severity) filters.SEVERITY = [severity];
    const base = getVulnStateScopedQueryString(filters, vulnerabilityState);
    return isTombstonesEnabled
        ? getDeploymentStatusScopedQueryString(
              base,
              searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
          )
        : base;
}
```

**Fallback behavior when `DEPLOYMENT_STATUS` is absent from URL:** When navigating directly to a CVE detail page without prior use of the overview page, `searchFilter.DEPLOYMENT_STATUS` is `undefined`. `getDeploymentStatusScopedQueryString` treats `undefined` as `['Deployed']` (no tombstone addition). This is intentional — the correct default is to show only active deployments. No default-filter sync is needed on `ImageCvePage`.

### 6. `WorkloadCves/Overview/DeploymentsTableContainer.tsx`

**Remove:**
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'`
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'`
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'`
- `const deploymentStatus = useDeploymentStatus()`
- The `getDeploymentStatusQueryString(...)` call and its result variable
- The `DeploymentStatusFilter` `ToolbarItem` JSX block
- `isTombstonesEnabled` local variable if it was only used for the filter (check if used elsewhere)

`deploymentsQueryString` becomes simply:

```typescript
const deploymentsQueryString = workloadCvesScopedQueryString;
```

The deployment status filter is applied upstream in `WorkloadCvesOverviewPage` before `workloadCvesScopedQueryString` reaches this container.

---

## Files to Remove

| File | Reason |
|------|--------|
| `WorkloadCves/components/DeploymentStatusFilter.tsx` | Replaced by DefaultFilterModal checkboxes |
| `WorkloadCves/components/DeploymentStatusFilter.test.tsx` | Component deleted |
| `hooks/useDeploymentStatus.ts` | No longer needed |
| `hooks/useDeploymentStatus.test.tsx` | Hook deleted |
| `types/deploymentStatus.ts` | Merged into `Containers/Vulnerabilities/types.ts` |
| `types/deploymentStatus.test.ts` | Deleted with the above |

---

## Files to Create / Modify

| File | Change |
|------|--------|
| `Containers/Vulnerabilities/types.ts` | Add `deploymentStatusLabels`, type, guard; extend Yup schema and `DefaultFilters` |
| `Containers/Vulnerabilities/utils/searchUtils.tsx` | Add `getDeploymentStatusScopedQueryString`; remove `getDeploymentStatusQueryString` |
| `Containers/Vulnerabilities/utils/searchUtils.test.ts` | Replace old function tests with new ones (5 cases: Deployed, Deleted, both, undefined, empty) |
| `WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx` | Add `isTombstonesEnabled`, extend `defaultStorage` + `mergeDefaultAndLocalFilters`, wrap `workloadCvesScopedQueryString`, pass prop to `DefaultFilterModal` |
| `WorkloadCves/components/DefaultFilterModal.tsx` | Add `isTombstonesEnabled` prop, "Deployment status" form group, extend badge count |
| `WorkloadCves/ImageCve/ImageCvePage.tsx` | Replace hook-based filter with URL search filter, strip DEPLOYMENT_STATUS from querySearchFilter, remove `DeploymentStatusFilter` SplitItem |
| `WorkloadCves/Overview/DeploymentsTableContainer.tsx` | Remove per-container tombstone logic; `deploymentsQueryString = workloadCvesScopedQueryString` |

---

## Non-Goals

- No changes to the backend Go code.
- No changes to other VM pages (node vulnerabilities, platform CVEs, etc.).
- No changes to the advanced search toolbar (deployment status is not an autocomplete field).
- The `DeploymentStatus` proto field and datastore write paths added in the previous session are kept as-is.
