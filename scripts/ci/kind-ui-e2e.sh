#!/usr/bin/env bash
# Experiment: Run UI e2e (Cypress) tests against a minimal StackRox deployment on KinD.
# Designed to validate feasibility on a 4-core / 16GB GHA runner.
#
# Usage:
#   ./scripts/ci/kind-ui-e2e.sh                          # full run: build, deploy, test
#   ./scripts/ci/kind-ui-e2e.sh --skip-build             # reuse existing images
#   ./scripts/ci/kind-ui-e2e.sh --skip-deploy            # reuse existing deployment (implies --skip-build)
#   ./scripts/ci/kind-ui-e2e.sh --pull-images <tag>      # pull pre-built images from quay.io instead of building
#   ./scripts/ci/kind-ui-e2e.sh --cleanup                # delete the KinD cluster and exit
#
# Environment variables:
#   IMAGE_REGISTRY    - registry to pull from (default: quay.io/rhacs-eng)
#   KIND_CLUSTER_NAME - KinD cluster name (default: kind-ui-e2e)
#   KIND_MEMORY_LIMIT - hard memory cap for KinD container (default: 6g)
#   MEMORY_WARN_MB    - log warning when KinD exceeds this (default: 4096)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind-ui-e2e}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-quay.io/rhacs-eng}"
KIND_MEMORY_LIMIT="${KIND_MEMORY_LIMIT:-6g}"
MEMORY_WARN_MB="${MEMORY_WARN_MB:-4096}"
SKIP_BUILD="${SKIP_BUILD:-false}"
SKIP_DEPLOY="${SKIP_DEPLOY:-false}"
PULL_TAG=""
MONITOR_LOG="/tmp/kind-ui-e2e-resources.log"
MONITOR_PID=""
PF_PID=""

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --skip-build)  SKIP_BUILD=true ;;
        --skip-deploy) SKIP_DEPLOY=true; SKIP_BUILD=true ;;
        --pull-images) SKIP_BUILD=true ;;  # tag follows as next arg
        --cleanup)     kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true; echo "Cleaned up."; exit 0 ;;
        *)
            if [[ -z "$PULL_TAG" && "$arg" != --* ]]; then
                PULL_TAG="$arg"
            else
                echo "Unknown argument: $arg"; exit 1
            fi
            ;;
    esac
done

info() { echo "==> $(date +%H:%M:%S) $*"; }
die()  { echo "FATAL: $*" >&2; exit 1; }

cleanup() {
    kill "$MONITOR_PID" 2>/dev/null || true
    kill "$PF_PID" 2>/dev/null || true
}
trap cleanup EXIT

########################################
# Step 1: Preflight checks
########################################
info "Preflight checks"
command -v kind    >/dev/null || die "kind not found — install from https://kind.sigs.k8s.io"
command -v kubectl >/dev/null || die "kubectl not found"
command -v docker  >/dev/null || die "docker not found"
docker info >/dev/null 2>&1  || die "Docker daemon not running (start colima?)"

if [[ -n "$PULL_TAG" ]]; then
    TAG="$PULL_TAG"
else
    TAG="$(make --quiet --no-print-directory -C "$ROOT" tag)"
fi
info "Image tag: $TAG"

MAIN_IMG="stackrox/main:${TAG}"
CENTRAL_DB_IMG="stackrox/central-db:${TAG}"

########################################
# Step 2: Get images (build or pull)
########################################
if [[ -n "$PULL_TAG" ]]; then
    info "Pulling pre-built images from ${IMAGE_REGISTRY}..."
    docker pull "${IMAGE_REGISTRY}/main:${TAG}"
    docker pull "${IMAGE_REGISTRY}/central-db:${TAG}"
    docker tag "${IMAGE_REGISTRY}/main:${TAG}" "$MAIN_IMG"
    docker tag "${IMAGE_REGISTRY}/central-db:${TAG}" "$CENTRAL_DB_IMG"
    info "Images pulled and tagged."
elif [[ "$SKIP_BUILD" == "false" ]]; then
    info "Building images (this takes a while on first run)..."
    make -C "$ROOT" image
