# Go Version Upgrade Guide

This document describes the process for upgrading the Go version in the StackRox project.

## Overview

Upgrading Go involves updating version declarations across the codebase, addressing breaking changes in the standard library, and ensuring CI infrastructure supports the new version. The process typically takes several iterations as test failures reveal behavioral changes in Go's standard library.

## Planning the Upgrade

### Before You Start

1. **Review Go Release Notes**
   - Identify breaking changes in the standard library
   - Check for deprecated features being removed
   - Note new language features that may conflict with existing code
   - Review security fixes (CVEs) driving the upgrade

2. **Check Infrastructure Support**
   - Verify the apollo-ci container supports the target version (or plan to use setup-go action)
   - Confirm Konflux go-builder image supports the target version (critical blocker)
   - Ensure local development environments can access the new version

3. **Plan for Breaking Changes**
   - Standard library behavioral changes (URL parsing, validation logic, etc.)
   - Runtime changes (goroutine scheduling, timing assumptions)
   - Compiler changes (stricter checks, new error messages)

## Version Declaration Updates

### 1. Primary Version Declaration

**File:** `go.mod`
```go
go X.Y.Z
```

This is the single source of truth for the Go version.

### 2. CI Workflow Configurations

**Principle:** Use `go-version-file: go.mod` to avoid version duplication.

**Workflows to update:**

- `.github/workflows/golangci-lint.yaml` - Linter must use same version as codebase
- `.github/workflows/unit-tests.yaml` - Test jobs (if not using go-version-file)

**Recommended approach:**
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: go.mod  # Single source of truth
```

**Alternative (hardcoded):**
```yaml
- uses: actions/setup-go@v5
  with:
    go-version: 'X.Y.Z'  # Must be updated manually
```

### 3. Tool Module Versions

Tool modules in `tools/` directories have their own `go.mod` files that should match the main version for consistency.

**Find and update:**
```bash
find . -name go.mod -path '*/tools/*' -exec sed -i 's/^go .*/go X.Y.Z/' {} \;
```

Common locations:
- `tools/generate-helpers/go.mod`
- `tools/roxvet/go.mod`
- Any other tool directories with go.mod files

## Understanding Test Failures

Go upgrades commonly expose test failures. These fall into three categories:

### 1. Standard Library Behavioral Changes

The Go standard library evolves between versions. Common areas of change:

**URL Parsing (`net/url` package):**
- Error message wording and priority changes
- Stricter validation rules
- IPv6 address handling

**Example pattern:**
```go
// Before: errString := `parse: invalid URL escape`
// After:  errString := `parse: invalid port after host`
```

**When this occurs:**
- Verify the new behavior is correct (often it's stricter/better)
- Update test expectations to match new error messages
- Consider if you should use project-specific utilities for consistency
- Document why the change was made

**Timing and Scheduling:**
- Go runtime scheduler changes between versions
- Tests with timing assumptions may become flaky
- Exposed race conditions that were previously masked

**When this occurs:**
- Increase timeout buffers for legitimate timing tests
- Add explicit synchronization instead of relying on timing
- Consider if the test is actually revealing a real race condition

### 2. Compiler and Type System Changes

**Stricter Type Checking:**
- More rigorous nil checks
- Improved escape analysis
- Generics-related inference changes

**When this occurs:**
- Fix the underlying issue rather than working around it
- The compiler is usually catching a real problem

### 3. Deprecated Feature Removal

**Common deprecations:**
- Old API methods removed
- Package reorganizations
- Build tag syntax changes

**When this occurs:**
- Migrate to the recommended replacement
- Check the Go release notes for migration guidance

## General Test Failure Strategy

For each failure during upgrade:

1. **Isolate:** Run the specific failing test locally
2. **Investigate:** Determine if it's a Go change or real bug
3. **Validate:** Check if the new behavior is more correct
4. **Fix:** Update tests to match new behavior (don't work around it)
5. **Document:** Add comments explaining the Go version dependency
6. **Commit:** Commit each fix individually for easier review

## CI Infrastructure Updates

### Understanding the Container Environment

**The Problem:**
GitHub Actions jobs can run in containers with pre-installed Go versions. When upgrading, you may encounter:
- Container has Go X.Y.Z baked in
- `go.mod` declares Go A.B.C (newer)
- `GOTOOLCHAIN=auto` downloads Go A.B.C
- But the container's Go X.Y.Z is still used from PATH

**The Symptom:**
```
compile: version "goA.B.C" does not match go tool version "goX.Y.Z"
```

**The Solution - Use setup-go Action:**

The `setup-go` action installs the specified Go version at the front of PATH, ensuring it's used instead of the container's version.

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
```

