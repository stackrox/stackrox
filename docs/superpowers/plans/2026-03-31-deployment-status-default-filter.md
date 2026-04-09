# Deployment Status Default Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the standalone `DeploymentStatusFilter` ToggleGroup with multi-select checkboxes in the Default Filters modal, following the same pattern as CVE severity and CVE status filters.

**Architecture:** `DEPLOYMENT_STATUS` is added to the `DefaultFilters` schema and URL search filter. The `getDeploymentStatusScopedQueryString` utility wraps `workloadCvesScopedQueryString` on the overview page (scoping all three entity tabs) and wraps `getDeploymentSearchQuery` on the CVE detail page. All old `DeploymentStatusFilter`, `useDeploymentStatus`, and `getDeploymentStatusQueryString` code is removed.

**Tech Stack:** React 18, TypeScript, PatternFly 6, Apollo Client, Yup, Vitest. No backend changes.

**Spec:** `docs/superpowers/specs/2026-03-31-deployment-status-default-filter-design.md`

---

## File Map

| File | Action |
|------|--------|
| `src/Containers/Vulnerabilities/types.ts` | Add `deploymentStatusLabels` type + Yup schema extension |
| `src/Containers/Vulnerabilities/utils/searchUtils.tsx` | Add `getDeploymentStatusScopedQueryString`; remove old function |
| `src/Containers/Vulnerabilities/utils/searchUtils.test.ts` | Replace old tests with new 5-case tests |
| `src/Containers/Vulnerabilities/WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx` | Add flag, extend defaultStorage + merge, wrap query, pass prop |
| `src/hooks/useAnalytics.ts` | Extend `WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED` event properties type |
| `src/Containers/Vulnerabilities/WorkloadCves/components/DefaultFilterModal.tsx` | Add `isTombstonesEnabled` prop + "Deployment status" form group |
| `src/Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage.tsx` | Replace hook with URL filter; strip DEPLOYMENT_STATUS |
| `src/Containers/Vulnerabilities/WorkloadCves/Overview/DeploymentsTableContainer.tsx` | Remove per-container tombstone logic |
| **Delete:** `src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.tsx` | Remove component |
| **Delete:** `src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx` | Remove tests |
| **Delete:** `src/hooks/useDeploymentStatus.ts` | Remove hook |
| **Delete:** `src/hooks/useDeploymentStatus.test.tsx` | Remove hook tests |
| **Delete:** `src/types/deploymentStatus.ts` | Merged into `Containers/Vulnerabilities/types.ts` |
| **Delete:** `src/types/deploymentStatus.test.ts` | Remove tests |

All paths are relative to `ui/apps/platform/`.

---

