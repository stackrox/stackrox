# Broad Pattern Warnings for File Activity Policies

## What

We're adding non-blocking warning messages to the file path input in the policy wizard. When a user types a glob pattern that's structurally too broad (like `/**` or `/tmp/**`), they'll see a yellow warning below the input explaining the risk and suggesting a narrower alternative. They can still save the policy -- nothing is blocked.

## Why

Today, users can enter patterns like `/**` (matches every file on the system) with zero feedback about the alert volume it will produce. There's no runtime dry-run capability, so users have no way to gauge impact before enabling a policy. The most common catastrophic mistakes are preventable with simple structural checks.

## What it looks like

- **`/**`** shows: *"This pattern matches every file on the system and will generate extreme alert volume. Consider narrowing to a specific directory like /etc/\*\*."*
- **`/tmp/**`** shows: *"Patterns under /tmp typically generate very high alert volume due to frequent temporary file activity."*
- **`/etc/passwd`** shows nothing -- it's a specific, reasonable path.
- Validation errors (e.g., non-absolute path) still take precedence over warnings.

## Scope

- **Frontend only** -- no backend changes, no new API endpoints.
- Warns on two categories of patterns:
  1. **Root-level catch-alls**: `/**`, `/*`, `/**/foo` (searches every directory)
  2. **Known high-churn directories with globs**: `/tmp/**`, `/proc/*`, `/sys/**`, `/var/log/**`
- Exact paths under high-churn directories (e.g., `/tmp/specific.txt`, `/proc/1/status`) do **not** warn -- the user knows what they're targeting.
- The high-churn directory list (`/tmp`, `/proc`, `/sys`, `/var/log`) is hardcoded. These are standard Linux virtual/temp filesystems with stable conventions.

## What it is not

- Not a breadth "score" or estimation tool -- it only catches known-dangerous structural patterns.
- Does not detect overlap between policies.
- Does not cover patterns that are broad in practice but structurally ambiguous (e.g., `/etc/**` is scoped and acceptable).

## Files changed (3 files)

1. **`policyCriteriaDescriptors.tsx`** -- Add `warn` property to the `TextDescriptor` type, create the `warnBroadFilePath` function, apply it to both File Path descriptors (deployment events and node events).
2. **`PolicyCriteriaFieldInput.tsx`** -- Render the warning using PatternFly's `HelperTextItem variant="warning"` with yellow input border.
3. **`policyCriteriaDescriptors.test.ts`** -- Unit tests for the new warning function.

## Questions for the team

1. **Does the high-churn directory list (`/tmp`, `/proc`, `/sys`, `/var/log`) match what backend engineers see causing alert storms in production?** Are there other directories we should include?
2. **Should `/**/foo` (recursive search for a specific filename) warn?** The plan currently warns because it searches every directory, but it is more targeted than `/**`. If backend telemetry suggests these patterns are fine in practice, we could skip the warning.
3. **Any plans for backend-side pattern validation?** This feature is purely client-side heuristics. If there's appetite for a server-side "estimated match breadth" endpoint in the future, we'd want to know so this UI can evolve to consume it.

## Full implementation plan

The detailed plan with code examples, edge case matrix, and test cases is at [docs/plans/2026-03-25-001-feat-file-path-broad-pattern-warnings-plan.md](./2026-03-25-001-feat-file-path-broad-pattern-warnings-plan.md).
