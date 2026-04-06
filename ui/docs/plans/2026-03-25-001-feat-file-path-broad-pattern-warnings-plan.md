---
title: "feat: Add broad pattern warnings to file path inputs"
type: feat
status: active
date: 2026-03-25
origin: docs/brainstorms/2026-03-23-simple-pattern-guardrails-requirements.md
---

# feat: Add broad pattern warnings to file path inputs

## Enhancement Summary

**Deepened on:** 2026-03-25
**Sections enhanced:** 6
**Research agents used:** TypeScript reviewer, pattern recognition specialist, code simplicity reviewer, security sentinel, architecture strategist, frontend races reviewer, performance oracle, PatternFly framework docs researcher

### Key Improvements
1. **Regex correctness fix**: The root-level catch-all regex conflates `/*` (single-level) with `/**` (recursive) -- `/*/bar` would incorrectly get the "searches every directory" message. Split the checks.
2. **Accessibility**: Add `isLiveRegion` to `HelperText` wrapper so screen readers announce warning changes dynamically.
3. **Layout stability**: The File Path descriptors already have `helperText`, so the helper text area is always rendered -- no layout shift when warnings appear.

### New Considerations Discovered
- PatternFly's `TextInput` does NOT set `aria-invalid` for `validated="warning"` (only for `"error"`). This is correct behavior.
- `HelperTextItem variant="warning"` renders an `ExclamationTriangleIcon` automatically with bold font weight.
- The `case 'text'` JSX must preserve its existing `<Flex grow>` wrapper -- plan snippets previously omitted it.

---

## Overview

Add non-blocking warning messages to file path inputs in the policy wizard when users enter glob patterns that are structurally too broad (e.g., `/**`, `/tmp/**`). Warnings describe the risk and suggest a narrower alternative. This is a frontend-only change -- no backend modifications required.

## Problem Statement / Motivation

Users authoring file activity policies can enter glob patterns like `/**` or `/tmp/**` with zero feedback about the alert volume these patterns will produce. The existing `validateFilePath` function only checks for absolute paths and directory traversal -- it does not assess pattern breadth. Since runtime dry-run is unsupported, users have no way to gauge impact before enabling a policy. The most catastrophic mistakes (root-level wildcards, known high-churn directories) are preventable with simple structural checks (see origin: `docs/brainstorms/2026-03-23-simple-pattern-guardrails-requirements.md`).

## Proposed Solution

Add a `warn` property to `TextDescriptor` (parallel to the existing `validate` property) and create a `warnBroadFilePath` function that detects structurally broad patterns. Render warnings in `PolicyCriteriaFieldInput.tsx` using PatternFly's `HelperTextItem variant="warning"`.

## Technical Considerations

### Architecture

- **No new abstractions**: The `warn` property mirrors the existing `validate` signature (`(value: string) => string | undefined`), keeping the descriptor contract simple.
- **No backend changes**: All logic is client-side pattern matching.
- **No feature flag**: The warning function is applied directly to descriptors. The deployment event file path descriptor is already gated by `ROX_SENSITIVE_FILE_ACTIVITY`; the node event descriptor has no gate (it exists independently). The warning is unconditional on both descriptors.
- **Keep `warn` on `TextDescriptor`, not `BaseDescriptor`**: Only text inputs accept freeform user entry. Select, multiselect, radioGroup, and tableModal constrain input to predefined options where validation/warnings are irrelevant. Promoting `warn` to `BaseDescriptor` would add dead weight to 6 other descriptor types. If `NumberDescriptor` later needs warnings, the refactor is trivial and non-breaking.

### Key Files

| File | Change |
|---|---|
| `apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx` | Add `warn` to `TextDescriptor` type, create `warnBroadFilePath`, apply to both File Path descriptors |
| `apps/platform/src/Containers/Policies/Wizard/Step3/PolicyCriteriaFieldInput.tsx` | Render warning state in the `text` case (lines 58-85) |
| `apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.test.ts` | Add tests for `warnBroadFilePath` |

### Rendering Priority

Errors > Warnings > Default `helperText`. Only one message displays at a time (see origin: R2).

### `TextInput validated` Prop

Set `validated="warning"` on the `TextInput` when a warning is active. This provides a yellow/amber border consistent with the error state's red border. Precedent exists in `ClusterLabelsTable.tsx` which uses `ValidatedOptions.warning`.

#### Research Insights: PatternFly Behavior

- **`validated="warning"`** adds CSS class `pf-m-warning`, renders an `ExclamationTriangleIcon` inside the input, sets amber border color via `--pf-t--global--border--color--status--warning--default`.
- **Does NOT set `aria-invalid="true"`** -- only `validated="error"` sets `aria-invalid`. This is correct: warnings are not errors.
- **`HelperTextItem variant="warning"`** automatically renders `ExclamationTriangleIcon`, applies bold font weight via `pf-m-warning` class. A visually-hidden `": warning status;"` is appended for screen readers.
- **`HelperText` wrapper should use `isLiveRegion`** to announce dynamic changes to assistive technologies. Add `isLiveRegion` prop so screen readers announce when warnings appear/disappear.

