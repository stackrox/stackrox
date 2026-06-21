#!/usr/bin/env bash
#
# Integration tests for the KinD local development workflow.
#
# Launches kind-dev.sh in the background (same as a developer would run it),
# waits for the deployment to come up, runs tests, then stops Skaffold.
#
# Usage:
#   ./dev-tools/test-kind-dev.sh              # run all tests
#   ./dev-tools/test-kind-dev.sh test_build   # run single test (cluster must exist)
#   ./dev-tools/test-kind-dev.sh teardown     # clean up only

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-stackrox-test}"
NAMESPACE=stackrox
GOARCH="$(go env GOARCH)"
REG_PORT="${KIND_REGISTRY_PORT:-5000}"
KIND_DEV_PID=""

PASS=0
FAIL=0
SKIP=0
ERRORS=()

# --- Harness ---

_now_ms() { python3 -c 'import time; print(int(time.time()*1000))'; }
_elapsed_ms() { local n; n=$(_now_ms); echo $(( n - $1 )); }
_pass() { PASS=$((PASS + 1)); printf "  %-35s \033[32mPASS\033[0m  %s\n" "$1" "${2:-}"; }
_fail() { FAIL=$((FAIL + 1)); ERRORS+=("$1: ${2:-}"); printf "  %-35s \033[31mFAIL\033[0m  %s\n" "$1" "${2:-}"; }
_skip() { SKIP=$((SKIP + 1)); printf "  %-35s \033[33mSKIP\033[0m  %s\n" "$1" "${2:-}"; }
_rt() { if command -v docker >/dev/null 2>&1; then docker "$@"; else podman "$@"; fi; }

_require_deploy() {
    if ! kubectl -n "$NAMESPACE" get deploy/central >/dev/null 2>&1; then
        _skip "$1" "not deployed"; return 1
    fi; return 0
}

_stop_kind_dev() {
    if [[ -n "$KIND_DEV_PID" ]] && kill -0 "$KIND_DEV_PID" 2>/dev/null; then
        echo "Stopping kind-dev.sh (pid $KIND_DEV_PID)..."
        kill -INT "$KIND_DEV_PID" 2>/dev/null
        wait "$KIND_DEV_PID" 2>/dev/null || true
        KIND_DEV_PID=""
    fi
}

# ==========================================================================
# SETUP — launch kind-dev.sh and wait for Central to be ready
# ==========================================================================

test_setup() {
    local name="test_setup"
    local start; start=$(_now_ms)

    echo "    Launching kind-dev.sh in background..."
    ./dev-tools/kind-dev.sh > /tmp/kind-dev-test.log 2>&1 &
    KIND_DEV_PID=$!

    echo "    Waiting for Central to be ready (up to 5 min)..."
    local deadline=$((SECONDS + 300))
    while [[ $SECONDS -lt $deadline ]]; do
        if kubectl -n "$NAMESPACE" get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null | grep -q "1"; then
            _pass "$name" "$(_elapsed_ms "$start")ms — kind-dev.sh + skaffold deploy complete"
            return 0
        fi
        if ! kill -0 "$KIND_DEV_PID" 2>/dev/null; then
            echo "    kind-dev.sh exited unexpectedly. Last 20 lines:"
            tail -20 /tmp/kind-dev-test.log
            _fail "$name" "kind-dev.sh exited"
            return 1
        fi
        sleep 3
    done

    echo "    Timed out. Last 20 lines of kind-dev.sh output:"
    tail -20 /tmp/kind-dev-test.log
    _fail "$name" "central not ready after 5 minutes"
}

# ==========================================================================
# INFRASTRUCTURE
# ==========================================================================

test_local_registry() {
    local name="test_local_registry"
    if curl -s "http://localhost:${REG_PORT}/v2/" >/dev/null 2>&1; then
        _pass "$name" "localhost:${REG_PORT}"
    else
        _fail "$name" "registry not responding"
    fi
}

