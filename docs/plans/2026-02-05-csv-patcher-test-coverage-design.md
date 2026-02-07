# Test Coverage Improvements for csv-patcher

**Date:** 2026-02-05
**Status:** Approved
**Coverage Target:** 50% of diff (currently 37.96%)

## Problem

CodeCov is failing PR checks with 37.96% coverage of the diff (target: 49.43%). The csv-patcher tool has two functions with 0% coverage that are actively used in production:

- `injectRelatedImageEnvVars()` - Used in konflux build mode
- `constructRelatedImages()` - Used in konflux build mode

## Solution

Add focused unit tests for these two functions only. Skip testing CLI entry points and defensive error handling that's already validated by operator-sdk.

## Test Design

### 1. `injectRelatedImageEnvVars()` Tests

**Location:** `operator/bundle_helpers/csv-patcher/patch_test.go`

**Function behavior:** Recursively traverses CSV spec and injects environment variable values for any field with `name` starting with `RELATED_IMAGE_`.

**Test cases:**

1. **TestInjectRelatedImageEnvVars_SingleEnvVar**
   - Set `RELATED_IMAGE_MAIN` env var
   - Create spec with deployment containing `{name: "RELATED_IMAGE_MAIN"}`
   - Verify `value` field is set from env var

2. **TestInjectRelatedImageEnvVars_MultipleNested**
   - Set multiple RELATED_IMAGE_* env vars (MAIN, SCANNER, SCANNER_DB)
   - Create nested spec structure (deployment → containers → env)
   - Verify all RELATED_IMAGE_* fields get values injected

3. **TestInjectRelatedImageEnvVars_MissingEnvVar**
   - Create spec with `{name: "RELATED_IMAGE_NONEXISTENT"}`
   - Don't set the env var
   - Verify error includes variable name

4. **TestInjectRelatedImageEnvVars_NoRelatedImages**
   - Create spec without RELATED_IMAGE_* references
   - Verify no errors, spec unchanged

### 2. `constructRelatedImages()` Tests

**Location:** `operator/bundle_helpers/csv-patcher/patch_test.go`

**Function behavior:** Collects all `RELATED_IMAGE_*` environment variables, builds `relatedImages` array with lowercase names, adds manager image.

**Test cases:**

1. **TestConstructRelatedImages_MultipleEnvVars**
   - Set RELATED_IMAGE_MAIN, RELATED_IMAGE_SCANNER env vars
   - Call with manager image
   - Verify `spec["relatedImages"]` contains entries with lowercase names ("main", "scanner") plus "manager"

2. **TestConstructRelatedImages_NoEnvVars**
   - Unset all RELATED_IMAGE_* env vars
   - Verify only "manager" entry exists

3. **TestConstructRelatedImages_NameTransformation**
   - Set `RELATED_IMAGE_SCANNER_DB_SLIM`
   - Verify name becomes "scanner_db_slim" (lowercase with underscores)

## Expected Coverage Impact

- `injectRelatedImageEnvVars()`: 0% → ~100%
- `constructRelatedImages()`: 0% → ~100%
- Overall csv-patcher coverage: 44.0% → ~60%+
- Diff coverage: 37.96% → >49.43% (target met)

## Non-Goals

- Testing `main()` functions (CLI entry points)
- Testing defensive error handling in `addSecurityPolicyCRD()` (operator-sdk validates CSV format)
- Testing `echoReplacedVersion()` or `splitComma()` (can add later if needed)

## Implementation Notes

- Use testify for assertions (already in use)
- Clean up env vars in test teardown
- Use table-driven tests where appropriate
