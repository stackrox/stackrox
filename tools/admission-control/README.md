# Admission Controller Tools

Scripts for benchmarking and profiling the admission controller, Sensor, and
Central around image cache and reprocessor ("reassess all") workflows.

## Prerequisites

- Kubernetes/OpenShift cluster with StackRox deployed
- `admissionControl.enforcement` enabled in the SecuredCluster CR
- `spec.monitoring.exposeEndpoint: Enabled` on **both** the SecuredCluster and Central CRs
  (AC metrics for all modes; Central metrics required for `prescan-slow-path`)

## Common Environment Variables

Shared by `bench-reassess.sh` and `profile-reassess.sh`:

| Variable | Default | Description |
|----------|---------|-------------|
| `ROX_PASSWORD` | *(required)* | Central admin password |
| `ROX_CENTRAL_ADDRESS` | *(required)* | Central host:port |
| `ROX_ADMIN_USER` | `admin` | Central admin username |
| `BURST_SIZE` | varies | Number of deployments per burst |
| `UNIQUE_PCT` | varies | % of BURST_SIZE used as distinct images |
| `IMAGES_FILE` | *(unset)* | File with one image per line (overrides generation) |
| `PARALLEL` | `50` | Max concurrent `kubectl create` calls |
| `NAMESPACE` | varies | Namespace for test deployments |
| `ROX_NAMESPACE` | `stackrox` | StackRox namespace |
| `REASSESS_WAIT_TIMEOUT` | `300` | Max seconds to wait for reprocessor |

## Image Pool

All slow-path scripts derive the unique image count from `BURST_SIZE` and
`UNIQUE_PCT`:

```
UNIQUE_COUNT = BURST_SIZE * UNIQUE_PCT / 100
```

`generate-image-pool.sh` is called automatically to fetch exactly
`UNIQUE_COUNT` images from `quay.io` (counts <= 20 use a hardcoded fallback,
counts > 20 query the quay.io tag API; results cached in `/tmp` for 1 hour).
Set `IMAGES_FILE` to skip generation and read from a file instead.

## Cross-Branch Comparison

All scripts follow the same pattern:

1. Deploy with **master** image, run the script, save output
2. Swap to **PR branch** image (policies persist in Central)
3. Run again, save output
4. `diff` the two outputs (or use `go tool pprof -diff_base` for profiles)

### Example: prescan-slow-path comparison

This workflow proves whether Central skips Scanner `GetScan` for tag-only
admission after a Central pre-scan (`ScanImageInternalForAdmission`).

**Policy (required):** enable **90-Day Image Age**, Deploy lifecycle, **Fail**
on admission. Spec-only policies (Privileged Container, Latest tag) will not
fetch images and the run is invalid.

**Why these flags:** `UNIQUE_PCT=100` and `PARALLEL=1` make every review a
Central fetch of a distinct pre-scanned tag. Cache hits and coalescing no
longer move the averages. Failed prescans are dropped from the burst so the
PR cannot pick up extra Scanner calls for uncached tags.

Scrape happens **after** prescan so `scan_duration_count` from
`POST /v1/images/scan` is not counted in the burst delta.

```bash
# 1. Deploy master image. Enforce 90-Day Image Age (Fail on admission).
ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=<host:port> \
  BURST_SIZE=20 UNIQUE_PCT=100 PARALLEL=1 \
  ./tools/admission-control/burst-test.sh prescan-slow-path \
  | tee /tmp/prescan-master.txt

# 2. Redeploy Central + Sensor with PR branch image
#    (AC pods will be restarted by the script itself).
#    Re-export ROX_PASSWORD if the redeploy minted a new one.

# 3. Run again on PR branch
ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=<host:port> \
  BURST_SIZE=20 UNIQUE_PCT=100 PARALLEL=1 \
  ./tools/admission-control/burst-test.sh prescan-slow-path \
  | tee /tmp/prescan-pr.txt

# 4. Compare
diff -y /tmp/prescan-master.txt /tmp/prescan-pr.txt
```

Pass bar (both runs must have `image_fetch_total` ≈ unique images):

| Metric | Master | PR branch |
|--------|--------|-----------|
| `scan_duration_count` (Central) | ≈ unique images | **~0** |
| `image_fetch_duration (avg)` | metadata + GetScan | metadata only (lower) |
| `image_fetch_total` | ≈ unique | ≈ unique |
| `review_duration_seconds (avg)` | ignore | ignore (diluted by hits) |

`scan_duration_count` is the Scanner-trip proof. Fetch duration is supporting
latency. Do not treat `review_duration_seconds` as a Scanner proxy: after
prescan, Scanner's index is warm so GetScan can be cheap, and review avg mixes
in cache hits.

---

### `burst-test.sh`

Burst of deployment creates against a live cluster with AC metric deltas.
Run with `-h` for all options.

| Mode | What it tests | Required policies |
|------|---------------|-------------------|
| `fast-path` | Spec-only evaluation (no image fetching) | Privileged Container, Latest tag |
| `slow-path` | Image coalescing + caching (enrichment) | Any enrichment-required (e.g. Image Age) |
| `prescan-slow-path` | Scan reuse for pre-scanned tag-only images | Any enrichment-required (e.g. Image Age) |

Slow-path runs two phases automatically: **cold cache** (pods restarted) then
**warm cache** (immediate re-burst).

```bash
# Fast-path
VIOLATION_PCT=50 ./burst-test.sh fast-path

# Slow-path (125 unique images = 25% of 500)
BURST_SIZE=500 UNIQUE_PCT=25 ./burst-test.sh slow-path

# Slow-path with custom images
IMAGES_FILE=/tmp/my-images.txt BURST_SIZE=500 UNIQUE_PCT=50 ./burst-test.sh slow-path

# Create persistent deployments (replicas=0) for reprocessor profiling
BURST_SIZE=1000 UNIQUE_PCT=60 ./burst-test.sh slow-path --no-dry-run
```