else
    info "Skipping image build (--skip-build)"
fi

docker image inspect "$MAIN_IMG" >/dev/null 2>&1 \
    || die "Image ${MAIN_IMG} not found. Use --pull-images <tag> or run without --skip-build."
docker image inspect "$CENTRAL_DB_IMG" >/dev/null 2>&1 \
    || die "Image ${CENTRAL_DB_IMG} not found. Use --pull-images <tag> or run without --skip-build."

########################################
# Step 3: Create KinD cluster
########################################
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    info "KinD cluster '$CLUSTER_NAME' already exists"
else
    info "Creating KinD cluster '$CLUSTER_NAME'..."
    kind create cluster --name "$CLUSTER_NAME" --wait 60s
fi

kubectl cluster-info --context "kind-${CLUSTER_NAME}" >/dev/null \
    || die "Cannot connect to KinD cluster"

# Set a hard memory cap on the KinD container to protect the host/runner.
# On Linux, this sets a cgroup limit — OOM kills happen inside the container,
# not on the host. KinD sets no limit by default (docker inspect shows 0).
info "Setting KinD container memory limit to ${KIND_MEMORY_LIMIT}"
docker update --memory "${KIND_MEMORY_LIMIT}" --memory-swap "${KIND_MEMORY_LIMIT}" \
    "${CLUSTER_NAME}-control-plane" >/dev/null

# Install metrics-server for kubectl top (per-pod CPU/memory metrics).
# KinD doesn't include it by default.
if ! kubectl -n kube-system get deployment metrics-server >/dev/null 2>&1; then
    info "Installing metrics-server..."
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml \
        >/dev/null 2>&1
    # KinD uses self-signed certs — metrics-server needs --kubelet-insecure-tls
    kubectl -n kube-system patch deployment metrics-server --type=json \
        -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]' \
        >/dev/null 2>&1
    kubectl -n kube-system rollout status deployment/metrics-server --timeout=60s >/dev/null 2>&1 || true
fi

########################################
# Step 4: Load images into KinD
########################################
if [[ "$SKIP_DEPLOY" == "false" ]]; then
    info "Loading images into KinD..."
    # Use ctr pull inside KinD node instead of "kind load docker-image" to avoid
    # multi-arch digest issues (same approach as sensor-integration-tests in unit-tests.yaml).
    NODE="${CLUSTER_NAME}-control-plane"
    for img in "$MAIN_IMG" "$CENTRAL_DB_IMG"; do
        if docker exec "$NODE" crictl images -o json 2>/dev/null | grep -q "$(echo "$img" | cut -d: -f1)"; then
            info "  $img already loaded, skipping"
        else
            info "  Loading $img..."
            docker save "$img" | docker exec -i "$NODE" ctr --namespace=k8s.io images import -
        fi
    done
    info "Images loaded."
fi

