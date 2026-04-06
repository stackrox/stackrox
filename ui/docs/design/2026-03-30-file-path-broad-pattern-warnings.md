---
date: 2026-03-30
topic: file-path-broad-pattern-warnings
epic: ROX-33110
target-release: "4.11"
status: Draft
---

# Broad Pattern Warnings for File Path Inputs - Design Document

**Feature:** Non-blocking warnings on overly broad glob patterns in policy wizard file path inputs\
**Target release:** ACS 4.11\
**Epic:** ROX-33110\
**Status:** Draft\
**Stakeholders:**
- TBD (PM)
- TBD (UX)
- TBD (Backend)
- TBD (UI)

---

## Background

Users authoring file activity policies in the policy wizard can enter glob patterns like `/**` or `/tmp/**` with zero feedback about the alert volume these patterns will produce. The existing `validateFilePath` function only checks for absolute paths and directory traversal -- it does not assess pattern breadth. Since runtime dry-run is explicitly unsupported, users have no way to gauge impact before enabling a policy.

The most catastrophic mistakes -- root-level wildcards and known high-churn directories -- are preventable with simple structural checks. Without guardrails, users discover the problem only after enabling a policy and being flooded with alerts.

## What We're Building

A frontend-only warning system that detects structurally broad file path patterns and displays non-blocking warning messages in the policy wizard. Warnings describe the risk and suggest a narrower alternative. No backend changes are required.

The implementation adds a `warn` property to the `TextDescriptor` type (parallel to the existing `validate` property) and a `warnBroadFilePath` function that checks for two categories of risky patterns:

1. **Root-level catch-all patterns** (`/**`, `/*`, `/*/...`) -- match across the entire filesystem
2. **High-churn directory globs** (`/tmp/**`, `/proc/*`, `/sys/**`, `/var/log/**`) -- directories with frequent activity in normal container operation

Warnings are applied to both File Path descriptors: the deployment event descriptor (gated by `ROX_SENSITIVE_FILE_ACTIVITY`) and the node event descriptor.

### What it looks like

When a user types a broad pattern, a yellow/amber warning border appears on the text input with a warning icon, and a helper text message appears below explaining the risk:

```
+----------------------------------------------------------+
| /tmp/**                                            [!]   |  <- amber border + warning icon
+----------------------------------------------------------+
  /!\ Patterns under /tmp typically generate very high        <- warning helper text
      alert volume due to frequent temporary file activity.
```

When the input has a validation error (e.g., non-absolute path), the error state takes precedence and the warning is suppressed. When the input is valid and non-broad, the default helper text ("Enter an absolute file path. Supports glob patterns.") displays normally.

## Success Criteria

- Users entering overly broad file path patterns (`/**`, `/tmp/**`, etc.) see a visible, actionable warning before saving the policy
- Users entering specific paths (`/etc/passwd`, `/tmp/specific.txt`) see no warning
- Validation errors still take precedence over warnings -- no disruption to existing behavior
- Warnings are non-blocking -- users can still save policies with broad patterns
- Screen readers announce warning changes dynamically via live region

## Non-Goals

- **Not a breadth estimator:** This does not score or estimate pattern breadth. It only catches known-dangerous structural patterns.
- **No policy overlap detection:** Does not detect overlap between multiple policies targeting the same paths (separate concern).
- **No backend changes:** All logic is client-side pattern matching. No new API endpoints.
- **No blocking of policy creation:** Warnings are advisory only. Users can always save.
- **Structurally ambiguous patterns are ignored:** Patterns like `/etc/**` are scoped and acceptable even though they may match many files.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Separate `warn` property on `TextDescriptor`, not a validation severity level | Keeps the existing `validate` contract intact. Warnings are a parallel concern. Avoids breaking changes to the `validate` return type. |
| `warn` on `TextDescriptor` only, not `BaseDescriptor` | Only text inputs accept freeform entry. Select, multiselect, radioGroup, and tableModal constrain input to predefined options where warnings are irrelevant. Avoids dead weight on 6 other descriptor types. |
| Hardcoded high-churn directory list (`/tmp`, `/proc`, `/sys`, `/var/log`) | Container filesystem conventions are stable. Configuring via UI/API adds unnecessary complexity for a heuristic warning. The list can be expanded in a follow-up with a code change. |
| String checks instead of regex for pattern detection | `===` and `startsWith` are more readable, easier to debug, and correctly distinguish between `/*` and `/**` patterns. No ReDoS risk. |
| Warning messages include risk description AND narrower alternative | Makes warnings actionable rather than just alarming. Users know both why the pattern is risky and what to do instead. |
| No feature flag for the warning itself | The deployment event file path descriptor is already gated by `ROX_SENSITIVE_FILE_ACTIVITY`; the node event descriptor has no gate. The warning is unconditional on both. |
| No debouncing | The warn function is pure, synchronous, sub-microsecond. React's own reconciliation costs orders of magnitude more per keystroke. |
| Warnings shown in read-only mode | Provides useful context for why a policy may generate excessive alerts, even if the user cannot edit it. |