## Task 1: Types + search utility

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/types.ts`
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.tsx`
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.test.ts`
- Delete: `ui/apps/platform/src/types/deploymentStatus.ts`
- Delete: `ui/apps/platform/src/types/deploymentStatus.test.ts`

> **Background:** `types.ts` already has `VulnerabilitySeverityLabel` (lines 5-15) and `FixableStatus` (lines 17-21) as the pattern to follow. `searchUtils.tsx` has `getDeploymentStatusQueryString` at line 245 to remove. The old test cases for `getDeploymentStatusQueryString` are in `searchUtils.test.ts` at lines 126-146. Vitest globals (`describe`, `it`, `expect`) are enabled — no import needed. `+` is the AND-conjunction separator; comma within one field value is an OR disjunction.

- [ ] **Step 1: Add `deploymentStatusLabels` to `types.ts`**

In `ui/apps/platform/src/Containers/Vulnerabilities/types.ts`, after the `isFixableStatus` block (around line 21), add:

```typescript
export const deploymentStatusLabels = ['Deployed', 'Deleted'] as const;
export type DeploymentStatusLabel = (typeof deploymentStatusLabels)[number];
export function isDeploymentStatusLabel(value: unknown): value is DeploymentStatusLabel {
    return deploymentStatusLabels.some((s) => s === value);
}
```

- [ ] **Step 2: Extend Yup schema in `types.ts`**

In the same file, find the `defaultFilters: yup.object({` block (around line 38). Add `DEPLOYMENT_STATUS` after the `FIXABLE` line:

```typescript
DEPLOYMENT_STATUS: yup
    .array(yup.string().required().oneOf(deploymentStatusLabels))
    .required(),
```

- [ ] **Step 3: Replace old test cases in `searchUtils.test.ts`**

Find the `describe('getDeploymentStatusQueryString', ...)` block (lines 126-146). Replace the entire block **and update the import** on line 5 to remove `getDeploymentStatusQueryString` and add `getDeploymentStatusScopedQueryString`:

In the existing import on line 5, change:
```typescript
// from:
    getDeploymentStatusQueryString,
// to:
    getDeploymentStatusScopedQueryString,
```

Replace the old describe block with:

```typescript
describe('getDeploymentStatusScopedQueryString', () => {
    it('returns baseQuery unchanged when status is Deployed only', () => {
        expect(getDeploymentStatusScopedQueryString('CVE:CVE-2025-1234', ['Deployed'])).toBe(
            'CVE:CVE-2025-1234'
        );
    });

    it('appends Tombstone Deleted At:* when status is Deleted only', () => {
        expect(getDeploymentStatusScopedQueryString('CVE:CVE-2025-1234', ['Deleted'])).toBe(
            'CVE:CVE-2025-1234+Tombstone Deleted At:*'
        );
    });

    it('appends Tombstone Deleted At:*,-* when both Deployed and Deleted are selected', () => {
        expect(
            getDeploymentStatusScopedQueryString('CVE:CVE-2025-1234', ['Deployed', 'Deleted'])
        ).toBe('CVE:CVE-2025-1234+Tombstone Deleted At:*,-*');
    });

    it('returns baseQuery unchanged when selectedStatuses is undefined', () => {
        expect(getDeploymentStatusScopedQueryString('CVE:CVE-2025-1234', undefined)).toBe(
            'CVE:CVE-2025-1234'
        );
    });

    it('returns baseQuery unchanged when selectedStatuses is empty', () => {
        expect(getDeploymentStatusScopedQueryString('CVE:CVE-2025-1234', [])).toBe(
            'CVE:CVE-2025-1234'
        );
    });

    it('handles empty baseQuery for Deleted only', () => {
        expect(getDeploymentStatusScopedQueryString('', ['Deleted'])).toBe(
            'Tombstone Deleted At:*'
        );
    });
});
```

- [ ] **Step 4: Run tests to confirm failure**

```bash
cd ui/apps/platform && npm run test -- src/Containers/Vulnerabilities/utils/searchUtils.test.ts
```

Expected: FAIL — `getDeploymentStatusScopedQueryString` is not exported yet.

- [ ] **Step 5: Add `getDeploymentStatusScopedQueryString` to `searchUtils.tsx`**

First, add the import for `DeploymentStatusLabel` at the top of `searchUtils.tsx` with the other type imports:

```typescript
import type { DeploymentStatusLabel } from '../types';
```

Then find `getDeploymentStatusQueryString` (line 245) and **replace the entire function** with:

```typescript
/**
 * Wraps a base query string to scope results by deployment status.
 * - `['Deployed']` or unset: no addition (view default excludes tombstoned records).
 * - `['Deleted']`: appends `+Tombstone Deleted At:*` (IS NOT NULL → only tombstoned).
 * - `['Deployed', 'Deleted']`: appends `+Tombstone Deleted At:*,-*`.
 *   Comma-separated values for one field form a disjunction: (IS NOT NULL OR IS NULL) = all rows.
 * The '+' character is the backend's AND-conjunction separator between fields.
 */
export function getDeploymentStatusScopedQueryString(
    baseQuery: string,
    selectedStatuses: DeploymentStatusLabel[] | undefined
): string {
    const showDeployed =
        !selectedStatuses || selectedStatuses.length === 0 || selectedStatuses.includes('Deployed');
    const showDeleted = selectedStatuses?.includes('Deleted') ?? false;

    if (showDeployed && showDeleted) {
        return [baseQuery, 'Tombstone Deleted At:*,-*'].filter(Boolean).join('+');
    }
    if (showDeleted) {
        return [baseQuery, 'Tombstone Deleted At:*'].filter(Boolean).join('+');
    }
    return baseQuery;
}
```

Also remove the old `import type { DeploymentStatus } from 'types/deploymentStatus'` line if present in `searchUtils.tsx` (search for it first).

- [ ] **Step 6: Run tests to confirm passing**

```bash
npm run test -- src/Containers/Vulnerabilities/utils/searchUtils.test.ts
```

Expected: all tests PASS (6 new tests replace 4 old ones; net +2 tests).

- [ ] **Step 7: Delete old standalone files**

```bash
git rm ui/apps/platform/src/types/deploymentStatus.ts
git rm ui/apps/platform/src/types/deploymentStatus.test.ts
```

- [ ] **Step 8: TypeScript check**

```bash
npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 9: Commit**

```bash
git add \
  ui/apps/platform/src/Containers/Vulnerabilities/types.ts \
  ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.tsx \
  ui/apps/platform/src/Containers/Vulnerabilities/utils/searchUtils.test.ts
git commit -m "feat(vm-ui): add DeploymentStatusLabel to types and getDeploymentStatusScopedQueryString

- Add deploymentStatusLabels, DeploymentStatusLabel type, isDeploymentStatusLabel
  to Containers/Vulnerabilities/types.ts (extends DefaultFilters via Yup schema)
- Replace getDeploymentStatusQueryString with getDeploymentStatusScopedQueryString
  supporting multi-select: Deployed, Deleted, both (disjunction), or unset
- Remove standalone types/deploymentStatus.ts (merged into Vulnerabilities/types.ts)

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: DefaultFilterModal — add Deployment status form group

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DefaultFilterModal.tsx`
- Modify: `ui/apps/platform/src/hooks/useAnalytics.ts`

> **Background:** `DefaultFilterModal.tsx` currently accepts `{ defaultFilters: DefaultFilters; setLocalStorage: (values: DefaultFilters) => void }` as props. It has `handleFixableChange` (lines 69-77) as the pattern to follow for `handleDeploymentStatusChange`. The `totalFilters` badge count is at line 37. The form ends with a "CVE status" `FormGroup` (line 153) — the new group goes after it, before `</Form>`. `DEPLOYMENT_STATUS` is now part of `DefaultFilters` (from Task 1), so Formik's `values.DEPLOYMENT_STATUS` is available.
>
> **Analytics:** `analyticsTrackDefaultFilters` (lines 11-27) calls `analyticsTrack` for event `WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED`. The event's `properties` type in `src/hooks/useAnalytics.ts` (lines 229-240) is an exact object shape — adding new analytics properties requires updating BOTH the type definition AND the tracking call. Do this BEFORE adding the DEPLOYMENT_STATUS form group, or TypeScript will error.

- [ ] **Step 1: Update `useAnalytics.ts` — extend the analytics event type**

In `ui/apps/platform/src/hooks/useAnalytics.ts`, find the `WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED` event type (around line 229-240). Add two new properties inside the `properties` object:

```typescript
DEPLOYMENT_STATUS_DEPLOYED: AnalyticsBoolean;
DEPLOYMENT_STATUS_DELETED: AnalyticsBoolean;
```

The full updated block:
```typescript
| {
      event: typeof WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED;
      properties: {
          SEVERITY_CRITICAL: AnalyticsBoolean;
          SEVERITY_IMPORTANT: AnalyticsBoolean;
          SEVERITY_MODERATE: AnalyticsBoolean;
          SEVERITY_LOW: AnalyticsBoolean;
          SEVERITY_UNKNOWN: AnalyticsBoolean;
          CVE_STATUS_FIXABLE: AnalyticsBoolean;
          CVE_STATUS_NOT_FIXABLE: AnalyticsBoolean;
          DEPLOYMENT_STATUS_DEPLOYED: AnalyticsBoolean;
          DEPLOYMENT_STATUS_DELETED: AnalyticsBoolean;
      };
  }
```

- [ ] **Step 2: Update `analyticsTrackDefaultFilters` in `DefaultFilterModal.tsx`**

Find `analyticsTrackDefaultFilters` (lines 11-27). Add the two new properties to the `properties` object (gated on whether the flag is on — but since `analyticsTrackDefaultFilters` does not receive the flag, just include them unconditionally as `0` when the status array is empty):

```typescript
DEPLOYMENT_STATUS_DEPLOYED: filters.DEPLOYMENT_STATUS.includes('Deployed') ? 1 : 0,
DEPLOYMENT_STATUS_DELETED: filters.DEPLOYMENT_STATUS.includes('Deleted') ? 1 : 0,
```

- [ ] **Step 3: Add `isTombstonesEnabled` prop**

Find the `type DefaultFilterModalProps` block (around line 29). Add the new prop:

```typescript
type DefaultFilterModalProps = {
    defaultFilters: DefaultFilters;
    setLocalStorage: (values: DefaultFilters) => void;
    isTombstonesEnabled: boolean;
};
```

Update the function signature to destructure the new prop:

```typescript
function DefaultFilterModal({ defaultFilters, setLocalStorage, isTombstonesEnabled }: DefaultFilterModalProps) {
```

- [ ] **Step 4: Extend `totalFilters` count**

Find `const totalFilters = defaultFilters.SEVERITY.length + defaultFilters.FIXABLE.length;` (line 37). Replace with:

```typescript
const totalFilters =
    defaultFilters.SEVERITY.length +
    defaultFilters.FIXABLE.length +
    (isTombstonesEnabled ? defaultFilters.DEPLOYMENT_STATUS.length : 0);
```

- [ ] **Step 5: Add `handleDeploymentStatusChange`**

After the `handleFixableChange` function (around line 77), add:

```typescript
function handleDeploymentStatusChange(
    status: DeploymentStatusLabel,
    isChecked: boolean
) {
    let newValues = [...values.DEPLOYMENT_STATUS];
    if (isChecked) {
        newValues.push(status);
    } else {
        newValues = newValues.filter((val) => val !== status);
    }
    setFieldValue('DEPLOYMENT_STATUS', newValues).catch(() => {});
}
```

Add the import for `DeploymentStatusLabel` to the existing import from `'../../types'`:

```typescript
import type { DefaultFilters, FixableStatus, VulnerabilitySeverityLabel, DeploymentStatusLabel } from '../../types';
```

- [ ] **Step 6: Add the "Deployment status" form group**

After the closing `</FormGroup>` of "CVE status" (around line 170), add:

```tsx
{isTombstonesEnabled && (
    <FormGroup label="Deployment status" isInline>
        <Checkbox
            label="Deployed"
            id="deployed-status"
            isChecked={values.DEPLOYMENT_STATUS.includes('Deployed')}
            onChange={(_event, isChecked) => {
                handleDeploymentStatusChange('Deployed', isChecked);
            }}
        />
        <Checkbox
            label="Deleted"
            id="deleted-status"
            isChecked={values.DEPLOYMENT_STATUS.includes('Deleted')}
            onChange={(_event, isChecked) => {
                handleDeploymentStatusChange('Deleted', isChecked);
            }}
        />
    </FormGroup>
)}
```

- [ ] **Step 7: TypeScript check**

```bash
cd ui/apps/platform && npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 8: Commit**

```bash
git add \
  ui/apps/platform/src/hooks/useAnalytics.ts \
  ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DefaultFilterModal.tsx
git commit -m "feat(vm-ui): add Deployment status checkboxes to DefaultFilterModal

- Update useAnalytics.ts: add DEPLOYMENT_STATUS_DEPLOYED/DELETED to the
  WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED event properties type
- Add isTombstonesEnabled prop to DefaultFilterModal
- When flag is on, show 'Deployed'/'Deleted' checkboxes after 'CVE status'
- Extend badge count and analytics tracking to include DEPLOYMENT_STATUS

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: WorkloadCvesOverviewPage — wire deployment status to workloadCvesScopedQueryString

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`

> **Background:** `defaultStorage` is at line 118. `mergeDefaultAndLocalFilters` is at line 89. `workloadCvesScopedQueryString` is computed at line 178. `DefaultFilterModal` is at line 431. The page already calls `isFeatureFlagEnabled` from `useFeatureFlags` (line 26/130) but does NOT yet declare `isTombstonesEnabled`. Import `getDeploymentStatusScopedQueryString` from `searchUtils` (already imported at line 49-54). Import `DeploymentStatusLabel` from `'../../types'` (already imported at line 47-48).

- [ ] **Step 1: Update imports**

In the existing `import type { DefaultFilters, VulnMgmtLocalStorage, WorkloadEntityTab } from '../../types'` (line 48), add `DeploymentStatusLabel`:

```typescript
import type { DefaultFilters, DeploymentStatusLabel, VulnMgmtLocalStorage, WorkloadEntityTab } from '../../types';
```

In the existing import from `'../../utils/searchUtils'` (lines 49-54), add `getDeploymentStatusScopedQueryString`.

- [ ] **Step 2: Add `DEPLOYMENT_STATUS` to `defaultStorage`**

Find `defaultStorage` (line 118). Add `DEPLOYMENT_STATUS: ['Deployed']`:

```typescript
const defaultStorage: VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: {
            SEVERITY: ['Critical', 'Important'],
            FIXABLE: ['Fixable'],
            DEPLOYMENT_STATUS: ['Deployed'],
        },
    },
} as const;
```

- [ ] **Step 3: Extend `mergeDefaultAndLocalFilters`**

Find the function (line 89-108). Add `DEPLOYMENT_STATUS` merge logic after the `FIXABLE` block:

```typescript
let DEPLOYMENT_STATUS = (filter.DEPLOYMENT_STATUS ?? []) as string[];
DEPLOYMENT_STATUS = difference(
    DEPLOYMENT_STATUS,
    oldDefaults.DEPLOYMENT_STATUS,
    newDefaults.DEPLOYMENT_STATUS
);
DEPLOYMENT_STATUS = DEPLOYMENT_STATUS.concat(newDefaults.DEPLOYMENT_STATUS);
```

Update the return statement to include `DEPLOYMENT_STATUS`:

```typescript
return { ...filter, SEVERITY, FIXABLE, DEPLOYMENT_STATUS };
```

- [ ] **Step 4: Declare `isTombstonesEnabled`**

Inside `WorkloadCvesOverviewPage` function body, after the `useFeatureFlags` destructure (around line 130), add:

```typescript
const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');
```

- [ ] **Step 5: Update `workloadCvesScopedQueryString`**

Find the current `workloadCvesScopedQueryString` computation (line 178-189). Replace with a two-step approach that strips `DEPLOYMENT_STATUS` from `querySearchFilter` and wraps the result:

```typescript
// Strip DEPLOYMENT_STATUS — it's consumed by getDeploymentStatusScopedQueryString,
// not by getVulnStateScopedQueryString.
const {
    DEPLOYMENT_STATUS: _deploymentStatus,
    ...querySearchFilterWithoutStatus
} = querySearchFilter;

