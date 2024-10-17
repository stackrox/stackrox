#!/usr/bin/env bash
set -euo pipefail

# Asserts that scanner v2 is running and ready in the supplied namespace

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"

verify_scannerV2_deployed_and_ready() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: verify_scannerV2_deployed_and_ready <namespace>"
    fi
    local namespace=${1:-stackrox}
    info "Waiting for Scanner V2 deployment to appear in namespace ${namespace}..."
    wait_for_object_to_appear "$namespace" deploy/scanner-db 600
    wait_for_object_to_appear "$namespace" deploy/scanner 300
    info "** Scanner V2 is deployed in namespace ${namespace}"
    kubectl -n "${namespace}" wait --for=condition=ready pod -l app=scanner --timeout 5m
}

verify_scannerV2_deployed_and_ready "$@"
