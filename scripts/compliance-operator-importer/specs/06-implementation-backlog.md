# 06 - Implementation Backlog (Spec + Agentic Execution)

This backlog translates specs into delivery slices with strict requirement traceability.

## Working rules

- Implement slices in order.
- Implement production code in Go for Phase 1 (no bash/shell implementation).
- For each slice:
  1. write/enable failing tests first,
  2. implement minimum code to pass,
  3. run tests and capture evidence,
  4. list fulfilled requirement IDs in PR notes.
- Keep each slice in its own PR when possible.

## Slice A - CLI, config, and preflight

### A Goal

Provide a reliable entrypoint with strict validation and preflight checks.

### A Requirement IDs

- `IMP-CLI-001..016`
- `IMP-CLI-023..026`

### A Implementation targets (suggested)

- `scripts/compliance-operator-importer/cmd/importer/main.go`
- `scripts/compliance-operator-importer/internal/config/config.go`
- `scripts/compliance-operator-importer/internal/preflight/preflight.go`

### A Tests to add

- `internal/config/config_test.go`
- `internal/preflight/preflight_test.go`

### A Acceptance signal

- Valid flags/env parse and preflight probe behavior with correct exit pathing.
- Both auth modes behave correctly:
  - token mode default path,
  - basic mode local/dev path.

### A Agent prompt seed

- "Implement Slice A for create-only importer. Start with tests for IMP-CLI-001..016 and IMP-CLI-023..026, then implement CLI/config/preflight with HTTPS and both token/basic auth mode support."

## Slice B - CO discovery and mapping core

### B Goal

Discover CO resources and map into ACS create payloads.

### B Requirement IDs

- `IMP-MAP-001..015`

### B Implementation targets (suggested)

- `scripts/compliance-operator-importer/internal/cofetch/client.go`
- `scripts/compliance-operator-importer/internal/mapping/mapping.go`
- `scripts/compliance-operator-importer/internal/mapping/schedule.go`

### B Tests to add

- `internal/mapping/mapping_test.go`
- `internal/mapping/schedule_test.go`

### B Acceptance signal

- Deterministic payload creation from SSB/ScanSetting/Profile inputs.
- Invalid schedule path produces skip-worthy error with fix hint text.

### B Agent prompt seed

- "Implement Slice B with tests first for IMP-MAP-001..015. Ensure missing profile kind defaults to Profile and invalid schedule becomes skip+problem."

## Slice C - ACS create-only writer and idempotency

### C Goal

Create missing configs, skip existing names, never update.

### C Requirement IDs

- `IMP-IDEM-001..007`

### C Implementation targets (suggested)

- `scripts/compliance-operator-importer/internal/acs/client.go`
- `scripts/compliance-operator-importer/internal/reconcile/create_only.go`

### C Tests to add

- `internal/reconcile/create_only_test.go`
- `internal/acs/client_test.go`

### C Acceptance signal

- Existing `scanName` always skipped with conflict problem.
- No code path emits `PUT`.

### C Agent prompt seed

- "Implement Slice C as strict create-only. Test IMP-IDEM-001..007 first, especially: existing scanName => skip + conflict problem; never call PUT."

## Slice D - Problem list, reporting, and exit codes

### D Goal

Centralize error handling/reporting and enforce run outcomes.

### D Requirement IDs

- `IMP-CLI-017..022`
- `IMP-ERR-001..004`

### D Implementation targets (suggested)

- `scripts/compliance-operator-importer/internal/problems/problems.go`
- `scripts/compliance-operator-importer/internal/report/report.go`
- `scripts/compliance-operator-importer/internal/run/run.go`

### D Tests to add

- `internal/problems/problems_test.go`
- `internal/report/report_test.go`
- `internal/run/run_test.go`

### D Acceptance signal

- `problems[]` always emitted for problematic resources with `description` + `fixHint`.
- exit codes map correctly to all-success/fatal/partial outcomes.

### D Agent prompt seed

- "Implement Slice D with tests first for IMP-CLI-017..022 and IMP-ERR-001..004. Ensure problem list and exit code semantics exactly match spec."

## Slice E - End-to-end acceptance and tooling

### E Goal

Make real-cluster validation repeatable and scriptable.

### E Requirement IDs

- `IMP-ACC-001..012`

### E Implementation targets (suggested)

- `scripts/compliance-operator-importer/hack/acceptance-run.sh`
- `scripts/compliance-operator-importer/hack/check-report.sh`

### E Tests/checks to add

- lightweight script tests where practical.
- documented manual acceptance evidence for cluster runs.

### E Acceptance signal

- all commands/checks in `specs/04-validation-and-acceptance.md` are reproducible.
- include at least one real-cluster proof run against a live ACS endpoint (for example localhost:8443) with artifact output.

### E Agent prompt seed

- "Implement Slice E automation helpers for IMP-ACC-001..012 and produce run artifacts paths for dry-run/apply/second-run checks."

## Cross-slice conventions

- Requirement IDs must appear in test names or comments.
- Keep mapping logic side-effect free where possible.
- Wrap external clients (k8s/ACS) behind interfaces for deterministic tests.
- Never mutate CO resources.
- Keep create-only invariant explicit (guard rail test that fails on any `PUT` path).
- Verify behavior with real-world examples early and often, not only mocked tests.
- Capture smoke-test commands and outputs in PR notes for traceability.

## Suggested execution order and ownership

1. Slice A (platform/entrypoint)
2. Slice B (domain mapping)
3. Slice C (ACS reconciliation)
4. Slice D (reporting + run orchestration)
5. Slice E (acceptance automation)

One agent per slice is ideal; if sequential, complete one slice fully before next.