########################################
# Step 5: Deploy StackRox (minimal)
########################################
if [[ "$SKIP_DEPLOY" == "false" ]]; then
    info "Deploying StackRox with minimal resources..."

    # Minimal deployment: no scanner, no collector, no admission controller.
    # LOCAL_DEPLOYMENT=true triggers deploy/common/local-dev-values.yaml which sets:
    #   central:    500m CPU / 1Gi memory (4Gi limit)
    #   central-db: 500m CPU / 1Gi memory (4Gi limit)
    #   sensor:     500m CPU / 500Mi memory (patched by sensor-local-patch.yaml)
    # Actual measured usage: ~790 MiB total across all pods (2026-05-01).
    export LOCAL_DEPLOYMENT=true
    export SCANNER_SUPPORT=false
    export ROX_SCANNER_V4=false
    export SENSOR_SCANNER_SUPPORT=false
    export SENSOR_SCANNER_V4_SUPPORT=false
    export ADMISSION_CONTROLLER=false
    export ADMISSION_CONTROLLER_UPDATES=false
    export COLLECTION_METHOD=NO_COLLECTION
    export LOAD_BALANCER=none
    export STORAGE=none
    export MONITORING_SUPPORT=false
    export OUTPUT_FORMAT=helm
    export ORCHESTRATOR_FLAVOR=k8s
    export POD_SECURITY_POLICIES=false

    export MAIN_IMAGE_TAG="$TAG"
    export MAIN_IMAGE="$MAIN_IMG"
    export CENTRAL_DB_IMAGE="$CENTRAL_DB_IMG"

    # roxctl is needed by deploy scripts to generate configs.
    if [[ -z "$PULL_TAG" ]]; then
        if ! command -v roxctl >/dev/null 2>&1 || [[ "$(roxctl version 2>/dev/null)" != "$TAG" ]]; then
            info "Building roxctl..."
            make -C "$ROOT" cli-build
            export PATH="${ROOT}/bin:${PATH}"
        fi
    else
        info "Using docker-based roxctl from main image"
        export ROXCTL_IMAGE_REPO="stackrox/main"
    fi

    "$ROOT/deploy/k8s/deploy-local.sh"

    info "Waiting for Central rollout..."
    kubectl -n stackrox rollout status deploy/central --timeout=5m

    info "Waiting for Sensor rollout..."
    kubectl -n stackrox rollout status deploy/sensor --timeout=3m || true

    # Scale down admission-control and collector — sensor manifest deploy
    # ignores ADMISSION_CONTROLLER=false and COLLECTION_METHOD=NO_COLLECTION.
    info "Scaling down unnecessary components..."
    kubectl -n stackrox scale deploy/admission-control --replicas=0 2>/dev/null || true
    kubectl -n stackrox delete daemonset/collector 2>/dev/null || true
else
    info "Skipping deployment (--skip-deploy)"
fi

########################################
# Step 6: Port-forward Central
########################################
pkill -f "kubectl.*port-forward.*central" 2>/dev/null || true
sleep 1

info "Port-forwarding Central to localhost:8000..."
kubectl -n stackrox port-forward svc/central 8000:443 &
PF_PID=$!
sleep 3

# Wait for Central API
info "Waiting for Central API..."
for i in $(seq 1 60); do
    if curl -sk "https://localhost:8000/v1/metadata" >/dev/null 2>&1; then
        break
    fi
    sleep 5
    echo -n "."
done
echo

curl -sk "https://localhost:8000/v1/metadata" | jq . || die "Central API not reachable"
info "Central is up!"

########################################
# Step 7: Resource monitor
########################################
# Background monitor that:
# - Logs per-pod CPU/memory (kubectl top) and docker-level memory every 15s
# - Warns if KinD container memory exceeds MEMORY_WARN_MB
# - Detects OOM kills and pod restarts
info "Starting resource monitor -> $MONITOR_LOG (warn at ${MEMORY_WARN_MB}MB)"
NODE="${CLUSTER_NAME}-control-plane"
(
    echo "=== Resource monitoring started at $(date) ==="
    echo "Memory limit: ${KIND_MEMORY_LIMIT}, warn threshold: ${MEMORY_WARN_MB}MB"
    PEAK_MEM_MB=0

    while true; do
        ts="$(date +%H:%M:%S)"
        echo "--- ${ts} ---"

        # Per-pod metrics (requires metrics-server, may take ~30s to become available)
        echo "## kubectl top pods"
        kubectl -n stackrox top pods --containers 2>/dev/null || echo "(waiting for metrics-server)"

        # Docker-level memory for the KinD container
        mem_raw=$(docker stats --no-stream --format "{{.MemUsage}}" "$NODE" 2>/dev/null || echo "0MiB / 0MiB")
        echo "## docker: ${mem_raw}"

        # Parse current memory in MiB and track peak
        mem_val=$(echo "$mem_raw" | grep -oE '^[0-9.]+' || echo 0)
        mem_unit=$(echo "$mem_raw" | grep -oE '(MiB|GiB)' | head -1 || echo "MiB")
        if [[ "$mem_unit" == "GiB" ]]; then
            mem_mb=$(awk "BEGIN{printf \"%d\", $mem_val * 1024}")
        else
            mem_mb="${mem_val%.*}"
        fi
        if [[ "$mem_mb" -gt "$PEAK_MEM_MB" ]]; then
            PEAK_MEM_MB="$mem_mb"
            echo "## NEW PEAK: ${PEAK_MEM_MB}MB"
        fi
        if [[ "$mem_mb" -gt "$MEMORY_WARN_MB" ]]; then
            echo "## WARNING: KinD memory ${mem_mb}MB exceeds threshold ${MEMORY_WARN_MB}MB"
        fi

        # Check for OOM kills and restarts
        restarts=$(kubectl -n stackrox get pods -o jsonpath='{range .items[*]}{.metadata.name}{" restarts="}{range .status.containerStatuses[*]}{.restartCount}{" "}{end}{"\n"}{end}' 2>/dev/null)
        oom=$(kubectl -n stackrox get pods -o jsonpath='{range .items[*]}{range .status.containerStatuses[*]}{.lastState.terminated.reason}{end}{end}' 2>/dev/null)
        if echo "$restarts" | grep -qv "restarts=0 $\|restarts=$"; then
            echo "## RESTARTS DETECTED:"
            echo "$restarts" | grep -v "restarts=0 $"
        fi
        if echo "$oom" | grep -qi "oom"; then
            echo "## OOM KILL DETECTED"
        fi

        echo
        sleep 15
    done
) > "$MONITOR_LOG" 2>&1 &
MONITOR_PID=$!