const rawScopedQuery = isViewingWithCves
    ? getVulnStateScopedQueryString(
          {
              ...baseSearchFilter,
              ...querySearchFilterWithoutStatus,
          },
          currentVulnerabilityState
      )
    : getZeroCveScopedQueryString({
          ...baseSearchFilter,
          ...querySearchFilterWithoutStatus,
      });

const workloadCvesScopedQueryString = isTombstonesEnabled
    ? getDeploymentStatusScopedQueryString(
          rawScopedQuery,
          searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
      )
    : rawScopedQuery;
```

- [ ] **Step 6: Pass `isTombstonesEnabled` to `DefaultFilterModal`**

Find the `<DefaultFilterModal` at line 431. Add the prop:

```tsx
<DefaultFilterModal
    defaultFilters={localStorageValue.preferences.defaultFilters}
    setLocalStorage={updateDefaultFilters}
    isTombstonesEnabled={isTombstonesEnabled}
/>
```

- [ ] **Step 7: TypeScript check**

```bash
cd ui/apps/platform && npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 8: Commit**

```bash
git add ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx
git commit -m "feat(vm-ui): wire DEPLOYMENT_STATUS into workloadCvesScopedQueryString

- Add isTombstonesEnabled constant
- Add DEPLOYMENT_STATUS: ['Deployed'] to defaultStorage
- Extend mergeDefaultAndLocalFilters to sync DEPLOYMENT_STATUS
- Strip DEPLOYMENT_STATUS from querySearchFilter before getVulnStateScopedQueryString
- Wrap workloadCvesScopedQueryString with getDeploymentStatusScopedQueryString
- Pass isTombstonesEnabled to DefaultFilterModal

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: ImageCvePage — replace hook with URL search filter

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage.tsx`

