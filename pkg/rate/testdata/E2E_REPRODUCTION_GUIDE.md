# Reproducing the rate-limiter comparison

Compares the completion-based limiter (this PR) with the time-based limiter (master) under VM
enrichment load. Review-only; remove before merge.

## Prerequisites

- A cluster with Central + Scanner V4 (e.g. `roxie deploy central --features +ROX_VIRTUAL_MACHINES`).
- Scale the real sensor to 0 but keep its certs: `kubectl -n stackrox scale deploy/sensor --replicas=0`
  (a SecuredCluster must have been registered so `tls-cert-sensor` exists).

## 1. Build the two Central binaries (same base commit)

```bash
BASE=$(git merge-base HEAD origin/master)
git worktree add /tmp/base $BASE && git worktree add /tmp/pr HEAD
for d in base pr; do
  (cd /tmp/$d && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
     -ldflags="-X github.com/stackrox/rox/pkg/version/internal.MainVersion=$(make --no-print-directory tag 2>/dev/null || echo 0.0.0-test)" \
     -o /tmp/central-$d ./central)
done
md5sum /tmp/central-base /tmp/central-pr   # must differ
```

## 2. Deploy each binary

Overlay the binary onto the running Central (read-only rootfs → subPath mount), with the operator
paused (`kubectl -n rhacs-operator-system scale deploy/rhacs-operator-controller-manager --replicas=0`)
so it is not reverted. Enable plaintext metrics for scraping:

```bash
kubectl -n stackrox set env deploy/central \
  ROX_ENABLE_SECURE_METRICS=false ROX_METRICS_PORT=:9090 \
  ROX_VM_INDEX_REPORT_RATE_LIMIT=0.3 ROX_VM_INDEX_REPORT_BUCKET_CAPACITY=30
```

Verify inside the pod: `md5sum /stackrox/central` matches the intended binary.

## 3. Generate load

```bash
cat > vm.yaml <<'EOF'
virtualMachineWorkload:
  poolSize: 3000          # unique VMs; >> concurrency avoids dedup collisions
  reportInterval: 1s
  numPackages: 508        # scale-test report size
  updateInterval: 10m
  lifecycleDuration: 120m
numNamespaces: 1
EOF

KUBECONFIG=<kubeconfig> ROX_VIRTUAL_MACHINES=true \
ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL=60s \
ROX_VIRTUAL_MACHINES_SCRAPER_TICK_INTERVAL=1s \
ROX_VIRTUAL_MACHINES_SCRAPER_CONCURRENCY=200 \
./local-sensor -connect-central <central-lb>:443 -namespace stackrox \
  -operator-install -with-fakeworkload vm.yaml -duration 20m -no-cpu-prof -no-mem-prof
```

Poll interval is clamped to 60 s, so offered rate ≈ `poolSize/60` ≈ 50/s — well above any admit rate.

## 4. Measure

Port-forward Central metrics and sample every 5 s:

```bash
kubectl -n stackrox port-forward deploy/central 9090:9090 &
watch -n5 'curl -s localhost:9090/metrics | grep -E \
  "rate_limiter_requests_total|rate_limiter_in_flight_tokens|sensor_event_queue.*VirtualMachineIndexReport"'
```

- **Throughput** = slope of `rate_limiter_requests_total{outcome="accepted"}` (steady state, drop
  first ~90 s). Do **not** use Scanner `GetVulnerabilities` counts — they are deduplicated per-VM and
  do not reflect the limiter.
- **Backlog** = `sensor_event_queue{Operation="Add",Type="VirtualMachineIndexReport"}` −
  `{Operation="Remove",...}`.

## 5. Scenarios

- **Upside**: give Scanner V4 headroom (raise scanner-v4-db CPU / matcher replicas). Completion
  admits at the Scanner ceiling; time-based stays at 0.3/s.
- **Downside**: starve Scanner V4 (scanner-v4-db 1 CPU) and set a low Central memory limit. With the
  time-based limiter the backlog and Central RSS grow unbounded → OOM; with completion they stay flat.
