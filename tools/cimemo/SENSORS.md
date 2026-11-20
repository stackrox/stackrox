# cimemo sensor architecture

Multi-sensor dependency discovery + fusion, with learned edges validated
on master/nightly runs (`--shadow`) and on demand (`cimemo audit`).

## Sensors

Active (exercised in tests):

| # | Sensor | Tier | Package | When |
|---|--------|------|---------|------|
| 1 | go-list closure (`go list -deps`, embed/cgo/test files expanded) | exact | `key.GoClosure` | pre-exec |
| 2 | lsof read observation | heuristic | `observe` | during |
| 3 | git-tracked fallback | sound/conservative | `cmd` | pre-exec |
| 4 | learned edges (changed→output history) | heuristic→validated | `model` | pre-exec |
| 5 | fsnotify change distribution | meta | `watch` | continuous |
| 6 | output content diffs (blob store) | ground truth | `store` | post-exec |

Planned (TODO, not exercised yet — see `sensor/sensor.go`):

| # | Sensor | Tier | Notes |
|---|--------|------|-------|
| 7 | compiler dep-files (gcc/clang `-MD`, rustc `dep-info`) | exact | TODO: c/c++ case study needed |
| 8 | dry-run/verbose traces (`go build -x`, `make -n`, `tsc --listFiles`) | exact | TODO |
| 9 | GOCACHEPROG pass-through proxy + miss log | exact | ACTIVE: `gocache` package, see below |
| 10 | coverage profiles (test selection) | exact | TODO |
| 11 | git history mining (seed model from commit co-changes) | statistical | TODO |
| 12 | BuildKit/Docker layer provenance | exact | TODO |

## Fusion rules

1. Union for coverage; exact beats heuristic. A heuristic never removes
   an edge an exact sensor found.
2. Negative evidence ("rebuilt to identical bytes") never becomes a
   learned absence — only downgrades urgency, never correctness.
3. Unknown files (no edges, not in any closure) → conservative: full rebuild.
4. Cheapest-sufficient: fsnotify/mtime delta → learned edges → closures.

## Validation modes

- `--shadow`: compute predictions, do NOT trust them, build fully,
  score predictions against ground truth, harvest evidence. Zero
  behavioral difference. Intended: always-on on master/nightly runs.
- `cimemo audit <cmd>`: force full rebuild now, score, repair model,
  mark edges validated, report divergences. For release certainty.

## Correctness rules (learned from v0.3 bug)

- Snapshots store OUTPUTS ONLY (files created/modified during the
  build that are not inputs). Restoring must never touch user files.
- Warm-path hit requires: read-set unchanged (mtime+hash), input
  directories unchanged (listing hash), toolchain unchanged.
- Learned edges serve hits only after surviving an audit/shadow
  validation (default off: `--trust-learned`).

## Test ladder

1. k8swatch (small): full scenario matrix + model learning + shadow + audit.
2. Prometheus (mid): same matrix, compare against v0.1 numbers.
3. StackRox (large): local main-binary builds, compare timing/cache
   against the custom CA solution (az-gocache / ca-build branches).

## Results (v0.4.0, 2025-11-20, macOS, local disk blob store)

| Scenario | k8swatch | Prometheus | StackRox (central, 4,590 inputs) |
|---|---|---|---|
| cold (full build + snapshot) | 138ms | 12.0s | 70.5s1 |
| warm hit (validate + restore) | ~0ms | 79ms | 203ms |
| README/non-input change -> hit | yes, edit survives restore | yes | yes |
| non-dep .go change -> hit | - | yes (promtool) | yes (roxctl) |
| dep .go change -> miss+rebuild | 147ms | ~11s | 32s |
| revert -> keyed hit (full restore) | 67ms | 422ms (136MB) | 1.6s (327MB) |
| audit: prediction + recall | 100% | 100% | 100% |
| model after audits | 1 edge, validated | 1 edge, validated | 1 edge, validated |

1 machine's go build cache partially warm; CI-cold would be minutes.

vs v0.1 (git-tracked keys): Prometheus warm 280ms -> 79ms; README and
non-dep changes now hit instead of missing (exact closure vs whole-tree).

vs custom CA solution (CI): structurally identical mechanisms
(content-addressed blobs, exact closures, per-input keys, warm
validation, provenance) achieved generically -- no Makefile recipes, no
per-project tooling. Absolute times are not comparable (local disk vs
CI runners + docker assembly), but the warm path does the same work:
validate inputs, restore/link outputs, skip the build.

## gocache (sensor #9) results — v0.4.0, 2025-11-20

Local pass-through proxy over the standard GOCACHE layout (cohabits
bare `go` builds in both directions — verified). go 1.27 accepts the
protocol (put responses MUST carry DiskPath+Size). Miss log feeds
cimemo manifests: exact work accounting per build (e.g. Prometheus
one-file change = 2 misses / 4 puts / 41MB; StackRox central = 8,838
actions tracked). BuildConfig (CGO_ENABLED/GOOS/GOARCH/GOFLAGS) folded
into the key — closes the env-change false-hit vector.

Remote tier (Azure, zero-dep net/http port of gocache-serve):
concurrent GET dispatch, concurrent upload flush, integrity check on
cloud GETs, module-identity manifests (go.mod + build config) with
startup prefetch. Correctness proven end-to-end: byte-identical
binaries served to a fresh local cache from the cloud, at k8swatch
(903 actions) and Prometheus (3,654-action closure) scale.

Honest performance finding: from this machine (220ms RTT to
westus3), the remote tier LOSES to local recompile (k8swatch: 19.7s
prefetch vs 5.3s recompile; Prometheus: 162s vs 40s). Root cause is
not bandwidth or the backend (86+ req/s available from parallel curl)
but request-rate: go's GOCACHEPROG client asks at ~GOMAXPROCS rate,
and cross-region RTT bounds bulk prefetch. The tier's value is
in-region (CI runners, 2-5ms RTT): the CA branch measured the same
architecture at 4.3m wall-clock vs hours of cold builds there.
Remote mode is opt-in (CIMEMO_GOCACHE_REMOTE=1); local pass-through
is the default and always on for Go commands.

Bugs found and fixed during the port: response-write race between
the main loop and dispatch goroutines (interleaved JSON killed a
Prometheus build — caught live, all writes now mutex-serialized,
-race clean); sequential upload flush (476s for one Prometheus
populate — now concurrent); empty-body put markers polluting the
manifest (330 of 1,233 actions uncacheable by nature — filtered);
container delete/create race in the test harness (all PUTs 404'd —
harness bug, not proxy).

Known wrinkle: the manifest merge is a union, so historically
recorded empty actions persist (one wasted tiny GET each at
prefetch); repopulate with a fresh m/ blob to clean.

Known bug fixed during this pass: filepath.SkipDir on a dot-FILE
(.DS_Store) in the repo root silently ended the tree walk -- outputs
were never captured. Dot-files are now skipped individually; only
dot-directories prune the walk (same fix applied to SnapshotDir and
key.Compute).
