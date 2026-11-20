# Test: Docs-Only Changes

This file is added to test the docs-only skip logic in ROX-39729.

## Expected CI Behavior

When only this markdown file is changed, the following jobs should **SKIP**:
- `golangci-lint` (saves 16 minutes)
- `check-generated-files` (saves 15 minutes)
- `check-crd-compatibility` (saves 6 minutes)

The following jobs should **RUN** (but optimized):
- `style-check` (should run but skip expensive dependency downloads)
- `misc-checks` (always runs)

The following test jobs should **SKIP** via ci-select:
- `go`, `go-postgres`, `go-bench`
- `local-roxctl-tests`
- `sensor-integration-tests`
- All E2E tests

## Validation

Check the CI run for this PR and verify:
1. Skipped jobs have "skip" status or don't appear
2. Total runtime is <10 minutes (vs ~180 minutes normally)
3. All running jobs complete successfully
