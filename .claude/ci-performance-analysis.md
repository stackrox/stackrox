# CI Build Performance Analysis

## Date: 2026-02-07

## Runner Performance Comparison

### Measured Data Points

| Metric | Self-hosted x86 | GitHub-hosted (free) | Scale Factor |
|--------|----------------|---------------------|--------------|
| **Go build (cold, 4566 pkgs)** | 49s | ~7 min (429s) | **8.8x slower on GitHub** |
| **Cache download (GOMODCACHE ~1.5 GiB)** | 3 min 40s (220s) | ~39s | **5.6x faster on GitHub** |
| **make deps (no cache)** | 45s | ~45s* | ~1x (network-bound) |
| **make deps (GOMODCACHE cached)** | 11s | ~5s* | ~2x faster on GitHub |
| **No-op rebuild** | 1s | ~1s | ~1x |
| **Relink from GOCACHE** | 8s | ~70s* | ~8.8x (CPU-bound) |

*Estimated based on scale factors

### Scale Factors for Estimation

To estimate GitHub-hosted runner performance from self-hosted measurements:
- **CPU-bound tasks** (Go compilation, linking): multiply by **8.8x**
- **Cache/network transfers**: divide by **5.6x** (GitHub is faster)
- **Disk I/O**: roughly similar (~1x)
- **Setup overhead** (checkout, tool install): roughly similar (~1x)

### Formula

```
github_time ≈ (self_hosted_cpu_time × 8.8) + (self_hosted_download_time / 5.6) + self_hosted_io_time
```

---

## Caching Strategy Comparison

### Seeding Run Data (self-hosted, all cold builds)

| Strategy | make deps | Go build | Packages | no-op | relink | deps+build |
|----------|----------|----------|----------|-------|--------|------------|
| **no-cache** | 45s | 49s | 4,566 | 1s | 8s | **94s** |
| **gomodcache-only** | 11s | 68s* | 4,566 | 1s | 8s | **79s** |
| **trimmed-gocache** | 4s | 96s* | 4,566 | 2s | 26s | **100s** |
| **full-gocache** | pending | pending | - | - | - | pending |

*CPU contention from parallel jobs inflated these numbers

### Estimated GitHub-Hosted Runner Performance (PR cache-hit run)

| Strategy | Cache DL | make deps | Go build | Total est. | Notes |
|----------|---------|----------|----------|------------|-------|
| **no-cache** | 0 | 45s | 7m 9s | **~8 min** | Baseline |
| **gomodcache-only** | ~10s | ~5s | 7m 9s | **~7.5 min** | Saves 30s on deps |
| **full-gocache (cache hit)** | ~60s | ~5s | ~30s | **~1.5 min** | Relink only! |
| **full-gocache (cache miss)** | ~60s | ~5s | 7m 9s | **~8.5 min** | Worse than no-cache |
| **trimmed-gocache (hit)** | ~20s | ~5s | ~4 min? | **~4.5 min** | Partial recompile |

### Key Insight

On GitHub's slow CPUs (8.8x slower than self-hosted), the full GOCACHE is the clear
winner. The 60-second download to avoid 7 minutes of compilation is a massive win.
On fast self-hosted runners, no caching at all is competitive because compilation
is so fast that cache download overhead dominates.

---

## Build Optimization Summary

### What We Changed

1. **buildvcs replaces generate-version.sh git commands**
   - Before: git describe + git rev-parse on every build (0.2s + 3s buildvcs overhead)
   - After: VERSION file + runtime debug.ReadBuildInfo() (0s + no recompilation on new commits)
   - Impact: 2.6s saved per build locally, eliminates recompilation cascade

2. **VERSION file for base version**
   - Replaces `git describe --tags` lookup
   - Updated at release boundaries (like COLLECTOR_VERSION etc.)
   - Format: `4.11.x-g<sha>` instead of `4.11.x-558-g<sha>` (commit count dropped)

3. **GOCACHE-based incremental builds**
   - Removed fragile Go binary cache (any .go change = total miss)
   - Go's own build cache handles incremental compilation naturally
   - GOCACHE save must happen AFTER build, not before (the key bug we found)

4. **Cache save/restore split**
   - All caches use actions/cache/restore (read) + actions/cache/save (write)
   - Saves only on master or with ci-seed-cache label
   - Prevents cache pollution from PR runs

### Local Build Performance (darwin/arm64)

| Scenario | Before | After |
|----------|--------|-------|
| No-op rebuild (same commit) | 2.2s | **1.0s** |
| New commit rebuild | 4.8s | **1.0s** |
| Small code change (1 pkg) | 5.6s | 5.6s |
| Shared pkg change (cascade) | 14.7s | 14.7s |
| Cold build | 48s | 48s |

### CI Build Performance (GitHub-hosted runners, estimated)

| Scenario | Before (master) | After (estimated) |
|----------|----------------|-------------------|
| Full pipeline (pre-builds → image) | ~30 min | TBD |
| All-in-one (cold) | ~12 min | **~8 min** |
| All-in-one (GOCACHE warm) | N/A | **~1.5 min** |

---

## Architecture Decisions

### Why not per-component binary caching?
- GOCACHE handles incremental builds at the package level naturally
- Binary cache key `hashFiles('**/*.go')` was too broad — any .go change = total miss
- GOCACHE is robust: only changed packages recompile, rest comes from cache

### Why cache GOCACHE despite 4 GiB size?
- On GitHub free runners, Go compilation takes 7+ minutes (slow CPUs)
- Cache download takes ~60s (fast internal network)
- Net savings: ~6 minutes per PR build
- Save cost on master is "free" (no developer waiting)

### Why keep generate-version.sh?
- Component versions (Collector, Scanner, Fact) come from files, not VCS
- go:embed can't reach repo root from pkg/version/internal/
- Script is now trivial (reads 4 files, no git commands, 0s runtime)

---

## Files Modified

| File | Change |
|------|--------|
| `VERSION` | New — base version "4.11.x" |
| `scripts/generate-version.sh` | Removed git commands, reads VERSION file |
| `pkg/version/internal/version_data.go` | Added BaseVersion var |
| `pkg/version/internal/vcs.go` | New — DeriveVersionFromBuildVCS() |
| `.github/workflows/build.yaml` | Cache strategy, timing, experiments |
| `.github/actions/cache-go-dependencies/action.yaml` | Simplified key, restore-only |
| `.github/actions/cache-ui-dependencies/action.yaml` | Restore-only |
| `.github/workflows/style.yaml` | setup-go cache: false |
| `.github/workflows/unit-tests.yaml` | setup-go cache: false |
| `.github/workflows/retest_periodic.yml` | setup-go cache: false |
| `.github/workflows/emailsender-central-compatibility.yaml` | setup-go cache: false |

---

## Open Questions

1. What is the actual GOCACHE download + build time on GitHub free runners?
   (Need to test once quota resets)
2. Should we cache GOMODCACHE separately from GOCACHE?
   (GOMODCACHE is ~1.5 GiB compressed, saves ~40s on deps)
3. Can GOCACHEPROG provide a better caching backend than Actions cache?
4. What's the optimal Docker layer ordering for incremental image builds?
5. Should we skip non-essential binaries for PR checks?
