#!/usr/bin/env bash
# Run e2e acceptance tests against a real ACS + Compliance Operator cluster.
#
# USAGE:
#   hack/run-e2e.sh                     # run all e2e tests
#   hack/run-e2e.sh -run TestIMP_ACC_003  # run a specific test
#
# Required environment:
#   ROX_ENDPOINT          ACS Central URL (bare hostname OK)
#   ROX_ADMIN_PASSWORD    Basic auth password  (or ROX_API_TOKEN for token auth)
#
# Optional:
#   CO_NAMESPACE          CO namespace (default: openshift-compliance)
#   E2E_KEEP_CONFIGS      Set to "1" to skip cleanup of created scan configs
#
# The script builds the importer, then runs `go test -tags e2e` against e2e/.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

# Validate prerequisites.
if [[ -z "${ROX_ENDPOINT:-}" ]]; then
    echo "ERROR: ROX_ENDPOINT not set" >&2
    echo "  export ROX_ENDPOINT=central-stackrox.apps.mycluster.example.com" >&2
    exit 1
fi

if [[ -z "${ROX_API_TOKEN:-}" ]] && [[ -z "${ROX_ADMIN_PASSWORD:-}" ]]; then
    echo "ERROR: neither ROX_API_TOKEN nor ROX_ADMIN_PASSWORD is set" >&2
    echo "  export ROX_API_TOKEN=<token>   # for token auth" >&2
    echo "  export ROX_ADMIN_PASSWORD=<pw> # for basic auth" >&2
    exit 1
fi

command -v kubectl >/dev/null 2>&1 || {
    echo "ERROR: kubectl not found in PATH" >&2
    exit 1
}

echo "=== E2E Test Configuration ==="
echo "  ROX_ENDPOINT:      ${ROX_ENDPOINT}"
echo "  Auth mode:         $(if [[ -n "${ROX_API_TOKEN:-}" ]]; then echo "token"; else echo "basic"; fi)"
echo "  CO_NAMESPACE:      ${CO_NAMESPACE:-openshift-compliance}"
echo "  E2E_KEEP_CONFIGS:  ${E2E_KEEP_CONFIGS:-0}"
echo ""

# Run tests. Pass through any extra args (e.g. -run, -v).
exec go test -tags e2e -v -count=1 -timeout 5m ./e2e/ "$@"
