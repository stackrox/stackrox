#!/usr/bin/env bash
# profile-reassess.sh -- CPU and heap profiling of Central + Sensor during reassess.
#
# 1. Creates real deployments (replicas=0) via the shared burst infrastructure
#    so that Central/Sensor have images to reprocess.
# 2. Captures Go pprof profiles (heap pre, CPU during reassess, heap post).
#
# Run once on master, once on the PR branch, then compare with go tool pprof.
#
# Central pprof: main API port (8443), HTTPS, requires Admin auth.
# Sensor pprof:  localhost:6060 inside the pod, HTTP, no auth.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

: "${ROX_PASSWORD:?ROX_PASSWORD must be set}"
: "${ROX_CENTRAL_ADDRESS:?ROX_CENTRAL_ADDRESS must be set}"
: "${ROX_ADMIN_USER:=admin}"
: "${ROX_NAMESPACE:=stackrox}"
: "${CPU_PROFILE_SECONDS:=60}"
: "${REASSESS_WAIT_TIMEOUT:=300}"

: "${BURST_SIZE:=500}"
: "${UNIQUE_PCT:=60}"
: "${PARALLEL:=50}"
: "${NAMESPACE:=burst-profile}"
: "${METRICS_PORT:=9090}"

DRY_RUN="false"
export DRY_RUN

SENSOR_PPROF_PORT=6060
SENSOR_LOCAL_PORT="${SENSOR_LOCAL_PORT:-6060}"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR="${OUTPUT_DIR:-/tmp/profiles/${BRANCH_NAME}-$(date +%Y%m%d-%H%M%S)}"

WORK_DIR=$(mktemp -d)

# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

cleanup() {
    kill_port_forwards
    kubectl delete namespace "$NAMESPACE" --ignore-not-found &>/dev/null || true
    [[ -n "${WORK_DIR:-}" ]] && rm -rf "$WORK_DIR"
}
trap cleanup EXIT

die() { echo "ERROR: $*" >&2; exit 1; }

central_pprof() {
    local path="$1" outfile="$2"
    curl -sk -u "${ROX_ADMIN_USER}:${ROX_PASSWORD}" \
        "https://${ROX_CENTRAL_ADDRESS}${path}" \
        -o "$outfile" 2>/dev/null
}

sensor_pprof() {
    local path="$1" outfile="$2"
    curl -s "http://localhost:${SENSOR_LOCAL_PORT}${path}" \
        -o "$outfile" 2>/dev/null
}

setup_sensor_pprof_port_forward() {
    local sensor_pod
    sensor_pod=$(kubectl -n "$ROX_NAMESPACE" get pod -l app=sensor \
        --field-selector=status.phase=Running \
        -o jsonpath='{.items[0].metadata.name}')
    [[ -n "$sensor_pod" ]] || die "no running Sensor pod found"

    echo "  Port-forwarding Sensor pprof: ${sensor_pod} ${SENSOR_LOCAL_PORT}:${SENSOR_PPROF_PORT}"
    kubectl -n "$ROX_NAMESPACE" port-forward "$sensor_pod" \
        "${SENSOR_LOCAL_PORT}:${SENSOR_PPROF_PORT}" >/dev/null 2>&1 &
    PF_PIDS+=($!)

    local reachable=false
    for _ in $(seq 1 15); do
        if curl -sf "http://localhost:${SENSOR_LOCAL_PORT}/debug/pprof" >/dev/null 2>&1; then
            reachable=true
            break
        fi
        sleep 1
    done
    $reachable || die "cannot reach Sensor pprof at localhost:${SENSOR_LOCAL_PORT} after 15s"
    echo "  Sensor pprof reachable."
}

verify_central_pprof() {
    echo "  Verifying Central pprof at ${ROX_CENTRAL_ADDRESS}..."
    local http_code
    http_code=$(curl -sk -o /dev/null -w '%{http_code}' \
        -u "${ROX_ADMIN_USER}:${ROX_PASSWORD}" \
        "https://${ROX_CENTRAL_ADDRESS}/debug/pprof/" || true)
    [[ "$http_code" == "200" ]] \
        || die "Central pprof returned HTTP ${http_code} (check ROX_CENTRAL_ADDRESS, ROX_PASSWORD)"
    echo "  Central pprof reachable."
}

