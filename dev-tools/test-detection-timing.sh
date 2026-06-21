#!/usr/bin/env bash
#
# Test StackRox detection timing — measures how fast violations are
# detected after deploying workloads or triggering runtime events.
#
# Usage:
#   ./dev-tools/test-detection-timing.sh [context]
#
# Default context: kind-stackrox-faster

set -euo pipefail

CTX="${1:-kind-stackrox-faster}"
NS="detection-test"
PASSWORD=$(cat deploy/k8s/central-deploy/password 2>/dev/null || echo "")
CENTRAL_EP="central.stackrox.svc:443"

ts_ms() { python3 -c "import time; print(int(time.time()*1000))"; }

log() { echo "[$(date +%H:%M:%S.%3N)] $*"; }

wait_for_central() {
    log "Waiting for Central API..."
    until kubectl --context "$CTX" -n stackrox exec deploy/central -- \
        roxctl --insecure-skip-tls-verify --insecure -e localhost:8443 -p "$PASSWORD" \
        central whoami 2>/dev/null | grep -q 'admin'; do
        sleep 1
    done
    log "Central API ready"
}

count_alerts() {
    local deployment="$1"
    kubectl --context "$CTX" -n stackrox exec deploy/central -- \
        roxctl --insecure-skip-tls-verify --insecure -e localhost:8443 -p "$PASSWORD" \
        alert list --json 2>/dev/null | grep -c "\"$deployment\"" 2>/dev/null || echo "0"
}

poll_for_alert() {
    local deployment="$1"
    local start_ms="$2"
    local timeout_s="${3:-30}"
    for _ in $(seq 1 $((timeout_s * 2))); do
        local count
        count=$(count_alerts "$deployment")
        if [ "$count" -gt 0 ]; then
            local now_ms
            now_ms=$(ts_ms)
            local elapsed=$((now_ms - start_ms))
            log "  DETECTED: $deployment — ${elapsed}ms (${count} alert(s))"
            return 0
        fi
        sleep 0.5
    done
    log "  TIMEOUT: $deployment — no alert after ${timeout_s}s"
    return 1
}

# --- Setup ---

kubectl --context "$CTX" create ns "$NS" 2>/dev/null || true
wait_for_central

echo ""
echo "=========================================="
echo "  StackRox Detection Timing Tests"
echo "=========================================="
echo ""

# --- Test 1: Deploy-time — Privileged container with latest tag ---

log "TEST 1: Deploy-time — privileged container with :latest tag"
T1=$(ts_ms)
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-privileged-latest
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-privileged-latest
  template:
    metadata:
      labels:
        app: test-privileged-latest
    spec:
      containers:
      - name: main
        image: nginx:latest
        securityContext:
          privileged: true
          runAsUser: 0
EOF
poll_for_alert "test-privileged-latest" "$T1"

# --- Test 2: Runtime — Package manager execution (apt-get) ---

log "TEST 2: Runtime — apt-get execution in Ubuntu container"
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-apt-exec
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-apt-exec
  template:
    metadata:
      labels:
        app: test-apt-exec
    spec:
      containers:
      - name: main
        image: ubuntu:22.04
        command: ["sleep", "3600"]
EOF
log "  Waiting for pod to be running..."
kubectl --context "$CTX" -n "$NS" wait --for=condition=ready pod -l app=test-apt-exec --timeout=60s 2>/dev/null || true
sleep 2
T2=$(ts_ms)
log "  Executing apt-get update..."
kubectl --context "$CTX" -n "$NS" exec deploy/test-apt-exec -- apt-get update 2>/dev/null || true
poll_for_alert "test-apt-exec" "$T2"

# --- Test 3: Runtime — Red Hat package manager (dnf/rpm) ---

log "TEST 3: Runtime — dnf execution in UBI container"
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-dnf-exec
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-dnf-exec
  template:
    metadata:
      labels:
        app: test-dnf-exec
    spec:
      containers:
      - name: main
        image: registry.access.redhat.com/ubi9-minimal:latest
        command: ["sleep", "3600"]
EOF
log "  Waiting for pod to be running..."
kubectl --context "$CTX" -n "$NS" wait --for=condition=ready pod -l app=test-dnf-exec --timeout=60s 2>/dev/null || true
sleep 2
T3=$(ts_ms)
log "  Executing rpm -qa..."
kubectl --context "$CTX" -n "$NS" exec deploy/test-dnf-exec -- rpm -qa 2>/dev/null | head -1 || true
poll_for_alert "test-dnf-exec" "$T3"

# --- Test 4: Runtime — netcat execution ---

log "TEST 4: Runtime — netcat to unexpected IP"
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-netcat
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-netcat
  template:
    metadata:
      labels:
        app: test-netcat
    spec:
      containers:
      - name: main
        image: ubuntu:22.04
        command: ["sleep", "3600"]
EOF
log "  Waiting for pod to be running..."
kubectl --context "$CTX" -n "$NS" wait --for=condition=ready pod -l app=test-netcat --timeout=60s 2>/dev/null || true
sleep 2
T4=$(ts_ms)
log "  Installing and running netcat..."
kubectl --context "$CTX" -n "$NS" exec deploy/test-netcat -- bash -c 'apt-get update -qq && apt-get install -qq -y netcat-openbsd && nc -z -w1 1.2.3.4 80 || true' 2>/dev/null || true
poll_for_alert "test-netcat" "$T4"

# --- Test 5: Deploy-time — container running as root ---

log "TEST 5: Deploy-time — container running as root"
T5=$(ts_ms)
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-root-user
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-root-user
  template:
    metadata:
      labels:
        app: test-root-user
    spec:
      containers:
      - name: main
        image: nginx:1.27
        securityContext:
          runAsUser: 0
EOF
poll_for_alert "test-root-user" "$T5"

# --- Test 6: Deploy-time — SSH port exposed ---

log "TEST 6: Deploy-time — SSH port 22 exposed"
T6=$(ts_ms)
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-ssh-exposed
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-ssh-exposed
  template:
    metadata:
      labels:
        app: test-ssh-exposed
    spec:
      containers:
      - name: main
        image: nginx:1.27
        ports:
        - containerPort: 22
          name: ssh
EOF
poll_for_alert "test-ssh-exposed" "$T6"

# --- Cleanup ---

echo ""
echo "=========================================="
echo "  Cleanup"
echo "=========================================="
kubectl --context "$CTX" delete ns "$NS" --timeout=30s 2>/dev/null || true
log "Done"
