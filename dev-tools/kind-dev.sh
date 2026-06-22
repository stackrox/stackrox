#!/usr/bin/env bash
#
# StackRox local development on KinD.
#
# One command to run the full StackRox stack locally with automatic rebuild
# on Go source changes. Designed for daily development — run it once in a
# terminal and leave it running all day.
#
# == Quick Start ==
#
#   ./dev-tools/kind-dev.sh              # start (or resume) dev loop
#   ./dev-tools/kind-dev.sh teardown     # delete cluster + registry
#
# First run (~5 min): creates KinD cluster, local registry, pulls base images,
# generates Helm charts, builds your code, deploys full stack, deploys Sensor.
# Subsequent runs: skips setup, goes straight to Skaffold watch loop.
#
# == What Gets Deployed ==
#
# Full stack in the "stackrox" namespace:
#   Central, Central-DB, Scanner (V2), Scanner-DB, Scanner V4 (indexer + matcher + DB),
#   Sensor, Collector (CORE_BPF), Admission-controller, Config-controller
# Total memory: ~3.3 GB
#
# == Go Developer Workflow ==
#
# With this script running, edit any .go file in central/, sensor/, pkg/, etc.
# Skaffold detects the change, cross-compiles all binaries, builds a container
# image (only changed binary layers rebuild via COPY --link), pushes to the
# local registry, and Helm-upgrades the deployment. ~20s from save to running.
#
#   Terminal 1:  ./dev-tools/kind-dev.sh
#   Terminal 2:  kubectl logs -f -l app=central -n stackrox --tail=100
#   Editor:      edit central/main.go, save → see rebuild in Terminal 1,
#                new log output in Terminal 2
#
# == UI Developer Workflow ==
#
# The UI dev server runs on your host (not in the cluster). Central in KinD
# serves as the API backend.
#
#   Terminal 1:  ./dev-tools/kind-dev.sh
#   Terminal 2:  kubectl -n stackrox port-forward svc/central 8000:443
#   Terminal 3:  cd ui/apps/platform && npm run start
#   Browser:     https://localhost:3000  (Vite HMR — sub-second updates)
#
# == Running E2E Tests ==
#
# Groovy (qa-tests-backend):
#   cd qa-tests-backend
#   JAVA_HOME=/path/to/jdk21 ROX_ADMIN_PASSWORD=$(cat deploy/k8s/central-deploy/password) \
#     API_HOSTNAME=localhost API_PORT=8443 CLUSTER=K8S \
#     POD_SECURITY_POLICIES=false REMOTE_CLUSTER_ARCH=aarch64 \
#     ./gradlew test --tests="PolicyConfigurationTest"
#
# Go e2e:
#   ROX_ADMIN_PASSWORD=$(cat deploy/k8s/central-deploy/password) \
#     API_ENDPOINT=localhost:8443 \
#     go test -v -tags test_e2e -run TestPing -count=1 ./tests/
#
# Cypress UI:
#   cd ui/apps/platform
#   ROX_ADMIN_PASSWORD=$(cat deploy/k8s/central-deploy/password) \
#     npm run cypress-spec -- "configmanagement/dashboard.test.js"
#
# Note: Groovy tests need a port-forward on 8443:
#   kubectl -n stackrox port-forward svc/central 8443:443
#
# Note: Cypress tests need the Vite dev server on :3000 (see UI workflow above)
#
# Note: Test images from quay.io/rhacs-eng need pull secrets. The script creates
# them in the "qa" namespace if ~/.config/containers/auth.json has quay.io creds.
#
# == Debugging Go Code ==
#
# Run in debug mode to attach a debugger (Delve) to Central:
#
#   DEBUG_BUILD=yes ./dev-tools/kind-dev.sh
#
# This builds with debug symbols (-gcflags="all=-N -l", no -s -w strip),
# installs Delve in the image, and runs `skaffold debug` which injects a
# Delve debug server on port 56268.
#
# IDE setup (VS Code):
#   Add to .vscode/launch.json:
#   {
#     "name": "Attach to Central (KinD)",
#     "type": "go",
#     "request": "attach",
#     "mode": "remote",
#     "port": 56268,
#     "host": "localhost",
#     "substitutePath": [
#       { "from": "${workspaceFolder}", "to": "/src" }
#     ]
#   }
#
# IDE setup (GoLand):
#   Run → Edit Configurations → + → Go Remote → Host: localhost, Port: 56268
#
# Note: debug builds are ~40% larger and ~10s slower to compile (no optimization).
# Auto-rebuild is disabled in debug mode to prevent tearing down debug sessions.
# Trigger manual rebuilds with `skaffold build` in another terminal.
#
# == Ctrl+C Behavior ==
#
# Pressing Ctrl+C stops Skaffold but leaves the cluster and deployment running
# (--cleanup=false). Re-run this script to resume watching. The cluster persists
# until you run: ./dev-tools/kind-dev.sh teardown
#
# == Environment Variables ==
#
#   KIND_CLUSTER_NAME    Cluster name (default: stackrox-dev)
#   KIND_REGISTRY_PORT   Local registry port (default: 5000)
#   MAIN_IMAGE_TAG       Base image tag (default: latest release git tag)
#   BUILDKITD_CONTAINER  BuildKit container name (default: buildkitd)
#
# == Prerequisites ==
#
#   kind, kubectl, helm, skaffold, roxctl, podman or docker
#   Optional: buildkitd container for fast image builds (~1s vs ~18s)
#
# == Troubleshooting ==
#
#   Disk pressure / pod evictions:
#     The full stack uses ~86% of KinD's 100GB disk. The kubelet threshold
#     is set to 1% to avoid false evictions. If pods get evicted, run:
#       kubectl taint node stackrox-dev-control-plane node.kubernetes.io/disk-pressure-
#
#   Sensor not connecting after Central restart:
#     Sensor reconnects within 10s (backoff: 1s initial, 10s max). If it's
#     stuck, restart the sensor pod:
#       kubectl -n stackrox delete pod -l app=sensor --grace-period=0
#
#   Version panic ("failed to parse main version"):
#     The build uses stable ldflags (0.0.0-dev). If you see this, the image
#     wasn't built by skaffold-build.sh. Re-run this script to rebuild.
#

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

