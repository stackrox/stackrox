#!/bin/bash
set -euo pipefail

# Check that ACS Central deployment is available
echo "Checking ACS Central deployment..."

NAMESPACE="${ACS_NAMESPACE:-stackrox}"

# Check if Central deployment exists and is available
if ! oc get deployment -n "${NAMESPACE}" central &>/dev/null; then
  echo "ERROR: ACS Central deployment not found in namespace ${NAMESPACE}"
  exit 1
fi

# Check deployment status
if ! oc wait --for=condition=Available --timeout=60s deployment/central -n "${NAMESPACE}"; then
  echo "ERROR: ACS Central deployment is not available"
  oc get deployment central -n "${NAMESPACE}" -o yaml
  exit 1
fi

echo "✓ ACS Central is deployed and available"