### Edge Cases Resolved

| Input | Behavior | Rationale |
|---|---|---|
| `""` or `" "` | No warning | Same as `validateFilePath` -- empty inputs are a no-op |
| `/**`, `/*` | Warn (root catch-all) | Matches everything under root |
| `/` | No warning | Literal root path, not a glob pattern |
| `/**/foo` | No warning | Scoped recursive search with a specific target file |
| `/*/bar` | Warn (root single-level) | `*` as first segment matches all immediate subdirectories of root |
| `/tmp/**`, `/tmp/*` | Warn (high-churn) | Glob under known noisy directory |
| `/tmp` | No warning | Exact path, not a glob -- user knows what they want |
| `/tmp/specific.txt` | No warning | Exact path under high-churn dir is intentional |
| `/var/log/syslog` | No warning | Specific file, not a glob |
| `/TMP/**` | No warning | Case-sensitive matching (Linux filesystems are case-sensitive) |
| `/proc/1/status` | No warning | Specific path under high-churn dir |
| `home/**` | No warning (but validation error) | `validate` catches non-absolute path; warning is irrelevant |

### Decisions on Outstanding Questions

**High-churn directory list: hardcoded** (see origin: Deferred to Planning). Container filesystem conventions are stable. The list is: `/tmp`, `/proc`, `/sys`, `/var/log`. If it needs to change, it is a code change. Configuring this via UI or API adds unnecessary complexity for a heuristic warning.

> **Research note**: Security review noted `/run`, `/var/run`, `/var/cache`, `/dev` as other high-churn candidates. These are omitted intentionally for MVP -- the list can be expanded in a follow-up if backend telemetry confirms they cause alert storms.

**High-churn matching logic**: Warn only when the path starts with a high-churn prefix AND the remaining portion contains a glob wildcard (`*` or `**`). Exact paths under high-churn directories are intentional and should not warn.

**`warnBroadFilePath` should trim input**: Yes, same as `validateFilePath`, to ensure consistent behavior on the same raw input.

**Read-only mode**: Do not suppress warnings in read-only mode. Showing a warning on an existing broad pattern provides useful context for why a policy may generate excessive alerts, even if the user cannot edit it in that moment.

### Regex Correctness

The original regex `/^\/((\*\*|\*)(\/.*)?)$/` conflates single-level wildcards (`/*`) with recursive wildcards (`/**`). For example, `/*/bar` matches only immediate subdirectories, not "every directory on the system." The implementation should use separate checks with distinct messages:

- `/**` and `/*` (no trailing path segments) -- "matches every file on the system"
- `/*/...` (single-level with trailing path) -- "matches all immediate subdirectories of root"

#### Research Insights: Security

- **No ReDoS risk**: The regex has no nested or overlapping quantifiers. `.*` is anchored to `$` and consumes greedily in a single pass. Linear time even on adversarial input.
- **No XSS risk**: Warning messages are hardcoded string literals or interpolate from the hardcoded `highChurnPrefixes` array. React's JSX auto-escapes string content.
- **Pattern matching bypass is acceptable**: Since warnings are advisory-only, missing a warning has zero security impact. The bypass vectors (Unicode lookalikes, path traversal combinations) would produce non-functional policies.

#### Research Insights: Performance

- **No debouncing needed**: The function is pure, synchronous, sub-microsecond. React's own reconciliation costs orders of magnitude more per keystroke.
- **Regex literals inline are fine**: V8 caches regex literals in hot functions -- no benefit to hoisting to module scope.
- **No unnecessary allocations**: Template literals in the loop create small strings (6-9 bytes) that are collected immediately. Negligible.

#### Research Insights: Frontend Races

- **No render-cycle issues**: The conditional evaluation is synchronous derivation from props/Formik state within a single render pass. Error-before-warning precedence is guaranteed by sequential evaluation.
- **No stale state risk**: `useField` subscribes to Formik state synchronously. The `warn` call runs against the same value as `validate` in the same render.
- **No layout shift for File Path fields**: Both File Path descriptors already have `helperText: 'Enter an absolute file path. Supports glob patterns.'`, so the `FormHelperText` wrapper is always rendered. Switching between helperText/warning/error only changes the content and variant, not the DOM presence.

## Acceptance Criteria

