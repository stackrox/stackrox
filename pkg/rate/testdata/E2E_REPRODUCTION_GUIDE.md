# E2E Reproduction Guide: Rate Limiter Comparison

Step-by-step instructions for comparing the time-based vs completion-based
VM index report rate limiter on an OCP cluster with Scanner V4.

## Prerequisites

- An OCP cluster with ACS installed (Central + Scanner V4)
- `kubectl` configured to access the cluster
- `crane` CLI installed (`go install github.com/google/go-containerregistry/cmd/crane@latest`)
- Go toolchain (for building `local-sensor`)
- Central API credentials (default: `admin` / password from the `central-htpasswd` secret)
- The StackRox repo checked out with the completion-based branch

## Step 1: Build Central Images

Build two Central images: one with the time-based limiter (baseline from master)
and one with the completion-based limiter (this branch).

```bash
cd <stackrox-repo>

# Get the current Central image tag from the cluster
CURRENT_IMAGE=$(kubectl get deployment -n stackrox central \
  -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "Current image: $CURRENT_IMAGE"

# Build the baseline (time-based) central binary from master
git stash  # save any changes
git checkout origin/master -- pkg/rate/limiter.go pkg/rate/metrics.go
go build -o /tmp/central-baseline ./cmd/central/
git checkout HEAD -- pkg/rate/limiter.go pkg/rate/metrics.go
git stash pop

# Build the completion-based central binary from this branch
go build -o /tmp/central-completion ./cmd/central/

# Create a layer for each binary and push to ttl.sh
BASELINE_TAG="ttl.sh/stackrox-central-baseline-$(date +%s):24h"
COMPLETION_TAG="ttl.sh/stackrox-central-completion-$(date +%s):24h"

# Baseline image
crane mutate "$CURRENT_IMAGE" \
  --append <(cd /tmp && tar cf - central-baseline) \
  --entrypoint /central-baseline \
  --tag "$BASELINE_TAG"
crane push "$BASELINE_TAG"

# Completion image
crane mutate "$CURRENT_IMAGE" \
  --append <(cd /tmp && tar cf - central-completion) \
  --entrypoint /central-completion \
  --tag "$COMPLETION_TAG"
crane push "$COMPLETION_TAG"

echo "Baseline: $BASELINE_TAG"
echo "Completion: $COMPLETION_TAG"
```

**Alternative (simpler):** If you only want to test the completion-based limiter
against the stock Central, just build and push one image:

```bash
# Build the modified binary
GOARCH=amd64 GOOS=linux go build -o /tmp/central-db ./cmd/central/

# Append it to the existing image
IMAGE_TAG="ttl.sh/stackrox-central-test-$(date +%s):24h"
crane mutate "$CURRENT_IMAGE" \
  --append <(cd /tmp && tar czf - central-db) \
  --tag "$IMAGE_TAG"

echo "Test image: $IMAGE_TAG"
```

## Step 2: Configure Cluster

```bash
# Scale down the real sensor to avoid connection conflicts
kubectl scale deployment sensor -n stackrox --replicas=0

# Set environment variables on Central
kubectl set env deployment/central -n stackrox \
  ROX_VIRTUAL_MACHINES=true \
  ROX_VM_TEST_MODE=true \
  ROX_VM_INDEX_REPORT_BUCKET_CAPACITY=30

# Deploy the completion-based image first
kubectl set image deployment/central -n stackrox \
  central=$COMPLETION_TAG
kubectl rollout status deployment/central -n stackrox --timeout=180s
```

## Step 3: Create Workload Configuration

```bash
cat > /tmp/vm-high-load-test.yaml << 'EOF'
nodeWorkload:
  numNodes: 4
numNamespaces: 1
virtualMachineWorkload:
  poolSize: 100
  updateInterval: 5m
  lifecycleDuration: 30m
  numLifecycles: 0
  reportInterval: 10s
  numPackages: 500
  initialReportDelay: 2s
EOF
```

This creates 100 fake VMs, each sending 500-package index reports every 10 seconds
= 10 reports/second incoming rate.

## Step 4: Build local-sensor

```bash
cd <stackrox-repo>
go build -o ./tools/local-sensor/local-sensor ./tools/local-sensor/
```

## Step 5: Run Completion-Based Test

