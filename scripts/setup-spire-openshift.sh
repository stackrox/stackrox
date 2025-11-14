#!/usr/bin/env bash
# Setup SPIRE on OpenShift 4.19 via CLI

set -eo pipefail

echo "=========================================="
echo "SPIRE Setup for OpenShift 4.19"
echo "=========================================="
echo ""

# Step 1: Get operator details
echo "Step 1: Checking operator availability..."
oc get packagemanifest openshift-zero-trust-workload-identity-manager -n openshift-marketplace -o yaml | grep -A 10 "channels:"

echo ""
echo "Step 2: Creating operator subscription..."

# Create namespace and subscription
cat <<EOF | oc apply -f -
---
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-zero-trust-workload-identity-manager
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: openshift-zero-trust-workload-identity-manager
  namespace: openshift-zero-trust-workload-identity-manager
spec:
  targetNamespaces:
  - openshift-zero-trust-workload-identity-manager
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openshift-zero-trust-workload-identity-manager
  namespace: openshift-zero-trust-workload-identity-manager
spec:
  channel: stable
  name: openshift-zero-trust-workload-identity-manager
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF

echo ""
echo "Waiting for operator to install (this may take 2-3 minutes)..."
sleep 10

# Wait for CSV to be ready
for i in {1..30}; do
  CSV_STATUS=$(oc get csv -n openshift-zero-trust-workload-identity-manager -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "Waiting")
  if [ "$CSV_STATUS" == "Succeeded" ]; then
    echo "✅ Operator installed successfully!"
    break
  fi
  echo "  Status: $CSV_STATUS (attempt $i/30)"
  sleep 10
done

if [ "$CSV_STATUS" != "Succeeded" ]; then
  echo "⚠️  Operator installation taking longer than expected. Check status with:"
  echo "   oc get csv -n openshift-zero-trust-workload-identity-manager"
  exit 1
fi

echo ""
echo "Step 3: Deploying SPIRE Server, Agent, and CSI Driver..."

# Deploy SPIRE components
cat <<EOF | oc apply -f -
---
apiVersion: operator.openshift.io/v1alpha1
kind: SpireServer
metadata:
  name: cluster
  namespace: openshift-zero-trust-workload-identity-manager
spec:
  trustDomain: stackrox.local
  clusterName: demo-cluster
---
apiVersion: operator.openshift.io/v1alpha1
kind: SpireAgent
metadata:
  name: cluster
  namespace: openshift-zero-trust-workload-identity-manager
spec:
  trustDomain: stackrox.local
  clusterName: demo-cluster
  nodeAttestor:
    k8sPSATEnabled: "true"
  workloadAttestors:
    k8sEnabled: "true"
---
apiVersion: operator.openshift.io/v1alpha1
kind: SpiffeCSIDriver
metadata:
  name: cluster
  namespace: openshift-zero-trust-workload-identity-manager
spec:
  agentSocketPath: '/run/spire/agent-sockets/spire-agent.sock'
EOF

echo ""
echo "Waiting for SPIRE components to be ready..."
sleep 5

# Wait for SPIRE Server
echo "  Waiting for SPIRE Server..."
oc wait --for=condition=ready pod -l app=spire-server -n openshift-zero-trust-workload-identity-manager --timeout=300s || true

# Check SPIRE Agent DaemonSet
echo "  Checking SPIRE Agent DaemonSet..."
for i in {1..30}; do
  DESIRED=$(oc get daemonset spire-agent -n openshift-zero-trust-workload-identity-manager -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
  READY=$(oc get daemonset spire-agent -n openshift-zero-trust-workload-identity-manager -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
  if [ "$DESIRED" != "0" ] && [ "$DESIRED" == "$READY" ]; then
    echo "  ✅ SPIRE Agent DaemonSet ready: $READY/$DESIRED"
    break
  fi
  echo "    SPIRE Agent: $READY/$DESIRED ready (attempt $i/30)"
  sleep 10
done

echo ""
echo "Step 4: Verifying SPIRE installation..."
echo ""

# Show pods
echo "SPIRE Pods:"
oc get pods -n openshift-zero-trust-workload-identity-manager

echo ""
echo "Testing SPIRE Server health:"
SPIRE_SERVER_POD=$(oc get pods -n openshift-zero-trust-workload-identity-manager -l app=spire-server -o name | head -1)
if [ -n "$SPIRE_SERVER_POD" ]; then
  oc exec -n openshift-zero-trust-workload-identity-manager "$SPIRE_SERVER_POD" -- \
    /opt/spire/bin/spire-server healthcheck || echo "Health check command not available"
fi

echo ""
echo "=========================================="
echo "✅ SPIRE Installation Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Push your branch to GitHub: git push origin vb/spire-spiffe-hackathon"
echo "2. Wait for CI to build images"
echo "3. Deploy StackRox with the new images"
echo "4. Run: ./scripts/demo-spire.sh"
echo ""
