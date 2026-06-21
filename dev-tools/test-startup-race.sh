#!/usr/bin/env bash
#
# Startup race condition test — deploys violating workloads BEFORE
# StackRox is fully started, then measures how fast they're detected.
#
# This simulates the worst case: an attacker deploys bad pods during
# a StackRox restart window.
#
# Usage:
#   ./dev-tools/test-startup-race.sh [context]

set -euo pipefail

CTX="${1:-kind-stackrox-faster}"
NS="race-test"
PASSWORD=$(cat deploy/k8s/central-deploy/password 2>/dev/null || echo "")

ts_ms() { python3 -c "import time; print(int(time.time()*1000))"; }
log() { echo "[$(date +%H:%M:%S.%3N)] $*"; }

count_alerts() {
    local deployment="$1"
    kubectl --context "$CTX" -n stackrox exec deploy/central -- \
        roxctl --insecure-skip-tls-verify --insecure -e localhost:8443 -p "$PASSWORD" \
        alert list --json 2>/dev/null | grep -c "\"$deployment\"" 2>/dev/null || echo "0"
}

echo "=========================================="
echo "  StackRox Startup Race Condition Test"
echo "=========================================="
echo ""

# --- Step 1: Deploy violating workloads ---

kubectl --context "$CTX" create ns "$NS" 2>/dev/null || true

log "STEP 1: Deploying violating workloads..."
kubectl --context "$CTX" -n "$NS" apply -f - <<'EOF'
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pre-existing-violation
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pre-existing-violation
  template:
    metadata:
      labels:
        app: pre-existing-violation
    spec:
      containers:
      - name: main
        image: nginx:latest
        securityContext:
          privileged: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pre-existing-ssh
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pre-existing-ssh
  template:
    metadata:
      labels:
        app: pre-existing-ssh
    spec:
      containers:
      - name: main
        image: nginx:1.27
        ports:
        - containerPort: 22
EOF
log "Violating workloads deployed"

# --- Step 2: Restart Central (simulates StackRox restart) ---

log "STEP 2: Restarting Central pod..."
RESTART_START=$(ts_ms)
kubectl --context "$CTX" -n stackrox delete pod -l app=central 2>/dev/null || true

# --- Step 3: Wait for Central to be ready ---

log "STEP 3: Waiting for Central to recover..."
until kubectl --context "$CTX" -n stackrox get pod -l app=central -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q True; do
    sleep 0.5
done
CENTRAL_READY=$(ts_ms)
log "Central ready — took $((CENTRAL_READY - RESTART_START))ms"

# --- Step 4: Wait for Sensor to reconnect ---

log "STEP 4: Waiting for Sensor to reconnect..."
sleep 5  # Give Sensor time to reconnect and sync

# --- Step 5: Check if pre-existing violations were detected ---

log "STEP 5: Checking for pre-existing violation detection..."

DETECTION_START=$(ts_ms)
DETECTED_PRIV=false
DETECTED_SSH=false

for _ in $(seq 1 60); do
    if [ "$DETECTED_PRIV" = false ]; then
        COUNT=$(count_alerts "pre-existing-violation")
        if [ "$COUNT" -gt 0 ]; then
            NOW=$(ts_ms)
            log "  DETECTED: pre-existing-violation — $((NOW - RESTART_START))ms from restart, $((NOW - CENTRAL_READY))ms from Central ready"
            DETECTED_PRIV=true
        fi
    fi

    if [ "$DETECTED_SSH" = false ]; then
        COUNT=$(count_alerts "pre-existing-ssh")
        if [ "$COUNT" -gt 0 ]; then
            NOW=$(ts_ms)
            log "  DETECTED: pre-existing-ssh — $((NOW - RESTART_START))ms from restart, $((NOW - CENTRAL_READY))ms from Central ready"
            DETECTED_SSH=true
        fi
    fi

    if [ "$DETECTED_PRIV" = true ] && [ "$DETECTED_SSH" = true ]; then
        break
    fi
    sleep 0.5
done

echo ""
echo "=========================================="
echo "  Results"
echo "=========================================="
echo "Central restart duration: $((CENTRAL_READY - RESTART_START))ms"
[ "$DETECTED_PRIV" = true ] && echo "Privileged container: DETECTED" || echo "Privileged container: NOT DETECTED"
[ "$DETECTED_SSH" = true ] && echo "SSH port exposed: DETECTED" || echo "SSH port exposed: NOT DETECTED"
echo ""

# --- Cleanup ---

kubectl --context "$CTX" delete ns "$NS" --timeout=30s 2>/dev/null || true
log "Done"
