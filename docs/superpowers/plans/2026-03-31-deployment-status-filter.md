# Deployment Status Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the session-local "Show deleted" switch with a URL-persisted "Deployment status" ToggleGroup filter (Deployed/Deleted) that scopes all entity tabs on the CVE detail page and the overview deployments table.

**Architecture:** A new `DeploymentStatus` proto enum is stored on the `Deployment` message and set on write paths; the frontend reads a `deploymentStatus` URL param (defaulting to `DEPLOYED`) and passes it through a new `getDeploymentStatusQueryString` utility to all GQL query variables. A shared `DeploymentStatusFilter` component owns the URL write; pages consume `useDeploymentStatus` (read-only) to build queries.

**Tech Stack:** Go 1.24, protobuf/buf, React 18, TypeScript, PatternFly 6, Apollo Client, Vitest.

**Spec:** `docs/superpowers/specs/2026-03-31-deployment-status-filter-design.md`

---

## File Map

| File | Action |
|------|--------|
| `proto/storage/deployment.proto` | Add enum + field |
| `central/deployment/datastore/datastore_impl.go` | Set status on write paths |
| `ui/apps/platform/src/types/deploymentStatus.ts` | **New** — type, const, guard |
| `ui/apps/platform/src/types/deploymentStatus.test.ts` | **New** — type guard tests |
| `ui/apps/platform/src/hooks/useDeploymentStatus.ts` | **New** — read-only URL hook |
| `ui/apps/platform/src/hooks/useDeploymentStatus.test.ts` | **New** — hook tests |
| `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.tsx` | Add `getDeploymentStatusQueryString` |
| `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.test.ts` | Extend with new function tests |
| `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.tsx` | **New** — ToggleGroup component |
| `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx` | **New** — component tests |
| `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage.tsx` | Replace `showDeleted`/`Switch` |
| `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/DeploymentsTableContainer.tsx` | Replace `showDeleted`/`Switch` |

---

## Task 1: Proto enum + stored field

**Files:**
- Modify: `proto/storage/deployment.proto`

> **Background:** The project uses `buf` to generate Go from proto. The `@gotags` annotation on a field line controls how it appears in the StackRox search layer — it must be on the **same line** as the field, not on a separate comment line. The `// Next available tag:` comment in the `Deployment` message header is a convention that must be kept up to date.

- [ ] **Step 1: Add the enum and field**

In `proto/storage/deployment.proto`:

1. Add the enum at file scope, after the `option java_package` line and before the `// Next available tag: 37` comment that opens the `Deployment` message. The file currently has two `option` lines followed immediately by the `Deployment` message — insert between them and the message:

```proto
enum DeploymentStatus {
    DEPLOYMENT_STATUS_UNSPECIFIED = 0; // Zero value; retained for backward compatibility.
    DEPLOYMENT_STATUS_DEPLOYED    = 1; // Deployment is active.
    DEPLOYMENT_STATUS_DELETED     = 2; // Deployment has been soft-deleted (tombstoned).
}
```

2. Inside the `Deployment` message, after the existing line `Tombstone tombstone = 36;`, add:

```proto
  DeploymentStatus deployment_status = 37; // @gotags: search:"Deployment Status,hidden"
```

3. In the `Deployment` message header comment, update `// Next available tag: 37` to `// Next available tag: 38`. This comment is on the line immediately before `message Deployment {`.

- [ ] **Step 2: Run code generation**

```bash
# from repo root
make proto-generated-srcs
```

Expected: exits 0. Generated files under `generated/storage/` are updated (e.g., `deployment.pb.go` gains the new enum and field).

- [ ] **Step 3: Verify build**

