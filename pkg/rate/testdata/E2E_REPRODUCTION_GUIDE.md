# E2E Reproduction Guide: Rate Limiter Comparison

Step-by-step instructions for comparing the time-based vs completion-based
VM index report rate limiter on an OCP cluster with Scanner V4.

## Prerequisites

- An OCP cluster with ACS installed (Central + Scanner V4)
- `kubectl` configured to access the cluster
- `crane` CLI installed (`go install github.com/google/go-containerregistry/cmd/crane@latest`)
- Go toolchain (for building `local-sensor`)
- Central API credentials (default: `admin` / `admin`; if this doesn't work, ask the cluster operator for the admin password)
- The StackRox repo checked out with the completion-based branch

## Step 1: Build Central Images

Build two Central images: one with the time-based limiter (baseline from master)
and one with the completion-based limiter (this branch).

**Important:** The deployment runs `/stackrox/central` via the entrypoint
scripts (`central-entrypoint.sh` → `start-central.sh` → `exec /stackrox/central`).
The binary must be placed at exactly this path inside the image — the image
entrypoint is not used.

```bash
cd <stackrox-repo>

# Get the current Central image tag from the cluster
CURRENT_IMAGE=$(kubectl get deployment -n stackrox central \
  -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "Current image: $CURRENT_IMAGE"

# Build the baseline (time-based) central binary from master
# Cross-compile for linux/amd64 if building from macOS or ARM
git stash  # save any changes
git checkout origin/master -- pkg/rate/limiter.go pkg/rate/metrics.go
GOOS=linux GOARCH=amd64 go build -o /tmp/central-baseline ./cmd/central/
git checkout HEAD -- pkg/rate/limiter.go pkg/rate/metrics.go
git stash pop

# Build the completion-based central binary from this branch
GOOS=linux GOARCH=amd64 go build -o /tmp/central-completion ./cmd/central/

# Create a layer for each binary and push to ttl.sh
BASELINE_TAG="ttl.sh/stackrox-central-baseline-$(date +%s):24h"
COMPLETION_TAG="ttl.sh/stackrox-central-completion-$(date +%s):24h"

# Baseline image — binary must land at /stackrox/central
crane mutate "$CURRENT_IMAGE" \
  --append <(mkdir -p /tmp/central-layer/stackrox && \
    cp /tmp/central-baseline /tmp/central-layer/stackrox/central && \
    tar -cf - -C /tmp/central-layer stackrox/central) \
  --tag "$BASELINE_TAG"

# Completion image — binary must land at /stackrox/central
crane mutate "$CURRENT_IMAGE" \
  --append <(mkdir -p /tmp/central-layer/stackrox && \
    cp /tmp/central-completion /tmp/central-layer/stackrox/central && \
    tar -cf - -C /tmp/central-layer stackrox/central) \
  --tag "$COMPLETION_TAG"

echo "Baseline: $BASELINE_TAG"
echo "Completion: $COMPLETION_TAG"
```

**Alternative (simpler):** If you only want to test the completion-based limiter
against the stock Central, just build and push one image:

```bash
GOOS=linux GOARCH=amd64 go build -o /tmp/central-completion ./cmd/central/

IMAGE_TAG="ttl.sh/stackrox-central-test-$(date +%s):24h"
crane mutate "$CURRENT_IMAGE" \
  --append <(mkdir -p /tmp/central-layer/stackrox && \
    cp /tmp/central-completion /tmp/central-layer/stackrox/central && \
    tar -cf - -C /tmp/central-layer stackrox/central) \
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

## Step 4: Build local-sensor and Extract Certificates

```bash
cd <stackrox-repo>
go build -o ./tools/local-sensor/local-sensor ./tools/local-sensor/
```

When using `-with-fakeworkload`, local-sensor creates a fake Kubernetes client
that cannot access the real cluster's secrets. You must pre-extract the sensor
TLS certificates so local-sensor can authenticate to Central:

```bash
mkdir -p /tmp/sensor-certs

# Try the current secret name first, fall back to legacy name
SECRET_NAME="tls-cert-sensor"
if ! kubectl get secret -n stackrox "$SECRET_NAME" &>/dev/null; then
  SECRET_NAME="sensor-tls"
fi

kubectl get secret -n stackrox "$SECRET_NAME" \
  -o jsonpath='{.data.ca\.pem}' | base64 -d > /tmp/sensor-certs/ca.pem
kubectl get secret -n stackrox "$SECRET_NAME" \
  -o jsonpath='{.data.sensor-cert\.pem}' | base64 -d > /tmp/sensor-certs/sensor-cert.pem
kubectl get secret -n stackrox "$SECRET_NAME" \
  -o jsonpath='{.data.sensor-key\.pem}' | base64 -d > /tmp/sensor-certs/sensor-key.pem

# For tls-cert-sensor, the key names are cert.pem / key.pem instead
# If the files are empty, try:
kubectl get secret -n stackrox "$SECRET_NAME" \
  -o jsonpath='{.data.cert\.pem}' | base64 -d > /tmp/sensor-certs/sensor-cert.pem
kubectl get secret -n stackrox "$SECRET_NAME" \
  -o jsonpath='{.data.key\.pem}' | base64 -d > /tmp/sensor-certs/sensor-key.pem

echo "Certificates extracted to /tmp/sensor-certs/"
ls -la /tmp/sensor-certs/
```

## Step 5: Run Completion-Based Test

**Important:** `ROX_VIRTUAL_MACHINES=true` must be set on the local-sensor
process, not just on Central. Without it, the sensor's feature flag check
prevents VM index report generation. The log message "VM index reports enabled"
appears during config parsing _before_ the feature flag check and is misleading —
reports will not actually flow unless the feature flag is enabled.

```bash
# Get Central endpoint
CENTRAL_EP=$(kubectl get route -n stackrox central \
  -o jsonpath='{.spec.host}'):443

# Start local-sensor with pre-extracted certificates
ROX_LOCAL_SENSOR=true \
ROX_VIRTUAL_MACHINES=true \
ROX_VM_TEST_MODE=true \
ROX_MTLS_CA_FILE=/tmp/sensor-certs/ca.pem \
ROX_MTLS_CERT_FILE=/tmp/sensor-certs/sensor-cert.pem \
ROX_MTLS_KEY_FILE=/tmp/sensor-certs/sensor-key.pem \
LOGLEVEL=info \
  ./tools/local-sensor/local-sensor \
  -connect-central "$CENTRAL_EP" \
  -with-fakeworkload /tmp/vm-high-load-test.yaml &
LS_PID=$!

# Wait for connection and initial reports to flow
sleep 45
```

### Verify Reports Are Flowing

Before starting measurements, confirm enrichment is happening:

```bash
POD=$(kubectl get pods -n stackrox -l app=central \
  -o jsonpath='{.items[0].metadata.name}')
kubectl logs -n stackrox $POD --since=30s | grep -c "Successfully enriched"
```

If the count is 0 after 45 seconds, see the Troubleshooting section below.

### Monitor Memory (run in a separate terminal, 15 minutes)

```bash
echo "=== COMPLETION-BASED RATE LIMITER ==="
for i in $(seq 1 30); do
  echo "Sample $i: $(date -u +%H:%M:%S)"
  kubectl top pod -n stackrox -l app=central --no-headers
  sleep 30
done
```

### Measure Throughput

```bash
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
    scan = vm.get('scan') or {}
    comps = scan.get('components') or []
    vulns = sum(len(c.get('vulns') or []) for c in comps)
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

# Remove all test env vars
kubectl set env deployment/central -n stackrox \
  ROX_VIRTUAL_MACHINES- \
  ROX_VM_TEST_MODE- \
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

### local-sensor crashes with certificate fetch error

```
panic: failed to fetch certificates from any source:
  secrets "tls-cert-sensor" not found
```

When using `-with-fakeworkload`, local-sensor creates a fake Kubernetes client
that cannot access the real cluster's secrets. Pre-extract certificates as
described in Step 4 and pass them via `ROX_MTLS_CA_FILE`,
`ROX_MTLS_CERT_FILE`, and `ROX_MTLS_KEY_FILE` environment variables.

### Port 8443/9443 conflicts

If local-sensor crashes with port binding errors, set `ROX_LOCAL_SENSOR=true`
(already included in the commands above) which makes the gRPC server endpoints
optional.

### No reports flowing

The log message "VM index reports enabled" is printed during configuration
parsing **before** the feature flag check. Reports will not actually be sent
unless all three conditions are met:

1. `ROX_VIRTUAL_MACHINES=true` is set **on the local-sensor process** (not just
   Central). Without this, the sensor never registers the VM index report
   handler and the report goroutines block forever.
2. `ROX_VM_TEST_MODE=true` is set on Central to allow accepting fake workloads.
3. The sensor has successfully connected to Central (check logs for
   "successfully connected to Central").

### Rate limiter not constraining

If you see no rate-limit warnings in Central logs, the bucket capacity may be
larger than the number of concurrent reports. Lower
`ROX_VM_INDEX_REPORT_BUCKET_CAPACITY` (e.g., to 30) or increase `poolSize` in
the workload YAML.

### Stock binary running instead of custom binary

If both baseline and completion tests show the same throughput (~0.3 rps),
the custom binary is not being used. Verify the `crane mutate --append` tar
places the binary at `stackrox/central` (not at the root). The deployment runs
`/stackrox/central` via the entrypoint scripts; the image entrypoint is ignored.
