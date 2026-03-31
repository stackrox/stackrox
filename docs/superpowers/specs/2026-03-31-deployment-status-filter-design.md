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
| R8 | The filter UI is gated on the `ROX_DEPLOYMENT_TOMBSTONES` feature flag. When the flag is off, the `deploymentStatus` URL parameter is ignored and tombstone exclusion is always applied (existing default behavior). |

---

## Backend

### Proto changes

**New enum and field** added to `proto/storage/deployment.proto`. The `@gotags` annotation must appear inline on the same field line (as per project convention throughout `deployment.proto`):

```proto
enum DeploymentStatus {
    DEPLOYMENT_STATUS_UNSPECIFIED = 0; // Zero value; retained for backward compatibility.
    DEPLOYMENT_STATUS_DEPLOYED    = 1; // Deployment is active.
    DEPLOYMENT_STATUS_DELETED     = 2; // Deployment has been soft-deleted (tombstoned).
}

// In the Deployment message (tag 37 is next available):
DeploymentStatus deployment_status = 37; // @gotags: search:"Deployment Status,hidden"
```

The `hidden` search tag makes the field queryable internally but not surfaced in UI search autocomplete. The enum definition should appear before the `Deployment` message, following the project's pattern for enums (e.g., `PermissionLevel`).

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

The `+` character is the backend's AND-conjunction separator for search query strings. `Tombstone Deleted At:*` means the field is non-null (i.e., tombstone is set). These conventions are established throughout the codebase.

`deployment_status` is stored for future use. Once natural record turnover eliminates UNSPECIFIED entries, the filter can migrate cleanly to `Deployment Status:DEPLOYED` / `Deployment Status:DELETED` without a user-facing change.

After adding the field, update the `// Next available tag: 37` comment at the top of the `Deployment` message to `// Next available tag: 38`.

### Code generation

Run `make proto-generated-srcs` after proto changes.

---

## Frontend

### Types (`src/types/deploymentStatus.ts`)

New file. Note `useDeploymentStatus` is intentionally read-only (no setter exported), matching the `useVulnerabilityState` pattern — mutations are handled by `DeploymentStatusFilter` directly via its own `useURLStringUnion` call.

```typescript
export const deploymentStatuses = ['DEPLOYED', 'DELETED'] as const;
export type DeploymentStatus = (typeof deploymentStatuses)[number];

export function isDeploymentStatus(value: unknown): value is DeploymentStatus {
    return deploymentStatuses.some((s) => s === value);
}
```

### Hook (`src/hooks/useDeploymentStatus.ts`)

New file. Mirrors `useVulnerabilityState`. Reads `deploymentStatus` from the URL, defaulting to `'DEPLOYED'` (first element of the const array, per `useURLStringUnion` convention).

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

New exported function, added alongside `getVulnStateScopedQueryString`. Wraps any existing query string. The `+` separator is the backend's AND-conjunction operator for search strings:

```typescript
export function getDeploymentStatusQueryString(
    baseQuery: string,
    deploymentStatus: DeploymentStatus
): string {
    if (deploymentStatus === 'DELETED') {
        // '+' is the AND-conjunction separator for backend search strings.
        // 'Tombstone Deleted At:*' opts into all tombstoned records.
        return [baseQuery, 'Tombstone Deleted At:*'].filter(Boolean).join('+');
    }
    return baseQuery; // 'DEPLOYED' → tombstone exclusion applied by the view layer.
}
```

### Filter component (`WorkloadCves/components/DeploymentStatusFilter.tsx`)

New file. A PatternFly `ToggleGroup` — two buttons, "Deployed" and "Deleted". Lighter than full Tabs (which are page-level navigation); appropriate for a secondary scope filter. Each `ToggleGroupItem` requires an `id` prop for correct ARIA labeling.

```tsx
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { deploymentStatuses } from 'types/deploymentStatus';

type DeploymentStatusFilterProps = {
    onChange?: () => void;
};

export default function DeploymentStatusFilter({ onChange }: DeploymentStatusFilterProps) {
    const [status, setStatus] = useURLStringUnion('deploymentStatus', deploymentStatuses);
    return (
        <ToggleGroup aria-label="Deployment status">
            <ToggleGroupItem
                id="deployment-status-deployed"
                text="Deployed"
                isSelected={status === 'DEPLOYED'}
                onChange={() => { setStatus('DEPLOYED'); onChange?.(); }}
            />
            <ToggleGroupItem
                id="deployment-status-deleted"
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

**Remove:**
- `showDeleted` `useState`
- The `Switch` import and its JSX
- `isTombstonesEnabled` local variable and all its guards (both inside `getDeploymentSearchQuery` and in the JSX)

**Add:**
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'`
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'`
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'`
- `import useFeatureFlags from 'hooks/useFeatureFlags'` (for the filter gate)