> **Background:** Current state: `useDeploymentStatus` hook (line 42, 200), `getDeploymentStatusQueryString` (lines 48, 209, 275), `DeploymentStatusFilter` component (lines 50, 515-519) are all present. Line 63 imports from `'../../types'` — add `DeploymentStatusLabel` there. `isTombstonesEnabled` is declared at line 199 and must be kept. `searchFilter` and `querySearchFilter` are available (lines 207-208).

- [ ] **Step 1: Update imports**

Remove these three import lines:
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'` (line 42)
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'` (line 50)
- Remove `getDeploymentStatusQueryString` from the import at line 48 (replace with `getDeploymentStatusScopedQueryString`)

In line 63 (`import type { VulnerabilitySeverityLabel, WorkloadEntityTab } from '../../types'`), add `DeploymentStatusLabel`:

```typescript
import type { DeploymentStatusLabel, VulnerabilitySeverityLabel, WorkloadEntityTab } from '../../types';
```

- [ ] **Step 2: Remove `useDeploymentStatus` call and add queryFilter stripping**

Remove: `const deploymentStatus = useDeploymentStatus();` (line 200).

After `const querySearchFilter = parseQuerySearchFilter(searchFilter);` (line 208), add:

```typescript
// Strip DEPLOYMENT_STATUS — it's consumed by getDeploymentStatusScopedQueryString,
// not by getVulnStateScopedQueryString.
const {
    DEPLOYMENT_STATUS: _deploymentStatus,
    ...querySearchFilterWithoutStatus
} = querySearchFilter;
```

