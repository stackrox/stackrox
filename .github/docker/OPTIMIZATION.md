# Docker Build Optimization Guide

This document captures lessons learned from optimizing DB image builds and provides guidelines for maintaining efficient Docker builds.

## Quick Reference: Dockerfile Layer Ordering

**Golden Rule**: Order layers from least-to-most frequently changing.

```dockerfile
# ✅ GOOD: Optimized layer order
FROM base:latest
ARG BUILD_ARG_THAT_NEVER_CHANGES

# Static content first
COPY --link static-file.txt /
ENV STATIC_VAR=value

# Expensive operations
RUN dnf upgrade -y && \
    # ... expensive package operations

# Content that changes occasionally
COPY --link scripts/ /usr/local/bin/

# Build args and labels LAST (change every build)
ARG VERSION
ARG GIT_SHA
LABEL version="${VERSION}" git-sha="${GIT_SHA}"

USER appuser
ENTRYPOINT ["/usr/local/bin/app"]
```

```dockerfile
# ❌ BAD: Labels before RUN invalidate cache
FROM base:latest
ARG VERSION  # Changes every build
LABEL version="${VERSION}"  # Invalidates all layers below!

RUN dnf upgrade -y  # Rebuilds every time despite no changes
COPY scripts/ /usr/local/bin/
```

## Debugging Cache Issues

### Symptom: "Some layers cache, others don't"

**Check for**:
1. **ARG/LABEL placement** - ARGs that change (git SHA, timestamps) invalidate all subsequent layers
2. **COPY order with --link** - COPY before RUN can break multi-platform caching
3. **Build args in bake config** - Check if bake passes changing values that affect early layers

**How to diagnose**:
```bash
# Compare two build logs
gh api /repos/stackrox/stackrox/actions/jobs/JOB_ID/logs | grep -E "CACHED|#[0-9]+ \["

# Look for pattern: early COPY marked CACHED, but RUN rebuilds
# This suggests COPY --link ordering issue or ARG invalidation between them
```

### Symptom: "Build rebuilds everything despite no code changes"

**Root cause**: Build args or context changes invalidating base layers.

**Check**:
1. Bake config `args = {}` - are you passing git SHA or timestamps to ARGs used in early layers?
2. `.dockerignore` - is build context including generated files that change every run?
3. Base image digests - using `FROM base:latest` without digest pins?

**Fix**:
```hcl
# ❌ BAD: Passes changing value to early-layer ARG
target "myapp" {
  args = {
    VERSION = "${GIT_SHA}"  # Used in LABEL before RUN
  }
}

# ✅ GOOD: ARG only used in final layer
# Move LABEL to end of Dockerfile
```

### Symptom: "COPY --link layer rebuilds every time"

**Root cause**: COPY --link placed BEFORE operations that invalidate in multi-platform builds.

**Pattern we saw**:
```dockerfile
# ❌ BAD: COPY scripts before RUN in multi-platform build
COPY --link init-bundle.tar /
COPY --link scripts/ /usr/local/bin/  # Rebuilds every time!
RUN dnf upgrade -y
```

**Fix**:
```dockerfile
# ✅ GOOD: RUN before COPY scripts
COPY --link init-bundle.tar /
RUN dnf upgrade -y  # Caches properly
COPY --link scripts/ /usr/local/bin/  # Now caches!
```

## Layer Ordering Checklist

When writing/reviewing Dockerfiles, verify this order:

1. **FROM and stage declarations**
2. **Static ARGs** (PG_VERSION, etc. - things that rarely change)
3. **Static ENV variables** (LANG, PATH)
4. **Static COPY operations** (files that never/rarely change)
5. **Expensive RUN operations** (package installs, compilations)
6. **Dynamic COPY operations** (application code, scripts)
7. **Dynamic ARGs** (VERSION, GIT_SHA, timestamps)
8. **LABEL with dynamic values**
9. **Metadata only** (USER, WORKDIR, EXPOSE, CMD, ENTRYPOINT)

**Why**: Docker caches layers sequentially. First layer change invalidates all subsequent layers. Expensive operations should come before frequently-changing content.

## COPY --link Best Practices

`COPY --link` creates independent layers for better caching and parallelism, but:

**✅ DO**:
- Use `--link` for files that change independently (scripts vs. binaries)
- Place `COPY --link` AFTER expensive RUN operations
- Use for files from different contexts in multi-stage builds