**Why this works:**
- Downloads and installs the correct Go version
- Prepends it to PATH ahead of the container's Go
- Works even when container has older version

**Alternative Approaches:**

1. **Wait for container update** - If apollo-ci container is updated with the new Go version
2. **Remove containers** - Switch to native GitHub runners (ubuntu-latest) instead of containers
3. **Use GOTOOLCHAIN alone** - Only works if container doesn't have Go pre-installed

**Recommendation:** Use setup-go action for fastest resolution and explicit version control.

### Build Cache Invalidation

**Why it's needed:**
Go build caches include metadata tied to the Go version. Upgrading Go without invalidating the cache can cause:
- Stale cached artifacts from old Go version
- Incompatible object files
- Mysterious compilation failures

**How to invalidate:**

Bump the cache key version in your cache action:

```yaml
# .github/actions/cache-go-dependencies/action.yaml
key: go-build-vN-${{ github.job }}...  # Increment N
```

**When to do it:**
- Always when upgrading minor versions (1.25 → 1.26)
- Usually when upgrading patch versions (1.26.1 → 1.26.2) to be safe
- After seeing cache-related failures

## External Build System Dependencies

### Red Hat Konflux

**Critical Constraint:** StackRox uses Red Hat Konflux for builds. Konflux has its own Go builder image that is updated independently.

**The Blocker:**
- Your PR updates `go.mod` to Go X.Y.Z
- Konflux go-builder only supports up to Go A.B.C
- Result: Konflux builds fail

**How to identify:**
```
Red Hat Konflux / checks: FAILURE
```

This typically means the Konflux builder doesn't support the Go version declared in go.mod.

**Resolution Path:**
1. Check current Konflux go-builder version
2. If it doesn't support your target version:
   - File a request to update the Konflux go-builder
   - Wait for the update (external dependency)
   - Cannot merge PR until Konflux is updated
3. If urgent (security CVEs), coordinate with Konflux team for expedited update

**Planning Tip:**
Check Konflux builder support BEFORE starting the upgrade to avoid delays.

## Upgrade Process Checklist

### Pre-Upgrade
- [ ] Review Go release notes for breaking changes
- [ ] Verify Konflux go-builder supports target version
- [ ] Check if security CVEs are driving the upgrade (affects urgency)
- [ ] Identify likely areas of breakage from release notes

### Code Changes
- [ ] Update `go.mod` with new Go version
- [ ] Update `.github/workflows/golangci-lint.yaml` (if not using go-version-file)
- [ ] Update all tool module `go.mod` files
- [ ] Bump cache version in `.github/actions/cache-go-dependencies/action.yaml`
- [ ] Ensure workflows use setup-go action with go-version-file

### Testing
- [ ] Run `make go-unit-tests` locally to identify obvious failures
- [ ] Push to branch and monitor all CI jobs
- [ ] Fix test failures iteratively (one commit per logical fix)
- [ ] Verify no behavioral regressions in passing tests
- [ ] Ensure all CI jobs pass (including Konflux)

### Documentation
- [ ] Update this guide with new patterns discovered
- [ ] Document any significant test changes with comments
- [ ] Note any workarounds in commit messages

## Troubleshooting Common Issues

### Version Mismatch Compilation Errors

**Symptom:**
```
compile: version "goX.Y.Z" does not match go tool version "goA.B.C"
```
Thousands of these errors during compilation.