## Alternatives Considered

| Approach | Why we rejected it |
|----------|-------------------|
| Add severity levels to `validate` return type (e.g., `{ level: 'error' \| 'warning', message: string }`) | Breaking change to every descriptor's validate contract. Higher blast radius for a feature that only applies to two text inputs. |
| Promote `warn` to `BaseDescriptor` | Adds dead weight to 6 descriptor types that constrain input to predefined options. Trivial to refactor later if needed. |
| Configurable high-churn directory list | Adds UI/API complexity for a heuristic that covers stable container filesystem conventions. Code change is sufficient for updates. |
| Use regex for all pattern matching | Original regex conflated `/*` (single-level) with `/**` (recursive). Simple string checks are clearer and correctly distinguish the cases. |

## High-Level Implementation Overview

Three files are modified:

**Descriptor type and warning function** -- The `TextDescriptor` type gains an optional `warn` property with the same signature as `validate`. A new `warnBroadFilePath` function uses string checks to detect root-level catch-alls, root-level single-level globs, and high-churn directory globs. The function trims input (matching `validateFilePath` behavior) and returns a descriptive warning message or `undefined`. Both File Path descriptors (deployment event and node event) reference this function.

**Field input rendering** -- The text input case in `PolicyCriteriaFieldInput` evaluates `warn` only when `validate` returns no error, enforcing error-over-warning precedence in a single synchronous render pass. When a warning is active, `TextInput` receives `validated="warning"` (amber border) and a `HelperTextItem variant="warning"` displays the message. The `HelperText` wrapper uses `isLiveRegion` for screen reader announcements. No layout shift occurs because both File Path descriptors already render a `helperText` area.

**Tests** -- Unit tests for `warnBroadFilePath` cover the full edge case matrix:

| Input | Expected | Rationale |
|-------|----------|-----------|
| `""`, `"   "` | No warning | Empty inputs are a no-op |
| `/**`, `/*` | Warn (root catch-all) | Matches everything under root |
| `/` | No warning | Literal root path, not a glob |
| `/**/foo` | No warning | Scoped recursive search with a specific target |
| `/*/bar` | Warn (root single-level) | Matches all immediate subdirectories of root |
| `/tmp/**`, `/proc/*`, `/sys/**`, `/var/log/**` | Warn (high-churn) | Glob under known noisy directory |
| `/tmp`, `/tmp/specific.txt`, `/proc/1/status` | No warning | Exact paths, not globs |
| `/etc/passwd`, `/var/log/syslog` | No warning | Specific file paths |
| `/TMP/**` | No warning | Case-sensitive matching (Linux filesystems) |
| `  /**  ` | Warn | Whitespace is trimmed before checking |

## Future Phases / Out of Scope

- **Expanded high-churn list:** `/run`, `/var/run`, `/var/cache`, `/dev` are candidates if backend telemetry confirms they cause alert storms.
- **Pattern overlap detection:** Detecting when multiple policies target overlapping file paths is a separate feature.
- **Backend dry-run:** Runtime impact estimation would require backend support and is out of scope.

## Dependencies

| Item | Status | Blocking? |
|------|--------|-----------|
| PatternFly `HelperTextItem variant="warning"` | Available in PF6 (`@patternfly/react-core ^6.4.1`) | No |
| PatternFly `TextInput validated="warning"` | Available in PF6 | No |

## Risks

| Risk | Mitigation |
|------|-----------|
| False positive warnings on legitimate broad patterns | Warnings are non-blocking and advisory only. Low cost if a user sees a warning they choose to ignore. |
| Missing warning on a pattern that should warn | Advisory-only -- missing a warning has zero security or functional impact. Pattern list can be expanded. |

## Open Questions

- [Non-blocking][PM] Should additional high-churn directories (`/run`, `/var/run`, `/var/cache`, `/dev`) be included in the initial release or deferred?