- [ ] **Step 3: Update the top-level `query` variable**

Find the `const query = getDeploymentStatusQueryString(...)` block (lines 209-219). Replace with:

```typescript
const query = isTombstonesEnabled
    ? getDeploymentStatusScopedQueryString(
          getVulnStateScopedQueryString(
              {
                  CVE: [exactCveIdSearchRegex],
                  ...baseSearchFilter,
                  ...querySearchFilterWithoutStatus,
              },
              vulnerabilityState
          ),
          searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
      )
    : getVulnStateScopedQueryString(
          {
              CVE: [exactCveIdSearchRegex],
              ...baseSearchFilter,
              ...querySearchFilterWithoutStatus,
          },
          vulnerabilityState
      );
```

- [ ] **Step 4: Update `getDeploymentSearchQuery`**

Find `function getDeploymentSearchQuery` (around line 264). Replace its body:

```typescript
function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
    const filters = {
        CVE: [exactCveIdSearchRegex],
        ...baseSearchFilter,
        ...querySearchFilterWithoutStatus,
    };
    if (severity) {
        filters.SEVERITY = [severity];
    }
    const base = getVulnStateScopedQueryString(filters, vulnerabilityState);
    return isTombstonesEnabled
        ? getDeploymentStatusScopedQueryString(
              base,
              searchFilter.DEPLOYMENT_STATUS as DeploymentStatusLabel[] | undefined
          )
        : base;
}
```

