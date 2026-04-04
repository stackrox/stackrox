# Go Build Cache Strategy for CI

## Summary

This document describes the Go build cache optimization for StackRox CI,
including the `netgo,osusergo` discovery that enables full GOCACHE sharing
between CGO_ENABLED=0 (CLI) and CGO_ENABLED=1 (main binaries) builds.

### Key results

- **CLI linux/amd64**: 172s cold → **24s warm** (7x speedup, avg over 8 runs)
- **CLI linux/arm64**: 182s cold → **12s warm** (15x speedup)
- **go-binaries amd64**: 353s cold → **22s warm** (16x speedup, avg over 8 runs)
- **Critical path**: ~22min (old single-job CLI) → **~5min** (matrix + shared cache)
- **Cache budget**: ~4.2 GB of 10 GB used (vs ~9 GB before optimization)

## Background

Go provides two caches relevant to CI builds:

- **GOCACHE** — compiled package objects, keyed by content hash
- **GOMODCACHE** — downloaded module source code

Both can be persisted across CI runs with `actions/cache`.

### Go cache documentation

From [`go help cache`](https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching):

> The go command caches build outputs for reuse in future builds [...] The
> build cache correctly accounts for changes to Go source files, compilers,
> compiler options, and so on.

Go's GOCACHE is content-addressed. Cache keys are computed from compiler
version, GOOS/GOARCH, build flags, source file content hashes, and import
dependency hashes. CGO_ENABLED is **not** directly included — it only affects
cache keys indirectly by changing which source files are selected via build
constraints.

### GODEBUG cache diagnostics

- `GODEBUG=gocacheverify=1` — rebuild everything, verify cached results match
- `GODEBUG=gocachehash=1` — print cache key inputs for all packages
- `GODEBUG=gocachetest=1` — print test cache reuse decisions

### GitHub Actions cache limits

- **10 GB per repository** across all branches and workflows
- Entries not accessed in **7 days** are evicted (LRU)
- All branches share the same pool, but access is scoped:
  - PRs can only access caches from their own ref or the default branch
  - Caches from other PRs are invisible but still consume budget

Source: [GitHub docs — Caching dependencies](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/caching-dependencies-to-speed-up-workflows)

## The `netgo,osusergo` Discovery

### Problem

On Linux, the `net` package compiles different source files depending on
CGO_ENABLED:

- **CGO_ENABLED=1**: `cgo_unix.go` + 5 C source files (system DNS resolver)
- **CGO_ENABLED=0**: `cgo_stub.go` (pure Go DNS resolver)

Different source files → different content hash → different cache key.
This cascades: any package importing `net` (directly or transitively) also
gets a different cache key. Analysis showed **793 of 1,569 roxctl packages**
(50%) are affected.

Only 4 stdlib packages have CGO-conditional source files:
`net`, `os/user`, `plugin`, `runtime/cgo`.

### Solution

Go provides the `netgo` and `osusergo` build tags to force the pure Go
implementations regardless of CGO_ENABLED:

```
//go:build !cgo || netgo    ← cgo_stub.go (selected with netgo tag)
//go:build cgo && !netgo    ← cgo_unix.go (excluded with netgo tag)
```

With `-tags=netgo,osusergo`:
- `net` compiles **identical files** regardless of CGO_ENABLED
- `os/user` compiles **identical files** regardless of CGO_ENABLED
- All 1,569 roxctl packages produce **identical cache entries**

Verified empirically — cache export paths are identical for `net`, `net/http`,
`os/user`, and `google.golang.org/grpc` between CGO=0 and CGO=1 when these
tags are used.

**Important**: The build tags themselves are part of the cache key for packages
where they change the build constraint evaluation. Both CGO=0 and CGO=1 builds
must use the **same tags** for sharing to work.

### Implementation

Tags are set globally in two places, **guarded to CI only**:

