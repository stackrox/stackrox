# Go Version Upgrade Guide

This document describes the process for upgrading the Go version used in the StackRox project.

## Prerequisites

- Update the apollo-ci container image (optional but recommended)
- Ensure Konflux go-builder image supports the target Go version
- Review Go release notes for breaking changes

## Files to Update

### 1. Core Go Version Declaration

**File:** `go.mod`
```go
go 1.26.2
```

### 2. CI Workflows

**File:** `.github/workflows/golangci-lint.yaml`

Update the `go-version` parameter:
```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.26.2'
```

**File:** `.github/workflows/unit-tests.yaml`

The workflow uses `go-version-file: go.mod` to automatically read the version from go.mod, so no changes needed if already configured.

If not using `go-version-file`, add setup-go action to jobs that run in containers:
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
```

### 3. Tool Modules

Update all tool module `go.mod` files to match the main Go version:

```bash
find . -name go.mod -path '*/tools/*' -exec sed -i 's/^go .*/go 1.26.2/' {} \;
```

Tool modules to update:
- `tools/generate-helpers/go.mod`
- `tools/roxvet/go.mod`
- Other tool directories with go.mod files

## Expected Test Failures and Fixes

When upgrading Go versions, expect test failures due to standard library behavioral changes. Common patterns:

### 1. URL Parser Changes

**Location:** `pkg/clientconn/client_test.go`

Go may change error message priority in URL parsing. Update test expectations to match new error messages.

### 2. IPv6 Validation Changes

**Location:** `pkg/tlscheck/tlscheck.go`

Go may change how `url.Parse()` validates IPv6 addresses. Consider using project-specific utilities like `netutil.ParseEndpoint()` for consistency.

### 3. Timing/Scheduling Changes

**Location:** Test files with timing assumptions (e.g., `central/processindicator/datastore/datastore_impl_test.go`)

Go runtime changes can expose timing race conditions. Increase timeouts or add explicit synchronization as needed.

### 4. General Strategy

For each test failure:
1. Verify it's a Go behavioral change, not a real bug
2. Check if the new behavior is more correct
3. Update test expectations rather than working around the new behavior
4. Add comments explaining why the behavior changed

## CI Infrastructure Considerations

### Container vs Native Runners

The apollo-ci container may have an older Go version baked in. When upgrading Go:

**Option 1: Use setup-go action (recommended)**
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
```

This installs the correct Go version ahead of the container's Go in PATH.

**Option 2: Wait for apollo-ci update**

Wait for the apollo-ci container to be updated with the new Go version.

**Option 3: Remove container**

Switch to native ubuntu-latest runners (see `davdhacs/remove-container-unittests` branch for reference).

### Cache Invalidation

When upgrading Go versions, invalidate the build cache:

**File:** `.github/actions/cache-go-dependencies/action.yaml`

Bump the cache version:
```yaml
key: go-build-v4-${{ github.job }}...  # Increment version number
```

## Konflux Integration

### Red Hat Konflux Builder

**CRITICAL:** Konflux uses a separate Go builder image that must support the target Go version.

**Issue:** The Konflux `checks` job will fail if the go-builder image doesn't support the new Go version.

**Resolution:**
1. Check if the Konflux go-builder supports the target version
2. If not, file a request to update the Konflux go-builder image
3. The PR cannot be merged until Konflux supports the new Go version

**Error Example:**
```
Red Hat Konflux / checks: FAILURE
```

This indicates the Konflux builder doesn't support the Go version in go.mod.

## Upgrade Checklist

- [ ] Update `go.mod` with new Go version
- [ ] Update `.github/workflows/golangci-lint.yaml` go-version
- [ ] Update all tool module `go.mod` files
- [ ] Bump cache version in `.github/actions/cache-go-dependencies/action.yaml`
- [ ] Ensure `.github/workflows/unit-tests.yaml` uses setup-go action or wait for apollo-ci update
- [ ] Run tests locally to identify behavioral changes
- [ ] Fix test failures caused by Go stdlib changes
- [ ] Verify Konflux go-builder supports the new version
- [ ] Monitor CI for additional failures
- [ ] Update this guide with new failure patterns discovered

## Common Issues

### Issue: "compile: version go1.26.2 does not match go tool version go1.25.7"

**Cause:** Container has older Go baked in, GOTOOLCHAIN=auto downloads new Go but container's Go is still in PATH.

**Solution:** Add setup-go action to install the correct Go version:
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
```

### Issue: Tests pass but make returns Error 1

**Cause:** Version mismatch errors during compilation cause non-zero exit status.

**Solution:** Same as above - install correct Go version with setup-go.

### Issue: Konflux checks fail

**Cause:** Konflux go-builder doesn't support the new Go version yet.

**Solution:** Wait for Konflux builder update or coordinate with Konflux team.

## Testing Strategy

1. **Local testing:** Run `make go-unit-tests` locally to catch obvious failures
2. **CI monitoring:** Push to a branch and monitor all CI jobs
3. **Iterative fixes:** Fix test failures one at a time, committing after each fix
4. **Verification:** Ensure all tests pass before requesting review

## Additional Notes

- Always check Go release notes for breaking changes
- Security-critical Go updates (CVE fixes) should be prioritized
- Consider the impact on downstream consumers of StackRox
- Update developer documentation if new Go features are adopted