CLUSTER_NAME="${KIND_CLUSTER_NAME:-stackrox-dev}"
NODE_NAME="${CLUSTER_NAME}-control-plane"
REG_NAME="${CLUSTER_NAME}-registry"
REG_PORT="${KIND_REGISTRY_PORT:-5000}"

_rt() {
    if command -v docker >/dev/null 2>&1; then docker "$@"; else podman "$@"; fi
}

for arg in "$@"; do
    case "$arg" in
        teardown|delete|destroy)
            echo "Deleting cluster + registry..."
            kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
            _rt rm -f "$REG_NAME" 2>/dev/null || true
            exit 0
            ;;
        -h|--help)
            sed -n '2,/^[^#]/{ /^#/s/^# \?//p }' "$0"
            exit 0
            ;;
    esac
done

# --- Ensure local registry ---
if ! _rt inspect "$REG_NAME" >/dev/null 2>&1; then
    echo "=== Starting local registry (localhost:${REG_PORT}) ==="
    _rt run -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" \
        --name "$REG_NAME" docker.io/library/registry:2
fi

# --- Ensure KinD cluster ---
if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "=== Creating KinD cluster '$CLUSTER_NAME' ==="
    mkdir -p bin
    kind_config="$(mktemp)"
    sed "s|REPO_ROOT|${REPO_ROOT}|g" dev-tools/kind-config.yaml > "$kind_config"
    kind create cluster --name "$CLUSTER_NAME" --config "$kind_config"
    rm -f "$kind_config"

    _rt network connect "kind" "$REG_NAME" 2>/dev/null || true
    _rt exec "$NODE_NAME" mkdir -p "/etc/containerd/certs.d/localhost:${REG_PORT}"
    _rt exec "$NODE_NAME" bash -c \
        "echo '[host.\"http://${REG_NAME}:5000\"]' > /etc/containerd/certs.d/localhost:${REG_PORT}/hosts.toml"

    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REG_PORT}"
EOF
fi

# --- Ensure base images are available in KinD node ---
MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(git tag --sort=-version:refname | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | head -1)}"
COLLECTOR_TAG="${COLLECTOR_IMAGE_TAG:-$(make --quiet --no-print-directory collector-tag 2>/dev/null)}"
REGISTRY="${DEFAULT_IMAGE_REGISTRY:-$(make --quiet --no-print-directory default-image-registry)}"
goarch="$(go env GOARCH)"

pull_if_missing() {
    local img="$1"
    if ! _rt exec "$NODE_NAME" crictl img --no-trunc 2>/dev/null | grep -q "$img"; then
        echo "  Pulling $img..."
        _rt exec "$NODE_NAME" ctr -n k8s.io images pull --platform "linux/${goarch}" "$img" 2>&1 | tail -1 &
    fi
}

