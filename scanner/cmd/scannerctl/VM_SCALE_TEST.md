# Scanner V4 VM Scale Test Guide

This guide documents how to run isolated load tests against Scanner V4 Matcher using `scannerctl vm-scale`.

## Overview

The `scannerctl vm-scale` command sends `GetVulnerabilities` requests with synthetic VM index reports directly to Scanner V4 Matcher, bypassing Central and Sensor. This allows isolated testing of Scanner V4 performance.

## Prerequisites

- Access to a Kubernetes cluster with StackRox deployed
- `kubectl` configured to access the cluster
- Go toolchain for building `scannerctl`

## Step 1: Build scannerctl for Linux

```bash
cd /path/to/stackrox
GOOS=linux GOARCH=amd64 go build -o bin/scannerctl-linux ./scanner/cmd/scannerctl
```

## Step 2: Create a Load Test Pod

The pod must have `app=central` label to pass NetworkPolicy:

```bash
# Create pod with correct label
kubectl run loadtest -n stackrox \
  --image=alpine:latest \
  --restart=Never \
  --labels="app=central" \
  -- sleep 3600

# Wait for pod to be ready
kubectl wait --for=condition=Ready pod/loadtest -n stackrox

# Copy scannerctl binary
kubectl cp bin/scannerctl-linux stackrox/loadtest:/scannerctl
kubectl exec -n stackrox loadtest -- chmod +x /scannerctl
```

## Step 3: Run the Scale Test

### Basic Test (verify connectivity)
```bash
kubectl exec -it -n stackrox loadtest -- /scannerctl vm-scale \
  --matcher-address scanner-v4-matcher.stackrox.svc.cluster.local:8443 \
  --insecure-skip-tls-verify \
  --direct-pod-ips \
  --requests 10 \
  --workers 5 \
  --packages 500 \
  --verbose
```

### Sustained Load Test (rate-limited)
```bash
kubectl exec -it -n stackrox loadtest -- /scannerctl vm-scale \
  --matcher-address scanner-v4-matcher.stackrox.svc.cluster.local:8443 \
  --insecure-skip-tls-verify \
  --direct-pod-ips \
  --rate 3 \
  --duration 2m \
  --workers 20 \
  --packages 500 \
  --verbose
```

### High Load Test
```bash
kubectl exec -it -n stackrox loadtest -- /scannerctl vm-scale \
  --matcher-address scanner-v4-matcher.stackrox.svc.cluster.local:8443 \
  --insecure-skip-tls-verify \
  --direct-pod-ips \
  --rate 5 \
  --duration 2m \
  --workers 30 \
  --packages 500 \
  --verbose
```

## Step 4: Monitor During Test

In separate terminals:

```bash
# Watch matcher pod CPU
watch -n2 'kubectl top pods -n stackrox -l app=scanner-v4-matcher --use-protocol-buffers'

# Watch DB CPU
watch -n2 'kubectl top pods -n stackrox -l app=scanner-v4-db --use-protocol-buffers'

# Watch matcher logs
kubectl logs -f -n stackrox -l app=scanner-v4-matcher --prefix | grep -i "GetVulnerabilities"
```

## Step 5: Cleanup

```bash
kubectl delete pod loadtest -n stackrox
```

## Command Options

| Flag | Default | Description |
|------|---------|-------------|
| `--requests` | 100 | Total requests (ignored if `--duration` set) |
| `--workers` | 15 | Number of parallel workers |
| `--packages` | 2000 | Packages per VM index report |
| `--repos` | 10 | Repositories per report |
| `--rate` | 0 | Target req/s (0 = unlimited) |
| `--duration` | 0 | Run duration (0 = use `--requests`) |
| `--direct-pod-ips` | false | **Required for load distribution** - resolves DNS and connects directly to pod IPs |
| `--verbose` | false | Print each request result |

## Important Notes

### Load Balancing Issue

The `--direct-pod-ips` flag is **required** for proper load distribution across multiple matcher pods. Without it, gRPC's round-robin load balancing doesn't work correctly with headless services, and all requests go to a single pod.

With `--direct-pod-ips`:
1. DNS is resolved to get all pod IPs
2. Workers are assigned to pods round-robin
3. Each worker connects directly to its assigned pod

### Worker Calculation

Workers are needed to sustain the target rate when requests have latency:

```
Workers needed = rate × avg_latency
```

Example:
- Target rate: 3 req/s
- Average latency: 5 seconds
- Workers needed: 3 × 5 = 15 workers

### NetworkPolicy

The loadtest pod must have `app=central` label because the Scanner V4 Matcher NetworkPolicy only allows ingress from pods with this label.

## Example Output

```
2025/12/11 11:42:52 Resolved 5 pod IPs: [10.129.2.53:8443 10.128.2.57:8443 10.131.1.200:8443 10.131.1.201:8443 10.128.2.58:8443]
2025/12/11 11:42:52 VM Scale Test Configuration:
2025/12/11 11:42:52   Workers: 10
2025/12/11 11:42:52   Packages per report: 500
2025/12/11 11:42:52   Direct pod IPs: true
2025/12/11 11:42:52   Total requests: 25
2025/12/11 11:42:52 [worker-0] connecting to pod 10.129.2.53:8443
2025/12/11 11:42:52 [worker-1] connecting to pod 10.128.2.57:8443
...
2025/12/11 11:42:55 [worker-7] [pod=10.131.1.200:8443] req=3 OK (3.01s)
2025/12/11 11:42:55 [worker-3] [pod=10.131.1.201:8443] req=6 OK (3.14s)
...
=== VM Scale Test Results ===
2025/12/11 11:43:01 Total time: 9.288152021s
2025/12/11 11:43:01 Total requests: 25
2025/12/11 11:43:01 Success: 25, Failure: 0
2025/12/11 11:43:01 Throughput: 2.69 req/s
```

## Troubleshooting

### Connection Timeout Errors

If you see `dial tcp <ip>:8443: i/o timeout`:
1. Check the loadtest pod has `app=central` label
2. Verify NetworkPolicy allows traffic: `kubectl get networkpolicy -n stackrox scanner-v4-matcher -o yaml`

### DNS Resolution Fails

If DNS resolution fails:
- Use full FQDN: `scanner-v4-matcher.stackrox.svc.cluster.local:8443`
- Verify service exists: `kubectl get svc -n stackrox scanner-v4-matcher`

### Only One Matcher Gets Load

Make sure to use `--direct-pod-ips` flag. Without it, gRPC load balancing doesn't work correctly with headless services.

