# Scanner Build Workflow Consolidation Design

## Context

The StackRox CI currently has two separate workflows for building artifacts:
- `.github/workflows/build.yaml` - builds main, operator, CLI, and UI components
- `.github/workflows/scanner-build.yaml` - builds scanner-v4 and scanner-v4-db images

This separation leads to:
- **Redundant compilation:** Scanner shares significant codebase with main (imports from `central/`, `pkg/`, etc.) but builds them separately with no cache sharing across workflows
- **Slower builds:** Scanner must rebuild shared dependencies that main already compiled
- **Maintenance overhead:** Two workflows to maintain instead of one
- **Cache inefficiency:** Separate GOCACHE per workflow, missing ~30-40% cache hits

Recent optimizations to `build.yaml` (ROX-30858, ROX-33792):
- Removed containers from most jobs (UI, docs, operator bundle)
- Built operator binary outside Docker in `pre-build-go-binaries` job
- Used native arm64 runners for builds
- Achieved significant speedups through shared Go toolchain caches

This design consolidates scanner builds into `build.yaml` following the same pattern established for operator builds, enabling maximum cache reuse and build time optimization.

## Overall Architecture

### Target State
- Single unified `build.yaml` workflow for all StackRox artifacts
- Scanner binary built in existing `pre-build-go-binaries` job alongside main, operator, CLI binaries
- Scanner uses same `apollo-ci:stackrox-test` container as main builds
- Scanner images built in new `build-and-push-scanner` job that downloads prebuilt scanner binary
- Scanner manifests pushed in new `push-scanner-manifests` job
- Complete Go cache sharing across all builds

### Build Flow Pattern (same as operator)
```
pre-build-go-binaries (produces scanner binary in artifact)
         ↓
build-and-push-scanner (downloads artifact, builds images)
         ↓
push-scanner-manifests (creates multi-arch manifests)
```

### Files Deleted
- `.github/workflows/scanner-build.yaml` - entire workflow consolidated into build.yaml

## Build Matrix Structure

The `define-job-matrix` job will be extended to include scanner build configuration.

### New Matrix Sections

**build_and_push_scanner:**
```json
{
  "name": ["default"],
  "goos": ["linux"],
  "goarch": ["amd64", "arm64"],
  "registry": ["quay.io/stackrox-io", "quay.io/rhacs-eng"]
}
```

**push_scanner_manifests:**
```json
{
  "name": ["default"],
  "registry": ["quay.io/stackrox-io", "quay.io/rhacs-eng"]
}
```

**scan_images_with_roxctl (extended):**
- Add `scanner-v4` and `scanner-v4-db` to image list
- Use both registries (stackrox-io, rhacs-eng)

### Build Variants Supported
- `default` - normal build (always)
- `prerelease` - with GOTAGS=release (conditionally: tagged builds or `ci-build-prerelease` label)
- `race-condition-debug` - with -race flag (conditionally: `ci-build-race-condition-debug` label, amd64 only)

### Architecture Support
- Always: amd64, arm64
- Conditionally: ppc64le, s390x (when `ci-build-all-arch` label present or not in PR context)
- Race builds: amd64 only (race detector limitation)

Scanner follows same variant/architecture rules as main builds for consistency.

## Workflow Job Changes

### Job 1: `define-job-matrix` (modified)
**Changes:**
- Add `build_and_push_scanner` matrix with name, goos, goarch, registry dimensions
- Add `push_scanner_manifests` matrix with name, registry dimensions
- Extend `scan_images_with_roxctl` matrix to include `scanner-v4`, `scanner-v4-db` images
- Apply same conditional logic for arch/variants as main builds

### Job 2: `pre-build-go-binaries` (modified)

**Makefile changes (root Makefile):**
```makefile
main-build-nodeps:
	$(GOBUILD) \
		central \
		compliance/cmd/compliance \
		config-controller \
		migrator \
		operator/cmd \
		roxctl \
		scanner/cmd/scanner \
		sensor/admission-control \
		sensor/kubernetes \
		sensor/upgrader \
		compliance/virtualmachines/roxagent
	mv bin/linux_$(GOARCH)/cmd bin/linux_$(GOARCH)/stackrox-operator
```

