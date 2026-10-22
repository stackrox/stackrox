#!/bin/bash
set -euo pipefail

# Check that ACS init-bundle and TLS secrets are configured
echo "Checking ACS init-bundle and TLS secrets..."

NAMESPACE="${ACS_NAMESPACE:-stackrox}"

# Check for required TLS secrets
REQUIRED_SECRETS=(
  "central-tls"
  "scanner-tls"
  "scanner-db-tls"
)

for secret in "${REQUIRED_SECRETS[@]}"; do
  if ! oc get secret -n "${NAMESPACE}" "${secret}" &>/dev/null; then
    echo "ERROR: Required TLS secret '${secret}' not found in namespace ${NAMESPACE}"
    exit 1
  fi
  echo "✓ Found TLS secret: ${secret}"
done

# Check for init-bundle secret (collector-tls or similar)
if oc get secret -n "${NAMESPACE}" collector-tls &>/dev/null; then
  echo "✓ Found init-bundle secret: collector-tls"
elif oc get secret -n "${NAMESPACE}" admission-control-tls &>/dev/null; then
  echo "✓ Found init-bundle secret: admission-control-tls"
else
  echo "WARNING: No collector or admission-control init-bundle secrets found, but may not be required for all deployments"
fi

echo "✓ ACS init-bundle and TLS secrets are configured"
