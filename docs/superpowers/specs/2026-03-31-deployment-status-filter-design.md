# Deployment Status Filter — Design Spec

**Date:** 2026-03-31
**Branch:** deployment-tombstones
**Status:** Approved

---

## Problem

The VM UI exposes tombstoned (soft-deleted) deployments behind a `Switch` toggle ("Show deleted") that is session-local, Deployment-tab-only, and does not scope the CVE or Image entity tabs. The toggle needs to be replaced with a proper URL-persisted "Deployment status" filter that:

- Applies to all entity tabs (Image, Deployment, CVE) on the CVE detail page.
- Applies to the deployment table on the overview page.
- Uses a backend proto enum for type safety and future searchability.
- Defaults to showing only active ("Deployed") deployments.

---

## Requirements

| # | Requirement |
|---|-------------|
| R1 | Replace the "Show deleted" switch with a "Deployment status" filter on both the CVE detail page and the overview deployments table. |
| R2 | Options: **Deployed** (default) and **Deleted**. |
| R3 | Filter state is URL-persisted via the `deploymentStatus` query parameter. |
| R4 | When "Deleted" is selected, only data from tombstoned deployments is shown across all entity tabs (CVEs, Images, Deployments). |
| R5 | When "Deployed" is selected, only data from active (non-tombstoned) deployments is shown. |
| R6 | A `DeploymentStatus` proto enum is added to the backend with an UNSPECIFIED zero value for backward compatibility. |
| R7 | No DB migration. Existing records retain the zero value (UNSPECIFIED), treated as DEPLOYED. |
| R8 | The filter is gated on the `ROX_DEPLOYMENT_TOMBSTONES` feature flag. |

---

## Backend

### Proto changes

**New enum** — added to `proto/storage/deployment.proto` (or a dedicated `deployment_status.proto`):

```proto
enum DeploymentStatus {
    DEPLOYMENT_STATUS_UNSPECIFIED = 0; // Zero value; retained for backward compatibility.
    DEPLOYMENT_STATUS_DEPLOYED    = 1; // Deployment is active.
    DEPLOYMENT_STATUS_DELETED     = 2; // Deployment has been soft-deleted (tombstoned).
}
```

**New field on `Deployment`** (tag 37, the next available):

```proto
// deployment_status reflects the lifecycle state of the deployment.
// UNSPECIFIED is the zero value for records written before this field existed.
DeploymentStatus deployment_status = 37;
// @gotags: search:"Deployment Status,hidden"
```

The `hidden` search tag makes the field queryable internally but not surfaced in the UI search autocomplete.

### Write path

| Operation | Value set |
|-----------|-----------|
| `UpsertDeployment` | `DEPLOYMENT_STATUS_DEPLOYED` |
| `TombstoneDeployment` | `DEPLOYMENT_STATUS_DELETED` |

No migration is run. Existing active records get the zero value (`UNSPECIFIED`), which the filter treats as `DEPLOYED` (via the existing tombstone null-check). Existing tombstoned records carry `UNSPECIFIED` but are still discovered by the `Tombstone Deleted At:*` mechanism.

### Search/filter mechanism

Because existing tombstoned records may carry `UNSPECIFIED` (not `DELETED`), `Tombstone Deleted At` remains the authoritative discriminator:

| Filter selection | Backend query addition |
|-----------------|------------------------|
| "Deployed" | No addition — tombstone exclusion (`tombstone IS NULL`) applied by `deploymentView.withTombstoneExclusion` |
| "Deleted" | `+Tombstone Deleted At:*` — opts into all tombstoned records regardless of `deployment_status` value |

`deployment_status` is stored for future use. Once natural record turnover eliminates UNSPECIFIED entries, the filter can migrate cleanly to `Deployment Status:DEPLOYED` / `Deployment Status:DELETED` without a user-facing change.

### Code generation

Run `make proto-generated-srcs` after proto changes.

---

## Frontend

### Types (`src/types/deploymentStatus.ts`)

```typescript
export const deploymentStatuses = ['DEPLOYED', 'DELETED'] as const;
export type DeploymentStatus = (typeof deploymentStatuses)[number];

export function isDeploymentStatus(value: unknown): value is DeploymentStatus {
    return deploymentStatuses.some((s) => s === value);
}
```

### Hook (`src/hooks/useDeploymentStatus.ts`)

Mirrors `useVulnerabilityState`. Reads `deploymentStatus` from the URL, defaulting to `'DEPLOYED'` (first element of the const array).