- [ ] **Step 5: Remove `DeploymentStatusFilter` JSX**

Find the `{isTombstonesEnabled && (<SplitItem><DeploymentStatusFilter .../>...)}` block (around lines 515-519). Delete the entire block.

Also remove `Switch` from the PatternFly import if it was ever present (search for it first to confirm).

- [ ] **Step 6: TypeScript check**

```bash
cd ui/apps/platform && npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 7: Commit**

```bash
git add ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage.tsx
git commit -m "feat(vm-ui): replace useDeploymentStatus hook with URL search filter in ImageCvePage

- Remove useDeploymentStatus hook, DeploymentStatusFilter, getDeploymentStatusQueryString
- Strip DEPLOYMENT_STATUS from querySearchFilter before getVulnStateScopedQueryString
- Wrap query and getDeploymentSearchQuery with getDeploymentStatusScopedQueryString
- DEPLOYMENT_STATUS=undefined defaults to Deployed-only (no tombstone addition)

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: DeploymentsTableContainer + delete removed files

**Files:**
- Modify: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/DeploymentsTableContainer.tsx`
- Delete: `src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.tsx`
- Delete: `src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx`
- Delete: `src/hooks/useDeploymentStatus.ts`
- Delete: `src/hooks/useDeploymentStatus.test.tsx`

> **Background:** `DeploymentsTableContainer.tsx` has three imports to remove (lines 2-5: `useDeploymentStatus`, `getDeploymentStatusQueryString`, `DeploymentStatusFilter`), plus `deploymentStatus` state, the `getDeploymentStatusQueryString` call, and the `DeploymentStatusFilter` toolbar item. `isTombstonesEnabled` is used ONLY for the filter gate — it can be removed too. `ToolbarItem` is still needed for the `ColumnManagementButton` wrapper (line 94) so keep it in the PatternFly import.

- [ ] **Step 1: Clean up `DeploymentsTableContainer.tsx`**

Remove imports:
- `import useDeploymentStatus from 'hooks/useDeploymentStatus'` (line 3)
- `import { getDeploymentStatusQueryString } from '../../utils/searchUtils'` (line 4)
- `import DeploymentStatusFilter from '../components/DeploymentStatusFilter'` (line 5)

Remove from the function body:
- `const { isFeatureFlagEnabled } = useFeatureFlags();` and `const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');` (lines 48-49) — only if `isTombstonesEnabled` is not used elsewhere in this file (search first)
- `const deploymentStatus = useDeploymentStatus();` (line 51)

