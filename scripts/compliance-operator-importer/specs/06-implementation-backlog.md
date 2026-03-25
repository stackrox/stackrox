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

## Slice E - Multi-cluster support and auto-discovery

### E Goal

Support multiple source clusters, auto-discover ACS cluster IDs, merge SSBs across clusters.

### E Requirement IDs

- `IMP-CLI-003` (updated), `IMP-CLI-005` (updated), `IMP-CLI-027`
- `IMP-MAP-016..021`
- `IMP-ACC-015..017`

### E Implementation targets (suggested)

- `scripts/compliance-operator-importer/internal/config/config.go` (new flags)
- `scripts/compliance-operator-importer/internal/discover/discover.go` (new package: ACS cluster ID auto-discovery)
- `scripts/compliance-operator-importer/internal/cofetch/client.go` (multi-context support)
- `scripts/compliance-operator-importer/internal/merge/merge.go` (new package: SSB merging + mismatch detection)
- `scripts/compliance-operator-importer/internal/run/run.go` (orchestrate multi-cluster flow)

### E Tests to add

- `internal/discover/discover_test.go`
- `internal/merge/merge_test.go`
- `internal/config/config_test.go` (new flag tests)
- `internal/run/run_test.go` (multi-cluster integration)

### E Acceptance signal

- Auto-discovery resolves ACS cluster IDs from admission-control ConfigMap on real clusters.
- SSBs with same name across clusters produce one merged scan config.
- SSBs with same name but different profiles/schedule produce an error.

### E Agent prompt seed

- "Implement Slice E: multi-cluster support. Add --kubecontext (repeatable), auto-discover ACS cluster ID via admission-control ConfigMap (fallback: ClusterVersion, helm-effective-cluster-name), merge SSBs by name across clusters, error on profile/schedule mismatch."

## Slice F - Overwrite-existing mode (PUT support)

### F Goal

Allow importer to update existing ACS scan configs instead of skipping them.

### F Requirement IDs

- `IMP-CLI-027`, `IMP-IDEM-008..009`, `IMP-ACC-014`

### F Implementation targets (suggested)

- `scripts/compliance-operator-importer/internal/models/models.go` (add UpdateScanConfiguration to ACSClient interface)
- `scripts/compliance-operator-importer/internal/acs/client.go` (implement PUT)
- `scripts/compliance-operator-importer/internal/reconcile/create_only.go` (rename to reconciler.go, add update path)
- `scripts/compliance-operator-importer/internal/config/config.go` (--overwrite-existing flag)

### F Tests to add

- `internal/reconcile/reconciler_test.go` (update path tests)
- `internal/acs/client_test.go` (PUT tests)

### F Acceptance signal

- With `--overwrite-existing`, existing scan configs are updated via PUT.
- Without the flag, behavior is unchanged (skip + conflict problem).

### F Agent prompt seed

- "Implement Slice F: --overwrite-existing flag. Add PUT to ACS client, update reconciler to call PUT when flag is set and scanName exists. Add UpdateScanConfiguration and DeleteScanConfiguration to ACSClient interface."

## Slice G - End-to-end acceptance and tooling

### G Goal

Make real-cluster validation repeatable and scriptable.

### G Requirement IDs

- `IMP-ACC-001..017`

### G Implementation targets (suggested)

- `scripts/compliance-operator-importer/hack/acceptance-run.sh`
- `scripts/compliance-operator-importer/hack/check-report.sh`

### G Tests/checks to add

- lightweight script tests where practical.
- documented manual acceptance evidence for cluster runs.

### G Acceptance signal

- all commands/checks in `specs/04-validation-and-acceptance.md` are reproducible.
- include at least one real-cluster proof run against a live ACS endpoint with artifact output.
- multi-cluster and overwrite scenarios tested against real clusters.

### G Agent prompt seed

- "Implement Slice G automation helpers for IMP-ACC-001..017 and produce run artifacts paths for dry-run/apply/second-run/multi-cluster/overwrite checks."