```bash
go build ./central/deployment/...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add proto/storage/deployment.proto generated/
git commit -m "feat(storage): add DeploymentStatus enum to Deployment proto

Add DEPLOYMENT_STATUS_UNSPECIFIED/DEPLOYED/DELETED enum and
deployment_status field (tag 37, hidden search label) to the Deployment
message. No migration — existing records keep the zero value
(UNSPECIFIED), treated as DEPLOYED via the tombstone null-check.

User prompt: add DeploymentStatus proto enum for the deployment status
filter feature; add unspecified value for backward compatibility.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Set DeploymentStatus on write paths

**Files:**
- Modify: `central/deployment/datastore/datastore_impl.go`

> **Background:** `upsertDeployment` (lowercase, internal) sets all deployment fields before storing — find it by searching for `func (ds *datastoreImpl) upsertDeployment`. `TombstoneDeployment` soft-deletes by setting `deployment.Tombstone` inside a `DoStatusWithLock` closure — find the closure by searching for `deployment.Tombstone = &storage.Tombstone{`. The generated constants from Task 1 are `storage.DeploymentStatus_DEPLOYMENT_STATUS_DEPLOYED` and `storage.DeploymentStatus_DEPLOYMENT_STATUS_DELETED`.
>
> Note: the codebase uses direct field assignment on proto structs (e.g., `deployment.Tombstone = nil`, `deployment.RiskScore = ...`) — this is standard Go practice for setting proto fields, as Go protobuf does not generate setter methods.

- [ ] **Step 1: Set DEPLOYED in upsertDeployment**

Inside `upsertDeployment`, find the resurrection block that ends with `deployment.Tombstone = nil`. Immediately after that closing brace, add:

```go
// Mark as active, clearing any previous DELETED status from before resurrection.
deployment.DeploymentStatus = storage.DeploymentStatus_DEPLOYMENT_STATUS_DEPLOYED
```

- [ ] **Step 2: Set DELETED in TombstoneDeployment**

Inside the `DoStatusWithLock` closure in `TombstoneDeployment`, find the line `deployment.Tombstone = &storage.Tombstone{...}` (the multi-line struct literal ends with a closing `}`). Immediately after that closing `}`, add:

```go
deployment.DeploymentStatus = storage.DeploymentStatus_DEPLOYMENT_STATUS_DELETED
```

- [ ] **Step 3: Build and verify**

```bash
go build ./central/deployment/...
```

Expected: exits 0.

- [ ] **Step 4: Run existing datastore tests**

```bash
go test ./central/deployment/datastore/... -count=1
```

Expected: PASS (no new failures).

- [ ] **Step 5: Commit**

```bash
git add central/deployment/datastore/datastore_impl.go
git commit -m "feat(deployment): set DeploymentStatus on upsert and tombstone

Set DEPLOYMENT_STATUS_DEPLOYED in upsertDeployment and
DEPLOYMENT_STATUS_DELETED in TombstoneDeployment so newly written
records carry the correct status. Existing records retain UNSPECIFIED.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Frontend types, hook, and search utility

**Files:**
- Create: `ui/apps/platform/src/types/deploymentStatus.ts`
- Create: `ui/apps/platform/src/types/deploymentStatus.test.ts`
- Create: `ui/apps/platform/src/hooks/useDeploymentStatus.ts`
- Create: `ui/apps/platform/src/hooks/useDeploymentStatus.test.ts`
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.tsx`
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.test.ts`

> **Background:** `useURLStringUnion(key, values)` reads/writes a URL query param, defaulting to `values[0]`. See `hooks/useURLStringUnion.ts` for the API and `hooks/useURLStringUnion.test.tsx` for the test pattern (uses `MemoryRouter` + `renderHook` from `@testing-library/react` and `actAndFlushTaskQueue` from `test-utils/flushTaskQueue`). The existing `getVulnStateScopedQueryString` in `searchUtils.tsx` is the pattern to follow for the new search util. Frontend tests run with Vitest: `npm run test -- <file>` from `ui/apps/platform/`.

- [ ] **Step 1: Create `src/types/deploymentStatus.ts`**

```typescript
export const deploymentStatuses = ['DEPLOYED', 'DELETED'] as const;
export type DeploymentStatus = (typeof deploymentStatuses)[number];

export function isDeploymentStatus(value: unknown): value is DeploymentStatus {
    return deploymentStatuses.some((s) => s === value);
}
```

- [ ] **Step 2: Create `src/types/deploymentStatus.test.ts`**

```typescript
import { isDeploymentStatus } from './deploymentStatus';

describe('isDeploymentStatus', () => {
    it('returns true for valid values', () => {
        expect(isDeploymentStatus('DEPLOYED')).toBe(true);
        expect(isDeploymentStatus('DELETED')).toBe(true);
    });

    it('returns false for invalid values', () => {
        expect(isDeploymentStatus('UNSPECIFIED')).toBe(false);
        expect(isDeploymentStatus('')).toBe(false);
        expect(isDeploymentStatus(null)).toBe(false);
        expect(isDeploymentStatus(undefined)).toBe(false);
        expect(isDeploymentStatus(42)).toBe(false);
    });
});
```

