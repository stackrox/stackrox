#!/usr/bin/env bash
set -euo pipefail

# One-touch vsock load generator
# Usage: ./run-loadgen.sh [CONFIG_FILE]
#
# Deploys load generator DaemonSet with config from loadgen-config.yaml
# The DaemonSet automatically starts load generation on all worker nodes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/../deploy" && pwd)"
CONFIG_FILE="${1:-${DEPLOY_DIR}/loadgen-config.yaml}"
MANIFEST="${DEPLOY_DIR}/vsock-loadgen-daemonset.yaml"

IMAGE_NAME="${VSOCK_LOADGEN_IMAGE:-quay.io/gualvare/stackrox/vsock-loadgen}"
IMAGE_TAG="${VSOCK_LOADGEN_TAG:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Error: Config file not found: $CONFIG_FILE"
    echo ""
    echo "Usage: $0 [CONFIG_FILE]"
    echo ""
    echo "Default config: ${DEPLOY_DIR}/loadgen-config.yaml"
    exit 1
fi

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  Vsock Load Generator Deployment                             ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "📝 Config file: $CONFIG_FILE"
echo "📦 Image: $FULL_IMAGE"
echo ""

# Show config
echo "Configuration:"
echo "─────────────────────────────────────────────────────────────────"
cat "$CONFIG_FILE" | grep -v "^#" | grep -v "^$"
echo "─────────────────────────────────────────────────────────────────"
echo ""

# Create or update ConfigMap
echo "🔧 Creating ConfigMap..."
kubectl -n stackrox create configmap vsock-loadgen-config \
    --from-file=config.yaml="$CONFIG_FILE" \
    --dry-run=client -o yaml | kubectl apply -f -

echo "   ✓ ConfigMap created/updated"
echo ""

# Deploy DaemonSet
echo "🚀 Deploying DaemonSet..."
kubectl apply -f "$MANIFEST"

echo ""
echo "⏳ Waiting for pods to be ready..."
kubectl -n stackrox rollout status daemonset/vsock-loadgen --timeout=60s

echo ""
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  ✅ Load generator deployed successfully!                     ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "📊 Load generator pods:"
kubectl -n stackrox get pods -l app=vsock-loadgen -o wide

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "💡 TIP: To view logs from a pod:"
echo "   kubectl -n stackrox logs -f <pod-name>"
echo ""
echo "   Or use: kubectl -n stackrox logs -f -l app=vsock-loadgen --max-log-requests=10"
echo ""
echo "📈 To monitor metrics:"
echo "   cd /path/to/ROX-XXXXX-fake-vsock-load"
echo "   ./start-monitoring.sh"
echo "   # View: http://localhost:3001/d/vsock-relay-load"
echo ""
echo "🛑 To stop and cleanup:"
echo "   kubectl -n stackrox delete daemonset vsock-loadgen"
echo ""