**Root Cause:**
- CI container has Go version A.B.C pre-installed
- `go.mod` declares version X.Y.Z
- `GOTOOLCHAIN=auto` downloads X.Y.Z but doesn't override PATH
- Container's Go A.B.C is still used for compilation

**Solution:**
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
```

This installs the correct version ahead of the container's Go in PATH.

### Tests Pass Individually But Job Fails

**Symptom:**
All test output shows `PASS`, but `make` returns non-zero exit code.

**Root Cause:**
Version mismatch errors during compilation return non-zero status, failing the entire pipeline even though tests execute successfully.

**Solution:**
Fix the version mismatch (see above).

### Konflux Build Failures

**Symptom:**
```
Red Hat Konflux / checks: FAILURE
```

**Root Cause:**
Konflux go-builder doesn't support the Go version in go.mod yet.

**Solution:**
1. Verify Konflux builder version
2. Request update if needed
3. Wait for Konflux team to update builder
4. Cannot merge until resolved

### Flaky Tests After Upgrade

**Symptom:**
Tests that previously passed now occasionally fail.

**Root Cause:**
- Go runtime scheduler changes exposed timing assumptions
- Race conditions previously masked by timing

**Solution:**
- Add explicit synchronization instead of timing-based waits
- Increase timeout buffers for legitimate timing-sensitive tests
- Run with `-race` flag to detect actual race conditions

## Testing Strategy

### Local Testing First
1. **Build:** `make main-build` to catch compilation issues
2. **Unit tests:** `make go-unit-tests` to identify behavioral changes
3. **Linting:** `make golangci-lint` to catch new warnings
4. **Integration tests:** Run key integration tests if available

### CI-Driven Iteration
1. **Push to branch:** Don't push directly to main
2. **Monitor all jobs:** Check every CI job, not just unit tests
3. **Fix iteratively:** One logical fix per commit
4. **Document changes:** Commit messages should explain why tests changed

### Verification
1. **All tests pass:** Including Konflux, linters, integration tests
2. **No regressions:** Verify previously passing tests still pass
3. **Performance check:** Large Go upgrades can affect performance
4. **Local build works:** Ensure developers can build without CI

## Best Practices

### Single Source of Truth
Always use `go-version-file: go.mod` in workflows rather than hardcoding versions. This prevents version drift between go.mod and CI configuration.

### Atomic Commits
Each test fix should be a separate commit with clear explanation:
```
Fix pkg/foo/bar_test.go for Go X.Y.Z URL parser changes

Go X.Y.Z changed url.Parse() to validate ports before checking
escape sequences. Updated error expectations to match new behavior.

Ref: https://go.dev/doc/go1.XY#net/url
```

### Document Behavioral Changes
When fixing tests, add comments explaining the Go version dependency:
```go
// Go 1.26+ validates port syntax before URL escapes
errString := `parse: invalid port after host`
```

### Test Before Filing PR
Don't create the PR until:
- All CI jobs pass (or have documented known issues)
- Konflux builder support is confirmed
- Local builds work

## Security Considerations

### CVE-Driven Upgrades
When upgrading due to security fixes:
- Prioritize the upgrade
- Coordinate with Konflux team for faster builder updates
- Document which CVEs are being addressed
- Consider backporting to supported release branches

### Testing Security Fixes
- Verify the CVE is actually fixed in your codebase
- Check if the CVE affects your usage patterns
- Test security-critical code paths explicitly

## Impact on Downstream

### API Stability
Go upgrades can affect:
- Binary compatibility (if distributing compiled binaries)
- Generated code (protobuf, code generators)
- Vendored dependencies behavior

### Communication
Coordinate with teams that:
- Consume StackRox as a library
- Build against StackRox codebase
- Use StackRox-generated artifacts

## Updating This Guide

When you encounter new patterns during an upgrade:
1. Document the failure mode
2. Document the solution
3. Add to troubleshooting section
4. Include Go version where it occurred

This guide improves with each upgrade cycle.