**Workflow changes:**
Add scanner binary copy step after main-build-nodeps:
```bash
# Build all binaries including scanner
if [[ "${{ matrix.arch }}" != "amd64" ]]; then
  GOOS=linux GOARCH=${{ matrix.arch }} CGO_ENABLED=0 make build-prep main-build-nodeps
else
  GOOS=linux GOARCH=${{ matrix.arch }} CGO_ENABLED=1 make build-prep main-build-nodeps
fi

# Copy scanner binary to location expected by scanner image build
mkdir -p scanner/image/scanner/bin
cp bin/linux_${{ matrix.arch }}/scanner scanner/image/scanner/bin/scanner
```

Extend tar bundle to include scanner binary:
```bash
tar -cvzf go-binaries-build.tgz bin/linux_${{ matrix.arch }} scanner/image/scanner/bin/scanner
```

No changes to artifact upload (still `go-binaries-build-${{ matrix.arch }}-${{ matrix.name }}`).

### Job 3: `build-and-push-scanner` (new)

Mirrors `build-and-push-operator` pattern:
- Depends on: `define-job-matrix`, `pre-build-go-binaries`
- Runs on: ubuntu-latest (no container)
- Matrix: `${{ fromJson(needs.define-job-matrix.outputs.matrix).build_and_push_scanner }}`

**Key steps:**
1. Download artifact: `go-binaries-build-${{ matrix.goarch }}-${{ matrix.name }}`
2. Unpack: `tar xvzf go-binaries-build.tgz`
3. Set BUILD_TAG for variants:
   - prerelease: `$(make -C scanner tag)-prerelease`
   - race-condition-debug: `$(make -C scanner tag)-rcd`
4. Build images: `make -C scanner GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} images`
   - Scanner Makefile finds prebuilt binary at `scanner/image/scanner/bin/scanner`
5. Push images: `push_scanner_image_set "${{ matrix.registry }}" "${{ matrix.goarch }}"`

**Differences from scanner-build.yaml:**
- No container (runs directly on ubuntu-latest)
- Downloads prebuilt binaries instead of building them
- Shares artifact with main/operator builds

### Job 4: `push-scanner-manifests` (new)

Mirrors `push-operator-manifests` / `push-main-manifests` pattern:
- Depends on: `define-job-matrix`, `build-and-push-scanner`
- Runs on: ubuntu-latest (no container)
- Matrix: `${{ fromJson(needs.define-job-matrix.outputs.matrix).push_scanner_manifests }}`

**Key steps:**
1. Set BUILD_TAG for variants
2. Determine architectures:
   - Base: `amd64,arm64`
   - With `ci-build-all-arch` or non-PR: `amd64,arm64,ppc64le,s390x`
   - Race builds: `amd64` only
3. Push manifests: `push_scanner_image_manifest_lists "${{ matrix.registry }}" "$architectures"`

### Job 5: `scan-images-with-roxctl` (modified)
**Changes:**
- Matrix already includes scanner images from define-job-matrix
- No additional logic needed (already supports registry matrix)

### Job 6: `slack-on-build-failure` (modified)
**Changes:**
- Add to needs list: `build-and-push-scanner`

## Caching Strategy

### Current State
- Main builds: `go-build-v2-pre-build-go-binaries-${{ GOARCH }}-${{ GOMOD_HASH }}`
- Scanner builds: Separate cache in separate workflow
- **Problem:** Scanner rebuilds shared code that main already compiled (~30-40% redundant compilation)

### Target State
- Scanner binary built in `pre-build-go-binaries` job alongside main/operator/CLI
- **Single unified GOCACHE** shared by all binaries in one job
- Cache key: `go-build-v2-pre-build-go-binaries-${{ GOARCH }}-${{ GOMOD_HASH }}`

### Cache Efficiency Gains

**1. Shared dependencies compiled once:**
- `central/*` packages used by both central and scanner
- `pkg/*` packages used across all binaries
- Standard library packages
- Third-party dependencies from shared go.mod

