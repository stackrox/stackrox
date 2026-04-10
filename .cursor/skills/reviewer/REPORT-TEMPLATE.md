# Report Template

## Contents
- [Report Header](#report-header)
- [Executive Summary](#executive-summary)
- [Spec Deviations](#spec-deviations) *(omit entirely if no spec)*
- [Code Quality Findings](#code-quality-findings)
- [Test Coverage Map](#test-coverage-map)
- [Summary Scorecard](#summary-scorecard)
- [Appendices](#appendices)

---

## Report Header

```
# {Project Name} — Code Review Report

**Generated**: {date}
**Codebase**: {file_count} files, ~{loc} LOC, {test_count} test files
**Spec**: {path to spec, or "None — code-quality review only"}
**Modules applied**: {comma-separated list of checklist module IDs used}
```

---

## Executive Summary

```
## Executive Summary

**Overall Assessment**: {PASS | PASS WITH CONCERNS | NEEDS WORK | FAIL}

| Severity | Count |
|----------|-------|
| Critical | {n}   |
| High     | {n}   |
| Medium   | {n}   |
| Low      | {n}   |
| Info     | {n}   |

### Top Findings (by impact)
1. {Most critical finding — one sentence}
2. {Second most critical}
3. {Third most critical}
```

---

## Spec Deviations

*Omit this entire section if no spec was found. Do not leave it empty.*

```
## Spec Deviations

### Traceability Matrix

| Spec Section | Status | Implementation File(s) | Notes |
|-------------|--------|----------------------|-------|
| {section}   | PASS / PARTIAL / MISSING | {file:line} | {notes} |

### Deviations

For each deviation:

### [{SEVERITY}] {Spec Section}: {Title}
- **Spec ref**: {section or requirement ID}
- **Expected**: {what the spec requires}
- **Actual**: {what the code does}
- **File**: {path:line}
- **Recommendation**: {specific fix}

### Missing Features

{Spec requirements with no implementation — list each with spec ref and severity}
```

---

## Code Quality Findings

```
## Code Quality Findings

{Organize findings under the checklist module that surfaced them, e.g. "### Code Quality (U)", "### Testing (T)", "### Go"}

For each finding:

### [{SEVERITY}] {ID}: {Title}
- **File**: {path:line}
- **Issue**: {description}
- **Expected**: {correct pattern}
- **Actual**: {what the code does}
- **Recommendation**: {specific fix}
```

---

## Test Coverage Map

```
## Test Coverage Map

| Package / Module | Source Files | Test Files | Untested Areas | Gap Severity |
|-----------------|-------------|------------|----------------|-------------|
| {package}       | {n}         | {n}        | {description}  | High/Med/Low |
```

---

## Summary Scorecard

```
## Summary Scorecard

| Category              | Score (1–5) | Weight | Weighted Score |
|-----------------------|-------------|--------|----------------|
| Spec Compliance       | {score}     | 25%    | {weighted}     |
| Language Best Practices | {score}   | 20%    | {weighted}     |
| Framework Patterns    | {score}     | 15%    | {weighted}     |
| Production Readiness  | {score}     | 15%    | {weighted}     |
| Test Quality & Coverage | {score}   | 15%    | {weighted}     |
| Architecture          | {score}     | 10%    | {weighted}     |
| **Overall**           |             | **100%** | **{total}/5.00** |

### Scoring Rubric
- **5 — Exemplary**: No issues or only Info-level notes
- **4 — Good**: Minor issues only (Low severity)
- **3 — Adequate**: Some Medium issues, no Critical/High
- **2 — Needs Work**: High severity issues present
- **1 — Critical Gaps**: Critical issues or major spec deviations

### Category Weights
Default weights above. Adjust for project type:
- **No spec**: redistribute the 25% Spec Compliance weight across remaining categories
- **Library**: increase Language Best Practices weight (+10%), reduce Production Readiness (-10%)
- **Service**: keep defaults
- **Framework-heavy project**: increase Framework Patterns weight (+10%), reduce Architecture (-10%)
```

---

## Appendices

```
## Appendix A: Files Reviewed
{Complete list with line counts}

## Appendix B: Anti-Patterns Checked
{Each project-specific anti-pattern identified in Step 4, with FOUND / NOT FOUND status and file:line if found}

## Appendix C: Prioritized Recommendations
{All findings ordered by: Critical first, then High, then quick wins (Low-effort fixes regardless of severity), then remaining Medium/Low}
```