- [ ] `TextDescriptor` type has an optional `warn` property with signature `(value: string) => string | undefined`
- [ ] `PolicyCriteriaFieldInput.tsx` renders `HelperTextItem variant="warning"` when `warn` returns a string and `validate` returns `undefined`
- [ ] `TextInput` shows `validated="warning"` when a warning is active (yellow border)
- [ ] `warnBroadFilePath` warns on root-level catch-all patterns: `/**`, `/*`
- [ ] `warnBroadFilePath` warns on root-level single-level glob: `/*/bar`
- [ ] `warnBroadFilePath` warns on high-churn directory globs: `/tmp/**`, `/proc/*`, `/sys/**`, `/var/log/**`
- [ ] `warnBroadFilePath` does NOT warn on: `/etc/passwd`, `/tmp`, `/tmp/specific.txt`, `/proc/1/status`, empty strings
- [ ] Warning messages describe the risk AND suggest a narrower alternative (see origin: R4)
- [ ] `warnBroadFilePath` is applied to both File Path descriptors: `policyCriteriaDescriptors` (line ~1527) and `nodeEventDescriptor` (line ~1672)
- [ ] Validation errors still take precedence over warnings
- [ ] `HelperText` wrapper uses `isLiveRegion` for screen reader announcements
- [ ] No backend changes
- [ ] Unit tests for `warnBroadFilePath` cover the full edge case matrix in `policyCriteriaDescriptors.test.ts`

## Success Metrics

- Users entering overly broad file path patterns see a visible, actionable warning before saving the policy
- Zero disruption to existing validation behavior
- No increase in blocked policy submissions (warnings are non-blocking)

## Dependencies & Risks

- **PatternFly `HelperTextItem variant="warning"`**: Confirmed available in PF6 (`@patternfly/react-core ^6.4.1`). Variant type: `'default' | 'indeterminate' | 'warning' | 'success' | 'error'`.
- **Low risk**: This is a frontend-only, additive change. No existing behavior is modified -- only new warning rendering is added alongside the existing error/helperText logic.

## MVP

### `policyCriteriaDescriptors.tsx` -- Type change

```typescript
// ~line 388
export type TextDescriptor = {
    type: 'text';
    placeholder?: string;
    helperText?: string;
    validate?: (value: string) => string | undefined;
    warn?: (value: string) => string | undefined;
} & BaseDescriptor &
    DescriptorCanBoolean &
    DescriptorCanNegate;
```

### `policyCriteriaDescriptors.tsx` -- `warnBroadFilePath` function

```typescript
// After validateFilePath (~line 33)
const highChurnPrefixes = ['/tmp', '/proc', '/sys', '/var/log'];

export function warnBroadFilePath(value: string): string | undefined {
    const trimmed = value.trim();
    if (trimmed.length === 0) {
        return undefined;
    }

    // Root-level catch-all without additional path segments: /**, /*
    if (trimmed === '/**' || trimmed === '/*') {
        return 'This pattern matches every file on the system and will generate extreme alert volume. Consider narrowing to a specific directory like /etc/**.';
    }

    // Root-level single-level glob with trailing path: /*/bar
    if (trimmed.startsWith('/*/')) {
        return 'This pattern matches all immediate subdirectories of root. Consider scoping to a specific directory.';
    }

    // High-churn directories with glob wildcards
    for (const prefix of highChurnPrefixes) {
        if (trimmed.startsWith(`${prefix}/`) && /[*]/.test(trimmed)) {
            return `Patterns under ${prefix} typically generate very high alert volume due to frequent ${prefix === '/tmp' ? 'temporary file' : prefix === '/var/log' ? 'log file' : 'system'} activity.`;
        }
    }

    return undefined;
}
```

> **Simplicity improvement**: Replaced the original regex with simple string checks (`===`, `startsWith`). This is more readable, easier to debug, and correctly distinguishes between `/*` (single-level) and `/**` (recursive) patterns with different warning messages.

### `policyCriteriaDescriptors.tsx` -- Apply to descriptors

```typescript
// ~line 1527 (deployment event file path)
{
    label: 'File path',
    name: 'File Path',
    // ... existing properties ...
    validate: validateFilePath,
    warn: warnBroadFilePath,
    // ...
},

// ~line 1672 (node event file path)
{
    label: 'File path',
    name: 'File Path',
    // ... existing properties ...
    validate: validateFilePath,
    warn: warnBroadFilePath,
    // ...
},
```

### `PolicyCriteriaFieldInput.tsx` -- Warning rendering

