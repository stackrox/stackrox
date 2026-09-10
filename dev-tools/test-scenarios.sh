#!/usr/bin/env bash
#
# Timing benchmarks for the local dev iteration cycle.
#
# Measures each stage from Go compile to log visibility, warns if
# any stage exceeds its target time. Does not fail — these are
# performance targets, not correctness tests.
#
# Prerequisites: running KinD dev cluster (./dev-tools/kind-dev.sh)
#                running buildkitd container
#
# Usage:
#   ./dev-tools/test-scenarios.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

GOARCH="$(go env GOARCH)"
OUTDIR="bin/linux_${GOARCH}"
NS=stackrox
CTX="${KIND_CLUSTER_NAME:+kind-}${KIND_CLUSTER_NAME:-kind-stackrox-dev}"
REG="${KIND_CLUSTER_NAME:-stackrox-dev}-registry:5000"
LDFLAGS="-s -w"

WARN_COUNT=0

_ms() { python3 -c 'import time; print(int(time.time()*1000))'; }

_check() {
    local name="$1" elapsed="$2" target="$3"
    if [[ $elapsed -gt $target ]]; then
        WARN_COUNT=$((WARN_COUNT + 1))
        printf "  %-45s %6sms  \033[33mWARN\033[0m (target: %sms)\n" "$name" "$elapsed" "$target"
    else
        printf "  %-45s %6sms  \033[32mOK\033[0m\n" "$name" "$elapsed"
    fi
}

_bk() {
    podman exec buildkitd buildctl build \
        --addr unix:///run/buildkit/buildkitd.sock \
        --frontend dockerfile.v0 \
        --local context=/context --local dockerfile=/context \
        --opt build-arg:TARGET_ARCH=${GOARCH} \
        --output "type=image,name=${REG}/main:$1,push=true,registry.insecure=true" \
        2>&1 | tail -1
}

_go1() {
    GOOS=linux GOARCH=$GOARCH CGO_ENABLED=0 \
        go build -buildvcs=false -trimpath -ldflags="$LDFLAGS" \
        -o "${OUTDIR}/$1" "$2" 2>&1
}

_goall() {
    for e in central:./central migrator:./migrator compliance:./compliance/cmd/compliance \
             kubernetes-sensor:./sensor/kubernetes sensor-upgrader:./sensor/upgrader \
             admission-control:./sensor/admission-control config-controller:./config-controller; do
        _go1 "${e%%:*}" "${e##*:}"
    done
}

echo "=== Local Dev Scenario Benchmarks ==="
echo ""

# --- Preflight ---
if ! kubectl --context "$CTX" -n "$NS" get deploy/central >/dev/null 2>&1; then
    echo "Central not deployed. Run ./dev-tools/kind-dev.sh first."
    exit 1
fi
if ! podman exec buildkitd buildctl debug workers --addr unix:///run/buildkit/buildkitd.sock >/dev/null 2>&1; then
    echo "BuildKit not running. Start buildkitd container first."
    exit 1
fi

# --- Warm up cache ---
echo "Warming up Go build cache..."
_go1 central ./central >/dev/null
_goall >/dev/null
echo ""

# --- S1: No-change compile (single binary) ---
T=$(_ms); _go1 central ./central >/dev/null
_check "S1 No-change compile (1 binary)" $(($(_ms)-T)) 5000

# --- S2: One-line change compile (single binary) ---
echo '// s2' >> central/main.go
T=$(_ms); _go1 central ./central >/dev/null
_check "S2 One-line change compile (1 binary)" $(($(_ms)-T)) 10000
git checkout central/main.go 2>/dev/null

# --- S3: No-change compile (all binaries) ---
T=$(_ms); _goall >/dev/null
_check "S3 No-change compile (all 7 binaries)" $(($(_ms)-T)) 15000

# --- S4: One-line change compile (all binaries) ---
echo '// s4' >> central/main.go
T=$(_ms); _goall >/dev/null
_check "S4 One-change compile (all 7, 1 changed)" $(($(_ms)-T)) 20000
git checkout central/main.go 2>/dev/null

# --- S5: No-change image build+push ---
T=$(_ms); _bk s5 >/dev/null
_check "S5 No-change image build+push (BuildKit)" $(($(_ms)-T)) 2000

# --- S6: One-binary-changed image build+push ---
touch image/rhel/bin/central
T=$(_ms); _bk s6 >/dev/null
_check "S6 One-binary image build+push (BuildKit)" $(($(_ms)-T)) 8000

# --- S7: kubectl set image + kill pod ---
T=$(_ms)
kubectl --context "$CTX" -n "$NS" set image deploy/central central=localhost:5000/main:s6 2>/dev/null
kubectl --context "$CTX" -n "$NS" delete pod -l app=central --grace-period=0 2>/dev/null
_check "S7 kubectl set image + kill pod" $(($(_ms)-T)) 3000

# --- S8: Pod restart until ready ---
T=$(_ms)
for i in $(seq 1 60); do
    ready=$(kubectl --context "$CTX" -n "$NS" get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
    [[ "$ready" == "1" ]] && break; sleep 1
done
_check "S8 Pod restart until Ready" $(($(_ms)-T)) 20000

echo ""

# --- S9: Full E2E — code change → log visible ---
marker="SCENARIO-$(date +%s)-$$"
marker_file="central/dev_scenario_marker.go"
cat > "$marker_file" <<GOFILE
package main
import ("fmt"; "os")
func init() { fmt.Fprintln(os.Stderr, "$marker") }
GOFILE
trap "rm -f '$marker_file'" EXIT INT TERM

TAG="e2e-$(date +%s)"
T=$(_ms)
_go1 central ./central >/dev/null
T1=$(_ms)
cp "${OUTDIR}/central" image/rhel/bin/central
_bk "$TAG" >/dev/null
T2=$(_ms)
kubectl --context "$CTX" -n "$NS" set image deploy/central "central=localhost:5000/main:${TAG}" 2>/dev/null
kubectl --context "$CTX" -n "$NS" delete pod -l app=central --grace-period=0 2>/dev/null
T3=$(_ms)
for i in $(seq 1 60); do
    kubectl --context "$CTX" -n "$NS" logs -l app=central --tail=500 2>/dev/null | grep -q "$marker" && break
    sleep 1
done
T4=$(_ms)
rm -f "$marker_file"

S9_COMPILE=$((T1 - T))
S9_IMAGE=$((T2 - T1))
S9_KUBECTL=$((T3 - T2))
S9_POD=$((T4 - T3))
S9_TOTAL=$((T4 - T))

echo "S9 Full E2E: code change → log visible"
_check "    compile" "$S9_COMPILE" 10000
_check "    image+push" "$S9_IMAGE" 8000
_check "    kubectl" "$S9_KUBECTL" 3000
_check "    pod restart" "$S9_POD" 20000
_check "    TOTAL" "$S9_TOTAL" 30000

echo ""
if [[ $WARN_COUNT -eq 0 ]]; then
    echo "All scenarios within target."
else
    echo "$WARN_COUNT scenario(s) exceeded target time."
fi
