#!/usr/bin/env bash
set -euo pipefail

# Wait for cluster to be registered in Central
#
# Usage:
#   cluster-registration-wait.sh <cluster-name> [<namespace>] [<max-wait-seconds>]
#
# Checks that a cluster with the given name is registered in Central's API.
# This is needed because sensor deployment can succeed (pod running) but the
# cluster registration in Central can take additional time.

CLUSTER_NAME="${1}"
CENTRAL_NAMESPACE="${2:-stackrox}"
MAX_WAIT="${3:-120}"

# Use API_ENDPOINT if set, otherwise fall back to localhost:8000
CENTRAL_API="${API_ENDPOINT:-localhost:8000}"

echo "Waiting for cluster '${CLUSTER_NAME}' to be registered in Central (max ${MAX_WAIT}s)..."
echo "Using Central API endpoint: ${CENTRAL_API}"

start_time="$(date '+%s')"

while true; do
    # Query Central API for clusters
    if response=$(curl -k -s -u "admin:${ROX_ADMIN_PASSWORD}" \
        "https://${CENTRAL_API}/v1/clusters" 2>/dev/null); then

        # Check if our cluster name exists
        if echo "$response" | jq -e ".clusters[] | select(.name == \"${CLUSTER_NAME}\")" >/dev/null 2>&1; then
            cluster_id=$(echo "$response" | jq -r ".clusters[] | select(.name == \"${CLUSTER_NAME}\") | .id")
            echo "✓ Cluster '${CLUSTER_NAME}' is registered with ID: ${cluster_id}"
            break
        fi

        echo "Cluster '${CLUSTER_NAME}' not yet registered..."
    else
        echo "Failed to query Central API, retrying..."
    fi

    elapsed=$(($(date '+%s') - start_time))
    if (( elapsed > MAX_WAIT )); then
        echo "ERROR: Timed out waiting for cluster '${CLUSTER_NAME}' to register after ${MAX_WAIT}s"
        echo "Registered clusters:"
        curl -k -s -u "admin:${ROX_ADMIN_PASSWORD}" "https://${CENTRAL_API}/v1/clusters" 2>/dev/null | jq -r '.clusters[] | .name' || echo "Failed to get cluster list"
        exit 1
    fi

    sleep 5
done

echo "Cluster registration verified successfully"