```tsx
// In the 'text' case (~line 58)
case 'text': {
    const validationError = descriptor.validate?.(String(value.value));
    const showError = Boolean(validationError);
    const warningMessage = !showError ? descriptor.warn?.(String(value.value)) : undefined;
    const showWarning = Boolean(warningMessage);

    return (
        <Flex grow={{ default: 'grow' }}>
            <TextInput
                value={value.value}
                type="text"
                id={name}
                isDisabled={readOnly}
                onChange={(_event, val) => handleChangeValue(val)}
                data-testid="policy-criteria-value-text-input"
                placeholder={descriptor.placeholder || ''}
                validated={showError ? 'error' : showWarning ? 'warning' : 'default'}
            />
            {(descriptor.helperText || showError || showWarning) && (
                <FormHelperText>
                    <HelperText isLiveRegion>
                        <HelperTextItem
                            variant={showError ? 'error' : showWarning ? 'warning' : 'default'}
                        >
                            {showError
                                ? validationError
                                : showWarning
                                  ? warningMessage
                                  : descriptor.helperText}
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            )}
        </Flex>
    );
}
```

> **Key changes from original plan**: (1) Preserved `<Flex grow>` wrapper. (2) Added `isLiveRegion` to `HelperText` for accessibility. (3) All existing props on `TextInput` preserved.

### `policyCriteriaDescriptors.test.ts` -- Tests for `warnBroadFilePath`

```typescript
describe('warnBroadFilePath', () => {
    it('should return undefined for an empty string', () => {
        expect(warnBroadFilePath('')).toBeUndefined();
    });

    it('should return undefined for a whitespace-only string', () => {
        expect(warnBroadFilePath('   ')).toBeUndefined();
    });

    it('should warn for /** (root catch-all)', () => {
        expect(warnBroadFilePath('/**')).toBeDefined();
    });

    it('should warn for /* (root catch-all)', () => {
        expect(warnBroadFilePath('/*')).toBeDefined();
    });

    it('should not warn for /**/foo (scoped recursive search)', () => {
        expect(warnBroadFilePath('/**/foo')).toBeUndefined();
    });

    it('should warn for /*/bar (root-level single-level search)', () => {
        expect(warnBroadFilePath('/*/bar')).toBeDefined();
    });

    it('should warn for /tmp/**', () => {
        expect(warnBroadFilePath('/tmp/**')).toBeDefined();
    });

    it('should warn for /proc/*', () => {
        expect(warnBroadFilePath('/proc/*')).toBeDefined();
    });

    it('should warn for /sys/**', () => {
        expect(warnBroadFilePath('/sys/**')).toBeDefined();
    });

    it('should warn for /var/log/**', () => {
        expect(warnBroadFilePath('/var/log/**')).toBeDefined();
    });

    it('should not warn for /etc/passwd (specific safe path)', () => {
        expect(warnBroadFilePath('/etc/passwd')).toBeUndefined();
    });

    it('should not warn for / (root path, not a glob)', () => {
        expect(warnBroadFilePath('/')).toBeUndefined();
    });

    it('should not warn for /tmp (exact path, no glob)', () => {
        expect(warnBroadFilePath('/tmp')).toBeUndefined();
    });

    it('should not warn for /tmp/specific.txt (exact path under high-churn)', () => {
        expect(warnBroadFilePath('/tmp/specific.txt')).toBeUndefined();
    });

    it('should not warn for /proc/1/status (specific path under high-churn)', () => {
        expect(warnBroadFilePath('/proc/1/status')).toBeUndefined();
    });

    it('should not warn for /home/**/.ssh/id_* (scoped pattern)', () => {
        expect(warnBroadFilePath('/home/**/.ssh/id_*')).toBeUndefined();
    });

    it('should handle leading/trailing whitespace', () => {
        expect(warnBroadFilePath('  /**  ')).toBeDefined();
    });
});
```

> **Test additions**: Added `/*/bar` case and whitespace-with-warning case. Tests follow existing `validateFilePath` conventions: `describe`/`it` blocks, `toBeUndefined()`/`toBeDefined()` assertions.

## Sources

- **Origin document:** [docs/brainstorms/2026-03-23-simple-pattern-guardrails-requirements.md](../brainstorms/2026-03-23-simple-pattern-guardrails-requirements.md) -- Key decisions carried forward: separate `warn` function on descriptor (not a validation severity), hardcoded high-churn directory list, risk + suggestion warning messages.
- `TextDescriptor` type: `apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx:388-395`
- `validateFilePath`: `apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx:19-31`
- `PolicyCriteriaFieldInput` text case: `apps/platform/src/Containers/Policies/Wizard/Step3/PolicyCriteriaFieldInput.tsx:58-85`
- File path descriptors: `policyCriteriaDescriptors.tsx:1527-1538` (deployment) and `:1672-1684` (node event)
- `ValidatedOptions.warning` precedent: `apps/platform/src/Containers/Clusters/ClusterLabelsTable.tsx`
- Existing tests: `apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.test.ts:62-98`
- PatternFly `HelperTextItem` source: `@patternfly/react-core/dist/esm/components/HelperText/HelperTextItem.js` (PF6)
- PatternFly `TextInput` validated prop: `@patternfly/react-core/dist/esm/components/TextInput/TextInput.js` (PF6)