Replace the `deploymentsQueryString` computation (lines 55-58):

```typescript
// Deployment status scoping is handled upstream in WorkloadCvesOverviewPage
// via getDeploymentStatusScopedQueryString applied to workloadCvesScopedQueryString.
const deploymentsQueryString = workloadCvesScopedQueryString;
```

Remove the `DeploymentStatusFilter` ToolbarItem JSX block (lines 89-93):
```tsx
// Remove:
{isTombstonesEnabled && (
    <ToolbarItem>
        <DeploymentStatusFilter onChange={() => pagination.setPage(1)} />
    </ToolbarItem>
)}
```

If `useFeatureFlags` is now unused (no remaining uses in the file), also remove `import useFeatureFlags from 'hooks/useFeatureFlags'` (line 7).

- [ ] **Step 2: Delete removed files**

```bash
git rm \
  ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.tsx \
  ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/components/DeploymentStatusFilter.test.tsx \
  ui/apps/platform/src/hooks/useDeploymentStatus.ts \
  ui/apps/platform/src/hooks/useDeploymentStatus.test.tsx
```

- [ ] **Step 3: TypeScript check**

```bash
cd ui/apps/platform && npm run tsc -- --noEmit
```

Expected: no new errors beyond pre-existing ones.

- [ ] **Step 4: Run all affected tests**

```bash
npm run test -- src/Containers/Vulnerabilities/utils/searchUtils.test.ts
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/DeploymentsTableContainer.tsx
git commit -m "feat(vm-ui): remove per-container deployment status filter; delete old files

- DeploymentsTableContainer: deploymentsQueryString = workloadCvesScopedQueryString
  (status scoping is now done upstream in WorkloadCvesOverviewPage)
- Delete DeploymentStatusFilter component + tests
- Delete useDeploymentStatus hook + tests

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Verification Checklist

After all tasks are complete:

- [ ] `npm run tsc -- --noEmit` exits 0 (pre-existing errors only).
- [ ] All searchUtils tests pass:
  ```bash
  npm run test -- src/Containers/Vulnerabilities/utils/searchUtils.test.ts
  ```
- [ ] With `ROX_DEPLOYMENT_TOMBSTONES=false`: "Deployment status" section absent from Default Filters modal; no tombstone filter applied.
- [ ] With `ROX_DEPLOYMENT_TOMBSTONES=true`:
  - "Default filters" modal shows "Deployment status" section with "Deployed" (checked) and "Deleted" (unchecked) by default.
  - Checking "Deleted" adds tombstoned deployments (and their images/CVEs) to all tabs.
  - Checking both shows all deployments.
  - Unchecking "Deployed" with "Deleted" checked shows only tombstoned.
  - Filter state persists in URL (`s[DEPLOYMENT_STATUS][0]=Deployed`).
  - CVE detail page (`ImageCvePage`) respects the same URL param automatically.
  - Pagination counts recompute when filter changes.
  - No `DeploymentStatusFilter` ToggleGroup visible anywhere.