## Slice H - UX alignment with roxctl conventions

### H Goal

Rename all flags and env vars to match roxctl conventions. Remove unnecessary
indirection flags. Simplify auth inference and endpoint handling.

### H Requirement IDs

- `IMP-CLI-001` (updated: `--endpoint` / `ROX_ENDPOINT`, auto-prepend `https://`)
- `IMP-CLI-002` (updated: auto-infer auth from env vars, no `--auth-mode`)
- `IMP-CLI-005` (updated: unified `--cluster` flag accepting UUID, name, or ctx=value)
- `IMP-CLI-013` (updated: bare hostnames get `https://` prepended)
- `IMP-CLI-023` (removed: `--auth-mode`)
- `IMP-CLI-024` (updated: `--username` / `ROX_ADMIN_USER`, default `admin`; password from `ROX_ADMIN_PASSWORD`)
- `IMP-CLI-025` (updated: ambiguous = both token+password set)
- `IMP-CLI-026` (removed: auth mode inferred)
- `IMP-MAP-022..023` (new: `--cluster` single-value shorthand with name or UUID)

### H Changes summary

| Old | New | Notes |
|-----|-----|-------|
| `--acs-endpoint` / `ACS_ENDPOINT` | `--endpoint` / `ROX_ENDPOINT` | aligned with roxctl |
| `--acs-auth-mode` | (removed) | auto-inferred |
| `--acs-token-env` | (removed) | always reads `ROX_API_TOKEN` |
| `--acs-password-env` | (removed) | always reads `ROX_ADMIN_PASSWORD` |
| `--acs-username` / `ACS_USERNAME` | `--username` / `ROX_ADMIN_USER` (default `admin`) | aligned with roxctl |
| `--acs-cluster-id` | `--cluster` (UUID, name, or ctx=value) | unified |
| `--source-kubecontext` | (removed) | redundant with `--kubecontext` |

### H Implementation targets

- `internal/config/config.go` (flag renames, auth inference, endpoint normalization)
- `internal/models/models.go` (remove AuthMode, TokenEnv, PasswordEnv fields)
- `internal/preflight/preflight.go` (auth inference)
- `internal/acs/client.go` (read from fixed env vars)
- `internal/run/cluster_source.go` (unified `--cluster` parsing)
- `cmd/importer/main.go`

### H Tests to update

- `internal/config/config_test.go`
- `internal/config/config_multicluster_test.go`
- `internal/preflight/preflight_test.go`
- `internal/acs/client_test.go`
- `internal/run/run_test.go`

### H Agent prompt seed

- "Implement Slice H: rename all ACS-prefixed flags/env vars to roxctl conventions per the table above. Remove --auth-mode, --token-env, --password-env, --source-kubecontext. Auto-infer auth from env vars. Auto-prepend https:// for bare hostnames. Unify --cluster to accept UUID, name, or ctx=value."

## Cross-slice conventions

- Requirement IDs must appear in test names or comments.
- Keep mapping logic side-effect free where possible.
- Wrap external clients (k8s/ACS) behind interfaces for deterministic tests.
- Never mutate CO resources.
- Guard rail test: without `--overwrite-existing`, no `PUT` is ever sent.
- Verify behavior with real-world examples early and often, not only mocked tests.
- Capture smoke-test commands and outputs in PR notes for traceability.

## Suggested execution order and ownership

1. Slice A (platform/entrypoint) -- DONE
2. Slice B (domain mapping) -- DONE
3. Slice C (ACS reconciliation) -- DONE
4. Slice D (reporting + run orchestration) -- DONE
5. Slice E (multi-cluster + auto-discovery)
6. Slice F (overwrite-existing / PUT support)
7. Slice G (acceptance automation)

Slices E and F are independent and can be implemented in parallel.
One agent per slice is ideal; if sequential, complete one slice fully before next.