**State:**
```typescript
const { isFeatureFlagEnabled } = useFeatureFlags();
const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');
const deploymentStatus = useDeploymentStatus();
```

**Top-level `query` variable** (scopes summary and image tab queries):
```typescript
const query = getDeploymentStatusQueryString(
    getVulnStateScopedQueryString(
        { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter },
        vulnerabilityState
    ),
    isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
);
```

When the feature flag is off, `deploymentStatus` is forced to `'DEPLOYED'` so the URL parameter has no effect.

**`getDeploymentSearchQuery`** (scopes deployment tab queries, including severity sub-queries):
```typescript
function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
    const filters = { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter };
    if (severity) filters.SEVERITY = [severity];
    const base = getVulnStateScopedQueryString(filters, vulnerabilityState);
    return getDeploymentStatusQueryString(
        base,
        isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
    );
}
```

**Filter UI** — rendered in a `SplitItem` above the entity toggle group, gated on `isTombstonesEnabled`:
```tsx
{isTombstonesEnabled && (
    <SplitItem>
        <DeploymentStatusFilter onChange={() => setPage(1)} />
    </SplitItem>
)}
```

### Overview page (`DeploymentsTableContainer`)

**Remove:**
- `showDeleted` `useState`
- The `Switch` import and its JSX

**Add:**
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'`
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'`
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'`

**State and query:**
```typescript
const { isFeatureFlagEnabled } = useFeatureFlags();
const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');
const deploymentStatus = useDeploymentStatus();

const deploymentsQueryString = getDeploymentStatusQueryString(
    workloadCvesScopedQueryString,
    isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
);
```

**Filter UI** — replace the `Switch` `ToolbarItem` with:
```tsx
{isTombstonesEnabled && (
    <ToolbarItem>
        <DeploymentStatusFilter onChange={() => pagination.setPage(1)} />
    </ToolbarItem>
)}
```

---

## Tests

| File | What to test |
|------|-------------|
| `src/types/deploymentStatus.test.ts` (new) | `isDeploymentStatus` returns true for valid values, false for others |
| `src/hooks/useDeploymentStatus.test.ts` (new) | Defaults to `'DEPLOYED'`; reflects `deploymentStatus` URL param |
| `Vulnerabilities/utils/searchUtils.test.ts` (extend) | `getDeploymentStatusQueryString`: returns base query unchanged for `DEPLOYED`; appends `+Tombstone Deleted At:*` for `DELETED`; handles empty base query |
| `WorkloadCves/components/DeploymentStatusFilter.test.tsx` (new) | Renders both options; clicking "Deleted" sets URL param; clicking "Deployed" clears it; `onChange` called on selection |

---

## Files to Create / Modify

| File | Change |
|------|--------|
| `proto/storage/deployment.proto` | Add `DeploymentStatus` enum + `deployment_status` field (tag 37), `@gotags` inline |
| `central/deployment/datastore/datastore_impl.go` | Set `DEPLOYED` in `UpsertDeployment`, `DELETED` in `TombstoneDeployment` |
| `src/types/deploymentStatus.ts` | New — `DeploymentStatus` type, const array, type guard |
| `src/hooks/useDeploymentStatus.ts` | New — URL-backed read-only hook |
| `WorkloadCves/components/DeploymentStatusFilter.tsx` | New — `ToggleGroup` filter component (exported default) |
| `Vulnerabilities/utils/searchUtils.tsx` | Add `getDeploymentStatusQueryString` |
| `WorkloadCves/ImageCve/ImageCvePage.tsx` | Wire `deploymentStatus`, update `query` and `getDeploymentSearchQuery`, remove `showDeleted`, add filter UI |
| `WorkloadCves/Overview/DeploymentsTableContainer.tsx` | Wire `deploymentStatus`, remove `showDeleted`, swap `Switch` for `DeploymentStatusFilter` |

---

## Non-Goals

- No DB migration for existing records.
- No changes to the `Deployment Status` search label behavior — it is `hidden` and not yet the primary filter mechanism.
- No changes to other VM pages beyond the two listed above.