echo "=== Pulling base images ==="
pull_if_missing "${REGISTRY}/central-db:${MAIN_IMAGE_TAG}"
pull_if_missing "${REGISTRY}/scanner:${MAIN_IMAGE_TAG}"
pull_if_missing "${REGISTRY}/scanner-db:${MAIN_IMAGE_TAG}"
pull_if_missing "${REGISTRY}/scanner-v4:${MAIN_IMAGE_TAG}"
pull_if_missing "${REGISTRY}/scanner-v4-db:${MAIN_IMAGE_TAG}"
pull_if_missing "${REGISTRY}/collector:${COLLECTOR_TAG}"
wait

# --- Ensure Helm chart is generated ---
if [[ ! -f deploy/k8s/central-deploy/chart/Chart.yaml ]]; then
    echo "=== Generating Helm chart ==="
    cd deploy/k8s
    USE_LOCAL_ROXCTL=true roxctl central generate k8s pvc \
        --output-format=helm \
        --main-image="${REGISTRY}/main:${MAIN_IMAGE_TAG}" \
        --central-db-image="${REGISTRY}/central-db:${MAIN_IMAGE_TAG}" \
        --output-dir=central-deploy none 2>&1 | tail -3
    cd "$REPO_ROOT"
fi

# --- Ensure secured-cluster chart exists ---
if [[ ! -f deploy/k8s/sensor-chart/Chart.yaml ]]; then
    echo "=== Generating secured-cluster-services chart ==="
    USE_LOCAL_ROXCTL=true roxctl helm output secured-cluster-services \
        --output-dir deploy/k8s/sensor-chart 2>&1 | tail -1
fi


# --- Initial deploy if Central isn't running ---
if ! kubectl --context "kind-${CLUSTER_NAME}" -n stackrox get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null | grep -q "1"; then
    echo "=== Initial deploy (skaffold run) ==="
    skaffold run -f dev-tools/skaffold.yaml --kube-context "kind-${CLUSTER_NAME}" 2>&1 | tail -5

    # Port-forward for init-bundle generation (skaffold run's forward dies on exit)
    kubectl -n stackrox port-forward svc/central 8000:443 >/dev/null 2>&1 &
    PF_PID=$!

    echo "=== Waiting for Central API ==="
    for i in $(seq 1 120); do
        curl -sk --connect-timeout 1 https://localhost:8000/v1/metadata >/dev/null 2>&1 && break
        sleep 1
    done
fi

# --- Deploy Sensor if not already running ---
if ! kubectl -n stackrox get deploy/sensor >/dev/null 2>&1; then
    echo "=== Deploying Sensor ==="

    # Ensure port-forward exists for Central API access
    if ! curl -sk --connect-timeout 1 https://localhost:8000/v1/ping >/dev/null 2>&1; then
        kubectl -n stackrox port-forward svc/central 8000:443 >/dev/null 2>&1 &
        PF_PID=$!
        for i in $(seq 1 30); do
            curl -sk --connect-timeout 1 https://localhost:8000/v1/ping >/dev/null 2>&1 && break
            sleep 1
        done
    fi

    ROX_ADMIN_PASSWORD="$(cat deploy/k8s/central-deploy/password)"

    # Generate init-bundle via Central API
    init_response=$(curl -sk -u "admin:${ROX_ADMIN_PASSWORD}" \
        https://localhost:8000/v1/cluster-init/init-bundles \
        -X POST -d "{\"name\":\"kind-dev-$(date +%s)\"}")

    echo "$init_response" | python3 -c "
import json, sys, base64
resp = json.load(sys.stdin)
bundle = base64.b64decode(resp['helmValuesBundle']).decode()
with open('/tmp/sensor-init-bundle.yaml', 'w') as f:
    f.write(bundle)
