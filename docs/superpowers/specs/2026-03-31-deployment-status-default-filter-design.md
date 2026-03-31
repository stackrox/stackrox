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

| Selected values | Backend query effect |
|----------------|---------------------|
| `['Deployed']` (default) | No tombstone query addition. `withTombstoneExclusion` in the view adds `tombstone_deleted_at IS NULL` → only active deployments. |
| `['Deleted']` | Appends `+Tombstone Deleted At:*` (`IS NOT NULL`). `withTombstoneExclusion` bypasses. Only tombstoned. |
| `['Deployed', 'Deleted']` | Appends `+Tombstone Deleted At:*+Tombstone Deleted At:-*`. The `-*` negation means `IS NULL`. Multiple values for the same field form a **disjunction**: `(IS NOT NULL OR IS NULL)` = all rows. `withTombstoneExclusion` bypasses because the field is mentioned. All records returned. |
| `[]` (empty, not reachable by default) | No tombstone addition. Treated same as `['Deployed']` — `withTombstoneExclusion` adds null filter → only active. |

The `+` character is the AND-conjunction separator for the backend search string. Multiple values for the same field key (e.g., two `Tombstone Deleted At` values) form a disjunction within that field.

**No backend changes are required.** The existing `withTombstoneExclusion` disjunction/negation semantics handle all cases.

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
// In vulnMgmtLocalStorageSchema:
defaultFilters: yup.object({
    SEVERITY: yup.array(yup.string().required().oneOf(vulnerabilitySeverityLabels)).required(),
    FIXABLE: yup.array(yup.string().required().oneOf(fixableStatuses)).required(),
    DEPLOYMENT_STATUS: yup.array(yup.string().required().oneOf(deploymentStatusLabels)).required(),
}),
```

`DefaultFilters` is inferred from this schema and gains `DEPLOYMENT_STATUS: DeploymentStatusLabel[]` automatically.

### 2. `Containers/Vulnerabilities/utils/searchUtils.tsx`

Add `getDeploymentStatusScopedQueryString` (replaces the deleted `getDeploymentStatusQueryString`):

```typescript
export function getDeploymentStatusScopedQueryString(
    baseQuery: string,
    selectedStatuses: DeploymentStatusLabel[] | undefined
): string {
    // When both or unset — no tombstone addition (view default handles active-only)
    const showDeployed = !selectedStatuses || selectedStatuses.includes('Deployed');
    const showDeleted = selectedStatuses?.includes('Deleted') ?? false;

    if (showDeployed && showDeleted) {
        // Disjunction: IS NOT NULL OR IS NULL = all rows.
        // Multiple values for same field form a disjunction in the backend parser.
        return [baseQuery, 'Tombstone Deleted At:*', 'Tombstone Deleted At:-*']
            .filter(Boolean)
            .join('+');
    }
    if (showDeleted) {
        return [baseQuery, 'Tombstone Deleted At:*'].filter(Boolean).join('+');
    }
    // 'Deployed' only or empty: no addition.
    return baseQuery;
}
```

Remove `getDeploymentStatusQueryString` (the old single-value function).

### 3. `WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`

**Default storage:** Add `DEPLOYMENT_STATUS: ['Deployed']` to `defaultStorage.preferences.defaultFilters`.

**`mergeDefaultAndLocalFilters`:** Add `DEPLOYMENT_STATUS` merge logic following the same `difference/concat` pattern as `SEVERITY` and `FIXABLE`:

```typescript
let DEPLOYMENT_STATUS = filter.DEPLOYMENT_STATUS ?? [];
DEPLOYMENT_STATUS = difference(DEPLOYMENT_STATUS, oldDefaults.DEPLOYMENT_STATUS, newDefaults.DEPLOYMENT_STATUS);
DEPLOYMENT_STATUS = DEPLOYMENT_STATUS.concat(newDefaults.DEPLOYMENT_STATUS);
return { ...filter, SEVERITY, FIXABLE, DEPLOYMENT_STATUS };
```

**`workloadCvesScopedQueryString`:** Wrap the existing query string with `getDeploymentStatusScopedQueryString`:

```typescript
const rawScopedQuery = isViewingWithCves
    ? getVulnStateScopedQueryString({ ...baseSearchFilter, ...querySearchFilter }, currentVulnerabilityState)
    : getZeroCveScopedQueryString({ ...baseSearchFilter, ...querySearchFilter });