```typescript
import useURLStringUnion from 'hooks/useURLStringUnion';
import { deploymentStatuses } from 'types/deploymentStatus';
import type { DeploymentStatus } from 'types/deploymentStatus';

export default function useDeploymentStatus(): DeploymentStatus {
    const [status] = useURLStringUnion('deploymentStatus', deploymentStatuses);
    return status;
}
```

### Search utility (`Vulnerabilities/utils/searchUtils.tsx`)

New exported function, wraps any existing query string:

```typescript
export function getDeploymentStatusQueryString(
    baseQuery: string,
    deploymentStatus: DeploymentStatus
): string {
    if (deploymentStatus === 'DELETED') {
        return [baseQuery, 'Tombstone Deleted At:*'].filter(Boolean).join('+');
    }
    return baseQuery;
}
```

Called after `getVulnStateScopedQueryString` to layer the deployment status scope on top.

### Filter component (`WorkloadCves/components/DeploymentStatusFilter.tsx`)

A PatternFly `ToggleGroup` — two buttons, "Deployed" and "Deleted". Lighter than full Tabs (which are page-level navigation); appropriate for a secondary scope filter.

```tsx
function DeploymentStatusFilter({ onChange }: { onChange?: () => void }) {
    const [status, setStatus] = useURLStringUnion('deploymentStatus', deploymentStatuses);
    return (
        <ToggleGroup aria-label="Deployment status">
            <ToggleGroupItem
                text="Deployed"
                isSelected={status === 'DEPLOYED'}
                onChange={() => { setStatus('DEPLOYED'); onChange?.(); }}
            />
            <ToggleGroupItem
                text="Deleted"
                isSelected={status === 'DELETED'}
                onChange={() => { setStatus('DELETED'); onChange?.(); }}
            />
        </ToggleGroup>
    );
}
```

The `onChange` callback is used by parent pages to reset pagination to page 1.

---

## Integration

### CVE detail page (`ImageCvePage`)

- Remove `showDeleted` useState, `isTombstonesEnabled`, and the `Switch` JSX.
- Add `const deploymentStatus = useDeploymentStatus()`.
- Wrap the base query in `getDeploymentStatusQueryString` in **all** query paths (deployment, image, summary), so all entity tabs reflect the deployment status scope.
- Render `<DeploymentStatusFilter onChange={() => setPage(1)} />` above the entity toggle group (`Image` / `Deployment`), gated on `isTombstonesEnabled`.

Updated `getDeploymentSearchQuery`:

```typescript
function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
    const filters = { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter };
    if (severity) filters.SEVERITY = [severity];
    const base = getVulnStateScopedQueryString(filters, vulnerabilityState);
    return getDeploymentStatusQueryString(base, deploymentStatus);
}
```

The top-level `query` variable (used for summary + image queries) also wraps through `getDeploymentStatusQueryString`.

### Overview page (`DeploymentsTableContainer`)

- Remove `showDeleted` useState and `Switch`.
- Add `const deploymentStatus = useDeploymentStatus()`.
- Apply `getDeploymentStatusQueryString(workloadCvesScopedQueryString, deploymentStatus)` to produce `deploymentsQueryString`.
- Render `<DeploymentStatusFilter onChange={() => pagination.setPage(1)} />` as a `ToolbarItem`, replacing the removed `Switch`.

---

## Files to Create / Modify

| File | Change |
|------|--------|
| `proto/storage/deployment.proto` | Add `DeploymentStatus` enum + `deployment_status` field (tag 37) |
| `central/deployment/datastore/datastore_impl.go` | Set status on `UpsertDeployment` and `TombstoneDeployment` |
| `src/types/deploymentStatus.ts` | New — `DeploymentStatus` type, const array, type guard |
| `src/hooks/useDeploymentStatus.ts` | New — URL-backed hook |
| `WorkloadCves/components/DeploymentStatusFilter.tsx` | New — `ToggleGroup` filter component |
| `Vulnerabilities/utils/searchUtils.tsx` | Add `getDeploymentStatusQueryString` |
| `WorkloadCves/ImageCve/ImageCvePage.tsx` | Wire `deploymentStatus`, remove `showDeleted`, add filter UI |
| `WorkloadCves/Overview/DeploymentsTableContainer.tsx` | Wire `deploymentStatus`, remove `showDeleted`, swap `Switch` for filter |

---

## Non-Goals

- No DB migration for existing records.
- No changes to the `Deployment Status` search label behavior — it is `hidden` and not yet the primary filter mechanism.
- No changes to other VM pages beyond the two listed above.