########################################
# Step 8: Run Cypress smoke test
########################################
info "Reading admin credentials..."
ROX_ADMIN_PASSWORD="$(cat "$ROOT/deploy/k8s/central-deploy/password" 2>/dev/null)" \
    || die "Cannot read admin password from deploy/k8s/central-deploy/password"
export ROX_USERNAME="admin"
export ROX_ADMIN_PASSWORD
export UI_BASE_URL="https://localhost:8000"
export CYPRESS_BASE_URL="https://localhost:8000"

info "Running Cypress release smoke test..."
cd "$ROOT/ui/apps/platform"

if [[ ! -d node_modules/.cache/Cypress ]]; then
    info "Installing Cypress binary..."
    npx cypress install
fi

export PATH="$ROOT/ui/apps/platform/node_modules/.bin:$PATH"
export CYPRESS_ROX_AUTH_TOKEN=$(./scripts/get-auth-token.sh 2>/dev/null || echo "")

set +e
TZ=UTC cypress run \
    --config "specPattern=cypress/integration/**/*.test.{js,ts}" \
    --spec "cypress/integration/release-smoke-test.test.js"
TEST_EXIT=$?
set -e

########################################
# Step 9: Report results
########################################
cd "$ROOT"

info "=== Final Resource Report ==="

echo ""
echo "## Per-pod metrics"
kubectl -n stackrox top pods --containers 2>/dev/null || echo "(metrics-server not available)"

echo ""
echo "## Docker container memory"
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" 2>/dev/null || true

echo ""
echo "## Pod status (restarts)"
kubectl -n stackrox get pods -o custom-columns='NAME:.metadata.name,STATUS:.status.phase,RESTARTS:.status.containerStatuses[0].restartCount'

echo ""
echo "## Peak memory from monitor log"
grep "NEW PEAK\|WARNING\|OOM\|RESTARTS DETECTED" "$MONITOR_LOG" 2>/dev/null || echo "(no alerts)"

echo ""
echo "## Resource requests vs actual"
echo "Configured requests: central 1Gi, central-db 1Gi, sensor 500Mi"
echo "Configured limits:   central 4Gi, central-db 4Gi, sensor 8Gi"
echo "KinD container limit: ${KIND_MEMORY_LIMIT}"

if [[ "$TEST_EXIT" -eq 0 ]]; then
    info "SUCCESS: Cypress smoke tests passed on KinD!"
else
    info "FAILED: Cypress tests exited with code $TEST_EXIT"
    info "Check Cypress artifacts in ui/apps/platform/cypress/test-results/"
fi

info "Full resource log: $MONITOR_LOG"
exit "$TEST_EXIT"