**❌ DON'T**:
- Place multiple `COPY --link` before RUN in multi-platform builds
- Assume `--link` automatically makes caching better (ordering still matters!)
- Use if files must be available during RUN (use regular COPY instead)

## Bake Configuration Patterns

### Cache Scopes

```hcl
# ✅ GOOD: Separate cache scopes per target
target "central-db" {
  cache-from = ["type=gha,scope=central-db"]
  cache-to = ["type=gha,mode=max,scope=central-db"]
}

target "scanner-v4-db" {
  cache-from = ["type=gha,scope=scanner-v4-db"]
  cache-to = ["type=gha,mode=max,scope=scanner-v4-db"]
}
```

**Why**: Separate scopes prevent cache pollution between images with different layer content.

### Build Args

```hcl
# ❌ BAD: Passing changing values to ARGs used early
target "app" {
  args = {
    VERSION = "${GIT_SHA}"  # If LABEL version="${VERSION}" is early in Dockerfile
  }
}

# ✅ GOOD: Only pass to ARGs used in final layers
# Or use --link and move LABEL to end
```

## Monitoring Build Performance

### Check cache hit rate:
```bash
# Count CACHED vs. total layers
gh api /repos/stackrox/stackrox/actions/jobs/JOB_ID/logs | \
  grep -E "#[0-9]+ \[" | \
  grep -c CACHED

# Expected for warm cache: ~90%+ layers cached
```

### Build time benchmarks:
- **Cold cache** (first build): ~2-3 minutes for DB images
- **Warm cache** (no changes): ~4-10 seconds for DB images
- **Partial cache** (code change): ~30-60 seconds

If warm cache builds take >30s, investigate cache invalidation.

## Common Pitfalls

### 1. Hidden ARG dependencies
```dockerfile
# ❌ ARG used in ENV, placed early
ARG VERSION=unknown
ENV APP_VERSION=${VERSION}  # Invalidates cache when VERSION changes!
RUN expensive-operation

# ✅ Move ENV after RUN, or use runtime config instead
```

### 2. Timestamp in LABEL
```dockerfile
# ❌ Changes every build
ARG BUILD_DATE
LABEL build-date="${BUILD_DATE}"  # Rebuilds everything below!
COPY app /app
RUN compile

# ✅ Move to end
RUN compile
COPY app /app
ARG BUILD_DATE
LABEL build-date="${BUILD_DATE}"
```

### 3. Multi-platform COPY order
```dockerfile
# ❌ Can break cache in multi-platform builds
COPY --link file1 /
COPY --link file2 /
RUN expensive-op

# ✅ Group static COPYs, RUN, then dynamic COPYs
COPY --link file1 /
RUN expensive-op
COPY --link file2 /
```

## Testing Cache Behavior

When making Dockerfile changes:

1. **First build** (establish cache):
   ```bash
   git commit -m "test: Establish cache baseline"
   git push
   # Wait for CI build to complete
   ```

2. **Second build** (verify cache):
   ```bash
   # Make trivial change (comment in non-Dockerfile)
   git commit -m "test: Verify cache works"
   git push
   # Check logs - should show CACHED for unchanged layers
   ```

3. **Compare logs**:
   ```bash
   # Build 1 (cache miss expected)
   grep CACHED build1.log | wc -l  # Should be low

   # Build 2 (cache hit expected)
   grep CACHED build2.log | wc -l  # Should be high
   ```

## Real-World Example: Scanner-v4-db Optimization

**Problem**: scanner-v4-db rebuilt RUN layer every build (~2min), central-db cached (~4s)

**Investigation**:
1. Both Dockerfiles nearly identical
2. Central-db showed `#17 CACHED`, scanner-v4-db ran `#26 RUN` (~90s)
3. Checked COPY order → scanner-v4-db had COPY scripts BEFORE RUN
4. Checked ARG usage → LABEL with changing ARGs BEFORE RUN

**Fixes**:
1. Moved `COPY --link scripts/` AFTER `RUN` (commit b1a7730417)
2. Moved ARG/LABEL to END of Dockerfile (commit c4728795ff)

**Result**: Build time 2min → 4s (97% reduction)

**Lessons**:
- COPY --link order matters in multi-platform builds
- ARG placement has cascading cache invalidation
- Compare working vs. broken Dockerfiles when debugging
- Test cache with minimal changes (comment-only commits)

## References

- Docker BuildKit cache: https://docs.docker.com/build/cache/
- COPY --link: https://docs.docker.com/reference/dockerfile/#copy---link
- GitHub Actions cache: https://docs.docker.com/build/ci/github-actions/cache/
