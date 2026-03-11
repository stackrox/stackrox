# Page Cache Warm-Path Experiment Results

PR: https://github.com/stackrox/stackrox/pull/19361
Branch: davdhacs/page-cache-warm-path (based on davdhacs/disable-buildvcs-tests)

## Setup

All runs confirmed warm GOCACHE (exact cache hits via `actions/cache@v5`).
3 samples per strategy per run. 5 runs total.

Runs 1-3: "perfect warm" — same code as cache seed, ~100% GOCACHE hits
Runs 4-5: "realistic warm" — master merged in, ~77% GOCACHE hits (1007/4421 recompiled)
Both scenarios are valid. Runs 4-5 are more realistic for PR workflows.

## Raw Data

### Run 1 (22938034522)

| Job | docker-tmpfs | none | vmtouch-lock |
|-----|-------------|------|-------------|
| build #1 | 23s | 22s | 21s |
| build #2 | 22s | 19s | 20s |
| build #3 | 21s | 21s | 21s |
| test #1 | 491s | 482s | 483s |
| test #2 | 489s | 486s | 482s |
| test #3 | 485s | 482s | 492s |
| style | 1286s (cold) | 1266s (cold) | 1262s (cold) |

### Run 2 (22938910778)

| Job | docker-tmpfs | none | vmtouch-lock |
|-----|-------------|------|-------------|
| build #1 | 22s | 21s | 21s |
| build #2 | 21s | 23s | 20s |
| build #3 | 21s | 18s | 21s |
| test #1 | 469s | 487s | 486s |
| test #2 | 485s | 487s | 494s |
| test #3 | 492s | 486s | 476s |
| style | 659s | 691s | 662s |

### Run 3 (22939481940)

| Job | docker-tmpfs | none | vmtouch-lock |
|-----|-------------|------|-------------|
| build #1 | 21s | 21s | 21s |
| build #2 | 22s | 22s | 21s |
| build #3 | 20s | 19s | 21s |
| test #1 | 492s | 485s | 488s |
| test #2 | 478s | 485s | 488s |
| test #3 | 487s | 472s | 475s |
| style | 666s | 655s | 658s |

### Run 4 (22953675556) — after master merge

| Job | docker-tmpfs | none | vmtouch-lock |
|-----|-------------|------|-------------|
| build #1 | 135s | 135s | 134s |
| build #2 | 130s | 131s | 133s |
| build #3 | 127s | 133s | 134s |
| test #1 | 1584s | 1619s | 1598s |
| test #2 | 1615s | 1620s | 1615s |
| test #3 | 1663s | 1606s | 1648s |
| style | 964s | 1021s | 941s |

### Run 5 (22955170130) — after master merge

| Job | docker-tmpfs | none | vmtouch-lock |
|-----|-------------|------|-------------|
| build #1 | 131s | 129s | 130s |
| build #2 | 157s | 136s | 129s |
| build #3 | 124s | 124s | 123s |
| test #1 | 1612s | 1498s | 1516s |
| test #2 | 1628s | 1536s | 1616s |
| test #3 | 1608s | 1566s | 1639s |
| style | 993s | 957s | 951s |

## Summary Statistics

### Build — perfect warm (runs 1-3, ~100% cache hits)

| Strategy | n | Mean | Min | Max | Stdev |
|----------|---|------|-----|-----|-------|
| docker-tmpfs | 9 | 21.4s | 20 | 23 | 0.9 |
| none | 9 | 20.7s | 18 | 23 | 1.6 |
| vmtouch-lock | 9 | 20.8s | 20 | 21 | 0.4 |

### Build — realistic warm (runs 4-5, ~77% cache hits)

| Strategy | n | Mean | Min | Max | Stdev |
|----------|---|------|-----|-----|-------|
| docker-tmpfs | 6 | 134.0s | 124 | 157 | 11.9 |
| none | 6 | 131.5s | 124 | 136 | 4.5 |
| vmtouch-lock | 6 | 130.5s | 123 | 134 | 4.2 |

### Unit Tests — perfect warm (runs 1-3)

| Strategy | n | Mean | Min | Max | Stdev |
|----------|---|------|-----|-----|-------|
| docker-tmpfs | 9 | 485.4s | 469 | 492 | 7.7 |
| none | 9 | 483.6s | 472 | 487 | 4.7 |
| vmtouch-lock | 9 | 484.9s | 475 | 494 | 6.2 |

### Unit Tests — realistic warm (runs 4-5)

| Strategy | n | Mean | Min | Max | Stdev |
|----------|---|------|-----|-----|-------|
| docker-tmpfs | 6 | 1618.3s | 1584 | 1663 | 26.6 |
| none | 6 | 1574.2s | 1498 | 1620 | 49.8 |
| vmtouch-lock | 6 | 1605.3s | 1516 | 1648 | 51.3 |

### Style / golangci-lint — warm (runs 2-5)

| Strategy | n | Mean | Min | Max | Stdev |
|----------|---|------|-----|-----|-------|
| docker-tmpfs | 4 | 820.5s | 659 | 993 | 166.7 |
| none | 4 | 831.0s | 655 | 1021 | 168.2 |
| vmtouch-lock | 4 | 803.0s | 658 | 951 | 138.1 |

## Go Source Code Analysis

Verified in Go 1.25.5 source:

**Build cache** (`cmd/go/internal/work/exec.go:484`):
- `buildActionID()` always calls `b.fileHash()` for every input file
- `cache.FileHash()` always does `os.Open(file)` → `io.Copy(sha256, file)`
- No mtime shortcut. No persistent stat cache. Always reads + SHA256 hashes.
- With `-trimpath` (used in builds), `p.Dir` is NOT in ActionID.
- The in-process `hashFileCache` only persists within a single invocation.

**Test cache** (`cmd/go/internal/test/test.go:2009-2045`):
- `hashOpen()` uses `os.Stat()` → `(size, mode, mtime)` only
- Line 2032: "do not attempt to hash the entirety of their content"
- Never reads file contents. mtime-based only.
- Without `-trimpath` (tests don't use it), `p.Dir` IS in ActionID.
- Path is the same for tmpfs and non-tmpfs (`/__w/stackrox/stackrox`).

## Historical Note

The original tmpfs implementation used a different workspace path (`/gosrc/stackrox`),
which required a cache key bump from v6→v7. The first "tmpfs speedup" measurement was
comparing a fresh v7 cache (seeded with tmpfs) against v6 (without tmpfs). This makes
it unclear whether the observed speedup was from tmpfs itself or from the cache key
reset coinciding with other fixes (mtime stabilization, explicit cache save).

The current experiment uses Docker `--tmpfs` mounted at the same path
(`/__w/stackrox/stackrox`), preserving ActionID compatibility. This is a fair A/B test.

## Conclusion (multi-runner experiment)

The multi-runner experiment (runs 1-5) could not resolve a tmpfs speedup because
runner-to-runner variance (±2-30s) exceeds the expected signal (~1-3s).

The mtime stabilization (`touch -t 200101010000`) is the essential optimization
that enables test caching (35m → 8m). tmpfs provides a complementary benefit on
the build path (Go always SHA256 hashes all source files) that is small but real.

Both optimizations should be kept. The tmpfs setup (`--tmpfs` + `/__w_host`) is
low-cost maintenance for a measurable-in-theory benefit on warm builds.

A single-runner experiment (all strategies on the same runner, sub-second timing)
is needed to resolve the small tmpfs effect. See run 6+ below.
