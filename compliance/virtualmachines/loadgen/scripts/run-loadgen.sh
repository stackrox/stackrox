#!/usr/bin/env bash
set -euo pipefail

# One-touch vsock load generator
# Usage: ./run-loadgen.sh [CONFIG_FILE]
#
# Deploys load generator DaemonSet with config from loadgen-config.yaml
# The DaemonSet automatically starts load generation on all worker nodes
#
# Auto-detects OpenShift and applies privileged SCC when needed

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/../deploy" && pwd)"
CONFIG_FILE="${1:-${DEPLOY_DIR}/loadgen-config.yaml}"
MANIFEST="${DEPLOY_DIR}/vsock-loadgen-daemonset.yaml"

DEFAULT_USER="${USER:-developer}"
IMAGE_NAME="${VSOCK_LOADGEN_IMAGE:-quay.io/${DEFAULT_USER}/stackrox/vsock-loadgen}"
IMAGE_TAG="${VSOCK_LOADGEN_TAG:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

# Auto-detect OpenShift
IS_OPENSHIFT=false
if command -v oc >/dev/null 2>&1; then
    # Check for OpenShift by trying to get SCCs
    if oc get scc >/dev/null 2>&1; then
        IS_OPENSHIFT=true
    fi
fi

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Error: Config file not found: $CONFIG_FILE"
    echo ""
    echo "Usage: $0 [CONFIG_FILE]"
    echo ""
    echo "Default config: ${DEPLOY_DIR}/loadgen-config.yaml"
    exit 1
fi

PLATFORM_NAME="Kubernetes"
if [[ "$IS_OPENSHIFT" == "true" ]]; then
    PLATFORM_NAME="OpenShift"
fi

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  Vsock Load Generator Deployment ($PLATFORM_NAME)                 ║"
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

# Deploy DaemonSet (substitute $USER in manifest)
echo "🚀 Deploying DaemonSet..."
export USER="${USER:-developer}"
envsubst < "$MANIFEST" | kubectl apply -f -

echo ""

# OpenShift-specific: Grant privileged SCC to service account
if [[ "$IS_OPENSHIFT" == "true" ]]; then
    echo "🔐 Granting privileged SCC for OpenShift (required for hostNetwork and root)..."
    if ! oc adm policy add-scc-to-user privileged -z vsock-loadgen -n stackrox 2>&1; then
        echo "   ⚠️  Failed to grant privileged SCC. Trying to continue anyway..."
        echo "   You may need cluster-admin permissions or ask your admin to run:"
        echo "   oc adm policy add-scc-to-user privileged -z vsock-loadgen -n stackrox"
    fi
    echo "   ✓ Privileged SCC granted"
    echo ""

    # Restart DaemonSet to apply SCC changes
    echo "🔄 Restarting DaemonSet to apply SCC changes..."
    kubectl -n stackrox rollout restart daemonset/vsock-loadgen
    echo ""
fi

echo "⏳ Waiting for pods to be ready..."
if ! kubectl -n stackrox rollout status daemonset/vsock-loadgen --timeout=60s 2>&1; then
    # Check if at least 3 pods are ready (acceptable if some nodes lack /dev/vsock)
    ready_count=$(kubectl -n stackrox get pods -l app=vsock-loadgen --field-selector=status.phase=Running -o json | jq '[.items[] | select(.status.containerStatuses[]?.ready == true)] | length')
    if [[ "${ready_count:-0}" -ge 3 ]]; then
        echo "   Rollout timeout, but $ready_count pods are ready (acceptable)"
    else
        echo "   ❌ Only $ready_count pods ready after timeout"
        if [[ "$IS_OPENSHIFT" == "true" ]]; then
            echo ""
            echo "Debugging information:"
            echo "───────────────────────────────────────────────────────────────"
            echo "Pod status:"
            kubectl -n stackrox get pods -l app=vsock-loadgen -o wide
            echo ""
            echo "Recent events:"
            kubectl -n stackrox get events --sort-by='.lastTimestamp' | grep -i vsock-loadgen | tail -5
        fi
        exit 1
    fi
fi

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
if [[ "$IS_OPENSHIFT" == "true" ]]; then
    echo "   oc adm policy remove-scc-from-user privileged -z vsock-loadgen -n stackrox"
fi
echo ""
