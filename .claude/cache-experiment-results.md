# Cache Experiment Results — 2026-02-07

## Cache-Hit Test (self-hosted x86 runners)

### Measured Results

| Strategy | Cache DL | make deps | Go build | Pkgs recompiled | no-op | relink | **Total job** |
|----------|---------|----------|----------|----------------|-------|--------|--------------|
| **no-cache** | 0s | 46s | 48s | 4,566 | 1s | 9s | **2m 10s** |
| **gomodcache-only** | 3m 48s | 4s | 45s | 4,566 | 1s | 5s | **6m 0s** |
| **trimmed-gocache** | 4m 4s + 1m 50s | 3s | **5s** | **9** | 0s | 5s | **6m 33s** |
| **full-gocache** | 3m 28s + 1m 32s | 3s | **6s** | **9** | 0s | 5s | **7m 26s** |

### Step-by-step breakdown

**no-cache (2m 10s total):**
- Cache restore: skipped
- make deps (download modules): 46s
- Go build: 48s (4,566 packages — full compile)
- Experiments: 1s no-op, 9s relink

**gomodcache-only (6m 0s total):**
- GOMODCACHE restore: 3m 48s (1.6 GiB download — slow self-hosted network)
- make deps: 4s (modules already cached)
- Go build: 45s (4,566 packages — full compile, no GOCACHE)
- Experiments: 1s no-op, 5s relink

**trimmed-gocache (6m 33s total):**
- GOMODCACHE restore: 4m 4s
- GOCACHE restore: 1m 50s (935 MiB)
- make deps: 3s
- **Go build: 5s (9 packages recompiled!)** — GOCACHE HIT!
- Experiments: 0s no-op, 5s relink

**full-gocache (7m 26s total):**
- GOMODCACHE restore: 3m 28s
- GOCACHE restore: 1m 32s (937 MiB)
- make deps: 3s
- **Go build: 6s (9 packages recompiled!)** — GOCACHE HIT!
- Experiments: 0s no-op, 5s relink

### Key Finding

The GOCACHE **works**. Both `trimmed-gocache` and `full-gocache` reduced the Go build
from **48s (4,566 packages)** to **5-6s (9 packages)**. The 9 recompiled packages are
the project's main packages (the binary entry points) which always relink.

The only reason the cached strategies show LONGER total times is the **cache download**
is slow on the self-hosted runners (~5 min for both caches). On GitHub runners with
their fast internal network, the download would be ~5.6x faster.

---

## Estimated GitHub Runner Performance

### Scale factors (measured)
- CPU: GitHub is **8.8x slower** than self-hosted
- Network: GitHub is **5.6x faster** for cache downloads

### Projected timings

| Strategy | Cache DL | make deps | Go build | **Est. total** | **vs no-cache** |
|----------|---------|----------|----------|---------------|----------------|
| **no-cache** | 0s | 46s | **7m 2s** | **~8 min** | baseline |
| **gomodcache-only** | ~41s | 4s | **6m 36s** | **~7.5 min** | -30s |
| **trimmed-gocache** | ~41s + ~20s | 3s | **~44s** | **~1m 48s** | **-6 min** |
| **full-gocache** | ~37s + ~16s | 3s | **~53s** | **~1m 49s** | **-6 min** |

Calculations:
- no-cache Go build: 48s × 8.8 = 422s = 7m 2s
- gomodcache-only Go build: 45s × 8.8 = 396s = 6m 36s
- GOMODCACHE download: 228s / 5.6 = 41s
- GOCACHE (trimmed) download: 110s / 5.6 = 20s
- GOCACHE (full) download: 92s / 5.6 = 16s
- Cached Go build: 5s × 8.8 = 44s (mostly linking, some CPU)
- make deps (cached): ~4s (mostly local, not CPU-bound)
- make deps (uncached): ~46s (network-bound, similar on both)

---

## Recommendation

### For GitHub-hosted runners: **full-gocache** or **trimmed-gocache**

Both reduce the Go build step from **~7 minutes to ~50 seconds** on GitHub runners.
The cache download adds ~55 seconds, for a net savings of **~6 minutes per PR build**.

- `full-gocache` (937 MiB): simplest, saves everything, slightly larger
- `trimmed-gocache` (935 MiB): almost identical size after trimming

Since they're nearly the same size and performance, **full-gocache is simpler** —
no trimming logic needed.

### Cache budget impact

| Cache entry | Size | Saved by |
|-------------|------|----------|
| GOMODCACHE | 1,630 MiB | master push |
| GOCACHE (full) | 937 MiB | master push |
| **Total** | **2,567 MiB** | ~2.5 GiB of 10 GiB budget |

### Implementation

1. On **master pushes**: save GOCACHE (post-build) + GOMODCACHE separately
2. On **PR runs**: restore both, build with warm cache (~50s instead of ~7 min)
3. Cache keys: `gocache-{platform}-{arch}-go-{version}-{sha}` with prefix fallback
4. Old cache cleanup: delete previous entries after saving new one

### What the 9 recompiled packages are

The 9 "recompiled" packages are the 9 binary entry points (central, compliance,
config-controller, migrator, admission-control, kubernetes, upgrader, roxctl,
roxagent). These always need relinking even with a warm GOCACHE, because the
output binary doesn't exist yet. This is the irreducible minimum.