- [ ] **Step 3: Run and confirm passing**

```bash
cd ui/apps/platform
npm run test -- src/types/deploymentStatus.test.ts
```

Expected: 2 tests PASS.

- [ ] **Step 4: Create `src/hooks/useDeploymentStatus.ts`**

```typescript
import useURLStringUnion from 'hooks/useURLStringUnion';
import { deploymentStatuses } from 'types/deploymentStatus';
import type { DeploymentStatus } from 'types/deploymentStatus';

/**
 * Reads the `deploymentStatus` URL parameter, defaulting to 'DEPLOYED'.
 * Read-only — mutations go through DeploymentStatusFilter which calls
 * useURLStringUnion directly.
 */
export default function useDeploymentStatus(): DeploymentStatus {
    const [status] = useURLStringUnion('deploymentStatus', deploymentStatuses);
    return status;
}
```

- [ ] **Step 5: Create `src/hooks/useDeploymentStatus.test.ts`**

Follow the same pattern as `hooks/useURLStringUnion.test.tsx` (MemoryRouter wrapper, renderHook, actAndFlushTaskQueue):

```typescript
import { MemoryRouter, useLocation } from 'react-router-dom-v5-compat';
import { renderHook } from '@testing-library/react';
import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';
import { URLSearchParams } from 'url';

import useDeploymentStatus from './useDeploymentStatus';

test('defaults to DEPLOYED when no URL param is set', async () => {
    let testLocation;
    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return useDeploymentStatus();
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['']}>{children}</MemoryRouter>
            ),
        }
    );

    await actAndFlushTaskQueue(() => {});

    expect(result.current).toBe('DEPLOYED');
    const params = new URLSearchParams(testLocation.search);
    expect(params.get('deploymentStatus')).toBe('DEPLOYED');
});

test('reflects DELETED when deploymentStatus=DELETED is in the URL', async () => {
    const { result } = renderHook(() => useDeploymentStatus(), {
        wrapper: ({ children }) => (
            <MemoryRouter initialEntries={['?deploymentStatus=DELETED']}>
                {children}
            </MemoryRouter>
        ),
    });

    await actAndFlushTaskQueue(() => {});

    expect(result.current).toBe('DELETED');
});

test('falls back to DEPLOYED for an invalid URL param value', async () => {
    const { result } = renderHook(() => useDeploymentStatus(), {
        wrapper: ({ children }) => (
            <MemoryRouter initialEntries={['?deploymentStatus=BOGUS']}>
                {children}
            </MemoryRouter>
        ),
    });

    await actAndFlushTaskQueue(() => {});

    expect(result.current).toBe('DEPLOYED');
});
```

- [ ] **Step 6: Run hook tests**

```bash
npm run test -- src/hooks/useDeploymentStatus.test.ts
```

Expected: 3 tests PASS.

- [ ] **Step 7: Write failing test for `getDeploymentStatusQueryString`**

In `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.test.ts`, the file already has a named import from `'./searchUtils'`. **Add `getDeploymentStatusQueryString` to the existing import destructure** (do not add a second `import` from the same module — that is a lint error). Then add a new `describe` block at the bottom of the file:

```typescript
// In the existing import at the top of the file, add getDeploymentStatusQueryString:
// import { getWorkloadEntityPagePath, ..., getDeploymentStatusQueryString } from './searchUtils';

describe('getDeploymentStatusQueryString', () => {
    it('returns baseQuery unchanged when status is DEPLOYED', () => {
        expect(getDeploymentStatusQueryString('CVE:CVE-2025-1234', 'DEPLOYED')).toBe(
            'CVE:CVE-2025-1234'
        );
    });

    it('appends tombstone filter when status is DELETED', () => {
        expect(getDeploymentStatusQueryString('CVE:CVE-2025-1234', 'DELETED')).toBe(
            'CVE:CVE-2025-1234+Tombstone Deleted At:*'
        );
    });

    it('handles empty baseQuery for DELETED (no leading +)', () => {
        expect(getDeploymentStatusQueryString('', 'DELETED')).toBe('Tombstone Deleted At:*');
    });

    it('handles empty baseQuery for DEPLOYED', () => {
        expect(getDeploymentStatusQueryString('', 'DEPLOYED')).toBe('');
    });
});
```

