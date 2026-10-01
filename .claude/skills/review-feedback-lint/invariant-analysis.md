# Review feedback lint invariant analysis

This document summarizes the high-feasibility invariants mined from the local review-feedback corpus and explains why the corresponding ESLint rules are worth enforcing.

## Corpus and selection method

The corpus was built from merged `stackrox/stackrox` pull requests that touched `ui/` between `2026-06-16` and `2026-09-16`.

Collection and filtering produced:

- 146 UI-touching merged PRs
- 232 raw review threads
- 395 raw review comments
- 195 filtered review threads after removing out-of-scope, generated-file, acknowledgement, bot-like, and low-signal discussion

Candidate invariants were selected by looking for feedback that was:

1. **Recurring or representative of a recurring class**: not just one-off style preference unless it exposed a generally applicable accessibility or correctness invariant.
2. **Static-analysis friendly**: detectable from local AST structure, TypeScript syntax, or JSX patterns without requiring whole-program execution.
3. **Important enough to block regressions**: likely to prevent user-visible bugs, accessibility regressions, security/authorization mistakes, or repeated reviewer churn.
4. **Low enough false-positive risk**: implementable with a narrow trigger and understandable remediation.

The high-feasibility rules implemented on this branch are the ones that met those criteria best. Other mined invariants, such as feature-flag consistency and API/proto contract matching, are valuable but need additional project metadata or generated API schemas before they can be linted safely.

## Implemented invariants

### `accessibility/Button-Tooltip-isAriaDisabled`

**Invariant:** A PatternFly `Button` wrapped in `Tooltip` or `ConditionalTooltip` should use `isAriaDisabled` instead of `isDisabled`.

**How it was determined:** Review feedback pointed out that a disabled button inside a tooltip loses keyboard focusability. That means keyboard users cannot reach the tooltip content, even though the UI visually exposes help text for the disabled action. The pattern is easy to recognize in JSX: a `Button` with `isDisabled` under a tooltip ancestor.

**Why it matters:** This is an accessibility correctness issue, not just a style preference. `isDisabled` removes the button from normal interaction and focus behavior, while `isAriaDisabled` preserves focusability and disabled semantics appropriate for tooltip explanations.

**How it improves the code base:** The rule prevents a subtle recurring accessibility regression at review time. It also teaches the intended PatternFly usage directly in the lint message, reducing the need for reviewers to repeatedly explain the distinction.

**Implementation boundary:** The rule intentionally checks only `Button` elements under `Tooltip` and `ConditionalTooltip`; it does not try to rewrite all disabled button usage.

### `generic/JSX-nullish-display-fallback`

**Invariant:** JSX display fallbacks should use `??` instead of `||` when rendering literal fallback text or numbers.

**How it was determined:** Several review comments flagged expressions that would hide valid falsy values, especially `0`, by using `value || '-'` or similar display fallbacks. The repeated underlying issue was the same: logical OR treats valid data as absent.

**Why it matters:** UI data fields often have meaningful falsy values. A CVSS score, count, percentage, or field that is legitimately `0` should render as `0`, not as `-`, `Unknown`, or another fallback. Incorrect fallbacks can misrepresent vulnerability, report, or configuration data.

**How it improves the code base:** The rule prevents a class of display correctness bugs and makes absence handling explicit. It uses a narrow trigger: JSX expression containers whose top-level operator is `||` and whose right-hand side is a literal display fallback. The auto-fix changes only the operator from `||` to `??`.

**Implementation boundary:** The rule avoids broad boolean/control-flow expressions, such as `isLoading || hasError` or `children || <EmptyState />`, because those are often intentional logical OR usages rather than display fallbacks.

### `generic/Partial-Record-sparse-map`

**Invariant:** A finite object literal used as a lookup map should not be typed as `Record<string, T>` unless every possible string key is actually valid. Use `Partial<Record<string, T>>` or `Record<string, T | undefined>` instead.

**How it was determined:** Review feedback called out sparse lookup objects that were typed as total string records. The type claimed every string key would produce a value, but the runtime object contained only a finite set of keys. That masks missing-key handling from TypeScript.

**Why it matters:** `Record<string, string>` tells TypeScript that `map[someRuntimeKey]` is always a `string`. For sparse maps, the real result can be `undefined`. The unsound type can lead to missing labels, bad API payloads, broken display values, or runtime errors that TypeScript could have helped prevent.

**How it improves the code base:** The rule forces sparse map types to encode absence. Once the type includes `undefined`, callers must handle missing keys explicitly. That shifts missing-case detection from review comments and runtime behavior into local type feedback.

**Implementation boundary:** The rule targets variable declarations where the initializer is an object literal and the annotation is `Record<string, T>` without `undefined` in `T`. It does not flag records with finite key unions, such as `Record<'critical' | 'important', string>`, because those can accurately model a total map.

### `accessibility/customIcon-ariaHidden` (`isAriaHidden` invariant)

**Invariant:** Decorative JSX elements passed to PatternFly-style `customIcon` props should include `aria-hidden`.

**How it was determined:** Review feedback identified a decorative spinner/icon used next to status text. Without `aria-hidden`, assistive technologies can announce both the icon and the nearby text, creating duplicate or confusing output. This maps cleanly to JSX: a `customIcon` prop whose value is an inline JSX element without `aria-hidden`.

**Why it matters:** Icons and spinners in `customIcon` positions are usually visual reinforcement for text already present in the alert/status component. Exposing them to the accessibility tree adds noise without adding information.

**How it improves the code base:** The rule keeps decorative status indicators quiet for screen readers and makes the accessible name/description come from the surrounding component text. It prevents regressions with a small, auditable pattern instead of relying on reviewer memory.

**Implementation boundary:** The rule only reports inline JSX elements passed directly to `customIcon`. It deliberately skips identifier expressions such as `customIcon={statusIcon}` because the icon may be defined elsewhere and needs separate inspection.

## Why these rules are the first batch

These rules were chosen before broader invariants because they have strong local signals:

- They are detectable from one JSX/TypeScript AST location plus, for the tooltip rule, JSX ancestry.
- The fix or remediation is obvious from the violation.
- They prevent real user-facing problems: inaccessible tooltips, noisy accessibility trees, incorrect display of valid falsy data, and unsound sparse-map typing.
- They encode feedback that reviewers otherwise repeat manually.

Rules that require cross-file route authorization modeling, feature-flag dependency graphs, generated API schemas, or runtime request semantics remain candidates for later work but need more infrastructure to avoid noisy false positives.