**Prescan-slow-path** pre-scans unique images via Central's `POST /v1/images/scan`
API (warming the DB), drops failures, restarts AC (not Scanner), then bursts
**one tag-only deployment per successful prescan**. Central `scan_duration_count`
during the burst is the Scanner-trip proof; AC `image_fetch_duration_seconds`
is the latency proof.

```bash
# Prescan + one-fetch-per-unique-image burst
ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=<host:port> \
  BURST_SIZE=20 UNIQUE_PCT=100 PARALLEL=1 PRESCAN_PARALLEL=5 \
  ./tools/admission-control/burst-test.sh prescan-slow-path
```

Requires **90-Day Image Age** (or another enrichment-required policy) enabled
with **Fail on admission**. `UNIQUE_PCT` is forced to 100.

Additional environment variables for `prescan-slow-path`:

| Variable | Default | Description |
|----------|---------|-------------|
| `ROX_CENTRAL_ADDRESS` | *(required)* | Central host:port |
| `ROX_PASSWORD` | *(required)* | Central admin password |
| `ROX_ADMIN_USER` | `admin` | Central admin username |
| `PRESCAN_PARALLEL` | `5` | Max concurrent prescan API calls |

---

### `bench-reassess.sh`

End-to-end benchmark: **burst → reassess → burst** cycle with Prometheus
metric deltas across AC, Sensor, and Central.

| Phase | What happens |
|-------|-------------|
| 1. Burst 1 | `--dry-run=server` deployments warm caches |
| 2. Snapshot | Scrape pre-reassess metrics |
| 3. Reassess | `POST /v1/policies/reassess` |
| 4. Snapshot | Scrape post-reassess metrics |
| 5. Burst 2 | Same manifests — measures cache survival |
| 6. Report | Print deltas and key comparisons |

```bash
ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=<host:port> ./bench-reassess.sh

# With more unique images or a custom list
BURST_SIZE=200 UNIQUE_PCT=50 ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=<addr> ./bench-reassess.sh
IMAGES_FILE=/tmp/my-images.txt BURST_SIZE=200 ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=<addr> ./bench-reassess.sh
```

Defaults: `BURST_SIZE=100`, `UNIQUE_PCT=25`, `METRICS_PORT=9090`,
`LOCAL_PORT=9090`.

**Metrics collected:**

| Component | Metric | What it tells you |
|-----------|--------|-------------------|
| Central | `reprocessor_duration_seconds` | Wall time of the reprocessor cycle |
| Central | `msg_to_sensor_not_sent_count` | Messages skipped due to errors |
| Sensor | `detector_deployment_processed` | Deployment re-detections triggered |
| Sensor | `detector_dedupe_cache_hits` | Deployments deduped (no re-detection) |
| Sensor | `sensor_events` | Total events sent to Central |
| Sensor | `component_process_message_duration_seconds` | Time processing Central messages |
| AC | `image_cache_operations_total{hit,miss,skip}` | Cache effectiveness |
| AC | `image_fetch_total` | Cold fetches from cache misses |
| AC | `policyeval_review_duration_seconds` | Per-review latency |

**Key comparisons** (master vs PR):

1. `reprocessor_duration_seconds` — lower = less serialization overhead
2. `deployment_processed` during reassess — lower = fewer redundant re-detections
3. AC cache hit rate on burst 2 — higher = cache survived reassess
4. AC `image_fetch_total` (burst 2) — lower = cache was warm
5. AC `image_fetch_total` (reassess) — lower = no unnecessary re-fetches

---

### `profile-reassess.sh`

Captures Go pprof CPU and heap profiles from Central and Sensor during reassess.
Creates real deployments (`replicas=0`) first, then profiles.

| Step | What happens |
|------|-------------|
| 1 | Verify pprof endpoints (fails fast if unreachable) |
| 2 | Create `BURST_SIZE` deployments (`replicas=0`), wait 30s |
| 3 | Capture pre-reassess heap snapshots |
| 4 | Start CPU profiling + trigger reassess |
| 5 | Capture post-reassess heap and goroutine snapshots |

```bash
ROX_PASSWORD=<pw> ROX_CENTRAL_ADDRESS=localhost:8000 \
  BURST_SIZE=500 UNIQUE_PCT=60 CPU_PROFILE_SECONDS=60 \
  ./profile-reassess.sh
```

Defaults: `BURST_SIZE=500`, `UNIQUE_PCT=60`, `CPU_PROFILE_SECONDS=60`,
`SENSOR_LOCAL_PORT=6060`, `OUTPUT_DIR=/tmp/profiles/<branch>-<timestamp>`.

**Output files:**

| File | When |
|------|------|
| `{central,sensor}-heap-pre.pb.gz` | Before reassess |
| `{central,sensor}-cpu.pb.gz` | During reassess |
| `{central,sensor}-heap-post.pb.gz` | After reassess |
| `{central,sensor}-goroutine.pb.gz` | After reassess |

**Comparison:**

```bash
# CPU diff between branches
go tool pprof -diff_base=./profiles/master-*/central-cpu.pb.gz \
                          ./profiles/<pr-branch>-*/central-cpu.pb.gz

# Heap growth within a single run
go tool pprof -diff_base=./profiles/<branch>/central-heap-pre.pb.gz \
                          ./profiles/<branch>/central-heap-post.pb.gz

# Interactive flamegraph
go tool pprof -http=:8080 ./profiles/<branch>/central-cpu.pb.gz
```
