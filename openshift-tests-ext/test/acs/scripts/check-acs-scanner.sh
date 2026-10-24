#!/bin/bash
set -euo pipefail

# Check that ACS Scanner deployment is available
echo "Checking ACS Scanner deployment..."

NAMESPACE="${ACS_NAMESPACE:-stackrox}"

# Check if Scanner deployment exists and is available
if ! oc get deployment -n "${NAMESPACE}" scanner &>/dev/null; then
  echo "ERROR: ACS Scanner deployment not found in namespace ${NAMESPACE}"
  exit 1
fi

# Check deployment status
if ! oc wait --for=condition=Available --timeout=60s deployment/scanner -n "${NAMESPACE}"; then
  echo "ERROR: ACS Scanner deployment is not available"
  oc get deployment scanner -n "${NAMESPACE}" -o yaml
  exit 1
fi

echo "✓ ACS Scanner is deployed and available"
