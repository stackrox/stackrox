#!/usr/bin/env bash
#
# StackRox local development on KinD.
#
# Run this once, leave it running. It sets up the infrastructure (if needed)
# and starts Skaffold in dev mode — watching Go source files, auto-rebuilding,
# and redeploying to the local cluster.
#
# Usage:
#   ./dev-tools/kind-dev.sh              # start (or resume) dev loop
#   ./dev-tools/kind-dev.sh teardown     # delete cluster + registry
#
# First run creates the KinD cluster, local registry, and generates the Helm
# chart (~2 min). Subsequent runs skip straight to Skaffold.
#
# Prerequisites: kind, kubectl, helm, skaffold, roxctl, podman or docker

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
            head -17 "$0" | tail -15 | sed 's/^# \?//'
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

# --- Ensure central-db base image is available ---
MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(git tag --sort=-version:refname | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | head -1)}"
REGISTRY="${DEFAULT_IMAGE_REGISTRY:-$(make --quiet --no-print-directory default-image-registry)}"
if ! _rt exec "$NODE_NAME" crictl img --no-trunc 2>/dev/null | grep -q "${REGISTRY}/central-db:${MAIN_IMAGE_TAG}"; then
    echo "=== Pulling central-db base image ==="
    goarch="$(go env GOARCH)"
    _rt exec "$NODE_NAME" ctr -n k8s.io images pull --platform "linux/${goarch}" \
        "${REGISTRY}/central-db:${MAIN_IMAGE_TAG}" 2>&1 | tail -1
fi

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
if ! kubectl -n stackrox get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null | grep -q "1"; then
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
        --set clusterName=local \
        --set centralEndpoint=central.stackrox.svc:443 \
        --set image.main.fullRef="${SENSOR_IMAGE}" \
        --set imagePullSecrets.allowNone=true \
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

exec skaffold dev -f dev-tools/skaffold.yaml --cleanup=false --kube-context "kind-${CLUSTER_NAME}"