- [ ] **Step 8: Run to confirm it fails**

```bash
npm run test -- src/Containers/Vulnerabilities/utils/searchUtils.test.ts
```

Expected: FAIL — `getDeploymentStatusQueryString` is not exported yet.

- [ ] **Step 9: Add `getDeploymentStatusQueryString` to searchUtils.tsx**

Add the type import near the top of `searchUtils.tsx` with the other type imports:

```typescript
import type { DeploymentStatus } from 'types/deploymentStatus';
```

Then add the function after `getVulnStateScopedQueryString` (search for that function to locate the right spot):

```typescript
/**
 * Wraps a base query string to scope results by deployment status.
 * DELETED appends '+Tombstone Deleted At:*' to opt into tombstoned records.
 * DEPLOYED returns the base query unchanged — tombstone exclusion is applied
 * by the backend view layer.
 * The '+' character is the backend's AND-conjunction separator.
 */
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

- [ ] **Step 10: Run tests to confirm passing**

```bash
npm run test -- src/Containers/Vulnerabilities/utils/searchUtils.test.ts
```

Expected: all tests PASS (existing + 4 new).

- [ ] **Step 11: TypeScript check**

```bash
npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 12: Commit**

```bash
git add \
  ui/apps/platform/src/types/deploymentStatus.ts \
  ui/apps/platform/src/types/deploymentStatus.test.ts \
  ui/apps/platform/src/hooks/useDeploymentStatus.ts \
  ui/apps/platform/src/hooks/useDeploymentStatus.test.ts \
  ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.tsx \
  ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.test.ts
git commit -m "feat(vm-ui): add DeploymentStatus types, hook, and search util

- types/deploymentStatus.ts: DeploymentStatus union type + isDeploymentStatus guard
- hooks/useDeploymentStatus.ts: read-only URL-backed hook (mirrors useVulnerabilityState)
- searchUtils.tsx: getDeploymentStatusQueryString appends tombstone filter for DELETED

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: DeploymentStatusFilter component + tests

**Files:**
- Create: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.tsx`
- Create: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx`

> **Background:** PatternFly `ToggleGroup`/`ToggleGroupItem` are in `@patternfly/react-core`. Each `ToggleGroupItem` needs an `id` prop for correct ARIA labeling — omitting it produces a console warning. The component writes URL state directly via `useURLStringUnion`; the `onChange` prop lets parents reset pagination. For component tests, use `render` from `@testing-library/react`, wrap in `MemoryRouter`, and use `userEvent` for interactions.

- [ ] **Step 1: Create the component**

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
                onChange={() => {
                    setStatus('DEPLOYED');
                    onChange?.();
                }}
            />
            <ToggleGroupItem
                id="deployment-status-deleted"
                text="Deleted"
                isSelected={status === 'DELETED'}
                onChange={() => {
                    setStatus('DELETED');
                    onChange?.();
                }}
            />
        </ToggleGroup>
    );
}
```

- [ ] **Step 2: Create `DeploymentStatusFilter.test.tsx`**

```tsx
import { MemoryRouter } from 'react-router-dom-v5-compat';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import DeploymentStatusFilter from './DeploymentStatusFilter';

function renderWithRouter(ui: React.ReactElement, initialEntry = '') {
    return render(
        <MemoryRouter initialEntries={[initialEntry]}>{ui}</MemoryRouter>
    );
}

describe('DeploymentStatusFilter', () => {
    it('renders both Deployed and Deleted options', () => {
        renderWithRouter(<DeploymentStatusFilter />);
        expect(screen.getByText('Deployed')).toBeInTheDocument();
        expect(screen.getByText('Deleted')).toBeInTheDocument();
    });

    it('selects Deployed by default', () => {
        renderWithRouter(<DeploymentStatusFilter />);
        // PatternFly ToggleGroupItem marks the selected item with aria-pressed="true"
        expect(screen.getByText('Deployed').closest('button')).toHaveAttribute(
            'aria-pressed',
            'true'
        );
        expect(screen.getByText('Deleted').closest('button')).toHaveAttribute(
            'aria-pressed',
            'false'
        );
    });

    it('selects Deleted when deploymentStatus=DELETED is in the URL', () => {
        renderWithRouter(<DeploymentStatusFilter />, '?deploymentStatus=DELETED');
        expect(screen.getByText('Deleted').closest('button')).toHaveAttribute(
            'aria-pressed',
            'true'
        );
    });

    it('calls onChange when a different option is selected', async () => {
        const onChange = vi.fn();
        renderWithRouter(<DeploymentStatusFilter onChange={onChange} />);
        await userEvent.click(screen.getByText('Deleted'));
        expect(onChange).toHaveBeenCalledTimes(1);
    });
});
```

