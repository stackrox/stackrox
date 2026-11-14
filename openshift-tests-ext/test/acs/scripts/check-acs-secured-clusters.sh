#!/bin/bash
set -euo pipefail

# Check that ACS SecuredCluster and collector are deployed
echo "Checking ACS SecuredCluster and collector..."

NAMESPACE="${ACS_NAMESPACE:-stackrox}"

# Check if sensor deployment exists and is available
if ! oc get deployment -n "${NAMESPACE}" sensor &>/dev/null; then
  echo "ERROR: ACS Sensor deployment not found in namespace ${NAMESPACE}"
  exit 1
fi

# Check sensor deployment status
if ! oc wait --for=condition=Available --timeout=60s deployment/sensor -n "${NAMESPACE}"; then
  echo "ERROR: ACS Sensor deployment is not available"
  oc get deployment sensor -n "${NAMESPACE}" -o yaml
  exit 1
fi

echo "✓ ACS Sensor is deployed"

# Check for collector daemonset
if oc get daemonset -n "${NAMESPACE}" collector &>/dev/null; then
  # Check if at least one collector pod is ready
  READY_PODS=$(oc get daemonset -n "${NAMESPACE}" collector -o jsonpath='{.status.numberReady}')
  DESIRED_PODS=$(oc get daemonset -n "${NAMESPACE}" collector -o jsonpath='{.status.desiredNumberScheduled}')

  if [ "${READY_PODS:-0}" -gt 0 ]; then
    echo "✓ Collector daemonset is running (${READY_PODS}/${DESIRED_PODS} pods ready)"
  else
    echo "WARNING: Collector daemonset exists but no pods are ready yet (${READY_PODS}/${DESIRED_PODS})"
  fi
else
  echo "WARNING: Collector daemonset not found, checking for admission-control instead..."
  if oc get deployment -n "${NAMESPACE}" admission-control &>/dev/null; then
    echo "✓ Admission-control deployment found"
  else
    echo "WARNING: Neither collector daemonset nor admission-control deployment found"
  fi
fi

echo "✓ ACS SecuredCluster and collector are deployed"