capture_heap() {
    local label="$1"
    echo "  Capturing ${label} heap profiles..."
    central_pprof "/debug/heap" "${OUTPUT_DIR}/central-heap-${label}.pb.gz"
    sensor_pprof  "/debug/heap" "${OUTPUT_DIR}/sensor-heap-${label}.pb.gz"
    echo "  Saved: central-heap-${label}.pb.gz, sensor-heap-${label}.pb.gz"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

cat <<EOF
================================================================================
  profile-reassess.sh
================================================================================
  Branch:                ${BRANCH_NAME}
  BURST_SIZE:            ${BURST_SIZE}
  UNIQUE_PCT:            ${UNIQUE_PCT}%
  CPU_PROFILE_SECONDS:   ${CPU_PROFILE_SECONDS}
  ROX_CENTRAL_ADDRESS:   ${ROX_CENTRAL_ADDRESS}
  REASSESS_WAIT_TIMEOUT: ${REASSESS_WAIT_TIMEOUT}s
  Output:                ${OUTPUT_DIR}
================================================================================

EOF

mkdir -p "$OUTPUT_DIR"

# ---- Step 1: Verify pprof endpoints ----
echo "=== Step 1: Verify pprof endpoints ==="
verify_central_pprof
setup_sensor_pprof_port_forward
echo ""

# ---- Step 2: Create burst deployments (replicas=0) ----
echo "=== Step 2: Create burst deployments (replicas=0) ==="
setup_namespace
generate_slow_path_manifests
run_burst "Creating deployments"

echo "  Waiting 30s for Central/Sensor to discover deployments..."
sleep 30
echo ""

# ---- Step 3: Pre-reassess heap ----
echo "=== Step 3: Pre-reassess heap snapshots ==="
capture_heap "pre"
echo ""

# ---- Step 4: Start CPU profiles + trigger reassess ----
echo "=== Step 4: CPU profiling (${CPU_PROFILE_SECONDS}s) + reassess ==="

echo "  Starting Central CPU profile (${CPU_PROFILE_SECONDS}s, background)..."
central_pprof "/debug/pprof/profile?seconds=${CPU_PROFILE_SECONDS}" \
    "${OUTPUT_DIR}/central-cpu.pb.gz" &
CENTRAL_CPU_PID=$!

echo "  Starting Sensor CPU profile (${CPU_PROFILE_SECONDS}s, background)..."
sensor_pprof "/debug/pprof/profile?seconds=${CPU_PROFILE_SECONDS}" \
    "${OUTPUT_DIR}/sensor-cpu.pb.gz" &
SENSOR_CPU_PID=$!

sleep 2

trigger_and_wait_for_reprocessor

echo ""
echo "  Waiting for CPU profiles to finish..."
wait "$CENTRAL_CPU_PID" 2>/dev/null || echo "  WARN: Central CPU profile request failed"
wait "$SENSOR_CPU_PID" 2>/dev/null || echo "  WARN: Sensor CPU profile request failed"
echo "  Saved: central-cpu.pb.gz, sensor-cpu.pb.gz"
echo ""

# ---- Step 5: Post-reassess heap ----
echo "=== Step 5: Post-reassess heap snapshots ==="
sleep 3
capture_heap "post"
echo ""

# ---- Step 6: Goroutine snapshots ----
echo "=== Step 6: Goroutine snapshots ==="
echo "  Capturing goroutine profiles..."
central_pprof "/debug/goroutine" "${OUTPUT_DIR}/central-goroutine.pb.gz"
sensor_pprof  "/debug/goroutine" "${OUTPUT_DIR}/sensor-goroutine.pb.gz"
echo "  Saved: central-goroutine.pb.gz, sensor-goroutine.pb.gz"
echo ""

# ---- Summary ----
echo "=== Summary ==="
echo ""
echo "  Output directory: ${OUTPUT_DIR}"
echo "  Deployments:      ${BURST_SIZE} (${UNIQUE_COUNT} unique images, replicas=0)"
echo ""
echo "  Files:"
for f in "${OUTPUT_DIR}"/*.pb.gz; do
    [[ -f "$f" ]] || continue
    local_size=$(wc -c < "$f" | awk '{printf "%.1fK", $1/1024}')
    printf "    %s (%s)\n" "$(basename "$f")" "$local_size"
done
echo ""

cat <<'EOF'
  ---- Comparison commands ----

  # Interactive CPU profile (single branch):
  go tool pprof -http=:8080 ./profiles/<branch>/central-cpu.pb.gz
  go tool pprof -http=:8081 ./profiles/<branch>/sensor-cpu.pb.gz

  # Compare CPU between master and PR (top diff):
  go tool pprof -diff_base=./profiles/master-*/central-cpu.pb.gz \
                            ./profiles/<pr-branch>-*/central-cpu.pb.gz

  go tool pprof -diff_base=./profiles/master-*/sensor-cpu.pb.gz \
                            ./profiles/<pr-branch>-*/sensor-cpu.pb.gz

  # Compare heap growth (post - pre, single branch):
  go tool pprof -diff_base=./profiles/<branch>/central-heap-pre.pb.gz \
                            ./profiles/<branch>/central-heap-post.pb.gz

  # Compare heap across branches:
  go tool pprof -diff_base=./profiles/master-*/central-heap-post.pb.gz \
                            ./profiles/<pr-branch>-*/central-heap-post.pb.gz

  # Text summary (no browser):
  go tool pprof -top ./profiles/<branch>/central-cpu.pb.gz
  go tool pprof -top -diff_base=./profiles/master-*/sensor-cpu.pb.gz \
                                ./profiles/<pr-branch>-*/sensor-cpu.pb.gz
EOF
echo ""
