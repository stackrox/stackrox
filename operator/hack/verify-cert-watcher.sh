#!/bin/bash
# Verifies that the cert-watcher sidecar container of the given deployment(s) started up correctly
# and is watching for TLS certificate changes, by polling its logs for the "cert-watcher: watching"
# message.
#
# Usage: verify-cert-watcher.sh <deployment-name> [<deployment-name> ...]
#
# Requires the NAMESPACE env var to be set (kuttl sets this automatically for TestStep commands).

set -eu -o pipefail

NAMESPACE=${NAMESPACE:-}
if [[ -z "$NAMESPACE" ]]; then
    echo >&2 "NAMESPACE must be set."
    exit 1
fi

if [[ $# -lt 1 ]]; then
    echo >&2 "Usage: $0 <deployment-name> [<deployment-name> ...]"
    exit 1
fi

verify_cert_watcher() {
    local deployment="$1"
    echo "Verifying cert-watcher sidecar is running in ${NAMESPACE}/${deployment}..."
    local retries=12
    local i
    for ((i = 1; i <= retries; i++)); do
        if retry-kubectl.sh </dev/null -n "$NAMESPACE" logs "deploy/${deployment}" -c cert-watcher 2>/dev/null \
                | grep -q "cert-watcher: watching"; then
            echo "cert-watcher is running in ${NAMESPACE}/${deployment}"
            return 0
        fi
        sleep 5
    done
    echo >&2 "ERROR: cert-watcher sidecar did not start correctly in ${NAMESPACE}/${deployment}"
    retry-kubectl.sh </dev/null -n "$NAMESPACE" logs "deploy/${deployment}" -c cert-watcher || true
    return 1
}

ret=0
for deployment in "$@"; do
    verify_cert_watcher "$deployment" || ret=1
done
exit "$ret"