```bash
# Get Central endpoint
CENTRAL_EP=$(kubectl get route -n stackrox central \
  -o jsonpath='{.spec.host}'):443

# Start local-sensor
ROX_LOCAL_SENSOR=true \
ROX_VIRTUAL_MACHINES=true \
ROX_VM_TEST_MODE=true \
LOGLEVEL=info \
  ./tools/local-sensor/local-sensor \
  -connect-central "$CENTRAL_EP" \
  -with-fakeworkload /tmp/vm-high-load-test.yaml &
LS_PID=$!

# Wait for connection and initial reports to flow
sleep 45
```

### Monitor Memory (run in a separate terminal)

```bash
echo "=== COMPLETION-BASED RATE LIMITER ==="
for i in $(seq 1 12); do
  kubectl top pod -n stackrox -l app=central --no-headers
  sleep 30
done
```

### Measure Throughput

```bash
# Get Central pod name
POD=$(kubectl get pods -n stackrox -l app=central \
  -o jsonpath='{.items[0].metadata.name}')

# Count enriched reports over 60 seconds
START=$(kubectl logs -n stackrox $POD | grep -c "Successfully enriched")
sleep 60
END=$(kubectl logs -n stackrox $POD | grep -c "Successfully enriched")
echo "Reports in 60s: $((END - START))"
```

### Check Vulnerability Matches

```bash
CENTRAL_URL=$(kubectl get route -n stackrox central \
  -o jsonpath='{.spec.host}')
curl -sk -u admin:admin \
  "https://${CENTRAL_URL}/v2/virtualmachines?pagination.limit=3" | \
  python3 -c "
import json, sys
data = json.load(sys.stdin)
for vm in data.get('virtualMachines', [])[:3]:
    scan = vm.get('scan', {})
    comps = scan.get('components', [])
    vulns = sum(len(c.get('vulns', [])) for c in comps)
    print(f'{vm[\"name\"]}: {len(comps)} components, {vulns} vulns')
"
```

### Record Scanner V4 Usage

```bash
kubectl top pods -n stackrox -l app=scanner-v4-matcher
kubectl top pods -n stackrox -l app=scanner-v4-db
```

### Stop Load Generator

```bash
kill $LS_PID
wait $LS_PID
```

## Step 6: Run Baseline (Time-Based) Test

```bash
# Switch to baseline image
kubectl set image deployment/central -n stackrox \
  central=$BASELINE_TAG
kubectl rollout status deployment/central -n stackrox --timeout=180s

# Repeat Step 5 with the same workload configuration
```

## Step 7: Compare Results

Expected results (from our test on ga-acp, 2026-05-15):

| Metric | Time-Based | Completion-Based |
|--------|-----------|-----------------|
| Reports enriched/min | 19 | 182 |
| Throughput (rps) | 0.3 | 3.0 |
| Central memory (avg) | 252 Mi | 522 Mi |
| Central memory (peak) | 263 Mi | 544 Mi |
| Matcher CPU (per pod) | 65-143m | 714-824m |
| DB CPU | 213m | 3,159m |
| Reports dropped/10s | ~97 | ~0 |

## Step 8: Cleanup

```bash
# Restore original Central image
kubectl set image deployment/central -n stackrox \
  central=$CURRENT_IMAGE

# Remove custom env vars
kubectl set env deployment/central -n stackrox \
  ROX_VM_INDEX_REPORT_BUCKET_CAPACITY-

# Scale sensor back up
kubectl scale deployment sensor -n stackrox --replicas=1

# Wait for rollout
kubectl rollout status deployment/central -n stackrox --timeout=180s
kubectl rollout status deployment/sensor -n stackrox --timeout=180s
```

## Troubleshooting

### local-sensor can't connect to Central

Check the endpoint is correct and the route is accessible:
```bash
curl -sk "https://$(kubectl get route -n stackrox central \
  -o jsonpath='{.spec.host}')/v1/ping"
```

### Port 8443/9443 conflicts

If local-sensor crashes with port binding errors, set `ROX_LOCAL_SENSOR=true`
(already included in the commands above) which makes the gRPC server endpoints
optional.

### No reports flowing

Check local-sensor logs for "Established connection to Central" and
"VM index reports enabled". If missing, verify `ROX_VIRTUAL_MACHINES=true`
and `ROX_VM_TEST_MODE=true` are set on both local-sensor env and Central
deployment.

### Rate limiter not constraining

If you see no rate-limit warnings in Central logs, the bucket capacity may be
larger than the number of concurrent reports. Lower
`ROX_VM_INDEX_REPORT_BUCKET_CAPACITY` (e.g., to 30) or increase `poolSize` in
the workload YAML.