const workloadCvesScopedQueryString = isTombstonesEnabled
    ? getDeploymentStatusScopedQueryString(
          rawScopedQuery,
          searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
      )
    : rawScopedQuery;
```

`isTombstonesEnabled` is already read from `useFeatureFlags` in `WorkloadCvesOverviewPage`.

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

Pass `isTombstonesEnabled` from `WorkloadCvesOverviewPage` to `DefaultFilterModal` at its call site.

### 5. `WorkloadCves/ImageCve/ImageCvePage.tsx`

Replace hook-based filter with URL search filter:

**Remove:**
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'`
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'`
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'`
- `const deploymentStatus = useDeploymentStatus()`
- The `DeploymentStatusFilter` `SplitItem` JSX block

**Add:**
- `import { getDeploymentStatusScopedQueryString } from '../../utils/searchUtils'`

**Update `query` (top-level, used by summary + image requests):**

```typescript
const query = isTombstonesEnabled
    ? getDeploymentStatusScopedQueryString(
          getVulnStateScopedQueryString(
              { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter },
              vulnerabilityState
          ),
          searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
      )
    : getVulnStateScopedQueryString(
          { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter },
          vulnerabilityState
      );
```

**Update `getDeploymentSearchQuery`:**

```typescript
function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
    const filters = { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter };
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

### 6. `WorkloadCves/Overview/DeploymentsTableContainer.tsx`

**Remove:**
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'`
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'`
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'`
- `const deploymentStatus = useDeploymentStatus()`
- The `getDeploymentStatusQueryString(...)` call
- The `DeploymentStatusFilter` `ToolbarItem` JSX block

The deployment status filter is now applied at the `workloadCvesScopedQueryString` level in `WorkloadCvesOverviewPage`, before it reaches `DeploymentsTableContainer`. `deploymentsQueryString` becomes simply:

```typescript
const deploymentsQueryString = workloadCvesScopedQueryString;
```

(No per-container tombstone logic needed — it is handled upstream.)

---

## Files to Remove

| File | Reason |
|------|--------|
| `WorkloadCves/components/DeploymentStatusFilter.tsx` | Replaced by DefaultFilterModal checkboxes |
| `WorkloadCves/components/DeploymentStatusFilter.test.tsx` | Component deleted |
| `hooks/useDeploymentStatus.ts` | No longer needed |
| `hooks/useDeploymentStatus.test.tsx` | Hook deleted |

---

## Files to Create / Modify

| File | Change |
|------|--------|
| `Containers/Vulnerabilities/types.ts` | Add `deploymentStatusLabels`, type, guard; extend Yup schema and `DefaultFilters` |
| `Containers/Vulnerabilities/utils/searchUtils.tsx` | Add `getDeploymentStatusScopedQueryString`; remove `getDeploymentStatusQueryString` |
| `Containers/Vulnerabilities/utils/searchUtils.test.ts` | Update tests: replace old function tests with new function tests |
| `WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx` | Extend `defaultStorage`, `mergeDefaultAndLocalFilters`, wrap `workloadCvesScopedQueryString` |
| `WorkloadCves/components/DefaultFilterModal.tsx` | Add `isTombstonesEnabled` prop, "Deployment status" form group, extend badge count |
| `WorkloadCves/ImageCve/ImageCvePage.tsx` | Replace hook-based filter with URL search filter, remove `DeploymentStatusFilter` SplitItem |
| `WorkloadCves/Overview/DeploymentsTableContainer.tsx` | Remove per-container tombstone logic |

---

## Non-Goals

- No changes to the backend Go code.
- No changes to other VM pages (node vulnerabilities, platform CVEs, etc.).
- No changes to the advanced search toolbar (deployment status is not an autocomplete field).
- The `DeploymentStatus` proto field and datastore write paths added in the previous session are kept as-is.
