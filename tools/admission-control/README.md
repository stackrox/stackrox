# Admission Controller Tools

Scripts for benchmarking and profiling the admission controller, Sensor, and
Central around image cache and reprocessor ("reassess all") workflows.

## Prerequisites

- Kubernetes/OpenShift cluster with StackRox deployed
- `admissionControl.enforcement` enabled in the SecuredCluster CR
- `spec.monitoring.exposeEndpoint: Enabled` in the SecuredCluster CR

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

---

### `burst-test.sh`

Burst of deployment creates against a live cluster with AC metric deltas.
Run with `-h` for all options.

| Mode | What it tests | Required policies |
|------|---------------|-------------------|
| `fast-path` | Spec-only evaluation (no image fetching) | Privileged Container, Latest tag |
| `slow-path` | Image coalescing + caching (enrichment) | Any enrichment-required (e.g. Image Age) |

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