- [ ] **Step 3: Run tests**

```bash
npm run test -- src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx
```

Expected: 4 tests PASS.

- [ ] **Step 4: TypeScript check**

```bash
npm run tsc -- --noEmit
```

Expected: no new errors.

- [ ] **Step 5: Commit**

```bash
git add \
  ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.tsx \
  ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx
git commit -m "feat(vm-ui): add DeploymentStatusFilter ToggleGroup component

URL-persisted Deployed/Deleted toggle that replaces the session-local
'Show deleted' Switch. Parents call onChange to reset pagination.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Wire ImageCvePage

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage.tsx`

> **Background:** The current file imports `useState` (line 1) and `Switch` (line 13). It has `showDeleted` state and `isTombstonesEnabled` (search for them). The top-level `query` variable (search for `const query = getVulnStateScopedQueryString`) and `getDeploymentSearchQuery` both need to wrap through `getDeploymentStatusQueryString`. `isTombstonesEnabled` is kept — it gates the filter UI and the `isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'` guard that ensures the URL param is ignored when the flag is off.

- [ ] **Step 1: Update imports**

- Remove `useState` from the React import (keep `useEffect`): change line 1 to `import { useEffect } from 'react';`
- Remove `Switch` from the PatternFly import block (keep all other PatternFly imports).
- Add these three imports anywhere in the import section:
  ```typescript
  import useDeploymentStatus from 'hooks/useDeploymentStatus';
  import DeploymentStatusFilter from '../components/DeploymentStatusFilter';
  import { getDeploymentStatusQueryString } from '../../utils/searchUtils';
  ```

- [ ] **Step 2: Replace showDeleted state**

Find and remove only the `showDeleted` line:
```typescript
const [showDeleted, setShowDeleted] = useState(false);
```

Keep `isTombstonesEnabled` (it still gates the filter UI). Add below the remaining two lines:
```typescript
const deploymentStatus = useDeploymentStatus();
```

So the block now reads:
```typescript
const { isFeatureFlagEnabled } = useFeatureFlags();
const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');
const deploymentStatus = useDeploymentStatus();
```

- [ ] **Step 3: Update the top-level `query` variable**

Find `const query = getVulnStateScopedQueryString(`. Wrap it:

```typescript
const query = getDeploymentStatusQueryString(
    getVulnStateScopedQueryString(
        {
            CVE: [exactCveIdSearchRegex],
            ...baseSearchFilter,
            ...querySearchFilter,
        },
        vulnerabilityState
    ),
    isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
);
```

- [ ] **Step 4: Update `getDeploymentSearchQuery`**

Find `function getDeploymentSearchQuery`. Replace the function body:

```typescript
function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
    const filters = { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter };
    if (severity) {
        filters.SEVERITY = [severity];
    }
    const base = getVulnStateScopedQueryString(filters, vulnerabilityState);
    return getDeploymentStatusQueryString(
        base,
        isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
    );
}
```

- [ ] **Step 5: Replace the Switch JSX with DeploymentStatusFilter**

Find and remove the entire Switch SplitItem block (search for `entityTab === 'Deployment' && isTombstonesEnabled`):
```tsx
{entityTab === 'Deployment' && isTombstonesEnabled && (
    <SplitItem>
        <Switch ... />
    </SplitItem>
)}
```

Replace with — placed in the `Split` before the ColumnManagementButton `SplitItem`:
```tsx
{isTombstonesEnabled && (
    <SplitItem>
        <DeploymentStatusFilter onChange={() => setPage(1)} />
    </SplitItem>
)}
```

The filter is no longer scoped to `entityTab === 'Deployment'` because it now scopes the top-level `query` (used by all entity tabs).

- [ ] **Step 6: TypeScript check**

