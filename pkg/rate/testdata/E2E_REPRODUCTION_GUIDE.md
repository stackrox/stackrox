# E2E Reproduction Guide: Rate Limiter Comparison

Step-by-step instructions for comparing the time-based vs completion-based
VM index report rate limiter on an OCP cluster with Scanner V4.

## Prerequisites

- An OCP cluster with ACS installed via operator (Central + Scanner V4)
- `kubectl` / `oc` configured to access the cluster
- `crane` CLI installed (`go install github.com/google/go-containerregistry/cmd/crane@latest`)
- Go toolchain (for building `local-sensor` and Central binaries)
- Central API credentials (from `roxie env` or the `central-htpasswd` secret)
- The StackRox repo checked out on this branch

## Step 1: Deploy ACS

Use [Roxie](https://github.com/stackrox/roxie) to deploy ACS with operator:

```bash
roxie deploy both \
  --tag <latest-master-tag> \
  --envrc /tmp/roxie-env.sh \
  --exposure loadbalancer \
  --resources auto
```

Find the latest master-based tag:

```bash
TAGS=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/main/tag/?limit=100&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)')
echo "$TAGS" | head -5
```

After deployment, read credentials:

```bash
source /tmp/roxie-env.sh
curl -sk -u "admin:${ROX_ADMIN_PASSWORD}" "https://${ROX_ENDPOINT}/v1/metadata"
```

## Step 2: Configure Cluster

Determine ACS namespace layout (Roxie defaults: `acs-central` + `acs-sensor`):

```bash
CENTRAL_NS=acs-central
SENSOR_NS=acs-sensor
```

Scale down real sensor and configure Central:

```bash
kubectl scale deployment sensor -n $SENSOR_NS --replicas=0

kubectl set env deployment/central -n $CENTRAL_NS \
  ROX_VIRTUAL_MACHINES=true \
  ROX_VM_INDEX_REPORT_BUCKET_CAPACITY=200
```

## Step 3: Build Central Images

Build two Central images: completion-based (this branch) and baseline (master).

```bash
# Detect cluster architecture
ARCH=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}')
# Falls back to amd64
ARCH=${ARCH:-amd64}

# Get current Central image
CURRENT_IMAGE=$(kubectl get deployment -n $CENTRAL_NS central \
  -o jsonpath='{.spec.template.spec.containers[0].image}')

# --- Completion-based binary (this branch) ---
GOOS=linux GOARCH=$ARCH CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o /tmp/central-completion ./central

# --- Baseline binary (master's rate limiter) ---
git stash
git checkout origin/master -- pkg/rate/limiter.go pkg/rate/metrics.go
GOOS=linux GOARCH=$ARCH CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o /tmp/central-baseline ./central
git checkout HEAD -- pkg/rate/limiter.go pkg/rate/metrics.go
git stash pop

# --- Create images with crane ---
cd /tmp
mkdir -p stackrox && cp central-completion stackrox/central && chmod +x stackrox/central
tar cf completion.tar stackrox/
COMPLETION_TAG="ttl.sh/rox-central-completion-$(date +%s):24h"
crane mutate "$CURRENT_IMAGE" --platform linux/$ARCH --set-platform linux/$ARCH \
  --append completion.tar --tag "$COMPLETION_TAG"

cp central-baseline stackrox/central && chmod +x stackrox/central
tar cf baseline.tar stackrox/
BASELINE_TAG="ttl.sh/rox-central-baseline-$(date +%s):24h"
crane mutate "$CURRENT_IMAGE" --platform linux/$ARCH --set-platform linux/$ARCH \
  --append baseline.tar --tag "$BASELINE_TAG"

rm -rf stackrox
echo "Completion: $COMPLETION_TAG"
echo "Baseline:   $BASELINE_TAG"
```

## Step 4: Build local-sensor

```bash
cd <stackrox-repo>
go build -o ./tools/local-sensor/local-sensor ./tools/local-sensor/
```

> **Note:** This branch includes a patch to `tools/local-sensor/main.go` that
> creates a real K8s client for cert fetching when using `-with-fakeworkload`
> combined with `-connect-central`. Without this patch, the fake workload
> manager's K8s client cannot access real cluster secrets (`tls-cert-sensor`).

## Step 5: Create Workload Configuration

```bash
cat > /tmp/vm-stress-test.yaml << 'EOF'
nodeWorkload:
  numNodes: 4
numNamespaces: 1
virtualMachineWorkload:
  poolSize: 400          # 400 simulated VMs
  updateInterval: 5m
  lifecycleDuration: 30m
  numLifecycles: 0
  reportInterval: 1s     # each VM reports every second → 400 rps
  numPackages: 500       # 500 real RHEL 9 packages per report
  initialReportDelay: 2s
EOF
```

This generates 400 reports/sec, well above the bucket capacity of 200. Both
rate limiter variants will drop reports at this load, confirming they are
actively engaging.

To see meaningful throughput differences between the variants, use a lower
load (e.g., `poolSize: 50`, `reportInterval: 1s`) where the completion-based
variant can keep pace while the time-based variant cannot.

## Step 6: Run Tests

For each variant (completion first, then baseline):

```bash
# Deploy the variant's image
kubectl set image deployment/central -n $CENTRAL_NS \
  central=$COMPLETION_TAG   # or $BASELINE_TAG
kubectl rollout status deployment/central -n $CENTRAL_NS --timeout=300s

# Wait for Central health
until curl -sk -u "admin:$ROX_ADMIN_PASSWORD" \
  "https://$ROX_ENDPOINT/v1/metadata" | grep -q version; do sleep 5; done

# Start local-sensor
ROX_LOCAL_SENSOR=true ROX_VIRTUAL_MACHINES=true LOGLEVEL=info \
  ./tools/local-sensor/local-sensor \
  -connect-central "$ROX_ENDPOINT" \
  -namespace $SENSOR_NS \
  -operator-install \
  -with-fakeworkload /tmp/vm-stress-test.yaml &
LS_PID=$!
```

### Monitor (15+ minutes per run)

In a separate terminal:

```bash
# Memory samples every 30s
for i in $(seq 1 30); do
  kubectl top pod -n $CENTRAL_NS -l app=central --no-headers
  sleep 30
done
```

### Collect Metrics

```bash
# Rate-limited reports (from Central logs)
CENTRAL_POD=$(kubectl get pods -n $CENTRAL_NS -l app=central \
  -o jsonpath='{.items[0].metadata.name}')
kubectl logs -n $CENTRAL_NS $CENTRAL_POD | \
  grep "log suppressed" | awk '{print $(NF-1)}' | \
  awk '{sum += $1} END {print "Dropped:", sum}'

# Throughput (from Scanner V4 Matcher logs)
MATCHER_POD=$(kubectl get pods -n $CENTRAL_NS -l app=scanner-v4-matcher \
  -o jsonpath='{.items[0].metadata.name}')
kubectl logs -n $CENTRAL_NS $MATCHER_POD | \
  grep -c "GetVulnerabilities"
```

### Stop and Switch

```bash
kill $LS_PID; wait $LS_PID 2>/dev/null
# If port 8443 is still held:
fuser -k 8443/tcp 2>/dev/null; sleep 5

# Deploy baseline image and repeat
kubectl set image deployment/central -n $CENTRAL_NS central=$BASELINE_TAG
```

## Step 7: Compare Results

Expected results at 400 rps (from our 6-run test on ga-ocp4-cron-2, 2026-06-23):

| Metric | Completion-Based (avg) | Time-Based Baseline (avg) |
|--------|----------------------|--------------------------|
| Reports dropped / 16 min | 10,736 | 11,573 |
| Scanner V4 vuln lookups / 16 min | 354 | 340 |
| Throughput (lookups/min) | 21.8 | 21.0 |
| Central memory peak | 604 Mi | 783 Mi |

At extreme saturation (400 rps >> 200 bucket capacity), both variants perform
similarly. The completion-based variant shows a modest +4% throughput edge and
lower peak memory.

For clearer throughput differentiation, use lower load (e.g., 50 rps) where the
completion-based variant can recycle tokens faster than the time-based refill.

## Step 8: Cleanup

```bash
# Restore original Central image
kubectl set image deployment/central -n $CENTRAL_NS central=$CURRENT_IMAGE
kubectl rollout status deployment/central -n $CENTRAL_NS --timeout=180s

# Remove custom env vars
kubectl set env deployment/central -n $CENTRAL_NS \
  ROX_VM_INDEX_REPORT_BUCKET_CAPACITY-

# Scale sensor back up
kubectl scale deployment sensor -n $SENSOR_NS --replicas=1
kubectl rollout status deployment/sensor -n $SENSOR_NS --timeout=180s
```

## Troubleshooting

### local-sensor can't connect to Central

Verify the endpoint is reachable:
```bash
curl -sk "https://$ROX_ENDPOINT/v1/ping"
```

For operator-deployed clusters, the `-operator-install` flag adjusts TLS
expectations. The `-namespace` flag must match the sensor namespace.

### Port 8443 conflicts

If local-sensor crashes with port binding errors, kill the old process:
```bash
fuser -k 8443/tcp
```
Setting `ROX_LOCAL_SENSOR=true` makes the local gRPC server endpoints optional.

### No reports flowing

Check local-sensor logs for "Established connection to Central". If VMs are
not registering, ensure the workload YAML uses `virtualMachineWorkload` (not
`vmWorkload`). Central must have `ROX_VIRTUAL_MACHINES=true`.

### Rate limiter not constraining

If Central logs show no rate-limit warnings, the incoming rate may be below the
bucket capacity. Increase `poolSize` or decrease `reportInterval` in the
workload YAML, or lower `ROX_VM_INDEX_REPORT_BUCKET_CAPACITY`.
