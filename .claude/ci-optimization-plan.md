# CI Build Optimization Plan: Sub-3-Minute Builds

## Context

The all-in-one CI job currently takes ~12 minutes even with caching enabled. For a "no code changes" scenario (e.g., documentation-only changes or re-running a failed job), we should be able to achieve sub-3-minute builds by fully leveraging caches.

**Current State:**
- Go dependency cache: Working (setup-go style keys)
- UI build cache: Working (skips build on hash match)
- Docker layer cache: Working (registry-based)
- Go build cache: Partially working (caches compiled packages, but still rebuilds)

**Goal:** When no Go/UI source files change, build should complete in under 3 minutes.

---

## Problem Analysis

### Why Go Builds Still Take 5+ Minutes with Cache Hits

Based on exploration of the codebase:

1. **Version file regeneration**: `scripts/generate-version.sh` runs on every build, regenerating `pkg/version/internal/version_data_generated.go`. This involves slow `git describe` operations.

2. **Cache key doesn't capture build outputs**: The Go cache key is based on `go.sum` hash, not source file hashes. Even with a dependency cache hit, Go must:
   - Validate all package freshness against source files
   - Recompile any changed packages
   - Re-link all binaries

3. **Environment variable differences**: CGO_ENABLED, GOTAGS, and DEBUG_BUILD flags can invalidate cache entries between different build types.

4. **No binary caching**: We cache dependencies but not compiled binaries. Each build recompiles from source.

### Current Caching Architecture

| Component | Cache Type | Cache Key | What's Cached |
|-----------|------------|-----------|---------------|
| Go deps | actions/cache | `setup-go-{platform}-{arch}-{goversion}-{go.sum hash}` | GOCACHE + GOMODCACHE |
| UI build | actions/cache | `built-ui-{branding}-{hashFiles(ui/**)}` | `ui/build/` directory |
| UI deps | actions/cache | `npm-v3-{package-lock hash}` | `node_modules/` |
| Docker layers | registry cache | `main-{arch}-master-{branding}` | Image layers in ghcr.io |

---

## Research Tasks

### Phase 1: Measurement & Baseline

1. **Add timing instrumentation to all-in-one job**
   - Measure each step: cache restore, Go build, UI build, Docker build
   - Identify which steps take longest on cache hits vs misses
   - Files: `.github/workflows/build.yaml` (all-in-one job)

2. **Analyze Go build cache effectiveness**
   - Check if GOCACHE is being used effectively
   - Verify cache key matches between runs
   - Test: Run same commit twice, compare build times

3. **Measure Docker layer cache hit rates**
   - Check which layers are being reused
   - Identify layers that always rebuild
   - Files: `image/rhel/Dockerfile`, Docker build logs

### Phase 2: Quick Wins

4. **Cache Go binaries directly**
   - Add cache for `bin/linux_amd64/` directory
   - Key: `go-binaries-{arch}-{hashFiles('**/*.go', 'go.sum', 'go.mod')}`
   - Skip Go build entirely if cache hits
   - Expected savings: 5+ minutes

5. **Optimize version generation**
   - Cache git describe output or compute once per workflow
   - Pass version via environment variable instead of regenerating
   - Files: `scripts/generate-version.sh`

6. **Skip builds based on path filters**
   - If only docs/tests changed, skip Go build
   - Use `dorny/paths-filter` action to detect changes
   - Files: `.github/workflows/build.yaml`

### Phase 3: Architectural Improvements

7. **Evaluate setup-go built-in caching vs custom action**
   - setup-go has built-in dependency caching
   - Compare performance with custom `cache-go-dependencies` action
   - Files: `.github/actions/cache-go-dependencies/action.yaml`

8. **Consider local Docker cache instead of registry**
   - Registry cache has network overhead
   - Local cache (`type=local`) is faster but doesn't persist
   - Hybrid approach: local for same-job, registry for cross-job

9. **Pre-built base images**
   - Create base images with common layers (RPMs, Go runtime)
   - Update weekly or on dependency changes
   - Reduces per-build layer count

---

## Immediate Fixes Needed (Priority 1)

### Fix Experimental Jobs (Blocking)

The experimental jobs are currently failing and must be fixed before we can compare approaches:

1. **experimental-go-in-docker**: Failing at "Build with Go-in-Docker" step
   - Investigate `image/rhel/Dockerfile.go-in-docker`
   - Check for missing files/dependencies
   - Verify COPY paths match actual binary locations
   - Files: `image/rhel/Dockerfile.go-in-docker`

2. **experimental-go-same-job**: Failing at "Extract UI" step
   - UI artifact download/extraction is broken
   - Check artifact name matches what pre-build-ui uploads
   - Verify extraction path is correct
   - Files: `.github/workflows/build.yaml` (lines 1651-1670)

**Action**: Debug these failures by checking workflow logs and fixing the configuration.

---

## Proposed Implementation Order

### Week 1: Measurement & Quick Fixes
- [ ] Add timing to all-in-one job
- [ ] Fix experimental job failures
- [ ] Implement Go binary caching

### Week 2: Optimization
- [ ] Optimize version generation
- [ ] Add path-based build skipping
- [ ] Evaluate setup-go caching

### Week 3: Architecture
- [ ] Evaluate Docker cache alternatives
- [ ] Implement pre-built base images
- [ ] Document final architecture

---

## Success Metrics

| Scenario | Current | Target |
|----------|---------|--------|
| No code changes | ~12 min | < 3 min |
| Go-only changes | ~12 min | < 5 min |
| UI-only changes | ~12 min | < 5 min |
| Full rebuild | ~12 min | < 10 min |

---

## Key Files

- `.github/workflows/build.yaml` - Main workflow, all-in-one job
- `.github/actions/cache-go-dependencies/action.yaml` - Go caching action
- `scripts/go-build.sh` - Go build script
- `scripts/generate-version.sh` - Version generation (optimization target)
- `image/rhel/Dockerfile` - Main image Dockerfile
- `image/rhel/Dockerfile.go-in-docker` - Experimental Go-in-Docker approach
- `Makefile` - Build targets and skip logic

---

## Open Questions

1. Should we cache binaries at the workflow level or job level?
2. Is the current Go cache key format optimal?
3. Should experimental jobs be removed or fixed?
4. What's the acceptable cache storage cost increase?