```bash
cd ui/apps/platform
npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 7: Commit**

```bash
git add ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage.tsx
git commit -m "feat(vm-ui): replace showDeleted switch with DeploymentStatusFilter in ImageCvePage

- Remove showDeleted useState and Switch JSX
- Add deploymentStatus from useDeploymentStatus hook (URL-persisted)
- Wrap top-level query and getDeploymentSearchQuery with
  getDeploymentStatusQueryString so all entity tabs (CVE, Image,
  Deployment) are scoped by the filter
- Filter appears above entity toggle, gated on ROX_DEPLOYMENT_TOMBSTONES

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Wire DeploymentsTableContainer

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/DeploymentsTableContainer.tsx`

> **Background:** The file currently has `import { useState } from 'react';` (line 1) and `import { Switch, ToolbarItem } from '@patternfly/react-core';` (line 2). It has `showDeleted` state and an inline tombstone query string. `ToolbarItem` is still needed (wraps `DeploymentStatusFilter`), so keep it.

- [ ] **Step 1: Update imports**

- Remove the entire `import { useState } from 'react';` line (useState is no longer needed).
- In the PatternFly import, remove `Switch` but keep `ToolbarItem`. Change line 2 from:
  ```typescript
  import { Switch, ToolbarItem } from '@patternfly/react-core';
  ```
  to:
  ```typescript
  import { ToolbarItem } from '@patternfly/react-core';
  ```
- Add:
  ```typescript
  import useDeploymentStatus from 'hooks/useDeploymentStatus';
  import { getDeploymentStatusQueryString } from '../../utils/searchUtils';
  import DeploymentStatusFilter from '../components/DeploymentStatusFilter';
  ```

- [ ] **Step 2: Replace state and query**

Find and remove these lines:
```typescript
const [showDeleted, setShowDeleted] = useState(false);

const deploymentsQueryString =
    isTombstonesEnabled && showDeleted
        ? [workloadCvesScopedQueryString, 'Tombstone Deleted At:*'].filter(Boolean).join('+')
        : workloadCvesScopedQueryString;
```

Replace with:
```typescript
const deploymentStatus = useDeploymentStatus();

const deploymentsQueryString = getDeploymentStatusQueryString(
    workloadCvesScopedQueryString,
    isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
);
```

- [ ] **Step 3: Replace Switch JSX with DeploymentStatusFilter**

Find the `ToolbarItem` containing the `Switch` (search for `<Switch` in this file). Replace the entire `{isTombstonesEnabled && ( <ToolbarItem> <Switch ... /> </ToolbarItem> )}` block with:

```tsx
{isTombstonesEnabled && (
    <ToolbarItem>
        <DeploymentStatusFilter onChange={() => pagination.setPage(1)} />
    </ToolbarItem>
)}
```

- [ ] **Step 4: TypeScript check**

```bash
cd ui/apps/platform
npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 5: Commit**

```bash
git add ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/DeploymentsTableContainer.tsx
git commit -m "feat(vm-ui): replace showDeleted switch with DeploymentStatusFilter in overview

Remove showDeleted useState and Switch; use URL-persisted deploymentStatus
from useDeploymentStatus hook and getDeploymentStatusQueryString to scope
the deployments table query. Filter gated on ROX_DEPLOYMENT_TOMBSTONES.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Verification Checklist

After all tasks are complete:

- [ ] `go build ./...` from repo root exits 0.
- [ ] `npm run tsc -- --noEmit` from `ui/apps/platform/` exits 0 (pre-existing errors only).
- [ ] All new tests pass:
  ```bash
  npm run test -- \
    src/types/deploymentStatus.test.ts \
    src/hooks/useDeploymentStatus.test.ts \
    src/Containers/Vulnerabilities/utils/searchUtils.test.ts \
    src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx
  ```
- [ ] With `ROX_DEPLOYMENT_TOMBSTONES=false`: no filter UI visible; tombstone exclusion always applied; `?deploymentStatus=DELETED` in URL has no effect.
- [ ] With `ROX_DEPLOYMENT_TOMBSTONES=true`:
  - "Deployed" (default): only active deployments, images from active deployments, CVEs from active deployments.
  - "Deleted": only tombstoned deployments and associated images/CVEs.
  - Selecting a filter option updates the URL (`?deploymentStatus=DELETED`).
  - Refreshing the page preserves the selected filter.
  - Filter state is shared between CVE detail page and overview page (same URL param).
