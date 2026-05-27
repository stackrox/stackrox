#!/usr/bin/env bash
set -euo pipefail

# Wait for critical OpenShift operators to become available on s390x
# Addresses: ROX-21457 - OPERATOR_DEGRADED failures
#
# This script explicitly waits for critical cluster operators to become available
# before proceeding with tests. This addresses the race condition on s390x where
# operators attempt to connect to the API server before it's ready.

OPERATOR_TIMEOUT="${OPERATOR_TIMEOUT:-20m}"

echo "Waiting for critical OpenShift operators to become available..."
echo "Timeout per operator: $OPERATOR_TIMEOUT"

# Critical operators that must be available before running tests
# These were identified as commonly failing on s390x (ROX-21457)
CRITICAL_OPERATORS=(
    "kube-apiserver"
    "kube-controller-manager"
    "authentication"
    "console"
    "storage"
    "monitoring"
    "machine-api"
    "ingress"
    "image-registry"
)

echo "Checking ${#CRITICAL_OPERATORS[@]} critical operators..."

failed_operators=()

for operator in "${CRITICAL_OPERATORS[@]}"; do
    echo ""
    echo "[$operator] Waiting for availability..."

    if oc wait --for=condition=Available \
        --timeout="$OPERATOR_TIMEOUT" \
        "clusteroperator/$operator" 2>&1; then

        echo "[$operator] ✓ Available"
    else
        echo "[$operator] ✗ FAILED to become available within $OPERATOR_TIMEOUT"
        failed_operators+=("$operator")

        echo "[$operator] Dumping operator status:"
        oc get clusteroperator "$operator" -o yaml 2>&1 || true

        echo "[$operator] Checking for degraded conditions:"
        oc get clusteroperator "$operator" -o jsonpath='{.status.conditions[?(@.type=="Degraded")]}' 2>&1 || true
        echo ""
    fi
done

echo ""
echo "========================================"
echo "Operator verification complete"
echo "========================================"

if [ ${#failed_operators[@]} -eq 0 ]; then
    echo "✓ All ${#CRITICAL_OPERATORS[@]} critical operators are healthy!"
    exit 0
else
    echo "✗ ${#failed_operators[@]} operator(s) failed to become available:"
    for operator in "${failed_operators[@]}"; do
        echo "  - $operator"
    done

    echo ""
    echo "Dumping all cluster operator statuses:"
    oc get clusteroperators -o wide 2>&1 || true

    exit 1
fi