print('Init bundle generated')
"

    # Add quay.io registry integration with credentials (if available) so Scanner
    # can pull and scan images from quay.io/stackrox-io and quay.io/rhacs-eng.
    if [[ -f "$HOME/.config/containers/auth.json" ]]; then
        QUAY_AUTH=$(python3 -c "import json; d=json.load(open('$HOME/.config/containers/auth.json')); print(d.get('auths',{}).get('quay.io',{}).get('auth',''))" 2>/dev/null | base64 -d 2>/dev/null)
        if [[ -n "$QUAY_AUTH" ]]; then
            QUAY_USER="${QUAY_AUTH%%:*}"
            QUAY_PASS="${QUAY_AUTH#*:}"
            curl -sk -u "admin:${ROX_ADMIN_PASSWORD}" -X POST https://localhost:8000/v1/imageintegrations \
                -d "{\"name\":\"StackRox Quay.io\",\"type\":\"docker\",\"categories\":[\"REGISTRY\"],\"docker\":{\"endpoint\":\"quay.io\",\"username\":\"${QUAY_USER}\",\"password\":\"${QUAY_PASS}\"},\"skipTestIntegration\":true}" >/dev/null 2>&1
            echo "Added quay.io registry integration for scanning"
        fi
    fi

    # Use dev image from local registry if available, else release image from quay.io
    SENSOR_IMAGE="${REGISTRY}/main:${MAIN_IMAGE_TAG}"
    if curl -s "http://localhost:${REG_PORT}/v2/main/tags/list" 2>/dev/null | grep -q '"tags"'; then
        DEV_TAG=$(curl -s "http://localhost:${REG_PORT}/v2/main/tags/list" 2>/dev/null | python3 -c "import json,sys; tags=json.load(sys.stdin).get('tags',[]); print(tags[0] if tags else '')" 2>/dev/null)
        if [[ -n "$DEV_TAG" ]]; then
            SENSOR_IMAGE="localhost:${REG_PORT}/main:${DEV_TAG}"
            echo "Using dev image for sensor: ${SENSOR_IMAGE}"
        fi
    fi

    # Pull the sensor image into KinD node if not using local registry
    if [[ "$SENSOR_IMAGE" == "${REGISTRY}"* ]]; then
        goarch="$(go env GOARCH)"
        _rt exec "$NODE_NAME" ctr -n k8s.io images pull --platform "linux/${goarch}" \
            "$SENSOR_IMAGE" 2>&1 | tail -1
    fi

    helm --kube-context "kind-${CLUSTER_NAME}" upgrade --install -n stackrox \
        stackrox-secured-cluster-services deploy/k8s/sensor-chart \
        --set clusterName=remote \
        --set centralEndpoint=central.stackrox.svc:443 \
        --set image.main.fullRef="${SENSOR_IMAGE}" \
        --set imagePullSecrets.allowNone=true \
        --set image.collector.fullRef="${REGISTRY}/collector:${COLLECTOR_TAG}" \
        --set customize.envVars.ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL=1s \
        --set customize.envVars.ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL=10s \
        --set collector.collectionMethod=CORE_BPF \
        --set admissionControl.replicas=1 \
        --set sensor.resources.requests.memory=100Mi \
        --set sensor.resources.requests.cpu=100m \
        --set sensor.resources.limits.memory=1Gi \
        --set collector.resources.requests.memory=100Mi \
        --set collector.resources.requests.cpu=50m \
        --set collector.resources.limits.memory=512Mi \
        --set admissionControl.resources.requests.memory=100Mi \
        --set admissionControl.resources.requests.cpu=50m \
        --set managedBy=MANAGER_TYPE_MANUAL \
        -f /tmp/sensor-init-bundle.yaml 2>&1 | tail -3

    echo "Sensor deployed"
fi

# Clean up any port-forward we started (skaffold dev manages its own)
[[ -n "${PF_PID:-}" ]] && kill "$PF_PID" 2>/dev/null && wait "$PF_PID" 2>/dev/null || true

# --- Run Skaffold dev loop ---
echo ""
echo "=== Starting Skaffold dev loop ==="
echo "  Central UI will be at https://localhost:8000 (port-forwarded)"
echo "  Password: $(cat deploy/k8s/central-deploy/password 2>/dev/null || echo '(see deploy/k8s/central-deploy/password)')"
echo "  Press Ctrl+C to stop (cluster stays running for next time)"
echo ""

if [[ "${DEBUG_BUILD:-}" == "yes" ]]; then
    echo "  DEBUG MODE: Delve debugger will be injected. Attach IDE to localhost:56268"
    echo "  Ctrl+C will clean up the debug deployment so normal mode works on next run."
    echo ""
    exec skaffold debug -f dev-tools/skaffold.yaml --cleanup=true --kube-context "kind-${CLUSTER_NAME}" --port-forward
else
    exec skaffold dev -f dev-tools/skaffold.yaml --cleanup=false --kube-context "kind-${CLUSTER_NAME}"
fi