test_resources() {
    local name="test_resources"
    _require_deploy "$name" || return 0

    local central_limit db_limit shared_buffers ok=true
    central_limit=$(kubectl -n "$NAMESPACE" get deploy/central -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')
    db_limit=$(kubectl -n "$NAMESPACE" get deploy/central-db -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')
    shared_buffers=$(kubectl -n "$NAMESPACE" get cm central-db-config -o jsonpath='{.data.postgresql\.conf}' 2>/dev/null | grep shared_buffers | awk '{print $3}')

    [[ "$central_limit" == "2Gi" ]] || { echo "    central=$central_limit expected 2Gi"; ok=false; }
    [[ "$db_limit" == "2Gi" ]] || { echo "    db=$db_limit expected 2Gi"; ok=false; }
    [[ "$shared_buffers" == "256MB" ]] || { echo "    shared_buffers=$shared_buffers expected 256MB"; ok=false; }

    $ok && _pass "$name" "central=${central_limit} db=${db_limit} shared_buffers=${shared_buffers}" \
        || _fail "$name" "incorrect"
}

test_memory() {
    local name="test_memory"
    _require_deploy "$name" || return 0
    local mem
    mem=$(_rt stats --no-stream --format '{{.MemUsage}}' "${KIND_CLUSTER_NAME}-control-plane" 2>/dev/null || echo "unknown")
    echo "    KinD container: $mem"
    _pass "$name" "$mem"
}

# ==========================================================================
# GO DEVELOPER
# ==========================================================================

test_skaffold_build() {
    local name="test_skaffold_build"
    if ! curl -s "http://localhost:${REG_PORT}/v2/" >/dev/null 2>&1; then
        _skip "$name" "no registry"; return 0
    fi

    local start; start=$(_now_ms)
    if ! IMAGE="localhost:${REG_PORT}/main:test-$(date +%s)" ./dev-tools/skaffold-build.sh 2>&1 | tail -3; then
        _fail "$name" "build failed"; return 1
    fi
    _pass "$name" "$(_elapsed_ms "$start")ms"
}

test_skaffold_rebuild_cached() {
    local name="test_skaffold_rebuild_cached"
    if ! curl -s "http://localhost:${REG_PORT}/v2/" >/dev/null 2>&1; then
        _skip "$name" "no registry"; return 0
    fi

    local start; start=$(_now_ms)
    IMAGE="localhost:${REG_PORT}/main:cached-$(date +%s)" ./dev-tools/skaffold-build.sh 2>&1 | tail -3
    local elapsed; elapsed=$(_elapsed_ms "$start")

    [[ $elapsed -lt 60000 ]] \
        && _pass "$name" "${elapsed}ms (under 60s)" \
        || _fail "$name" "${elapsed}ms (over 60s)"
}

test_rebuild_under_30s() {
    # Verifies that a single-binary Go change rebuilds + pushes + restarts in under 30s.
    # This is the developer iteration speed target.
    local name="test_rebuild_under_30s"
    _require_deploy "$name" || return 0
    if ! curl -s "http://localhost:${REG_PORT}/v2/" >/dev/null 2>&1; then
        _skip "$name" "no registry"; return 0
    fi

    local ready
    ready=$(kubectl -n "$NAMESPACE" get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
    [[ -n "$ready" ]] && [[ "$ready" -ge 1 ]] || { _skip "$name" "central not ready"; return 0; }

    # Need a warm build cache — run a baseline build first if not done
    local goarch; goarch="$(go env GOARCH)"
    if [[ ! -f "bin/linux_${goarch}/central" ]]; then
        _skip "$name" "no baseline build"; return 0
    fi

    # Drop a marker into central only (single binary change)
    local marker="SPEED-$(date +%s)-$$"
    local marker_file="central/dev_speed_marker.go"
    cat > "$marker_file" <<GOFILE
package main
import ("fmt"; "os")
func init() { fmt.Fprintln(os.Stderr, "$marker") }
GOFILE
    trap "rm -f '$marker_file'" EXIT INT TERM

    local start; start=$(_now_ms)

    # Build + push + restart — same as skaffold-build.sh but only central
    local goarch; goarch="$(go env GOARCH)"

    # Only recompile central (the changed binary)
    GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 \
        go build -buildvcs=false -trimpath -ldflags="-s -w" \
        -o "bin/linux_${goarch}/central" ./central 2>&1

    cp "bin/linux_${goarch}/central" image/rhel/bin/central

    local tag="speed-$(date +%s)"

    # Use BuildKit for fast cached image build+push
    if BUILDKIT_HOST="${BUILDKIT_HOST:-}" buildctl debug workers >/dev/null 2>&1; then
        buildctl build \
            --frontend dockerfile.v0 \
            --local context=image/rhel \
            --local dockerfile=image/rhel \
            --opt "build-arg:TARGET_ARCH=${goarch}" \
            --output "type=image,name=${KIND_CLUSTER_NAME}-registry:5000/main:${tag},push=true,registry.insecure=true" \
            2>&1 | tail -2
    else
        local rt; if command -v docker >/dev/null 2>&1; then rt=docker; else rt=podman; fi
        $rt build -t "localhost:${REG_PORT}/main:${tag}" --build-arg "TARGET_ARCH=${goarch}" \
            --file image/rhel/Dockerfile image/rhel 2>&1 | tail -2
        if [[ "$rt" == "podman" ]]; then
            $rt push "localhost:${REG_PORT}/main:${tag}" --tls-verify=false 2>&1 | tail -1
        else
            $rt push "localhost:${REG_PORT}/main:${tag}" 2>&1 | tail -1
        fi
    fi

    kubectl -n "$NAMESPACE" set image deploy/central "central=localhost:${REG_PORT}/main:${tag}" 2>/dev/null
    kubectl -n "$NAMESPACE" delete pod -l app=central --grace-period=0 2>/dev/null

    # Wait for marker in logs
    local deadline=$((SECONDS + 60))
    while [[ $SECONDS -lt $deadline ]]; do
        if kubectl logs -l app=central -n "$NAMESPACE" --tail=500 2>/dev/null | grep -q "$marker"; then
            local elapsed; elapsed=$(_elapsed_ms "$start")
            rm -f "$marker_file"
            if [[ $elapsed -lt 30000 ]]; then
                _pass "$name" "${elapsed}ms — rebuild + restart under 30s"
            else
                _fail "$name" "${elapsed}ms — over 30s target"
            fi
            return 0
        fi
        sleep 1
    done

    rm -f "$marker_file"
    _fail "$name" "marker not found in 60s"
}

test_selective_rebuild() {
    # Verifies that changing 2 binaries' source code only rebuilds those 2 binaries.
    # All others should remain byte-identical (stable ldflags + Go build cache).
    local name="test_selective_rebuild"

    local goarch; goarch="$(go env GOARCH)"
    local bindir="bin/linux_${goarch}"

    # Need a baseline build first
    if [[ ! -f "${bindir}/central" ]]; then
        _skip "$name" "no baseline binaries (run test_skaffold_build first)"
        return 0
    fi

    # Record md5 of all binaries before change
    declare -A before
    for bin in central migrator compliance kubernetes kubernetes-sensor upgrader sensor-upgrader admission-control config-controller roxagent roxctl; do
        [[ -f "${bindir}/${bin}" ]] && before[$bin]=$(md5sum "${bindir}/${bin}" | cut -d' ' -f1)
    done

    # Drop marker files into central and migrator (two separate binaries)
    local marker="SELECTIVE-$(date +%s)-$$"
    local central_marker="central/dev_selective_marker.go"
    local migrator_marker="migrator/dev_selective_marker.go"

    cat > "$central_marker" <<GOFILE
package main
import "fmt"
func init() { fmt.Print("$marker-central") }
GOFILE
    cat > "$migrator_marker" <<GOFILE
package main
import "fmt"
func init() { fmt.Print("$marker-migrator") }
GOFILE
    trap "rm -f '$central_marker' '$migrator_marker'" EXIT INT TERM

    # Rebuild
    export BUILD_TAG=0.0.0 SHORTCOMMIT=dev STABLE_COLLECTOR_VERSION=0.0.0 STABLE_FACT_VERSION=0.0.0 STABLE_SCANNER_VERSION=0.0.0
    export CGO_ENABLED=0 GOOS=linux
    make main-build-nodeps 2>&1 | tail -2

    rm -f "$central_marker" "$migrator_marker"

    # Compare md5s
    local changed=0
    local unchanged=0
    local changed_names=""
    for bin in "${!before[@]}"; do
        [[ -f "${bindir}/${bin}" ]] || continue
        local after
        after=$(md5sum "${bindir}/${bin}" | cut -d' ' -f1)
        if [[ "$after" != "${before[$bin]}" ]]; then
            changed=$((changed + 1))
            changed_names="${changed_names} ${bin}"
        else
            unchanged=$((unchanged + 1))
        fi
    done

    echo "    changed:${changed_names:-none} (${changed}), unchanged: ${unchanged}"

    if [[ $changed -eq 2 ]] && [[ "$changed_names" == *central* ]] && [[ "$changed_names" == *migrator* ]]; then
        _pass "$name" "only central + migrator changed (${changed} changed, ${unchanged} unchanged)"
    elif [[ $changed -eq 0 ]]; then
        _fail "$name" "no binaries changed — marker files not compiled?"
    else
        _fail "$name" "expected 2 changed (central, migrator), got ${changed}:${changed_names}"
    fi
}

test_iteration_e2e() {
    local name="test_iteration_e2e"
    _require_deploy "$name" || return 0
    if ! curl -s "http://localhost:${REG_PORT}/v2/" >/dev/null 2>&1; then
        _skip "$name" "no registry"; return 0
    fi

    local ready
    ready=$(kubectl -n "$NAMESPACE" get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
    [[ -n "$ready" ]] && [[ "$ready" -ge 1 ]] || { _skip "$name" "central not ready"; return 0; }

    local marker="E2E-$(date +%s)-$$"
    local marker_file="central/dev_test_marker.go"

    cat > "$marker_file" <<GOFILE
package main

import (
	"fmt"
	"os"
)

func init() {
	fmt.Fprintln(os.Stderr, "$marker")
}
GOFILE
    trap "rm -f '$marker_file'" EXIT INT TERM

    local start; start=$(_now_ms)
    local tag="e2e-$(date +%s)"

    echo "    Building..."
    if ! IMAGE="localhost:${REG_PORT}/main:${tag}" ./dev-tools/skaffold-build.sh 2>&1 | tail -3; then
        rm -f "$marker_file"; _fail "$name" "build failed"; return 1
    fi

    echo "    Deploying..."
    kubectl -n "$NAMESPACE" set image deploy/central "central=localhost:${REG_PORT}/main:${tag}" 2>/dev/null
    kubectl -n "$NAMESPACE" delete pod -l app=central --grace-period=0 2>/dev/null

    echo "    Waiting for marker..."
    local deadline=$((SECONDS + 90))
    while [[ $SECONDS -lt $deadline ]]; do
        if kubectl logs -l app=central -n "$NAMESPACE" --tail=500 2>/dev/null | grep -q "$marker"; then
            rm -f "$marker_file"
            _pass "$name" "$(_elapsed_ms "$start")ms — code change visible in pod"
            return 0
        fi
        sleep 1
    done

    rm -f "$marker_file"
    _fail "$name" "marker not found in 90s"
}

# ==========================================================================
# UI DEVELOPER
# ==========================================================================

test_ui_port_forward() {
    local name="test_ui_port_forward"
    _require_deploy "$name" || return 0

    kubectl -n "$NAMESPACE" port-forward svc/central 18443:443 >/dev/null 2>&1 &
    local pid=$!; sleep 2

    if curl -sk --connect-timeout 5 https://localhost:18443/v1/ping >/dev/null 2>&1; then
        _pass "$name" "/v1/ping OK"
    else
        _fail "$name" "cannot reach Central"
    fi
    kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
}

test_ui_dev_server() {
    local name="test_ui_dev_server"
    _require_deploy "$name" || return 0
    if [[ ! -d "ui/apps/platform/node_modules" ]]; then
        _skip "$name" "run: cd ui/apps/platform && npm ci"; return 0
    fi

    kubectl -n "$NAMESPACE" port-forward svc/central 8000:443 >/dev/null 2>&1 &
    local pf=$!; sleep 2

    (cd ui/apps/platform && NODE_ENV=development npm run start 2>&1) &
    local vite=$!

    local deadline=$((SECONDS + 60))
    while [[ $SECONDS -lt $deadline ]]; do
        curl -sk --connect-timeout 2 https://localhost:3000/ >/dev/null 2>&1 && break
        sleep 2
    done

    if curl -sk --connect-timeout 2 https://localhost:3000/ >/dev/null 2>&1; then
        _pass "$name" "Vite on :3000"
    else
        _fail "$name" "Vite did not start"
    fi

    kill "$vite" 2>/dev/null; wait "$vite" 2>/dev/null
    kill "$pf" 2>/dev/null; wait "$pf" 2>/dev/null
}

# ==========================================================================
# DEBUG MODE — Delve debugger
# ==========================================================================

test_debug_mode() {
    # Verifies that DEBUG_BUILD=yes produces a debug binary with symbols,
    # Delve can attach, and variables are inspectable.
    local name="test_debug_mode"
    _require_deploy "$name" || return 0

    local goarch; goarch="$(go env GOARCH)"

    # Build debug binary (with symbols, no strip)
    echo "    Building debug binary..."
    local start; start=$(_now_ms)
    GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 \
        go build -buildvcs=false -trimpath -gcflags="all=-N -l" \
        -ldflags="-X github.com/stackrox/rox/pkg/version/internal.MainVersion=0.0.0-debug -X github.com/stackrox/rox/pkg/version/internal.GitShortSha=debug" \
        -o "bin/linux_${goarch}/central" ./central 2>&1

    # Build dlv for the target platform
    if [[ ! -f "$HOME/.go/bin/linux_${goarch}/dlv" ]]; then
        echo "    Building Delve..."
        GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 go install github.com/go-delve/delve/cmd/dlv@latest 2>&1
    fi

    # Stage into image
    cp "bin/linux_${goarch}/central" image/rhel/bin/central
    cp "$HOME/.go/bin/linux_${goarch}/dlv" image/rhel/bin/dlv
    touch image/rhel/bin/dlv-placeholder
    chmod +x image/rhel/bin/*

    # Build + push debug image
    local tag="debug-test-$(date +%s)"
    local rt; if command -v docker >/dev/null 2>&1; then rt=docker; else rt=podman; fi

    if $rt inspect --format '{{.State.Running}}' buildkitd 2>/dev/null | grep -q true; then
        local reg="${KIND_CLUSTER_NAME}-registry:5000"
        $rt exec buildkitd buildctl build \
            --addr unix:///run/buildkit/buildkitd.sock \
            --frontend dockerfile.v0 \
            --local context=/context --local dockerfile=/context \
            --opt "build-arg:TARGET_ARCH=${goarch}" \
            --output "type=image,name=${reg}/main:${tag},push=true,registry.insecure=true" \
            2>&1 | tail -1
    else
        $rt build -t "localhost:5000/main:${tag}" \
            --build-arg "TARGET_ARCH=${goarch}" \
            --file image/rhel/Dockerfile image/rhel 2>&1 | tail -1
        if [[ "$rt" == "podman" ]]; then
            $rt push "localhost:5000/main:${tag}" --tls-verify=false 2>&1 | tail -1
        else
            $rt push "localhost:5000/main:${tag}" 2>&1 | tail -1
        fi
    fi

    # Deploy Central under Delve — use a unique annotation to force a rollout
    echo "    Deploying Central under Delve..."
    kubectl -n "$NAMESPACE" patch deploy/central --type=strategic -p "
spec:
  template:
    metadata:
      annotations:
        debug-test: '${tag}'
    spec:
      securityContext:
        runAsUser: 0
        fsGroup: 0
      containers:
      - name: central
        image: localhost:5000/main:${tag}
        securityContext:
          capabilities:
            add: [SYS_PTRACE]
        command: ['/bin/sh', '-c']
        args: ['restore-all-dir-contents; import-additional-cas; exec /stackrox/bin/dlv exec --headless --continue --accept-multiclient --listen=:56268 --api-version=2 -- /stackrox/central']
        readinessProbe:
          httpGet:
            scheme: HTTPS
            path: /v1/ping
            port: 8443
          timeoutSeconds: 30
          periodSeconds: 10
" 2>/dev/null

    # Wait for pod Running + Delve listening
    echo "    Waiting for pod + Delve..."
    sleep 15
    kubectl --context "kind-${KIND_CLUSTER_NAME}" -n "$NAMESPACE" port-forward deploy/central 56268:56268 >/dev/null 2>&1 &
    local pf_pid=$!
    sleep 5

    local dlv_listening=false
    local deadline=$((SECONDS + 120))
    while [[ $SECONDS -lt $deadline ]]; do
        if echo 'goroutines' | timeout 5 dlv connect localhost:56268 2>&1 | grep -q 'Goroutine'; then
            dlv_listening=true
            break
        fi
        # Port-forward may need restart
        kill "$pf_pid" 2>/dev/null; wait "$pf_pid" 2>/dev/null
        kubectl --context "kind-${KIND_CLUSTER_NAME}" -n "$NAMESPACE" port-forward deploy/central 56268:56268 >/dev/null 2>&1 &
        pf_pid=$!
        sleep 5
    done

    local dlv_output=""
    if $dlv_listening; then
        sleep 3
        dlv_output=$(echo 'vars -v github.com/stackrox/rox/pkg/version/internal
exit' | timeout 10 dlv connect localhost:56268 2>&1)
    fi

    kill "$pf_pid" 2>/dev/null; wait "$pf_pid" 2>/dev/null

    # Revert Central to normal mode
    kubectl -n "$NAMESPACE" patch deploy/central --type=strategic -p "
spec:
  template:
    metadata:
      annotations:
        debug-test: 'reverted-${tag}'
    spec:
      securityContext:
        runAsUser: 4000
        fsGroup: 4000
      containers:
      - name: central
        command: ['/stackrox/central-entrypoint.sh']
        args: null
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: [ALL]
        readinessProbe:
          httpGet:
            scheme: HTTPS
            path: /v1/ping
            port: 8443
          timeoutSeconds: 1
          periodSeconds: 10
" 2>/dev/null

    # Report result
    if ! $dlv_listening; then
        _fail "$name" "Delve API server did not start"
    elif echo "$dlv_output" | grep -q 'MainVersion'; then
        _pass "$name" "$(_elapsed_ms "$start")ms — Delve connected, variables inspectable"
    else
        _fail "$name" "Delve connected but could not inspect variables"
    fi
}

# ==========================================================================
# TEARDOWN — uses kind-dev.sh teardown
# ==========================================================================

test_teardown() {
    _stop_kind_dev
    local name="test_teardown"
    ./dev-tools/kind-dev.sh teardown 2>&1
    if ! kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
        _pass "$name" "cleaned up"
    else
        _fail "$name" "cluster still exists"
    fi
}

# ==========================================================================
# RUNNER
# ==========================================================================

ALL_TESTS=(
    test_setup
    test_local_registry
    test_resources
    test_memory
    test_skaffold_build
    test_skaffold_rebuild_cached
    test_selective_rebuild
    test_rebuild_under_30s
    test_iteration_e2e
    test_ui_port_forward
    test_ui_dev_server
    test_debug_mode
    test_teardown
)

print_summary() {
    echo ""
    echo "=== Results: $PASS passed, $FAIL failed, $SKIP skipped ==="
    if [[ ${#ERRORS[@]} -gt 0 ]]; then
        echo "Failures:"
        for err in "${ERRORS[@]}"; do echo "  - $err"; done
    fi
    echo ""
}

trap _stop_kind_dev EXIT

if [[ "${1:-}" == "teardown" ]]; then test_teardown; exit 0; fi
if [[ -n "${1:-}" ]]; then
    declare -f "$1" >/dev/null 2>&1 || { echo "Unknown: $1"; echo "Available: ${ALL_TESTS[*]}"; exit 1; }
    "$1"; print_summary; exit $FAIL
fi

echo "=== KinD Local Dev Tests ==="
echo ""
for t in "${ALL_TESTS[@]}"; do "$t" || true; done
_stop_kind_dev
print_summary
exit $FAIL