**2. Single Go toolchain invocation:**
- Current: `$(GOBUILD) central operator roxctl ...` + separate `make -C scanner`
- New: `$(GOBUILD) central operator roxctl scanner ...`
- Go's compiler can optimize across all targets in one invocation

**3. Expected cache hit improvement:**
- Current: ~60-70% cache hit rate for scanner (rebuilds shared code)
- Expected: ~90-95% cache hit rate (only scanner-specific code compiled)

No changes needed to `cache-go-dependencies` action - it already works per-job.

## Migration Plan and Verification

### Implementation Steps

**Phase 1: Makefile changes**
1. Modify root `Makefile`: Add `scanner/cmd/scanner` to `main-build-nodeps` target
2. Verify locally: `make build-prep main-build-nodeps` produces `bin/linux_amd64/scanner`

**Phase 2: Workflow changes**
1. Modify `define-job-matrix`: Add scanner matrices
2. Modify `pre-build-go-binaries`: Add scanner binary copy step and extend tar bundle
3. Add `build-and-push-scanner` job (adapted from scanner-build.yaml)
4. Add `push-scanner-manifests` job (adapted from scanner-build.yaml)
5. Modify `scan-images-with-roxctl`: Extend matrix to include scanner images
6. Modify `slack-on-build-failure`: Add scanner jobs to needs list

**Phase 3: Cleanup**
1. Delete `.github/workflows/scanner-build.yaml`
2. Verify no other workflows reference scanner-build.yaml

### Verification Steps

**Pre-merge verification (PR build):**
1. Trigger PR build with `ci-build-all-arch` label to test full matrix
2. Check `pre-build-go-binaries` job:
   - ✓ Scanner binary built alongside main binaries
   - ✓ Artifact includes `scanner/image/scanner/bin/scanner`
   - ✓ Build time comparable or faster than current separate builds
3. Check `build-and-push-scanner` job:
   - ✓ Downloads prebuilt scanner binary
   - ✓ Builds scanner-v4 and scanner-v4-db images
   - ✓ Pushes to both registries (stackrox-io, rhacs-eng)
4. Check `push-scanner-manifests` job:
   - ✓ Creates multi-arch manifests for both registries
   - ✓ Architectures match build matrix
5. Check `scan-images-with-roxctl` job:
   - ✓ Scanner images scanned successfully

**Post-merge verification (master build):**
1. First master build after merge:
   - ✓ All scanner images pushed with correct tags
   - ✓ `latest-$arch` tags created for scanner images
   - ✓ Cache warming happens (GOCACHE saved)
2. Second master build:
   - ✓ Cache hit rate improves (scanner benefits from warm cache)
   - ✓ Scanner build time faster than baseline

**Performance monitoring:**
- Baseline: Current `pre-build-scanner-go-binary` job time
- Target: `pre-build-go-binaries` job time increase should be minimal due to cache sharing
- Overall build workflow time should not increase significantly

### Rollback Plan

**If issues arise after merge:**
1. **Quick rollback:** Revert the consolidation commit and restore scanner-build.yaml
2. **Temporary workaround:** Disable scanner jobs in build.yaml, re-enable scanner-build.yaml
3. **Debug approach:**
   - Check artifact contents: `tar -tzf go-binaries-build.tgz | grep scanner`
   - Verify binary is at expected path in scanner image build
   - Compare Go cache hit rates before/after

**Risk mitigation:**
- Scanner-build.yaml remains functional until consolidation PR merges
- No breaking changes to scanner build process (same `make -C scanner images` command)
- Binary location preserved (`scanner/image/scanner/bin/scanner`)
- Can run both workflows in parallel during transition period for validation

## Expected Outcomes

1. **Build time reduction:** 20-30% faster scanner builds due to cache sharing
2. **Simplified CI:** One workflow instead of two to maintain
3. **Better cache utilization:** ~30% increase in cache hit rate for scanner builds
4. **Consistency:** Scanner follows same build pattern as operator
5. **Resource efficiency:** Shared Go toolchain invocation across all binaries