- **`make/env.mk`**: `ifdef CI` → `GOTAGS := ...netgo,osusergo`
  — applies to all `make` targets in CI via `go-build.sh` and `go-test.sh`
  — NOT applied in Konflux/release builds (which don't set `CI` or use
    their own Dockerfiles with explicit GOTAGS)
- **`.golangci.yml`**: `build-tags: [... netgo, osusergo]`
  — applies to all golangci-lint analysis passes (CI only)

The `ifdef CI` guard ensures Konflux/release builds are unaffected. Those
builds use `strictfipsruntime` and need the C DNS resolver for FIPS
compliance. Local developer builds are also unaffected.

### References

- Go build constraints: https://pkg.go.dev/go/build#hdr-Build_Constraints
- `net` package build tags: `$(go env GOROOT)/src/net/cgo_stub.go` (`//go:build !cgo || netgo`)
- Verified in CI runs:
  - Cold (tag mismatch): [Run 21807146741](https://github.com/stackrox/stackrox/actions/runs/21807146741) — CLI linux/amd64: 80s, go-binaries: 257s
  - Warm (cache hit): [Run 21832411602](https://github.com/stackrox/stackrox/actions/runs/21832411602) — CLI linux/amd64: 6s, go-binaries: 206s
  - Warm (consecutive): [Run 21832632139](https://github.com/stackrox/stackrox/actions/runs/21832632139) — CLI linux/amd64: 8s, go-binaries: 210s
  - Warm (consecutive): [Run 21832674828](https://github.com/stackrox/stackrox/actions/runs/21832674828) — CLI linux/amd64: 8s, go-binaries: 205s

## Current CI Setup

Defined in `.github/actions/cache-go-dependencies/action.yaml`.

### Cache key strategy

**GOMODCACHE:**
```
key:          go-mod-v1-${{ hashFiles('**/go.sum') }}
restore-keys: go-mod-v1-
```

**GOCACHE:**
```
key:          go-build-v1-${{ group }}-${{ GOARCH }}-${{ hashFiles('**/go.sum') }}
restore-keys: go-build-v1-${{ group }}-${{ GOARCH }}-
```

Where `group` defaults to `github.job` but can be overridden via `cache-group`
input. `pre-build-go-binaries` is the shared group used across build, CLI,
test, and operator jobs.

### Save/restore policy

- **Master pushes**: save (auto-saves via `actions/cache@v5` post-step)
- **PR builds**: restore-only (`actions/cache/restore@v4`)
- **`force-save` input**: bypasses event/ref check (for experiments only)

### CLI per-arch matrix

The CLI build (`pre-build-cli`) is split into a per-arch matrix with dynamic
entries defined in `define-job-matrix`:

| Context | Targets built |
|---------|--------------|
| **PR (default)** | linux/amd64, linux/arm64 |
| **PR + `ci-build-all-arch`** | + linux/ppc64le, linux/s390x, darwin/amd64, darwin/arm64, windows/amd64 |
| **Master / tag push** | All 7 targets |

A `collect-cli-builds` fan-in job combines per-arch artifacts into the unified
`cli-build` artifact expected by downstream `build-and-push-main` jobs.

Cache sharing per target:
- **linux/***: shared `pre-build-go-binaries` group (cache hits from go-binaries)
- **darwin/***: per-job cache (different GOOS, no sharing possible)
- **windows/amd64**: no cache (6,318 new entries, zero speedup — not worth caching)

## Measured Results

### Cache sizes

| Cache entry | Compressed | On disk | Entries |
|------------|-----------|---------|---------|
| `pre-build-go-binaries` amd64 | **927 MB** (full) | 4.1 GB | ~29,000-31,600 |
| `pre-build-go-binaries` arm64 | **890 MB** (full) | 4.0 GB | ~31,200 |
| CLI per-job (darwin, 1 arch) | ~280-294 MB | — | ~18,340 |
| GOMODCACHE | ~1,635 MB | — | — |
| Scanner | ~379 MB | — | ~31,400 |

### Build times: warm vs cold

Data confirmed across 8 consecutive warm runs with 100% cache hit rate
(21845188087, 21845317019, 21845447755, 21845582203, 21845878961,
21846000723, 21846130810, 21846251148), plus earlier runs (21832411602,
21832632139, 21832674828, 21836022036, 21836188666):

| Job | Cold time | Warm time | Speedup | New entries | Cache group |
|-----|-----------|-----------|---------|-------------|-------------|
| **CLI linux/amd64** | 172s | **6-8s** | 22-29x | +10 | `pre-build-go-binaries` |
| **CLI linux/arm64** | 182s | **6s+6s** | 15x | +10 | `pre-build-go-binaries` |
| **CLI linux/ppc64le** | 179s | **6s+7s** | 14x | +10 | `pre-build-go-binaries` |
| **CLI linux/s390x** | 169s | **7s+7s** | 12x | +10 | `pre-build-go-binaries` |
| **CLI darwin/amd64** | 115s | **7s** | 16x | +5 | per-job |
| **CLI darwin/arm64** | 115s | **7s** | 16x | +5 | per-job |
| CLI windows/amd64 | 110s | 114s | ~1x | +6,318 | none (skipped) |
| **go-binaries amd64** | 353s | **21-25s** | 14-17x | +9,620 | `pre-build-go-binaries` |
| **go-binaries arm64** | 327s | **15s** | 22x | — | `pre-build-go-binaries` |
| pre-build-docs | — | — | free | +0 | `pre-build-go-binaries` |
| operator (all 4 variants) | — | — | free | +0 | `pre-build-go-binaries` |

go-binaries warm time with the full 916 MB cache averages **22.5s** (range
21-25s) across 8 consecutive runs. CLI averages **24.5s** (range 24-26s).

On master, go-binaries is the sole saver of the `pre-build-go-binaries` cache
(CLI and other jobs are restore-only), ensuring the full 32K-entry cache is
always what gets saved.

### Aggregate results from monitoring

Over 8 consecutive warm runs with full 916 MB cache:
- **100% cache hit rate** on every run
- **CLI linux/amd64**: avg 24.5s (17-18s roxctl + 6-8s roxagent)
- **go-binaries amd64**: avg 22.5s (range 21-25s)
- Zero failures, zero new cache entries needed for CLI (+10 only)

### Cache download overhead

Restoring 927 MB compressed → ~31,600 entries / 4.1 GB on disk takes **~15 seconds**.
Restoring 461 MB compressed → 22,857 entries / 2.1 GB on disk takes **7-9 seconds**.

### Critical path comparison

**Old (single-job CLI, master):**
```
pre-build-cli (all 7 arches, sequential): ~22 min
```

**New (per-arch matrix, PR with 2 arches):**
```
max(CLI linux/amd64: 3m40s, CLI linux/arm64: 3m, go-binaries: 2m*) + collect: 1m = ~4m40s
```
*With full warm cache

**Savings: ~17 minutes on the pre-build critical path.**

### Cache utilization analysis

Jobs with **+0 to +10 new entries** are perfect consumers of the shared cache.
Jobs with **+9,620 entries** (go-binaries) are the primary producer.
Jobs with **+6,318 entries and no speedup** (windows) should not use the cache.

## Cache budget

### Projected budget with shared cache (on master)

| Entry | Size | Shared by |
|-------|------|-----------|
| `pre-build-go-binaries` amd64 | ~915 MB | go-binaries, CLI, unit tests, operator, docs, golangci-lint |
| `pre-build-go-binaries` arm64 | ~890 MB | go-binaries arm64, CLI arm64 |
| `go-mod-v1` | ~1,635 MB | all Go jobs |
| Scanner | ~379 MB | scanner jobs |
| npm, setup-go, buildx, binfmt | ~380 MB | UI, operator, misc |
| **Total** | **~4,200 MB** | **42% of 10 GB** |

Leaves ~5.8 GB for optional caches (ppc64le, s390x, darwin per-job) and
caches from other active branches.

### vs. previous budget (before optimization)

| Entry | Old size | New size | Change |
|-------|----------|----------|--------|
| CLI monolithic (all 7 arches) | 2,561 MB | 0 MB | Eliminated |
| `pre-build-go-binaries` amd64 | 922 MB | 915 MB | Shared (was per-job) |
| `pre-build-go-binaries` arm64 | 897 MB | 890 MB | Shared (was per-job) |
| `main-build` amd64 (duplicate) | 914 MB | 0 MB | Merged into above |
| `main-build` arm64 (duplicate) | 890 MB | 0 MB | Merged into above |
| golangci-lint own cache | 2,200 MB | 0 MB | Uses shared cache |
| **Total Go build caches** | **~8,384 MB** | **~1,805 MB** | **-78%** |

## Improvement Ideas

### Scheduled cache save workflow

Instead of saving on every master push, use a scheduled workflow. Benefits:

- **Prevents cache expiration**: Master commit frequency analysis shows 3,092
  commits/year with longest gap of 6 days — close to the 7-day TTL. A
  scheduled save removes this risk.
- **Reduces churn**: One entry per period instead of one per commit.
- **Frees budget**: Fewer master entries means more room for PR caches.
- **Staleness is acceptable**: Go's content-based hashing means stale entries
  are simply unused, not incorrectly reused.

Suggested schedule: every 3-4 days (inside the 7-day TTL with margin).

```yaml
name: Cache refresh
on:
  schedule:
    - cron: '0 4 * * 1,4'  # Monday and Thursday at 4am UTC
  workflow_dispatch: {}

jobs:
  refresh-go-cache:
    runs-on: ubuntu-latest
    container:
      image: quay.io/stackrox-io/apollo-ci:stackrox-test-0.5.1
    strategy:
      matrix:
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v6
      - uses: ./.github/actions/cache-go-dependencies
        with:
          cache-group: pre-build-go-binaries
        env:
          GOARCH: ${{ matrix.arch }}
      - run: |
          if [[ "${{ matrix.arch }}" == "amd64" ]]; then
            CGO_ENABLED=1 make build-prep main-build-nodeps
          else
            CGO_ENABLED=0 GOARCH=${{ matrix.arch }} make build-prep main-build-nodeps
          fi
```

### Cache health monitoring

The GOCACHE probe script (`.github/scripts/gocache-probe.sh`) measures:

- **Pre-build**: entry count, disk size, hit rate via `go build -v`
- **Post-build**: entry count growth, restored entries as % of final

This identifies jobs where the shared cache is overkill (large restore for
few hits) and should use their own per-job cache or skip caching entirely.

### Cache key debugging

```bash
GODEBUG=gocachehash=1 go build ./pkg/errox/... 2>/tmp/cachehash.log
grep "HASH\[build" /tmp/cachehash.log | head -200
```

Shows exact inputs for each package's cache key, including compiler version,
GOOS/GOARCH, build tags, file content hashes, and import dependency hashes.

## Phase 2: Simplified approach (PR 18984)

Based on code review feedback from PR 18935, a simpler approach was adopted:

### Changes from PR 18935

- **No shared cache group** — each job keeps its own `github.job` cache key
- **No `netgo,osusergo` tags** — deferred to a separate PR
- **No CLI matrix split** — deferred to a separate PR
- **No new action inputs** — minimal change to the action
- **`restore-keys` prefix** — added for partial matches across commits
- **`make tag` for GOCACHE key** — replaces `go.sum` hash; changes less often
- **Save only on default branch** — `actions/cache@v5` on master,
  `actions/cache/restore@v5` on PRs

### Cache key strategy (PR 18984)

```
key:          go-build-v1-{github.job}-{GOARCH}-{make tag}
restore-keys: go-build-v1-{github.job}-{GOARCH}-
```

The `make tag` (e.g., `4.11.x-85-gfcc96f3da4`) changes per commit but the
`restore-keys` prefix matches any previous tag, providing partial cache hits.

### Measured results (PR 18984)

Partial restore-key hits confirmed on [run 21921857050](https://github.com/stackrox/stackrox/actions/runs/21921857050):
`Cache hit for restore-key: go-build-v1-pre-build-go-binaries-amd64-4.11.x-93-g9c6aeec2dd`

| Job | Cold | Partial hit |
|-----|------|-------------|
| pre-build-go-binaries amd64 | 581s | **255s** |
| pre-build-cli | 1,258s | **448s** |
| operator RHACS amd64 | 1,134s | **725s** |
| operator STACKROX amd64 | 933s | **558s** |

### Cache budget problem (PR 18984)

Every job saves its own GOCACHE entry. Total per master commit:

| Entry | Size | Critical path? |
|-------|------|----------------|
| pre-build-go-binaries amd64 | 917 MB | yes |
| pre-build-go-binaries arm64 | 890 MB | yes |
| pre-build-cli | 1,250 MB | yes |
| pre-build-docs | 30 MB | yes |
| operator amd64 | 1,248 MB | parallel |
| operator arm64 | 1,467 MB | parallel |
| scanner | 387 MB | no |
| db-integration-tests | 62 MB | no |
| go unit tests | 2,000 MB | no |
| go-postgres | 913 MB | no |
| go-bench | 610 MB | no |
| local-roxctl-tests | 1,163 MB | no |
| sensor-integration-tests | 626 MB | no |
| check-generated-files | 752 MB | no |
| style-check | 2,183 MB | no |
| golangci-lint | ~1,400 MB | no |
| setup-go (golangci-lint) | 3,354 MB | no |
| **GOCACHE+setup-go subtotal** | **~19,252 MB** | |
| go-mod-v1 | 1,629 MB | shared |
| npm + lint + misc | ~380 MB | |
| **Total** | **~21,261 MB** | |
| **Over 10 GB budget** | **~11,261 MB** | |

### Next steps

To fit within 10 GB, disable saves on non-critical-path jobs (~8.3 GB):
- go unit tests, go-postgres, go-bench (test-dominated, no speedup)
- local-roxctl-tests, sensor-integration-tests (test-dominated)
- check-generated-files, style-check (style-dominated)
- golangci-lint (analysis-dominated)

This leaves ~7.6 GB of critical-path + operator saves. Still over 10 GB
with GOMODCACHE (1.6 GB), so may also need to drop operator arm64 (1.5 GB)
or investigate if GOMODCACHE can be replaced by Go module mirror downloads.

Future PRs in the stack:
1. **CLI matrix split** — parallel per-arch builds, ~17 min critical path savings
2. **`netgo,osusergo` tags** — full CGO cache sharing, 100% hit rate
3. **Operator cache optimization** — shared cache, Docker layer caching

## Files changed

### Core changes
- `make/env.mk` — add `netgo,osusergo` to GOTAGS (CI only, guarded by `ifdef CI`)
- `.golangci.yml` — add `netgo,osusergo` to build-tags
- `.github/actions/cache-go-dependencies/action.yaml` — add `cache-group`,
  save/restore policy, timing notice

### Workflow changes
- `.github/workflows/build.yaml` — CLI per-arch matrix, shared cache group,
  `collect-cli-builds` fan-in, dynamic matrix via `define-job-matrix`,
  windows cache skip
- `.github/workflows/unit-tests.yaml` — shared `pre-build-go-binaries` group
  for go, go-postgres, go-bench, local-roxctl-tests, sensor-integration
- `.github/workflows/style.yaml` — golangci-lint uses shared cache, removed
  broken per-job lint cache
- `.github/workflows/scanner-build.yaml` — GOCACHE probes
- `.github/workflows/scanner-db-integration-tests.yaml` — shared cache group
- `.github/workflows/emailsender-central-compatibility.yaml` — use setup-go
  built-in cache
- `.github/workflows/retest_periodic.yml` — use setup-go built-in cache

### New files
- `.github/scripts/gocache-probe.sh` — cache effectiveness measurement
